package tui

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

var updateScreenGoldens = flag.Bool("update-screens", false, "rewrite screen golden files")

var screenNow = time.Date(2026, 9, 11, 14, 0, 0, 0, time.UTC)

// fixtureTickResult builds a TickResult with ok healthz/metrics/cert for
// the given instance names (reusing view-test fixture content semantics).
func fixtureTickResult(names []string) collect.TickResult {
	res := collect.TickResult{Instances: map[string]*collect.InstanceSnapshot{}}
	for i, name := range names {
		unit := "sssonector@" + name + ".service"
		if name == "default" {
			unit = "sssonector.service"
		}
		h := collect.Healthz{Status: "ok", Mode: "server", TunnelState: "up", UptimeSeconds: 8040}
		snap := &collect.InstanceSnapshot{
			Name: name, Unit: unit,
			ActiveState: "active", SubState: "running", MainPID: 8100 + i,
			Healthz:        collect.SourceOK(h),
			Metrics:        collect.SourceOK(fixtureScreenSnapshot()),
			PrometheusAddr: fmt.Sprintf("http://127.0.0.1:%d", 9090+i),
			TunAddr:        fmt.Sprintf("10.77.0.%d/24", 1+i),
			ListenPort:     9443 + i,
		}
		if i == 1 { // client-b down this tick: per-source error, never stale
			snap.ActiveState = "inactive"
			snap.SubState = "dead"
			snap.MainPID = 0
			snap.Healthz = collect.SourceErr[collect.Healthz](fmt.Errorf("connection refused"))
			snap.Metrics = collect.SourceErr[collect.Snapshot](fmt.Errorf("connection refused"))
			snap.Cert = collect.SourceErr[collect.CertInfo](fmt.Errorf("cert unreadable: no such file"))
		} else {
			snap.Cert = collect.SourceOK(collect.CertInfo{
				Issuer:        "CN=sssonector-instance-ca,O=pa-fleet-11",
				NotAfter:      screenNow.Add(64 * 24 * time.Hour),
				DaysRemaining: 64,
			})
		}
		res.Instances[name] = snap
	}
	return res
}

// fixtureScreenSnapshot is a compact metrics snapshot (real parser).
func fixtureScreenSnapshot() collect.Snapshot {
	const m = `# TYPE sssonector_bytes_in_total counter
sssonector_bytes_in_total 123456789
# TYPE sssonector_connections_active gauge
sssonector_connections_active 3
# TYPE sssonector_nat_forwarded_packets_total counter
sssonector_nat_forwarded_packets_total 1204551
# TYPE sssonector_nat_return_packets_total counter
sssonector_nat_return_packets_total 1190003
# TYPE sssonector_nat_dropped_packets_total counter
sssonector_nat_dropped_packets_total 12
# TYPE sssonector_nat_acl_denied_total counter
sssonector_nat_acl_denied_total 8
# TYPE sssonector_nat_flows_active gauge
sssonector_nat_flows_active 47
# TYPE sssonector_nat_listener_accepts_total counter
sssonector_nat_listener_accepts_total 63
# TYPE sssonector_throttle_hits_total counter
sssonector_throttle_hits_total{direction="in"} 812
sssonector_throttle_hits_total{direction="out"} 634
# TYPE sssonector_throttle_effective_rate_bytes_per_second gauge
sssonector_throttle_effective_rate_bytes_per_second 52428800
# TYPE sssonector_throttle_burst_bytes gauge
sssonector_throttle_burst_bytes 104857600
`
	fams, err := collect.ParseExposition(m)
	if err != nil {
		panic(err)
	}
	snap, err := collect.NewSnapshot(fams)
	if err != nil {
		panic(err)
	}
	return snap
}

// driveTick pushes one tickMsg through the model (message-driven, no timer).
func driveTick(m dashboardModel) dashboardModel {
	next, _ := m.Update(tickMsg{})
	return next.(dashboardModel)
}

// ctx is a background context for poll seams in tests.
func ctx() context.Context { return context.Background() }

