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
	return dashboardModel{poll: poll, now: now, refresh: refresh}
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
		// tick only.
		names := sortedInstanceNames(m.result)
		if m.focus == "" && len(names) > 0 {
			m.focus = names[0]
		}
		return m, tickNow(m.refresh) // re-issue next tick (after refresh interval)
	case tea.KeyMsg:
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
		// Reserved for the config view (Phase 3): accepted no-op now.
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

// View composes the full screen from the WI 2.2 renderers.
func (m dashboardModel) View() string {
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

	// Daemon header from the focused instance's systemd state.
	b.WriteString(view.RenderDaemonHeader(collect.InstanceState{
		Name:        snap.Name,
		Unit:        snap.Unit,
		ActiveState: snap.ActiveState,
		SubState:    snap.SubState,
		MainPID:     snap.MainPID,
	}))
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

	// Shared panels: cert from the focused instance, logs from all.
	b.WriteString(view.RenderCert(snap.Cert, now))
	b.WriteString("\n")
	b.WriteString(view.RenderLog(allLogs(m.result), 12))

	// Read-only footer (no destructive actions yet — WI 2.4+/3).
	b.WriteString("\nfooter: [c]onfig [q]uit (read-only)\n")
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
		if m, ok := snap.Metrics.Get(); ok {
			if v, ok2 := m.Connections.Active.Get(); ok2 {
				in.PeerCounts[name] = int(v)
			}
		}
		i++
	}
	return in
}

// railInputSingle assembles the collapsed single-instance rail (client mode).
func railInputSingle(snap *collect.InstanceSnapshot, now time.Time) view.RailInput {
	in := view.RailInput{
		Instances: []collect.InstanceState{{
			Name: snap.Name, Unit: snap.Unit,
			ActiveState: snap.ActiveState, SubState: snap.SubState, MainPID: snap.MainPID,
		}},
		TUNAddresses:   map[string]string{},
		ListenPorts:    map[string]int{},
		PeerCounts:     map[string]int{},
		LastPeerChange: map[string]time.Time{},
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

// allLogs merges nothing yet — the log panel is fed from LogTail reads in
// WI 1.6's probe; for the screen the entries come with the tick result in
// a later wiring. Until then the panel renders its empty state.
func allLogs(res collect.TickResult) []collect.LogEntry {
	return nil
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
