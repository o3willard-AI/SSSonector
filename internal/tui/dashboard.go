// Package tui hosts the bubbletea dashboard Model (Phase 2 WI 2.3):
// composition of the internal/tui/view panel renderers into the full
// screen, wired to an injectable poll seam.
//
// No keymap/navigation yet (WI 2.4). The tick is message-driven — Init
// issues the first tick command, Update calls the poll seam and re-issues
// the next tick — so tests drive ticks directly with no sleeps.
package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
	"github.com/o3willard-AI/SSSonector/internal/tui/view"
)

// bubbleteaVersion is the pinned bubbletea version (go.mod).
const bubbleteaVersion = "v1.3.6"

// PollFunc is the injectable poll seam: one full dashboard tick.
type PollFunc func(ctx context.Context) collect.TickResult

// tickMsg drives one poll+render cycle (message-driven, no sleeps).
type tickMsg struct{}

// focusRegion tracks which region owns the navigation keys.
type focusRegion int

const (
	focusRail focusRegion = iota
	focusPanels
)

// defaultRefresh is the spec refresh interval (docs/tui.md §6: 1s).
const defaultRefresh = time.Second

// dashboardModel is the full-screen dashboard (WI 2.3 composition + WI 2.4
// read-only navigation + WI 2.5 timed re-tick).
type dashboardModel struct {
	poll     PollFunc
	now      func() time.Time // injectable clock (golden determinism)
	refresh  time.Duration    // re-tick interval (default 1s via tea.Tick)
	result   collect.TickResult
	focus    string // focused instance name (whose panels render)
	selected int    // highlighted rail row index
	region   focusRegion
	polled   bool
	lastNow  time.Time
	mode     viewMode
	config   configModel
	deps     ConfigDeps
	lc       lifecycleState
	logs     LogTailFunc // merged-log seam (WI 4.3)
}

// LogTailFunc is the merged-log seam: given the tick result and a bound,
// return the merged [inst]-tagged stream. Production wires a LogTail over
// the real CommandRunner; tests inject fixture entries.
type LogTailFunc func(res collect.TickResult, n int) []collect.LogEntry

// defaultLogTail is the production merge: one LogTail over the real
// OSCommandRunner, merging every discovered instance's unit journal into
// one [inst]-tagged time-ordered stream.
var productionLogTail = collect.NewLogTail(collect.OSCommandRunner{})

func defaultLogTail(res collect.TickResult, n int) []collect.LogEntry {
	return allLogs(res, n)
}

// lifecycleState holds WI 4.1 lifecycle deps + confirm dialog + banner.
type lifecycleState struct {
	deps    LifecycleDeps
	confirm confirmModel
	banner  lifecycleBanner
}

// NewDashboard builds the dashboard model with the given poll seam and the
// default 1s refresh (docs/tui.md §6). Tests pass their own clock; the
// refresh stays message-driven — tests drive tickMsg directly and never
// depend on the timer firing.
func NewDashboard(poll PollFunc, now func() time.Time) dashboardModel {
	return NewDashboardWithRefresh(poll, now, defaultRefresh)
}

// NewDashboardWithRefresh builds the dashboard with an explicit refresh
// interval (the re-tick waits this long between polls).
func NewDashboardWithRefresh(poll PollFunc, now func() time.Time, refresh time.Duration) dashboardModel {
	if refresh <= 0 {
		refresh = defaultRefresh
	}
	return dashboardModel{poll: poll, now: now, refresh: refresh, logs: defaultLogTail}
}

// NewDashboardWithConfig builds the dashboard with the config-view
// dependencies wired (WI 3.4). Production (cmd/daemon/tui.go) passes the
// real dump/validate/apply + collect.SignalHUP; the rig-gate wiring gap
// (zero-value deps => `c` silently did nothing) is structurally prevented
// because callers must supply ConfigDeps explicitly.
func NewDashboardWithConfig(poll PollFunc, now func() time.Time, refresh time.Duration, deps ConfigDeps) dashboardModel {
	m := NewDashboardWithRefresh(poll, now, refresh)
	m.deps = deps
	return m
}

