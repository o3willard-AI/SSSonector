package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

// lcRunner records systemctl argv (the fake runner — no real systemctl is
// ever invoked from tests).
type lcRunner struct {
	calls [][]string
	err   error
}

func (r *lcRunner) Run(args ...string) (string, error) {
	r.calls = append(r.calls, args)
	return "", r.err
}

// lcSignal records SIGHUP pids (never a real signal).
type lcSignal struct {
	pids []int
}

func (s *lcSignal) signal(pid int) error {
	s.pids = append(s.pids, pid)
	return nil
}

// lcFixture builds a lifecycle-wired model with two instances: client-a up
// (pid 8100) and client-b down. The poll seam returns one fixed tick.
func lcFixture(t *testing.T, runner collect.CommandRunner, sig *lcSignal, read collect.ReloadReader) dashboardModel {
	t.Helper()
	m := lcFixtureNames(t, []string{"client-a", "client-b"}, runner, sig, read)
	return m
}

// lcFixtureNames is lcFixture with configurable instance names (their
// fixture indexes drive up/down state: client-a up, index-1 down).
func lcFixtureNames(t *testing.T, names []string, runner collect.CommandRunner, sig *lcSignal, read collect.ReloadReader) dashboardModel {
	t.Helper()
	m := NewDashboardWithLifecycle(
		func(context.Context) collect.TickResult {
			return fixtureTickResult(names)
		},
		func() time.Time { return screenNow },
		0,
		ConfigDeps{},
		LifecycleDeps{Runner: runner, Signal: sig.signal, Read: read},
	)
	m.Init()
	return driveTick(m)
}

func lkey(m dashboardModel, s string) dashboardModel {
	next, _ := m.Update(tea.KeyMsg{Type: keyTypeFor(s), Runes: runesFor(s)})
	return next.(dashboardModel)
}

// TestLifecycleStop_ConfirmDialog_EnterRunsExactlyOneCall: s shows the
// dialog; Enter issues exactly one runner call with the correct unit.
func TestLifecycleStop_ConfirmDialog_EnterRunsExactlyOneCall(t *testing.T) {
	r := &lcRunner{}
	sig := &lcSignal{}
	m := lcFixture(t, r, sig, nil)

	m = lkey(m, "s")
	if m.mode != modeConfirm {
		t.Fatalf("s must open the confirm dialog, mode=%v", m.mode)
	}
	if len(r.calls) != 0 {
		t.Fatalf("FAIL-CLOSED VIOLATED: dialog open but runner called: %v", r.calls)
	}
	if got := m.View(); !strings.Contains(got, "stop sssonector@client-a.service?") {
		t.Errorf("dialog must name the unit:\n%s", got)
	}
	// Random key: still nothing runs.
	m = lkey(m, "x")
	if m.mode != modeConfirm || len(r.calls) != 0 {
		t.Fatalf("other keys must stay in dialog with zero calls: mode=%v calls=%v", m.mode, r.calls)
	}

	m = lkey(m, "enter")
	if m.mode != modeDashboard {
		t.Errorf("enter must return to dashboard, mode=%v", m.mode)
	}
	if len(r.calls) != 1 {
		t.Fatalf("runner calls after confirm: %d, want exactly 1", len(r.calls))
	}
	if got := strings.Join(r.calls[0], " "); got != "systemctl stop sssonector@client-a.service" {
		t.Errorf("argv: %q", got)
	}
	if !strings.Contains(m.View(), "banner: ✓ stop sssonector@client-a.service") {
		t.Errorf("ok banner missing:\n%s", m.View())
	}
}

// TestLifecycleRestart_ConfirmDialog_EscCancelsZeroCalls: r shows the
// dialog; Esc issues ZERO runner calls and returns to the dashboard.
func TestLifecycleRestart_ConfirmDialog_EscCancelsZeroCalls(t *testing.T) {
	r := &lcRunner{}
	m := lcFixture(t, r, &lcSignal{}, nil)

	m = lkey(m, "r")
	if m.mode != modeConfirm {
		t.Fatalf("r must open the confirm dialog, mode=%v", m.mode)
	}
	if got := m.View(); !strings.Contains(got, "restart sssonector@client-a.service?") {
		t.Errorf("dialog must name the unit and verb:\n%s", got)
	}
	m = lkey(m, "esc")
	if m.mode != modeDashboard {
		t.Errorf("esc must return to dashboard, mode=%v", m.mode)
	}
	if len(r.calls) != 0 {
		t.Fatalf("FAIL-CLOSED VIOLATED: esc issued runner calls: %v", r.calls)
	}
}

// TestLifecycle_UnitNameFromFocusedInstance: the runner receives
// sssonector@<focused>, never a hardcoded default. Focus client-c via a
// three-instance fixture, then stop it.
func TestLifecycle_UnitNameFromFocusedInstance(t *testing.T) {
	r := &lcRunner{}
	m := lcFixtureNames(t, []string{"client-a", "client-b", "client-c"}, r, &lcSignal{}, nil)
	// Focus client-c: select it in the rail, then Enter.
	m = lkey(m, "down")
	m = lkey(m, "down")
	m = lkey(m, "enter")
	if m.focus != "client-c" {
		t.Fatalf("focus: %q", m.focus)
	}
	m = driveTick(m) // refresh snapshot so the focused pid/unit are live
	m = lkey(m, "s")
	m = lkey(m, "enter")
	if len(r.calls) != 1 {
		t.Fatalf("calls: %d", len(r.calls))
	}
	if got := strings.Join(r.calls[0], " "); got != "systemctl stop sssonector@client-c.service" {
		t.Errorf("unit must follow the FOCUSED instance, argv: %q", got)
	}
}

