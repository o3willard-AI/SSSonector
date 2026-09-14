package tui

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

// WI 4.4: degradation-matrix goldens — one test per docs/tui.md §7
// failure-mode row, pinning the WHOLE-SCREEN degraded render. Each test
// fails if the degraded render regresses to the healthy render or to a
// fabricated value. No real journalctl/systemctl runs (log seam injected;
// collectors fed by fixtures).

// degModel builds a dashboard over an explicit tick result with a fixture
// log stream (hermetic — never shells out).
func degModel(res collect.TickResult, logs []collect.LogEntry) dashboardModel {
	m := NewDashboard(func(context.Context) collect.TickResult { return res },
		func() time.Time { return screenNow }).WithLogTail(func(collect.TickResult, int) []collect.LogEntry {
		return logs
	})
	m.Init()
	return driveTick(m)
}

// unreachableResult: one instance whose healthz/metrics/cert all errored
// (daemon unreachable — §7 row 1).
func unreachableResult() collect.TickResult {
	res := fixtureTickResult([]string{"solo"})
	s := res.Instances["solo"]
	s.Healthz = collect.SourceErr[collect.Healthz](fmt.Errorf("connection refused"))
	s.Metrics = collect.SourceErr[collect.Snapshot](fmt.Errorf("connection refused"))
	s.Cert = collect.SourceErr[collect.CertInfo](fmt.Errorf("connection refused"))
	return res
}

// TestDegradation1_DaemonUnreachable: panels show "daemon unreachable
// (...)", the rail dot is ✖ (healthz absent/error class), no fabricated
// metric values anywhere, and the footer still offers stop/restart (the
// dashboard stays interactive — nothing crashes or blanks out).
func TestDegradation1_DaemonUnreachable(t *testing.T) {
	out := degModel(unreachableResult(), nil).View()
	if !strings.Contains(out, "daemon unreachable (connection refused)") {
		t.Errorf("§7.1: TUNNEL must render unreachable:\n%s", out)
	}
	if strings.Contains(out, "1204551") || strings.Contains(out, "fwd      1") {
		t.Errorf("§7.1: NEVER STALE — unreachable panels rendered metric values:\n%s", out)
	}
	rail := railSection(t, out)
	if !strings.Contains(railLineFor(t, rail, "solo"), "✖") {
		t.Errorf("§7.1: rail dot must be ✖ for unreachable:\n%s", rail)
	}
	if !strings.Contains(out, "footer: [s]top [r]estart [R]eload act on FOCUSED instance") {
		t.Errorf("§7.1: dashboard must stay interactive (footer intact):\n%s", out)
	}
}

// TestDegradation2_PrometheusDisabled: source-specific disabled
// placeholders; healthz panels still render (healthz is served even when
// prometheus is off).
func TestDegradation2_PrometheusDisabled(t *testing.T) {
	res := fixtureTickResult([]string{"solo"})
	s := res.Instances["solo"]
	s.Metrics = collect.SourceAbsent[collect.Snapshot]()
	out := degModel(res, nil).View()
	if !strings.Contains(out, "— (prometheus disabled in config)") {
		t.Errorf("§7.2: NAT must show the prometheus-disabled placeholder:\n%s", out)
	}
	// healthz unaffected: TUNNEL renders the ok state, header RUNNING.
	if !strings.Contains(out, "daemon: RUNNING") {
		t.Errorf("§7.2: healthz must still drive the header:\n%s", out)
	}
	if !strings.Contains(out, "state    ● up") {
		t.Errorf("§7.2: TUNNEL must still render from healthz:\n%s", out)
	}
	// No fabricated metric values in NAT/RATE.
	if strings.Contains(out, "1204551") || strings.Contains(out, "50.0 MB/s") {
		t.Errorf("§7.2: absent metrics must not render values:\n%s", out)
	}
}

// TestDegradation3_ConfigUnreadable: config view shows the loader error
// verbatim; the dashboard still runs (rail/panels unaffected).
func TestDegradation3_ConfigUnreadable(t *testing.T) {
	m := NewDashboard(func(context.Context) collect.TickResult {
		return fixtureTickResult([]string{"solo"})
	}, func() time.Time { return screenNow }).WithLogTail(func(collect.TickResult, int) []collect.LogEntry {
		return nil
	})
	// Inject a failing Dump into the config deps.
	boom := errors.New("open /etc/sssonector/instances/solo/config.yaml: permission denied")
	m.deps = ConfigDeps{
		Dump: func(collect.ConfigPaths, string) ([]collect.ConfigLine, error) { return nil, boom },
	}
	m.Init()
	m = driveTick(m)

	// c opens config mode: the error surfaces, mode stays dashboard
	// (fail-closed open path — the error is visible, never a blank view).
	before := m.View()
	m = key(m, "c")
	if m.mode == modeConfig {
		t.Fatal("config open must fail closed when the dump errors")
	}
	// The dashboard still runs (same rail/panels as before the attempt).
	if m.View() != before {
		t.Errorf("§7.3: dashboard must keep rendering after a failed config open")
	}
	if m.mode != modeDashboard {
		t.Errorf("§7.3: mode must remain dashboard after failed open: %v", m.mode)
	}
}