// NewDashboardWithLifecycle builds the dashboard with BOTH the config-view
// and lifecycle deps wired (WI 4.1). Production passes the real
// OSCommandRunner + SignalHUP + LogReloadReader; tests inject fakes.
func NewDashboardWithLifecycle(poll PollFunc, now func() time.Time, refresh time.Duration, deps ConfigDeps, lc LifecycleDeps) dashboardModel {
	m := NewDashboardWithConfig(poll, now, refresh, deps)
	m.lc.deps = lc
	return m
}

// WithLogTail overrides the merged-log seam (WI 4.3). Production passes
// a LogTail over the real CommandRunner; tests inject fixture streams.
// The zero value keeps defaultLogTail (the merge over allLogs).
func (m dashboardModel) WithLogTail(f LogTailFunc) dashboardModel {
	if f != nil {
		m.logs = f
	}
	return m
}

// WithFocus pre-focuses an instance (WI 5.3: the wizard lands on the
// dashboard focused on the newly created instance). The focus applies on
// the first tick; empty is a no-op (default selection). Unknown names are
// ignored by the tick logic (it never fabricates an instance).
func (m dashboardModel) WithFocus(name string) dashboardModel {
	if name != "" {
		m.focus = name
		m.selected = -1 // set on the first tick from the sorted rail
	}
	return m
}

// Init issues the first tick (message-driven).
func (m dashboardModel) Init() tea.Cmd {
	return tickNow(m.refresh)
}

// tickNow returns the command that emits the next tick after the model's
// refresh interval (tea.Tick — no busy loop). The message type is unchanged
// (tickMsg), so tests keep driving ticks directly without a timer.
func tickNow(refresh time.Duration) tea.Cmd {
	return tea.Tick(refresh, func(time.Time) tea.Msg { return tickMsg{} })
}

// Update handles ticks and read-only keys (WI 2.4 subset of spec §4:
// navigation + tab + q; `c` reserved no-op for Phase 3; NO destructive
// actions s/r/R in this WI).
func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		m.result = m.poll(context.Background())
		m.lastNow = m.now()
		m.polled = true
		// Keep selection/focus stable across ticks; initialize on first
		// tick only. A pre-focus (WithFocus, WI 5.3) also snaps the
		// selection to the focused row on that first tick.
		names := sortedInstanceNames(m.result)
		if m.focus == "" && len(names) > 0 {
			m.focus = names[0]
		}
		if m.selected < 0 && len(names) > 0 {
			for i, n := range names {
				if n == m.focus {
					m.selected = i
					break
				}
			}
			if m.selected < 0 {
				m.selected = 0 // pre-focused name not discovered: default
			}
		}
		return m, tickNow(m.refresh) // re-issue next tick (after refresh interval)
	case tea.KeyMsg:
		if m.mode == modeConfig {
			next, cmd, _ := m.handleConfigKey(msg)
			return next, cmd
		}
		if m.mode == modeConfirm {
			next, cmd, _ := m.handleConfirmKey(msg)
			return next, cmd
		}
		// WI 4.1 lifecycle keys first (s/r/R on the focused instance).
		if next, handled := m.handleLifecycleKey(msg); handled {
			return next, nil
		}
		return m.handleKey(msg)
	default:
		return m, nil
	}
}

// handleKey applies the read-only keymap. Selection movement CLAMPS at the
// ends (documented choice: no wrap, so PgUp/PgDn paging is predictable).
func (m dashboardModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	names := sortedInstanceNames(m.result)
	switch msg.String() {
	case "tab":
		if m.region == focusRail {
			m.region = focusPanels
		} else {
			m.region = focusRail
		}
		return m, nil
	case "q", "ctrl+c":
		return m, tea.Quit
	case "c":
		// Open the config view for the focused instance (WI 3.4).
		if snap := m.focusedSnapshot(); snap != nil {
			cm, err := openConfig(m.deps, snap.Name)
			if err != nil {
				return m, nil // dump error: stay on dashboard (banner via probe/log tail)
			}
			m.mode = modeConfig
			m.config = cm
		}
		return m, nil
	}

	// Rail-selection keys act ONLY when the rail is focused.
	if m.region != focusRail {
		return m, nil
	}
	switch msg.String() {
	case "up", "k":
		if m.selected > 0 {
			m.selected--
		}
	case "down", "j":
		if m.selected < len(names)-1 {
			m.selected++
		}
	case "pgup":
		m.selected = 0
	case "pgdown":
		if len(names) > 0 {
			m.selected = len(names) - 1
		}
	case "enter":
		if m.selected >= 0 && m.selected < len(names) {
			m.focus = names[m.selected]
		}
	}
	return m, nil
}