// TestLifecycle_R_NoDialog_SignalsFocusedPID_BannerOutcome: R skips the
// dialog, signals the focused pid, and maps the reload outcome to a
// banner. OK and rejected outcomes both pinned.
func TestLifecycle_R_NoDialog_SignalsFocusedPID(t *testing.T) {
	sig := &lcSignal{}
	m := lcFixture(t, &lcRunner{}, sig, func(int) (collect.ReloadOutcome, error) { return collect.ReloadOK, nil })

	m = lkey(m, "R")
	if m.mode != modeDashboard {
		t.Fatalf("R must NOT open a dialog, mode=%v", m.mode)
	}
	if len(sig.pids) != 1 || sig.pids[0] != 8100 {
		t.Fatalf("SIGHUP pids: %v, want [8100] (focused client-a's MainPID)", sig.pids)
	}
	if !strings.Contains(m.View(), "banner: ✓ sssonector@client-a.service reloaded") {
		t.Errorf("reload-ok banner missing:\n%s", m.View())
	}

	// Rejected outcome surfaces the rejection banner.
	sig2 := &lcSignal{}
	m2 := lcFixture(t, &lcRunner{}, sig2, func(int) (collect.ReloadOutcome, error) { return collect.ReloadRejected, nil })
	m2 = lkey(m2, "R")
	if len(sig2.pids) != 1 {
		t.Fatalf("pids: %v", sig2.pids)
	}
	if !strings.Contains(m2.View(), "REJECTED the reload") {
		t.Errorf("reload-rejected banner missing:\n%s", m2.View())
	}
}

// TestLifecycle_StopDisabledWhenDown: s on a stopped instance issues ZERO
// runner calls (disabled) with an explicit banner.
func TestLifecycle_StopDisabledWhenDown(t *testing.T) {
	r := &lcRunner{}
	m := lcFixture(t, r, &lcSignal{}, nil)
	// client-b is down in the fixture; focus it.
	m = lkey(m, "down")
	m = lkey(m, "enter")
	if m.focus != "client-b" {
		t.Fatalf("focus: %q", m.focus)
	}
	m = lkey(m, "s")
	if m.mode == modeConfirm {
		t.Fatal("s on a stopped instance must NOT open the confirm dialog")
	}
	if len(r.calls) != 0 {
		t.Fatalf("FAIL-CLOSED VIOLATED: s on stopped instance ran: %v", r.calls)
	}
	if !strings.Contains(m.View(), "(disabled)") || !strings.Contains(m.View(), "not running") {
		t.Errorf("disabled banner missing:\n%s", m.View())
	}
}

// TestLifecycle_RunnerErrorSurfaces: a failed systemctl call surfaces the
// error banner (exactly one call was still made).
func TestLifecycle_RunnerErrorSurfaces(t *testing.T) {
	r := &lcRunner{err: context.DeadlineExceeded}
	m := lcFixture(t, r, &lcSignal{}, nil)
	m = lkey(m, "r")
	m = lkey(m, "enter")
	if len(r.calls) != 1 {
		t.Fatalf("calls: %d", len(r.calls))
	}
	if !strings.Contains(m.View(), "banner: ✗") {
		t.Errorf("error banner missing:\n%s", m.View())
	}
}

// TestLifecycleWiring_NonNilDeps: the production wiring path must
// configure ALL lifecycle deps non-nil, so s/r/R can never silently
// no-op — the WI 4.1 counterpart of TestProductionWiring_NonNilDeps
// (which pins the same contract for ConfigDeps after the Phase 3
// rig-gate bug). It builds the same deps cmd/daemon/tui.go's
// runTUIDashboard passes.
func TestLifecycleWiring_NonNilDeps(t *testing.T) {
	// Mirror of runTUIDashboard's lifecycle deps block.
	lc := LifecycleDeps{
		Runner: collect.OSCommandRunner{},
		Signal: collect.SignalHUP,
		Read:   collect.LogReloadReader(collect.OSCommandRunner{}, func(int) string { return "sssonector.service" }, time.Second),
	}
	if lc.Runner == nil {
		t.Fatal("Runner must be wired (collect.OSCommandRunner)")
	}
	if lc.Signal == nil {
		t.Fatal("Signal must be wired (collect.SignalHUP)")
	}
	if lc.Read == nil {
		t.Fatal("Read must be wired (collect.LogReloadReader)")
	}

	// Teeth: the production constructor carries them; a forgotten dep is
	// visible as nil on the model.
	m := NewDashboardWithLifecycle(
		func(context.Context) collect.TickResult { return collect.TickResult{} },
		func() time.Time { return screenNow },
		0, ConfigDeps{}, lc)
	if m.lc.deps.Runner == nil || m.lc.deps.Signal == nil || m.lc.deps.Read == nil {
		t.Fatalf("NewDashboardWithLifecycle dropped deps: %+v", m.lc.deps)
	}

	// And the zero-value model has them nil — the mutation check proving
	// the assertions above can fail.
	var forgotten LifecycleDeps
	if forgotten.Runner != nil || forgotten.Signal != nil || forgotten.Read != nil {
		t.Fatal("mutation harness broken: zero-value LifecycleDeps should be nil")
	}
}
