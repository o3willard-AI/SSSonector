package collect

import (
	"context"
	"strings"
	"testing"
)

// testCtx is a plain background context for PollOnce in tests.
func testCtx() context.Context { return context.Background() }

// buildProbeFixtures assembles a TickResult + logs from the existing
// fixture helpers (fixture daemon + fake runner), without any real
// systemd/journalctl/daemon.
func probeFixture(t *testing.T, healthy bool) (TickResult, map[string][]LogEntry) {
	t.Helper()
	d := newFixtureDaemon(t)
	d.healthy = healthy
	d.metricsUp = healthy

	root := t.TempDir()
	writeInstanceConfig(t, root, "client-a", d.port(t), true)
	writeInstanceConfig(t, root, "client-b", d.port(t), true)

	r := newFakeRunner()
	r.outputs["systemctl list-units sssonector@* --all --plain --no-legend"] =
		"sssonector@client-a.service loaded active running x\n" +
			"sssonector@client-b.service loaded inactive dead x\n"
	r.outputs["systemctl show sssonector@client-a.service -p ActiveState -p SubState -p MainPID -p LoadState"] =
		"ActiveState=active\nSubState=running\nMainPID=100\n"
	r.outputs["systemctl show sssonector@client-b.service -p ActiveState -p SubState -p MainPID -p LoadState"] =
		"ActiveState=inactive\nSubState=dead\nMainPID=0\n"
	r.outputs["systemctl show sssonector.service -p ActiveState -p SubState -p MainPID -p LoadState"] =
		"ActiveState=not-found\nSubState=dead\nMainPID=0\n"
	// journalctl fake: both units same fixture output (unit arg differs but
	// fake runner keys on full argv; add both).
	for _, u := range []string{"sssonector@client-a.service", "sssonector@client-b.service"} {
		r.outputs["journalctl -u "+u+" -n 12 --no-pager -o short-iso --show-cursor"] =
			"2026-09-13T02:05:26+00:00 qa-host " + strings.TrimSuffix(strings.TrimPrefix(u, "sssonector@"), ".service") + "[8123]: tunnel rekeyed peer 192.168.100.51\n" +
				"-- cursor: s=1;i=1;b=boot-1\n"
	}

	poller := NewPoller(SystemdCollector{Runner: r, Paths: DefaultSystemdPaths()},
		ConfigPaths{ConfigRoot: root}, d.srv.Client())
	res := poller.PollOnce(testCtx())

	logs := map[string][]LogEntry{}
	lt := NewLogTail(r)
	for name, snap := range res.Instances {
		read, err := lt.Read(snap.Unit, "", "boot-1", 12)
		if err == nil {
			logs[name] = read.Entries
		}
	}
	return res, logs
}

func TestFormatProbe_Healthy(t *testing.T) {
	res, logs := probeFixture(t, true)
	out, fatal := FormatProbe(res, logs)
	if fatal {
		t.Fatalf("healthy probe must not be fatal: %s", out)
	}

	for _, want := range []string{
		"== instance client-a ==",
		"== instance client-b ==",
		"unit:        sssonector@client-a.service",
		"systemd:     active/running pid=100",
		"healthz:     ok mode=server tunnel_state=up uptime=8040s",
		// a real metric value from the live fixture:
		"nat_forwarded    1204551",
		"connections_active 3",
		// a log line:
		"tunnel rekeyed peer 192.168.100.51",
		"[client-a]",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("probe text missing %q:\n%s", want, out)
		}
	}

	// Deterministic: two renders over the same result are identical (after
	// normalizing the timestamp line).
	normalize := func(s string) string {
		lines := strings.Split(s, "\n")
		lines[0] = "probe time: X"
		return strings.Join(lines, "\n")
	}
	out2, _ := FormatProbe(res, logs)
	if normalize(out) != normalize(out2) {
		t.Error("probe output must be deterministic")
	}

	// client-b is inactive: metrics source should still resolve (config
	// exists, prometheus enabled) but systemd state shows inactive.
	if !strings.Contains(out, "systemd:     inactive/dead pid=0") {
		t.Errorf("client-b state line missing:\n%s", out)
	}
}

func TestFormatProbe_PerSourceError(t *testing.T) {
	// Daemon down this tick: healthz and metrics render ERROR lines, not
	// values, not fabricated zeros.
	res, logs := probeFixture(t, false)
	out, fatal := FormatProbe(res, logs)
	if fatal {
		t.Fatal("per-source errors are not fatal")
	}
	if !strings.Contains(out, "healthz:     ERROR:") {
		t.Errorf("healthz error line missing:\n%s", out)
	}
	if !strings.Contains(out, "metrics:     ERROR:") {
		t.Errorf("metrics error line missing:\n%s", out)
	}
	// No fabricated values: the fixture's known value must NOT appear in
	// the metrics section (only its error line).
	if strings.Contains(out, "nat_forwarded  1204551") {
		t.Error("NEVER STALE: errored metrics must not render tick-1 values")
	}
	_ = logs
}

func TestFormatProbe_DiscoverFailure_IsFatal(t *testing.T) {
	res := TickResult{DiscoverErr: errForTest("no systemd")}
	out, fatal := FormatProbe(res, nil)
	if !fatal {
		t.Fatal("discovery failure must be fatal")
	}
	if !strings.Contains(out, "discovery: ERROR:") {
		t.Errorf("discovery error line missing: %s", out)
	}
}

func TestFormatProbe_PrometheusDisabled_Absent(t *testing.T) {
	d := newFixtureDaemon(t)
	root := t.TempDir()
	writeInstanceConfig(t, root, "client-a", d.port(t), false) // prometheus disabled
	r := newFakeRunner()
	r.outputs["systemctl list-units sssonector@* --all --plain --no-legend"] =
		"sssonector@client-a.service loaded active running x\n"
	r.outputs["systemctl show sssonector@client-a.service -p ActiveState -p SubState -p MainPID -p LoadState"] =
		"ActiveState=active\nSubState=running\nMainPID=1\n"
	r.outputs["systemctl show sssonector.service -p ActiveState -p SubState -p MainPID -p LoadState"] =
		"ActiveState=not-found\nSubState=dead\nMainPID=0\n"

	poller := NewPoller(SystemdCollector{Runner: r, Paths: DefaultSystemdPaths()},
		ConfigPaths{ConfigRoot: root}, d.srv.Client())
	res := poller.PollOnce(testCtx())
	out, _ := FormatProbe(res, nil)
	if !strings.Contains(out, "metrics:     absent") {
		t.Errorf("prometheus-disabled metrics must render absent:\n%s", out)
	}
}

func errForTest(msg string) error { return &strErr{msg} }

type strErr struct{ s string }

func (e *strErr) Error() string { return e.s }