func assertScreenGolden(t *testing.T, name, rendered string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *updateScreenGoldens {
		if err := os.MkdirAll("testdata", 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(rendered), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden %s missing (run -update-screens): %v", path, err)
	}
	if rendered != string(want) {
		t.Errorf("screen golden mismatch for %s:\n--- got ---\n%s\n--- want ---\n%s", name, rendered, string(want))
	}
}

func TestDashboard_ServerMode_Golden(t *testing.T) {
	var calls int
	m := NewDashboard(func(context.Context) collect.TickResult {
		calls++
		return fixtureTickResult([]string{"client-a", "client-b"})
	}, func() time.Time { return screenNow }).
		WithLogTail(func(collect.TickResult, int) []collect.LogEntry {
			// Hermetic fixture stream (WI 4.4): the golden must never
			// depend on host journalctl output.
			return []collect.LogEntry{
				{Instance: "client-a", Timestamp: screenNow.Add(-11 * time.Second), Unit: "sssonector@client-a", Pid: 8100, Message: "tunnel rekeyed peer 192.168.100.51"},
				{Instance: "client-b", Timestamp: screenNow.Add(-5 * time.Second), Unit: "sssonector@client-b", Pid: 8101, Message: "listener :9444 conn refused x1"},
			}
		})

	// Message-driven: Init issues the tick command; drive it explicitly.
	if m.Init() == nil {
		t.Fatal("Init must issue the first tick command")
	}
	m = driveTick(m)
	if calls != 1 {
		t.Fatalf("poll calls: %d, want 1", calls)
	}
	// Update re-issues the next tick command.
	next, cmd := m.Update(tickMsg{})
	if cmd == nil {
		t.Error("Update must re-issue the next tick command")
	}
	_ = next

	out := m.View()
	// Server-mode invariants: focused banner present (2 instances).
	if !strings.Contains(out, "--- focused: client-a ---") {
		t.Errorf("server mode must show focused banner:\n%s", out)
	}
	// Focused instance's panels show client-a's real values.
	if !strings.Contains(out, "fwd      1204551") {
		t.Errorf("focused metrics missing:\n%s", out)
	}
	// client-b (not focused) must not leak its errored state into the
	// focused panels; the rail shows its health dot though.
	if !strings.Contains(out, "client-b") {
		t.Errorf("rail must list client-b:\n%s", out)
	}
	assertScreenGolden(t, "screen_server", out)

	// Determinism: same tick result renders identically.
	if out != m.View() {
		t.Error("View must be deterministic for the same tick")
	}
}

func TestDashboard_ClientMode_Golden(t *testing.T) {
	m := NewDashboard(func(context.Context) collect.TickResult {
		return fixtureTickResult([]string{"default"})
	}, func() time.Time { return screenNow }).
		WithLogTail(func(collect.TickResult, int) []collect.LogEntry {
			// Hermetic fixture stream (WI 4.4): never shells out.
			return []collect.LogEntry{
				{Instance: "default", Timestamp: screenNow.Add(-3 * time.Second), Unit: "sssonector.service", Pid: 8100, Message: "listener :9443 ready"},
			}
		})
	m.Init()
	m = driveTick(m)

	out := m.View()
	// Degenerate single-instance view: NO focused banner (spec §3.2).
	if strings.Contains(out, "--- focused:") {
		t.Errorf("client mode must omit the focused banner:\n%s", out)
	}
	// Rail collapsed to one fixed row, still present.
	if !strings.Contains(out, "default") || !strings.Contains(out, "INSTANCES") {
		t.Errorf("client mode rail missing:\n%s", out)
	}
	// Panels render the single instance.
	if !strings.Contains(out, "fwd      1204551") {
		t.Errorf("client mode metrics missing:\n%s", out)
	}
	assertScreenGolden(t, "screen_client", out)
}

func TestDashboard_ErrorSource_NeverStale(t *testing.T) {
	m := NewDashboard(func(context.Context) collect.TickResult {
		return fixtureTickResult([]string{"default", "client-b"})
	}, func() time.Time { return screenNow })
	m.Init()
	m = driveTick(m)

	// Focus client-b (the errored instance) by making it first alphabetically
	// — use a model with client-b as the only-error case; simpler: assert
	// the errored instance's panels via a dedicated poll result where the
	// focused instance errors.
	m2 := NewDashboard(func(context.Context) collect.TickResult {
		res := fixtureTickResult([]string{"aaa-err", "zzz"})
		s := res.Instances["aaa-err"]
		s.Healthz = collect.SourceErr[collect.Healthz](fmt.Errorf("connection refused"))
		s.Metrics = collect.SourceErr[collect.Snapshot](fmt.Errorf("connection refused"))
		s.Cert = collect.SourceErr[collect.CertInfo](fmt.Errorf("unreadable"))
		return res
	}, func() time.Time { return screenNow })
	m2.Init()
	m2 = driveTick(m2)
	out := m2.View()

	if !strings.Contains(out, "daemon unreachable (connection refused)") {
		t.Errorf("errored focused instance must render unreachable:\n%s", out)
	}
	if !strings.Contains(out, "cert unreadable: unreadable") {
		t.Errorf("errored cert must render unreadable:\n%s", out)
	}
	// The focused (errored) instance's panels must not render fabricated
	// values: the real metric value appears nowhere on the screen because
	// BOTH instances error (zzz fixture also errors in this scenario).
	if strings.Contains(out, "1204551") {
		t.Errorf("NEVER STALE: errored focused panel rendered values:\n%s", out)
	}
}

func TestDashboard_DiscoverFailure(t *testing.T) {
	m := NewDashboard(func(context.Context) collect.TickResult {
		return collect.TickResult{DiscoverErr: fmt.Errorf("no systemd")}
	}, func() time.Time { return screenNow })
	m.Init()
	m = driveTick(m)
	out := m.View()
	if !strings.Contains(out, "discovery: ERROR: no systemd") {
		t.Errorf("discovery failure must render:\n%s", out)
	}
}

func TestDashboard_NoInstances(t *testing.T) {
	m := NewDashboard(func(context.Context) collect.TickResult {
		return collect.TickResult{Instances: map[string]*collect.InstanceSnapshot{}}
	}, func() time.Time { return screenNow })
	m.Init()
	m = driveTick(m)
	if out := m.View(); !strings.Contains(out, "no sssonector@* units on this host") {
		t.Errorf("empty rail line missing:\n%s", out)
	}
}
