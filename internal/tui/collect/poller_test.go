package collect

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureDaemon is a controllable fake daemon: it serves the exact wire
// fixtures on /healthz and /metrics, and tests can break it mid-test.
type fixtureDaemon struct {
	srv       *httptest.Server
	healthy   bool
	metricsUp bool
	reqs      int
}

const daemonHealthzBody = `{"status":"ok","mode":"server","tunnel_state":"up","uptime_seconds":8040}`

// daemonMetricsBody is the 21-metric live fixture from promparse_test.
const daemonMetricsBody = liveFixture

func newFixtureDaemon(t *testing.T) *fixtureDaemon {
	t.Helper()
	d := &fixtureDaemon{}
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		d.reqs++
		if !d.healthy {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(daemonHealthzBody + "\n"))
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		d.reqs++
		if !d.metricsUp {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte(daemonMetricsBody))
	})
	d.srv = httptest.NewServer(mux)
	d.healthy = true
	d.metricsUp = true
	t.Cleanup(d.srv.Close)
	return d
}

// port extracts the listen port from the httptest URL.
func (d *fixtureDaemon) port(t *testing.T) int {
	t.Helper()
	var p int
	if _, err := fmt.Sscanf(d.srv.URL, "http://127.0.0.1:%d", &p); err != nil {
		t.Fatalf("parse port from %s: %v", d.srv.URL, err)
	}
	return p
}

// writeInstanceConfig points instance <name> at the fixture daemon port.
func writeInstanceConfig(t *testing.T, root, name string, port int, promEnabled bool) {
	t.Helper()
	enabled := "true"
	if !promEnabled {
		enabled = "false"
	}
	yaml := fmt.Sprintf(`metadata:
  schema_version: "2.0.0"
type: server
config:
  mode: server
  monitor:
    enabled: true
    prometheus:
      enabled: %s
      port: %d
      path: /metrics
  auth:
    cert_file: ""
    key_file: ""
    ca_file: ""
`, enabled, port)
	p := filepath.Join(root, "instances", name, "config.yaml")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
}

// newTestPoller builds a poller over a fixture systemd runner and temp
// config root, with the fixture daemon's HTTP client.
func newTestPoller(t *testing.T, d *fixtureDaemon, root, instance string, port int, promEnabled bool) *Poller {
	t.Helper()
	writeInstanceConfig(t, root, instance, port, promEnabled)
	r := newFakeRunner()
	r.outputs["systemctl list-units sssonector@* --all --plain --no-legend"] =
		"sssonector@" + instance + ".service loaded active running SSSonector tunnel (" + instance + ")\n"
	r.outputs["systemctl show sssonector@"+instance+".service -p ActiveState -p SubState -p MainPID"] =
		"ActiveState=active\nSubState=running\nMainPID=8123\n"
	r.outputs["systemctl show sssonector.service -p ActiveState -p SubState -p MainPID"] =
		"ActiveState=not-found\nSubState=dead\nMainPID=0\n"
	return NewPoller(SystemdCollector{Runner: r, Paths: DefaultSystemdPaths()},
		ConfigPaths{ConfigRoot: root}, d.srv.Client())
}

func TestPoller_Success(t *testing.T) {
	d := newFixtureDaemon(t)
	p := newTestPoller(t, d, t.TempDir(), "client-a", d.port(t), true)

	res := p.PollOnce(context.Background())
	if res.DiscoverErr != nil {
		t.Fatalf("discover: %v", res.DiscoverErr)
	}
	snap, ok := res.Instances["client-a"]
	if !ok {
		t.Fatalf("instance missing: %+v", res.Instances)
	}
	if snap.Healthz.Status() != StatusOK {
		t.Fatalf("healthz status: %v err=%v", snap.Healthz.Status(), snap.Healthz.Err())
	}
	h, _ := snap.Healthz.Get()
	if h.TunnelState != "up" || h.Mode != "server" || h.UptimeSeconds != 8040 {
		t.Errorf("healthz values: %+v", h)
	}
	if snap.Metrics.Status() != StatusOK {
		t.Fatalf("metrics status: %v err=%v", snap.Metrics.Status(), snap.Metrics.Err())
	}
	m, _ := snap.Metrics.Get()
	if v, ok := m.NAT.ForwardedPackets.Get(); !ok || v != 1204551 {
		t.Errorf("metrics NAT forwarded: %v ok=%v", v, ok)
	}
	if v, ok := m.Network.BytesIn.Get(); !ok || v != 123456789 {
		t.Errorf("metrics bytes_in: %v ok=%v", v, ok)
	}
	if snap.ActiveState != "active" || snap.MainPID != 8123 {
		t.Errorf("systemd state: %+v", snap)
	}
}