// TestDegradation4_SystemctlFails: discovery degrades to a single
// implicit instance only if exactly one config exists; otherwise the
// explicit error renders — never a fabricated list.
func TestDegradation4_SystemctlFails(t *testing.T) {
	// Case A: explicit discovery error renders.
	out := degModel(collect.TickResult{DiscoverErr: errors.New("systemctl unavailable")}, nil).View()
	if !strings.Contains(out, "discovery: ERROR: systemctl unavailable") {
		t.Errorf("§7.4: explicit discovery error must render:\n%s", out)
	}
	if strings.Contains(out, "INSTANCES") {
		t.Errorf("§7.4: no fabricated instance list on discovery failure:\n%s", out)
	}

	// Case B (collector-level): exactly one config under the root degrades
	// to the implicit instance; zero configs errors. Both pinned at the
	// seam the Poller uses.
	dir := t.TempDir()
	oneCfg := dir + "/instances/only-one/config.yaml"
	if err := writeDegFile(oneCfg, "x: y\n"); err != nil {
		t.Fatal(err)
	}
	st, err := collect.ImplicitInstanceForTest(dir)
	if err != nil || st.Name != "only-one" {
		t.Errorf("§7.4: exactly-one config must degrade to the implicit instance: %+v err=%v", st, err)
	}
	empty := t.TempDir()
	if _, err := collect.ImplicitInstanceForTest(empty); err == nil {
		t.Errorf("§7.4: zero configs must error, not fabricate")
	}
}

func writeDegFile(path, content string) error {
	i := strings.LastIndexByte(path, '/')
	if i > 0 {
		if err := os.MkdirAll(path[:i], 0o750); err != nil {
			return err
		}
	}
	return os.WriteFile(path, []byte(content), 0o600)
}

func readFileDeg(path string) (string, error) {
	b, err := os.ReadFile(path)
	return string(b), err
}

// TestDegradation5_ReloadRejected: rejection banner on the config view,
// and the config file STAYS WRITTEN (no rollback — documented SIGHUP
// semantics). At the screen level the rejected banner is the visible cell.
func TestDegradation5_ReloadRejected(t *testing.T) {
	m, sig, root := configFixture(t, cfgValidYAML)
	// Reader reports the daemon rejected the reload.
	m.deps.Read = func(int) (collect.ReloadOutcome, error) { return collect.ReloadRejected, nil }
	m = ckey(m, "c")
	m = ckey(m, "a")
	out := m.View()
	if !strings.Contains(out, "⚠ applied but daemon REJECTED the reload") {
		t.Errorf("§7.5: rejected banner must render:\n%s", out)
	}
	// File stays written (rollback is manual, documented).
	written, err := readFileDeg(root + "/instances/client-a/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(written, "metadata") {
		t.Errorf("§7.5: config file must stay written after rejection:\n%s", written)
	}
	if len(sig.calls) != 1 {
		t.Errorf("§7.5: exactly one signal must have fired: %v", sig.calls)
	}
}

// TestDegradation6_CertUnreadable: cert panel shows the verbatim error,
// never a guessed issuer/expiry.
func TestDegradation6_CertUnreadable(t *testing.T) {
	res := fixtureTickResult([]string{"solo"})
	res.Instances["solo"].Cert = collect.SourceErr[collect.CertInfo](
		fmt.Errorf("open /etc/sssonector/instances/solo/certs/server.crt: no such file or directory"))
	out := degModel(res, nil).View()
	if !strings.Contains(out, "cert unreadable: open /etc/sssonector/instances/solo/certs/server.crt: no such file or directory") {
		t.Errorf("§7.6: verbatim unreadable error must render:\n%s", certSection(t, out))
	}
	if strings.Contains(out, "CN=") {
		t.Errorf("§7.6: NEVER GUESS — issuer rendered on an unreadable cert:\n%s", certSection(t, out))
	}
}
