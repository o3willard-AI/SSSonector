package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

// LifecycleDeps are the injectable lifecycle operations (WI 4.1 seams).
// Production wires collect.OSCommandRunner + collect.SignalHUP +
// collect.LogReloadReader in cmd/daemon/tui.go; tests inject fakes — no
// real systemctl runs from tests. Exported so the WI 3.4 rig-gate wiring
// gap (unwired deps silently no-oping) cannot regress: the constructor
// requires them explicitly.
type LifecycleDeps struct {
	// Runner executes systemctl stop/restart (collect.OSCommandRunner in
	// production).
	Runner collect.CommandRunner
	// Signal sends SIGHUP (collect.SignalHUP in production).
	Signal collect.SignalFunc
	// Read reads the reload outcome after the signal (collect.LogReloadReader).
	Read collect.ReloadReader
}

// confirmModel is the confirm-dialog sub-model for destructive lifecycle
// actions (WI 4.1).
type confirmModel struct {
	instance string
	unit     string // sssonector@<name>.service (always derived)
	action   collect.LifecycleAction
}

// prompt renders the confirm dialog text.
func (c confirmModel) prompt() string {
	return fmt.Sprintf("%s %s? [Enter] confirm [Esc] cancel", c.action, c.unit)
}

// lifecycleBanner holds the last lifecycle action outcome, surfaced on the
// dashboard (separate from the config-view banner so the two never fight).
type lifecycleBanner struct {
	kind bannerKind
	text string
}

// confirmLifecycle builds the confirm dialog for a destructive action on
// the named instance. The unit is ALWAYS derived from the instance name —
// never a hardcoded default.
func confirmLifecycle(instance string, action collect.LifecycleAction) confirmModel {
	return confirmModel{instance: instance, unit: collect.UnitNameFor(instance), action: action}
}

// handleLifecycleKey processes s/r/R when the dashboard is active. Returns
// (model, handled). Destructive actions route through the confirm dialog;
// R (reload) acts immediately (SIGHUP is non-destructive).
func (m dashboardModel) handleLifecycleKey(msg tea.KeyMsg) (tea.Model, bool) {
	snap := m.focusedSnapshot()
	if snap == nil {
		return m, false // nothing focused: no action possible
	}
	unit := collect.UnitNameFor(snap.Name)

	switch msg.String() {
	case "s":
		// s is disabled when the instance is already stopped: zero runner
		// calls, explicit banner.
		if !instanceRunning(snap) {
			m.lc.banner = lifecycleBanner{
				kind: bannerLifecycleDisabled,
				text: fmt.Sprintf("stop disabled — %s is not running", unit),
			}
			return m, true
		}
		if m.lc.deps.Runner == nil {
			m.lc.banner = lifecycleBanner{kind: bannerLifecycleError, text: "lifecycle runner not configured"}
			return m, true
		}
		// Destructive: confirm first (fail-closed — nothing runs until Enter).
		m.mode = modeConfirm
		m.lc.confirm = confirmLifecycle(snap.Name, collect.LifecycleStop)
		return m, true
	case "r":
		if m.lc.deps.Runner == nil {
			m.lc.banner = lifecycleBanner{kind: bannerLifecycleError, text: "lifecycle runner not configured"}
			return m, true
		}
		m.mode = modeConfirm
		m.lc.confirm = confirmLifecycle(snap.Name, collect.LifecycleRestart)
		return m, true
	case "R":
		return m.reloadFocused()
	}
	return m, false
}

// instanceRunning reports whether the snapshot shows a live daemon
// (active state with a real main pid).
func instanceRunning(snap *collect.InstanceSnapshot) bool {
	return snap != nil && snap.ActiveState == "active" && snap.MainPID > 0
}

