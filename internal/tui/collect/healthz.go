package collect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// healthzPath is the daemon's liveness/readiness endpoint
// (internal/monitor handleHealthz serves it).
const healthzPath = "/healthz"

// Healthz is the typed view of the daemon's /healthz JSON response. All
// four fields are always present in a 200 response (the wire contract), so
// absence is expressed as an error at decode time, not as Opt fields.
type Healthz struct {
	Status        string `json:"status"`
	Mode          string `json:"mode"`
	TunnelState   string `json:"tunnel_state"`
	UptimeSeconds int64  `json:"uptime_seconds"`
}

// ErrHealthzTimeout reports that the /healthz request exceeded the
// caller-supplied timeout.
var ErrHealthzTimeout = errors.New("healthz: request timed out")

// GetHealthz GETs <baseURL>/healthz and decodes the JSON response.
//
// baseURL is supplied by the caller (e.g. "http://127.0.0.1:9443"). The
// request carries the given timeout via context; a timeout returns an
// error wrapping ErrHealthzTimeout and never hangs. Non-2xx status,
// malformed JSON, and connection failures return explicit errors —
// never a partially-decoded struct and never a panic.
func GetHealthz(ctx context.Context, client *http.Client, baseURL string) (Healthz, error) {
	var h Healthz

	base := strings.TrimSuffix(baseURL, "/")
	url := base + healthzPath

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return h, fmt.Errorf("healthz: build request for %s: %w", url, err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		// Distinguish timeouts (deadline exceeded) from other failures.
		if errors.Is(err, context.DeadlineExceeded) {
			return h, fmt.Errorf("healthz: %s: %w", url, ErrHealthzTimeout)
		}
		var netErr interface{ Timeout() bool }
		if errors.As(err, &netErr) && netErr.Timeout() {
			return h, fmt.Errorf("healthz: %s: %w", url, ErrHealthzTimeout)
		}
		return h, fmt.Errorf("healthz: %s: %w", url, err)
	}
	defer resp.Body.Close() //nolint:errcheck // read-only body

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return h, fmt.Errorf("healthz: %s: unexpected status %d: %s",
			url, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return h, fmt.Errorf("healthz: %s: read body: %w", url, err)
	}
	if err := json.Unmarshal(body, &h); err != nil {
		return h, fmt.Errorf("healthz: %s: malformed JSON: %w", url, err)
	}
	return h, nil
}

// GetHealthzTimeout is a convenience wrapper that applies timeout to a
// fresh request context.
func GetHealthzTimeout(ctx context.Context, client *http.Client, baseURL string, timeout time.Duration) (Healthz, error) {
	tctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return GetHealthz(tctx, client, baseURL)
}