// focusedSnapshot returns the focused instance's snapshot.
func (m dashboardModel) focusedSnapshot() *collect.InstanceSnapshot {
	if m.focus == "" {
		return nil
	}
	return m.result.Instances[m.focus]
}

// View composes the full screen from the WI 2.2 renderers (or the config
// overlay / confirm dialog when in those modes).
func (m dashboardModel) View() string {
	if m.mode == modeConfig && m.polled {
		return m.config.render(m.focus)
	}
	if m.mode == modeConfirm && m.polled {
		return m.lc.confirm.render()
	}
	var b strings.Builder

	fmt.Fprintf(&b, "SSSonector tui (bubbletea %s)\n", bubbleteaVersion)

	if !m.polled {
		b.WriteString("waiting for first tick…\n")
		return b.String()
	}

	now := m.lastNow

	// Fatal discovery failure: explicit error, no fabricated screen.
	if m.result.DiscoverErr != nil {
		fmt.Fprintf(&b, "discovery: ERROR: %v\n", m.result.DiscoverErr)
		b.WriteString("footer: [q]uit\n")
		return b.String()
	}

	names := sortedInstanceNames(m.result)
	if len(names) == 0 {
		b.WriteString(view.RenderRail(view.RailInput{}, now))
		b.WriteString("footer: [q]uit\n")
		return b.String()
	}

	// Client mode (len==1): same code path; the rail renders one fixed row
	// and the focused-instance banner is omitted (spec §3.2).
	clientMode := len(names) == 1

	focus := m.focus
	snap := m.focusedSnapshot()
	if snap == nil {
		focus = names[0]
		snap = m.result.Instances[focus]
	}

	// Daemon header: liveness from HEALTHZ (platform-neutral); systemd
	// MainPID is still shown when > 0 but does NOT drive liveness.
	b.WriteString(view.RenderDaemonHeader(snap.Healthz, snap.MainPID))
	b.WriteString("\n")

	// Rail: collapsed in client mode (no focus marker), full in server mode.
	if !clientMode {
		fmt.Fprintf(&b, "\n--- focused: %s ---\n", focus)
		in := railInputFor(m.result, focus, now)
		in.FocusIdx = m.selected
		b.WriteString(view.RenderRail(in, now))
	} else {
		b.WriteString("\n")
		b.WriteString(view.RenderRail(railInputSingle(snap, now), now))
	}

	// Focused instance's panels.
	b.WriteString("\n")
	b.WriteString(view.RenderTunnel(snap.Healthz, snap.Metrics, tunAddrOf(snap), stateSince(snap, now), now))
	b.WriteString("\n")
	b.WriteString(view.RenderNAT(snap.Metrics))
	b.WriteString("\n")
	b.WriteString(view.RenderRate(snap.Metrics))
	b.WriteString("\n")

	// Shared panels: cert from the focused instance, logs merged across
	// ALL instances (WI 4.3).
	b.WriteString(view.RenderCert(snap.Cert, now))
	b.WriteString("\n")
	b.WriteString(view.RenderLog(m.logs(m.result, 12), 12))

	// Lifecycle outcome banner (WI 4.1), above the footer.
	if line := m.renderLifecycleBanner(); line != "" {
		b.WriteString("\n" + line + "\n")
	}

	// Footer (WI 4.1): lifecycle actions on the FOCUSED instance.
	b.WriteString("\nfooter: [s]top [r]estart [R]eload act on FOCUSED instance  [c]onfig  [q]uit\n")
	return b.String()
}