func TestPoller_NeverStale_ErrorThenRecovery(t *testing.T) {
	d := newFixtureDaemon(t)
	p := newTestPoller(t, d, t.TempDir(), "client-a", d.port(t), true)
	ctx := context.Background()

	// Tick 1: all ok.
	res1 := p.PollOnce(ctx)
	s1 := res1.Instances["client-a"]
	if s1.Healthz.Status() != StatusOK || s1.Metrics.Status() != StatusOK {
		t.Fatalf("tick1: healthz=%v metrics=%v", s1.Healthz.Status(), s1.Metrics.Status())
	}

	// Tick 2: daemon stops responding — sources must be ERROR, not the
	// tick-1 values.
	d.healthy = false
	d.metricsUp = false
	res2 := p.PollOnce(ctx)
	s2 := res2.Instances["client-a"]
	if s2.Healthz.Status() != StatusError {
		t.Fatalf("tick2 healthz: want error, got %v", s2.Healthz.Status())
	}
	if s2.Metrics.Status() != StatusError {
		t.Fatalf("tick2 metrics: want error, got %v", s2.Metrics.Status())
	}
	// The defining assertion: no value is carried forward.
	if _, ok := s2.Healthz.Get(); ok {
		t.Errorf("NEVER STALE violated: errored healthz returned a value")
	}
	if _, ok := s2.Metrics.Get(); ok {
		t.Errorf("NEVER STALE violated: errored metrics returned a value")
	}

	// Tick 3: daemon returns — errors clear with fresh data.
	d.healthy = true
	d.metricsUp = true
	res3 := p.PollOnce(ctx)
	s3 := res3.Instances["client-a"]
	if s3.Healthz.Status() != StatusOK || s3.Metrics.Status() != StatusOK {
		t.Fatalf("tick3: healthz=%v metrics=%v", s3.Healthz.Status(), s3.Metrics.Status())
	}
	h, _ := s3.Healthz.Get()
	if h.UptimeSeconds != 8040 {
		t.Errorf("tick3 healthz fresh value: %+v", h)
	}
}

func TestPoller_PrometheusDisabled_MetricsAbsentHealthzStillTried(t *testing.T) {
	d := newFixtureDaemon(t)
	p := newTestPoller(t, d, t.TempDir(), "client-a", d.port(t), false)

	res := p.PollOnce(context.Background())
	s := res.Instances["client-a"]
	if s.Metrics.Status() != StatusAbsent {
		t.Errorf("metrics: want absent, got %v", s.Metrics.Status())
	}
	if _, ok := s.Metrics.Get(); ok {
		t.Error("absent metrics must carry no value")
	}
	// Per the wire contract, with prometheus disabled there is no HTTP
	// listener at all — healthz cannot be served either, so it errors
	// (never stale, never silently empty).
	if s.Healthz.Status() != StatusError {
		t.Errorf("healthz with no listener: want error, got %v", s.Healthz.Status())
	}
}

