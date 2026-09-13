package collect

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

// SourceStatus is the per-tick status of one data source.
type SourceStatus int

const (
	// StatusOK means the source returned fresh data this tick.
	StatusOK SourceStatus = iota
	// StatusError means the source failed this tick. The value is NOT
	// carried forward from a previous tick — it is meaningless (zero) and
	// the previous value must not be shown (NEVER STALE invariant).
	StatusError
	// StatusAbsent means the source has no data by design (e.g.
	// prometheus disabled in config). Not an error.
	StatusAbsent
)

// String renders the status for diagnostics and views.
func (s SourceStatus) String() string {
	switch s {
	case StatusOK:
		return "ok"
	case StatusError:
		return "error"
	default:
		return "absent"
	}
}

// SourceOf pairs a per-tick value with its status. A non-OK source carries
// no value: Get returns the zero value and ok=false (never stale).
type SourceOf[T any] struct {
	status SourceStatus
	value  T
	err    error
}

// okSource wraps a fresh value.
func okSource[T any](v T) SourceOf[T] { return SourceOf[T]{status: StatusOK, value: v} }

// errSource marks a failed fetch; value is dropped (never stale).
func errSource[T any](err error) SourceOf[T] { return SourceOf[T]{status: StatusError, err: err} }

// absentSource marks a source with no data by design.
func absentSource[T any]() SourceOf[T] { return SourceOf[T]{status: StatusAbsent} }

// Status returns the per-tick status.
func (s SourceOf[T]) Status() SourceStatus { return s.status }

// Err returns the error when Status()==StatusError, else nil.
func (s SourceOf[T]) Err() error { return s.err }

// Get returns the value and whether it is fresh (StatusOK). Absent and
// errored sources return the zero value — callers must check Status.
func (s SourceOf[T]) Get() (T, bool) { return s.value, s.status == StatusOK }

// CertSource carries per-instance certificate status (issuer, expiry,
// rotation flag) read from local files — WI 1.4.
type CertSource = SourceOf[CertInfo]

// InstanceSnapshot is everything the dashboard shows for one instance,
// as of a single tick. Each source is independently ok/error/absent.
type InstanceSnapshot struct {
	// Name is the instance name (from discovery).
	Name string
	// Unit is the systemd unit name ("" when degraded).
	Unit string
	// Running is the systemd main-process state ("" when unknown).
	ActiveState string
	SubState    string
	MainPID     int

	// Healthz is the /healthz result.
	Healthz SourceOf[Healthz]
	// Metrics is the parsed /metrics snapshot.
	Metrics SourceOf[Snapshot]
	// PrometheusAddr is the base URL metrics/healthz were fetched from,
	// empty when prometheus is disabled or config resolution failed.
	PrometheusAddr string

	// Cert is the WI 1.4 seam (never populated by this WI).
	Cert CertSource

	// ConfigErr carries the config-resolution error when the instance's
	// config could not be loaded this tick (both sources then error).
	ConfigErr error
}

// Poller assembles per-instance dashboard snapshots on each tick. Single
// tick logic lives in PollOnce (testable with a fake clock, no sleeps);
// Run is the thin ticker wrapper.
type Poller struct {
	Systemd SystemdCollector
	Paths   ConfigPaths
	Client  *http.Client
}

// NewPoller builds a Poller with the given collectors and HTTP client.
func NewPoller(sys SystemdCollector, paths ConfigPaths, client *http.Client) *Poller {
	return &Poller{Systemd: sys, Paths: paths, Client: client}
}

// TickResult is the full dashboard state for one tick.
type TickResult struct {
	// Instances maps instance name to its per-tick snapshot.
	Instances map[string]*InstanceSnapshot
	// DiscoverErr is set when instance discovery itself failed this tick
	// (Instances is then empty — never stale).
	DiscoverErr error
}

