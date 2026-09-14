package view

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

// updateGoldens regenerates golden files: go test ./internal/tui/view/ -update
var updateGoldens = flag.Bool("update", false, "rewrite golden files")

// fixedNow is the deterministic clock for all golden fixtures.
var fixedNow = time.Date(2026, 9, 11, 14, 0, 0, 0, time.UTC)

// fixtureHealthzOK / fixtureMetricsOK are the shared ok fixtures.
func fixtureHealthzOK() collect.SourceOf[collect.Healthz] {
	return collect.SourceOK(collect.Healthz{Status: "ok", Mode: "server", TunnelState: "up", UptimeSeconds: 8040})
}

// metricsFixture is a compact exposition fixture with the key values the
// golden files assert (built with the real parser — no hand-rolled structs).
const metricsFixture = `# HELP sssonector_bytes_in_total bytes in.
# TYPE sssonector_bytes_in_total counter
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

func fixtureMetricsOK() collect.SourceOf[collect.Snapshot] {
	fams, err := collect.ParseExposition(metricsFixture)
	if err != nil {
		panic(err)
	}
	snap, err := collect.NewSnapshot(fams)
	if err != nil {
		panic(err)
	}
	return collect.SourceOK(snap)
}

func okCert() collect.SourceOf[collect.CertInfo] {
	return collect.SourceOK(collect.CertInfo{
		Issuer:        "CN=sssonector-instance-ca,O=pa-fleet-11",
		NotAfter:      fixedNow.Add(64 * 24 * time.Hour),
		DaysRemaining: 64,
		NeedsRotation: false,
	})
}

func errSource2[T any](msg string) collect.SourceOf[T] {
	return collect.SourceErr[T](errStr(msg))
}

func errStr(s string) error { return &staticErr{s} }

type staticErr struct{ s string }

func (e *staticErr) Error() string { return e.s }

// assertGolden compares rendered text to a golden file, regenerating with
// -update.
func assertGolden(t *testing.T, name, rendered string) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")
	if *updateGoldens {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(rendered), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden file %s missing (run with -update to seed): %v", path, err)
	}
	if got, wantS := rendered, string(want); got != wantS {
		t.Errorf("golden mismatch for %s:\n--- got ---\n%s\n--- want ---\n%s", name, got, wantS)
	}
}

func railFixture() RailInput {
	return RailInput{
		Instances: []collect.InstanceState{
			{Name: "client-a", Unit: "sssonector@client-a.service", ActiveState: "active", SubState: "running", MainPID: 8123},
			{Name: "client-b", Unit: "sssonector@client-b.service", ActiveState: "inactive", SubState: "dead", MainPID: 0},
			{Name: "client-c", Unit: "sssonector@client-c.service", ActiveState: "failed", SubState: "failed", MainPID: 0},
		},
		FocusIdx: 0,
		TUNAddresses: map[string]string{
			"client-a": "10.77.0.1/24",
			"client-b": "10.77.0.9/24",
			"client-c": "10.77.1.1/24",
		},
		ListenPorts: map[string]int{
			"client-a": 9443,
			"client-b": 9444,
			"client-c": 9445,
		},
		PeerCounts: map[string]int{"client-a": 2},
		LastPeerChange: map[string]time.Time{
			"client-a": fixedNow.Add(-2*time.Hour - 14*time.Minute),
			"client-b": fixedNow.Add(-41 * time.Minute),
			"client-c": fixedNow.Add(-5 * time.Minute),
		},
	}
}

func TestRenderRail_Golden(t *testing.T) {
	assertGolden(t, "rail_multi", RenderRail(railFixture(), fixedNow))
	assertGolden(t, "rail_empty", RenderRail(RailInput{}, fixedNow))
}

func TestRenderDaemonHeader_Golden(t *testing.T) {
	assertGolden(t, "header_running", RenderDaemonHeader(collect.InstanceState{ActiveState: "active", SubState: "running", MainPID: 8123}))
	assertGolden(t, "header_stopped", RenderDaemonHeader(collect.InstanceState{ActiveState: "inactive", SubState: "dead"}))
	assertGolden(t, "header_failed", RenderDaemonHeader(collect.InstanceState{ActiveState: "failed", SubState: "failed"}))
	assertGolden(t, "header_unknown", RenderDaemonHeader(collect.InstanceState{}))
}

func TestRenderTunnel_Golden(t *testing.T) {
	since := fixedNow.Add(-2*time.Hour - 14*time.Minute)
	assertGolden(t, "tunnel_ok", RenderTunnel(fixtureHealthzOK(), fixtureMetricsOK(), "10.77.0.1/24", since, fixedNow))
	assertGolden(t, "tunnel_error", RenderTunnel(errSource2[collect.Healthz]("connection refused"), fixtureMetricsOK(), "10.77.0.1/24", since, fixedNow))
	assertGolden(t, "tunnel_absent", RenderTunnel(collect.SourceAbsent[collect.Healthz](), fixtureMetricsOK(), "", time.Time{}, fixedNow))
	// listening state (no peer yet)
	lh := collect.SourceOK(collect.Healthz{Status: "ok", Mode: "server", TunnelState: "listening"})
	assertGolden(t, "tunnel_listening", RenderTunnel(lh, fixtureMetricsOK(), "10.77.0.1/24", since, fixedNow))
}

func TestRenderNAT_Golden(t *testing.T) {
	assertGolden(t, "nat_ok", RenderNAT(fixtureMetricsOK()))
	assertGolden(t, "nat_error", RenderNAT(errSource2[collect.Snapshot]("connection refused")))
	assertGolden(t, "nat_absent", RenderNAT(collect.SourceAbsent[collect.Snapshot]()))
}

func TestRenderRate_Golden(t *testing.T) {
	assertGolden(t, "rate_ok", RenderRate(fixtureMetricsOK()))
	assertGolden(t, "rate_error", RenderRate(errSource2[collect.Snapshot]("context deadline exceeded")))
	assertGolden(t, "rate_absent", RenderRate(collect.SourceAbsent[collect.Snapshot]()))
}

func TestRenderCert_Golden(t *testing.T) {
	assertGolden(t, "cert_ok", RenderCert(okCert(), fixedNow))
	// rotation due
	rot := collect.SourceOK(collect.CertInfo{
		Issuer:        "CN=sssonector-instance-ca,O=pa-fleet-11",
		NotAfter:      fixedNow.Add(14 * 24 * time.Hour),
		DaysRemaining: 14,
		NeedsRotation: true,
	})
	assertGolden(t, "cert_rotation", RenderCert(rot, fixedNow))
	assertGolden(t, "cert_error", RenderCert(errSource2[collect.CertInfo]("open /etc/sssonector/instances/client-a/certs/server.crt: no such file or directory"), fixedNow))
	assertGolden(t, "cert_absent", RenderCert(collect.SourceAbsent[collect.CertInfo](), fixedNow))
}

func TestRenderLog_Golden(t *testing.T) {
	entries := []collect.LogEntry{
		{Instance: "client-a", Timestamp: time.Date(2026, 9, 11, 14, 2, 11, 0, time.UTC), Unit: "sssonector@client-a", Pid: 8123, Message: "tunnel rekeyed peer 192.168.100.51"},
		{Instance: "client-b", Timestamp: time.Date(2026, 9, 11, 14, 1, 58, 0, time.UTC), Unit: "sssonector@client-b", Pid: 8124, Message: "listener :9445 conn refused x3"},
		{Instance: "client-a", Raw: "garbage line carried verbatim"},
	}
	assertGolden(t, "log_ok", RenderLog(entries, 12))
	// trimming: more than n keeps newest
	assertGolden(t, "log_trimmed", RenderLog(entries, 2))
	assertGolden(t, "log_empty", RenderLog(nil, 12))
}

// Mutation-check helper tests: the degenerate-case contracts are asserted
// directly (not just via goldens) so a reverted renderer fails loudly.
func TestRender_AntiMockContracts(t *testing.T) {
	// Error sources must NEVER render a value.
	tun := RenderTunnel(errSource2[collect.Healthz]("boom"), fixtureMetricsOK(), "", time.Time{}, fixedNow)
	if strings.Contains(tun, "up") || strings.Contains(tun, "8040") {
		t.Errorf("error tunnel must not render ok-state values:\n%s", tun)
	}
	if !strings.Contains(tun, "daemon unreachable (boom)") {
		t.Errorf("error tunnel must render unreachable:\n%s", tun)
	}
	nat := RenderNAT(errSource2[collect.Snapshot]("boom"))
	if strings.Contains(nat, "1204551") {
		t.Errorf("error NAT must not render stale values:\n%s", nat)
	}
	// Absent renders the disabled placeholder exactly.
	if !strings.Contains(RenderNAT(collect.SourceAbsent[collect.Snapshot]()), "— (prometheus disabled in config)") {
		t.Error("absent NAT must render the prometheus-disabled placeholder")
	}
	// Empty rail renders the spec line exactly.
	if got := RenderRail(RailInput{}, fixedNow); !strings.Contains(got, "no sssonector@* units on this host") {
		t.Errorf("empty rail line wrong: %s", got)
	}
	// Unreadable cert renders unreadable, never guessed.
	cert := RenderCert(errSource2[collect.CertInfo]("no such file"), fixedNow)
	if !strings.Contains(cert, "cert unreadable: no such file") || strings.Contains(cert, "issuer") && strings.Contains(cert, "CN=") {
		t.Errorf("cert error must render unreadable without issuer:\n%s", cert)
	}
}