func TestPoller_SourceIndependence(t *testing.T) {
	t.Run("healthz fails, metrics succeed", func(t *testing.T) {
		d := newFixtureDaemon(t)
		p := newTestPoller(t, d, t.TempDir(), "client-a", d.port(t), true)

		// Break healthz by wrapping the client transport to 503 it — the
		// fixture daemon routes by path, so use a stubbing transport.
		stub := &pathStubTransport{inner: d.srv.Client().Transport, failPath: "/healthz"}
		p.Client = &http.Client{Transport: stub}

		res := p.PollOnce(context.Background())
		s := res.Instances["client-a"]
		if s.Healthz.Status() != StatusError {
			t.Errorf("healthz: want error, got %v", s.Healthz.Status())
		}
		if s.Metrics.Status() != StatusOK {
			t.Errorf("metrics must be unaffected: got %v err=%v", s.Metrics.Status(), s.Metrics.Err())
		}
		m, _ := s.Metrics.Get()
		if v, ok := m.NAT.ActiveFlows.Get(); !ok || v != 47 {
			t.Errorf("metrics values clobbered: flows=%v ok=%v", v, ok)
		}
	})

	t.Run("metrics fail, healthz succeeds", func(t *testing.T) {
		d := newFixtureDaemon(t)
		p := newTestPoller(t, d, t.TempDir(), "client-a", d.port(t), true)
		stub := &pathStubTransport{inner: d.srv.Client().Transport, failPath: "/metrics"}
		p.Client = &http.Client{Transport: stub}

		res := p.PollOnce(context.Background())
		s := res.Instances["client-a"]
		if s.Metrics.Status() != StatusError {
			t.Errorf("metrics: want error, got %v", s.Metrics.Status())
		}
		if s.Healthz.Status() != StatusOK {
			t.Errorf("healthz must be unaffected: got %v", s.Healthz.Status())
		}
		h, _ := s.Healthz.Get()
		if h.TunnelState != "up" {
			t.Errorf("healthz values clobbered: %+v", h)
		}
	})
}

// pathStubTransport fails requests to one path, passes others through.
type pathStubTransport struct {
	inner    http.RoundTripper
	failPath string
}

func (s *pathStubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.HasSuffix(req.URL.Path, s.failPath) {
		return nil, fmt.Errorf("stubbed failure for %s", s.failPath)
	}
	return s.inner.RoundTrip(req)
}

func TestPoller_ConfigResolutionFailure_ErrorsSources(t *testing.T) {
	// Config missing entirely: both sources error with the config error,
	// nothing stale, nothing fabricated.
	d := newFixtureDaemon(t)
	root := t.TempDir() // no config written
	r := newFakeRunner()
	r.outputs["systemctl list-units sssonector@* --all --plain --no-legend"] =
		"sssonector@ghost.service loaded active running x\n"
	r.outputs["systemctl show sssonector@ghost.service -p ActiveState -p SubState -p MainPID"] =
		"ActiveState=active\nSubState=running\nMainPID=1\n"
	r.outputs["systemctl show sssonector.service -p ActiveState -p SubState -p MainPID"] =
		"ActiveState=not-found\nSubState=dead\nMainPID=0\n"
	p := NewPoller(SystemdCollector{Runner: r, Paths: DefaultSystemdPaths()},
		ConfigPaths{ConfigRoot: root}, d.srv.Client())

	res := p.PollOnce(context.Background())
	s := res.Instances["ghost"]
	if s == nil {
		t.Fatalf("ghost missing: %+v", res.Instances)
	}
	if s.Healthz.Status() != StatusError || s.Metrics.Status() != StatusError {
		t.Errorf("want both errored, got healthz=%v metrics=%v", s.Healthz.Status(), s.Metrics.Status())
	}
	if s.ConfigErr == nil {
		t.Error("ConfigErr must carry the resolution error")
	}
}

func TestPoller_DiscoverFailure_NeverStale(t *testing.T) {
	// Runner fails AND zero configs: DiscoverErr set, Instances empty.
	p := NewPoller(SystemdCollector{Runner: errRunner{}, Paths: DefaultSystemdPaths()},
		ConfigPaths{ConfigRoot: t.TempDir()}, http.DefaultClient)
	res := p.PollOnce(context.Background())
	if res.DiscoverErr == nil {
		t.Fatal("want discover error")
	}
	if len(res.Instances) != 0 {
		t.Errorf("failed discovery must not fabricate instances: %+v", res.Instances)
	}
}

func TestSourceStatus_String(t *testing.T) {
	for st, want := range map[SourceStatus]string{
		StatusOK: "ok", StatusError: "error", StatusAbsent: "absent",
	} {
		if got := st.String(); got != want {
			t.Errorf("status %d: got %q want %q", st, got, want)
		}
	}
}
