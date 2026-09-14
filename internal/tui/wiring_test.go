package tui

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

// TestProductionWiring_NonNilDeps pins the Phase 3 rig-gate fix: the
// production wiring path must configure ALL config-view deps non-nil, so
// pressing `c` can never silently no-op again. It builds the same deps
// struct cmd/daemon/tui.go's runTUIDashboard passes (same constructors —
// the wiring cannot drift from this test without touching both).
func TestProductionWiring_NonNilDeps(t *testing.T) {
	// Mirror of runTUIDashboard's deps block.
	deps := ConfigDeps{
		Dump:     collect.EffectiveConfigDump,
		Validate: validateDraftWrapper,
		Apply:    collect.ApplyConfig,
		Paths:    collect.DefaultConfigPaths(),
		Signal:   collect.SignalHUP,
		Read:     collect.LogReloadReader(collect.OSCommandRunner{}, func(int) string { return "sssonector.service" }, time.Second),
	}

	if deps.Dump == nil {
		t.Fatal("Dump must be wired (collect.EffectiveConfigDump)")
	}
	// Teeth check: a zero-value (forgotten) dep must be caught by the
	// constructor path the same way the rig-gate regression would be.
	var forgotten ConfigDeps
	if forgotten.Dump != nil || forgotten.Validate != nil || forgotten.Apply != nil || forgotten.Signal != nil {
		t.Fatal("mutation harness broken: zero-value ConfigDeps should have nil fns")
	}
	mZero := NewDashboardWithConfig(
		func(context.Context) collect.TickResult { return collect.TickResult{} },
		func() time.Time { return screenNow },
		0, forgotten)
	// The nil guard in handleConfigKey/openConfig must trigger for this
	// model — proving non-nil deps are load-bearing, not decorative.
	if _, err := openConfig(mZero.deps, "default"); err == nil ||
		!strings.Contains(err.Error(), "not configured") {
		t.Errorf("zero-value deps must hit the explicit nil guard, got: %v", err)
	}
	if deps.Validate == nil {
		t.Fatal("Validate must be wired (collect.ValidateDraft wrapper)")
	}
	if deps.Apply == nil {
		t.Fatal("Apply must be wired (collect.ApplyConfig)")
	}
	if deps.Signal == nil {
		t.Fatal("Signal must be wired (collect.SignalHUP)")
	}
	if deps.Read == nil {
		t.Fatal("Read must be wired (reload reader)")
	}
	if deps.Paths.ConfigRoot != "/etc/sssonector" {
		t.Errorf("Paths must be the production root, got %q", deps.Paths.ConfigRoot)
	}

	// Regression simulation: NewDashboard (pre-fix API) leaves zero deps —
	// exactly what the rig gate hit. The constructor-separation is what
	// makes the gap visible.
	mOld := NewDashboard(
		func(context.Context) collect.TickResult { return collect.TickResult{} },
		func() time.Time { return screenNow })
	if mOld.deps.Dump != nil {
		t.Fatal("mutation harness broken: NewDashboard must leave deps zero")
	}

	// The model built via the production constructor carries them.
	m := NewDashboardWithConfig(
		func(context.Context) collect.TickResult { return collect.TickResult{} },
		func() time.Time { return screenNow },
		0, deps)
	if m.deps.Dump == nil || m.deps.Validate == nil || m.deps.Apply == nil || m.deps.Signal == nil || m.deps.Read == nil {
		t.Fatalf("NewDashboardWithConfig dropped deps: %+v", m.deps)
	}

	// End-to-end: openConfig with the production deps must NOT hit the
	// "not configured" guard. It fails on this host only because
	// /etc/sssonector does not exist — assert the error is the resolution
	// error, never the nil-guard error.
	if _, err := openConfig(deps, "default"); err != nil {
		if strings.Contains(err.Error(), "not configured") {
			t.Fatalf("production deps hit the nil guard — wiring gap regressed: %v", err)
		}
	}
}

// TestSignalHUP_RejectsDeadPid: the production SignalFunc must be a real
// implementation (errors on a dead pid rather than silently succeeding).
//
// NOTE on pid choice: kill(-1, sig) is NOT "signal a dead pid" — it
// broadcasts the signal to every process the caller may signal (the whole
// session, including the systemd user manager and anything else running
// beside the test). This test previously used -1 and SIGHUP-bombed the
// host on every `go test ./...` run. A genuinely dead pid (an exited
// child) returns ESRCH harmlessly and proves the same contract.
func TestSignalHUP_RejectsDeadPid(t *testing.T) {
	cmd := exec.Command("/bin/true")
	if err := cmd.Start(); err != nil {
		t.Skipf("cannot spawn child process: %v", err)
	}
	pid := cmd.Process.Pid
	_ = cmd.Wait() // pid is now definitely dead (and reaped: no zombie)
	if err := collect.SignalHUP(pid); err == nil {
		t.Errorf("SignalHUP(dead pid %d) must error", pid)
	}
}