// railInputFor assembles rail input for server mode, marking focus.
func railInputFor(res collect.TickResult, focus string, now time.Time) view.RailInput {
	in := view.RailInput{
		TUNAddresses:   map[string]string{},
		ListenPorts:    map[string]int{},
		PeerCounts:     map[string]int{},
		LastPeerChange: map[string]time.Time{},
	}
	i := 0
	for _, name := range sortedInstanceNames(res) {
		snap := res.Instances[name]
		in.Instances = append(in.Instances, collect.InstanceState{
			Name: snap.Name, Unit: snap.Unit,
			ActiveState: snap.ActiveState, SubState: snap.SubState, MainPID: snap.MainPID,
		})
		if name == focus {
			in.FocusIdx = i
		}
		if snap.TunAddr != "" {
			in.TUNAddresses[name] = snap.TunAddr
		}
		if snap.ListenPort > 0 {
			in.ListenPorts[name] = snap.ListenPort
		}
		if in.HealthStatus == nil {
			in.HealthStatus = map[string]collect.SourceStatus{}
		}
		in.HealthStatus[name] = snap.Healthz.Status()
		if m, ok := snap.Metrics.Get(); ok {
			if v, ok2 := m.Connections.Active.Get(); ok2 {
				in.PeerCounts[name] = int(v)
			}
		}
		i++
	}
	return in
}

// railInputSingle assembles the collapsed single-instance rail (client
// mode): one fixed row, no focus marker (FocusIdx -1), and the healthz
// status carried through so the dot agrees with the daemon header (WI 4.2
// — both derive from the same healthz source).
func railInputSingle(snap *collect.InstanceSnapshot, now time.Time) view.RailInput {
	in := view.RailInput{
		Instances: []collect.InstanceState{{
			Name: snap.Name, Unit: snap.Unit,
			ActiveState: snap.ActiveState, SubState: snap.SubState, MainPID: snap.MainPID,
		}},
		FocusIdx:       -1, // collapsed mode: no focus marker
		TUNAddresses:   map[string]string{},
		ListenPorts:    map[string]int{},
		PeerCounts:     map[string]int{},
		LastPeerChange: map[string]time.Time{},
		HealthStatus:   map[string]collect.SourceStatus{snap.Name: snap.Healthz.Status()},
	}
	if snap.TunAddr != "" {
		in.TUNAddresses[snap.Name] = snap.TunAddr
	}
	if snap.ListenPort > 0 {
		in.ListenPorts[snap.Name] = snap.ListenPort
	}
	if m, ok := snap.Metrics.Get(); ok {
		if v, ok2 := m.Connections.Active.Get(); ok2 {
			in.PeerCounts[snap.Name] = int(v)
		}
	}
	return in
}

// tunAddrOf returns the instance's TUN address from the snapshot.
func tunAddrOf(snap *collect.InstanceSnapshot) string {
	if snap == nil {
		return ""
	}
	return snap.TunAddr
}

// stateSince derives the tunnel state age from uptime (uptime_seconds)
// against the injected clock (determinism).
func stateSince(snap *collect.InstanceSnapshot, now time.Time) time.Time {
	h, ok := snap.Healthz.Get()
	if !ok || h.UptimeSeconds <= 0 {
		return time.Time{}
	}
	return now.Add(-time.Duration(h.UptimeSeconds) * time.Second)
}

// allLogs merges every discovered instance's log tail into ONE
// time-ordered, [inst]-tagged stream (WI 4.3). Each unit is read through
// the LogTail runner seam (injectable — tests pass fakes); a unit that
// fails contributes nothing. n bounds the merged stream.
func allLogs(res collect.TickResult, n int) []collect.LogEntry {
	var units []string
	for _, name := range sortedInstanceNames(res) {
		if u := res.Instances[name].Unit; u != "" {
			units = append(units, u)
		}
	}
	return collect.MergeLogs(productionLogTail, units, n)
}

// sortedInstanceNames returns instance names sorted for determinism.
func sortedInstanceNames(res collect.TickResult) []string {
	names := make([]string, 0, len(res.Instances))
	for n := range res.Instances {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