// reloadFocused runs the R path: SignalHUP on the focused pid, then the
// reload reader; outcome mapped to a banner. No confirm dialog (SIGHUP is
// non-destructive).
func (m dashboardModel) reloadFocused() (tea.Model, bool) {
	snap := m.focusedSnapshot()
	if snap == nil {
		return m, true
	}
	if m.lc.deps.Signal == nil || m.lc.deps.Read == nil {
		m.lc.banner = lifecycleBanner{kind: bannerReloadError, text: "reload deps not configured"}
		return m, true
	}
	if snap.MainPID <= 0 {
		m.lc.banner = lifecycleBanner{kind: bannerReloadError, text: fmt.Sprintf("no live pid for %s — not reloaded", snap.Unit)}
		return m, true
	}
	if err := m.lc.deps.Signal(snap.MainPID); err != nil {
		m.lc.banner = lifecycleBanner{kind: bannerReloadError, text: fmt.Sprintf("SIGHUP pid %d failed: %v", snap.MainPID, err)}
		return m, true
	}
	outcome, err := m.lc.deps.Read(snap.MainPID)
	switch {
	case err != nil:
		m.lc.banner = lifecycleBanner{kind: bannerReloadError, text: fmt.Sprintf("reload outcome read failed: %v", err)}
	case outcome == collect.ReloadOK:
		m.lc.banner = lifecycleBanner{kind: bannerReloadOK, text: fmt.Sprintf("%s reloaded (SIGHUP pid %d)", snap.Unit, snap.MainPID)}
	default:
		m.lc.banner = lifecycleBanner{kind: bannerReloadRejected, text: fmt.Sprintf("%s REJECTED the reload — keeps its old config", snap.Unit)}
	}
	return m, true
}

// handleConfirmKey processes keys while the confirm dialog is up.
// Enter executes EXACTLY one systemctl call via the runner; Esc returns to
// the dashboard with ZERO runner calls. Destructive action runs ONLY after
// confirm (fail-closed).
func (m dashboardModel) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	c := m.lc.confirm
	switch msg.String() {
	case "esc":
		// Cancel: zero runner calls, back to dashboard.
		m.mode = modeDashboard
		m.lc.confirm = confirmModel{}
		m.lc.banner = lifecycleBanner{kind: bannerInfo, text: "cancelled — nothing done"}
		return m, nil, true
	case "enter":
		var err error
		switch c.action {
		case collect.LifecycleStop:
			err = collect.StopInstance(m.lc.deps.Runner, c.unit)
		case collect.LifecycleRestart:
			err = collect.RestartInstance(m.lc.deps.Runner, c.unit)
		default:
			err = fmt.Errorf("unknown lifecycle action %d", c.action)
		}
		if err != nil {
			m.lc.banner = lifecycleBanner{kind: bannerLifecycleError, text: err.Error()}
		} else {
			m.lc.banner = lifecycleBanner{kind: bannerLifecycleOK, text: fmt.Sprintf("%s %s — accepted", c.action, c.unit)}
		}
		m.mode = modeDashboard
		m.lc.confirm = confirmModel{}
		return m, nil, true
	}
	// Any other key: stay in the dialog (nothing runs).
	return m, nil, true
}

// renderConfirm renders the confirm dialog overlay.
func (c confirmModel) render() string {
	var b strings.Builder
	b.WriteString("CONFIRM\n")
	b.WriteString(strings.Repeat("─", 78) + "\n")
	fmt.Fprintf(&b, "%s\n", c.prompt())
	b.WriteString("footer: Enter confirm · Esc cancel\n")
	return b.String()
}

// renderLifecycleBanner renders the lifecycle outcome banner (or "").
func (m dashboardModel) renderLifecycleBanner() string {
	switch m.lc.banner.kind {
	case bannerLifecycleOK:
		return "banner: ✓ " + m.lc.banner.text
	case bannerLifecycleError:
		return "banner: ✗ " + m.lc.banner.text
	case bannerReloadOK:
		return "banner: ✓ " + m.lc.banner.text
	case bannerReloadRejected:
		return "banner: ⚠ " + m.lc.banner.text
	case bannerReloadError:
		return "banner: ✗ " + m.lc.banner.text
	case bannerLifecycleDisabled:
		return "banner: (disabled) " + m.lc.banner.text
	case bannerInfo:
		if m.lc.banner.text == "" {
			return ""
		}
		return "banner: " + m.lc.banner.text
	}
	return ""
}