// PollOnce runs one tick: discover instances, then fetch every source for
// each instance with independent per-source status. Sources that fail are
// marked error with their value dropped — the caller never sees a previous
// tick's data (NEVER STALE).
func (p *Poller) PollOnce(ctx context.Context) TickResult {
	res := TickResult{Instances: map[string]*InstanceSnapshot{}}

	states, err := p.Systemd.Discover()
	if err != nil {
		res.DiscoverErr = err
		return res
	}

	for _, st := range states {
		snap := &InstanceSnapshot{
			Name:        st.Name,
			Unit:        st.Unit,
			ActiveState: st.ActiveState,
			SubState:    st.SubState,
			MainPID:     st.MainPID,
		}
		p.pollInstance(ctx, snap)
		res.Instances[st.Name] = snap
	}
	return res
}

// pollInstance fills one instance's sources for this tick.
func (p *Poller) pollInstance(ctx context.Context, snap *InstanceSnapshot) {
	ic, err := ResolveInstanceConfig(p.Paths, snap.Name)
	if err != nil {
		snap.ConfigErr = err
		snap.Healthz = errSource[Healthz](fmt.Errorf("config: %w", err))
		snap.Metrics = errSource[Snapshot](fmt.Errorf("config: %w", err))
		snap.Cert = errSource[CertInfo](fmt.Errorf("config: %w", err))
		return
	}

	// Cert is read from local files (no HTTP) and is independent of the
	// Prometheus endpoint.
	snap.Cert = fetchCert(ic)

	if !ic.Prometheus.Enabled {
		// Prometheus disabled in config: metrics ABSENT by design, not an
		// error. With prometheus disabled there is no metrics listener at
		// all, so healthz is also unreachable by this transport; it is
		// marked error (the daemon may be running but has no HTTP surface).
		snap.Metrics = absentSource[Snapshot]()
		snap.Healthz = errSource[Healthz](fmt.Errorf("prometheus disabled in config: no HTTP endpoint"))
		return
	}

	addr := fmt.Sprintf("http://127.0.0.1:%d", ic.Prometheus.Port)
	snap.PrometheusAddr = addr

	// healthz and metrics are independent: one failing does not clobber
	// the other.
	snap.Healthz = fetchHealthz(ctx, p.Client, addr)
	snap.Metrics = fetchMetrics(ctx, p.Client, addr, ic.Prometheus.Path)
}

// fetchCert reads the instance's certificate status from local files.
func fetchCert(ic InstanceConfig) SourceOf[CertInfo] {
	info, err := ReadCertInfo(ic.CertPaths, ic.CertRotationInterval, time.Time{})
	if err != nil {
		return errSource[CertInfo](err)
	}
	return okSource(info)
}

// fetchHealthz GETs <addr>/healthz with per-tick status.
func fetchHealthz(ctx context.Context, client *http.Client, addr string) SourceOf[Healthz] {
	h, err := GetHealthz(ctx, client, addr)
	if err != nil {
		return errSource[Healthz](err)
	}
	return okSource(h)
}

// fetchMetrics GETs <addr><path>, parses the exposition, and builds the
// typed snapshot — reusing ParseExposition + NewSnapshot (no duplicated
// logic).
func fetchMetrics(ctx context.Context, client *http.Client, addr, path string) SourceOf[Snapshot] {
	if path == "" {
		path = "/metrics"
	}
	body, err := httpGet(ctx, client, addr+path)
	if err != nil {
		return errSource[Snapshot](err)
	}
	fams, err := ParseExposition(body)
	if err != nil {
		return errSource[Snapshot](fmt.Errorf("parse %s: %w", path, err))
	}
	snap, err := NewSnapshot(fams)
	if err != nil {
		return errSource[Snapshot](fmt.Errorf("snapshot %s: %w", path, err))
	}
	return okSource(snap)
}

// httpGet performs a bounded GET and returns the body text.
func httpGet(ctx context.Context, client *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("build request %s: %w", url, err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close() //nolint:errcheck // read-only body
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return "", fmt.Errorf("GET %s: unexpected status %d: %s", url, resp.StatusCode, string(b))
	}
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", fmt.Errorf("GET %s: read body: %w", url, err)
	}
	return string(b), nil
}

// Run is the thin tick-loop wrapper (owns the time.Ticker). PollOnce does
// the work; tests drive that directly. This loop is glue only.
func (p *Poller) Run(ctx context.Context, interval time.Duration, sink func(TickResult)) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			sink(p.PollOnce(ctx))
		}
	}
}
