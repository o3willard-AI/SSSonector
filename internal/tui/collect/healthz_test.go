package collect

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestGetHealthz_ValidResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if r := recover(); r != nil {
			t.Errorf("handler panic: %v", r)
		}
		// Exact wire shape from internal/monitor handleHealthz.
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","mode":"server","tunnel_state":"up","uptime_seconds":8040}` + "\n"))
	}))
	defer srv.Close()

	h, err := GetHealthzTimeout(context.Background(), srv.Client(), srv.URL, 2*time.Second)
	if err != nil {
		t.Fatalf("GetHealthz: %v", err)
	}
	if h.Status != "ok" {
		t.Errorf("status: %q", h.Status)
	}
	if h.Mode != "server" {
		t.Errorf("mode: %q", h.Mode)
	}
	if h.TunnelState != "up" {
		t.Errorf("tunnel_state: %q", h.TunnelState)
	}
	if h.UptimeSeconds != 8040 {
		t.Errorf("uptime_seconds: %d", h.UptimeSeconds)
	}
}

func TestGetHealthz_MalformedJSON(t *testing.T) {
	cases := map[string]string{
		"not json":    "this is not json",
		"truncated":   `{"status":"ok","mode":`,
		"wrong types": `{"status":1,"mode":"server","tunnel_state":"up","uptime_seconds":"x"}`,
		"array":       `[1,2,3]`,
		"empty body":  "",
		"html error":  "<html>502</html>",
	}
	for name, body := range cases {
		body := body
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(body))
			}))
			defer srv.Close()

			h, err := GetHealthzTimeout(context.Background(), srv.Client(), srv.URL, 2*time.Second)
			if err == nil {
				t.Fatalf("body %q: want error, got struct %+v", body, h)
			}
			// json.Unmarshal guarantees a valid partial decode; the caller
			// contract is: on error, no field is trustworthy. Assert the
			// documented zero-value contract fields (status, uptime) are
			// untouched rather than requiring the whole struct be zero.
			if h.Status != "" || h.UptimeSeconds != 0 {
				t.Errorf("malformed JSON must not decode trusted fields, got %+v", h)
			}
			if !strings.Contains(err.Error(), "malformed JSON") {
				t.Errorf("want malformed-JSON error, got: %v", err)
			}
		})
	}
}

func TestGetHealthz_Timeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(3 * time.Second):
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()

	start := time.Now()
	h, err := GetHealthzTimeout(context.Background(), srv.Client(), srv.URL, 150*time.Millisecond)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("want timeout error, got struct %+v", h)
	}
	if h != (Healthz{}) {
		t.Errorf("timeout must return zero struct, got %+v", h)
	}
	if !errors.Is(err, ErrHealthzTimeout) {
		t.Errorf("want ErrHealthzTimeout, got: %v", err)
	}
	if elapsed > time.Second {
		t.Errorf("timeout did not return promptly: %v", elapsed)
	}
}

func TestGetHealthz_Non2xx(t *testing.T) {
	for _, code := range []int{404, 500, 503} {
		code := code
		t.Run(http.StatusText(code), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "nope", code)
			}))
			defer srv.Close()

			h, err := GetHealthzTimeout(context.Background(), srv.Client(), srv.URL, 2*time.Second)
			if err == nil {
				t.Fatalf("status %d: want error, got struct %+v", code, h)
			}
			if h != (Healthz{}) {
				t.Errorf("non-2xx must return zero struct, got %+v", h)
			}
			if !strings.Contains(err.Error(), "unexpected status") {
				t.Errorf("want unexpected-status error, got: %v", err)
			}
		})
	}
}

func TestGetHealthz_ConnectionRefused(t *testing.T) {
	// Reserve then close a port to get a reliable connection failure.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	url := srv.URL
	srv.Close()

	_, err := GetHealthzTimeout(context.Background(), srv.Client(), url, 2*time.Second)
	if err == nil {
		t.Fatal("want connection error, got nil")
	}
	if errors.Is(err, ErrHealthzTimeout) {
		t.Errorf("connection refused must not be reported as timeout: %v", err)
	}
}
