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

// dashboardModel is the full-screen dashboard (WI 2.3: composition only).
type dashboardModel struct {
	poll    PollFunc
	now     func() time.Time // injectable clock (golden determinism)
	result  collect.TickResult
	focus   string // focused instance name (first instance in 2.3)
	polled  bool
	lastNow time.Time
}

// NewDashboard builds the dashboard model with the given poll seam.
func NewDashboard(poll PollFunc, now func() time.Time) dashboardModel {
	return dashboardModel{poll: poll, now: now}
}

// Init issues the first tick (message-driven).
func (m dashboardModel) Init() tea.Cmd {
	return tickNow()
}

// tickNow returns the command that emits the next tick.
func tickNow() tea.Cmd {
	return func() tea.Msg { return tickMsg{} }
}

// Update handles ticks (and only ticks — keymap is WI 2.4).
func (m dashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg.(type) {
	case tickMsg:
		m.result = m.poll(context.Background())
		m.lastNow = m.now()
		m.polled = true
		m.focus = firstInstanceName(m.result)
		return m, tickNow() // re-issue next tick
	default:
		return m, nil
	}
}

// firstInstanceName picks the focused instance (sorted for determinism;
// WI 2.4 makes it selectable).
func firstInstanceName(res collect.TickResult) string {
	if len(res.Instances) == 0 {
		return ""
	}
	names := make([]string, 0, len(res.Instances))
	for n := range res.Instances {
		names = append(names, n)
	}
	sort.Strings(names)
	return names[0]
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
		b.WriteString("footer: [c]onfig [q]uit\n")
		return b.String()
	}

	names := sortedInstanceNames(m.result)
	if len(names) == 0 {
		b.WriteString(view.RenderRail(view.RailInput{}, now))
		b.WriteString("footer: [c]onfig [q]uit\n")
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
		b.WriteString(view.RenderRail(railInputFor(m.result, focus, now), now))
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
		if snap.PrometheusAddr != "" {
			in.TUNAddresses[name] = snap.PrometheusAddr // placeholder until a TUN series exists
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
		PeerCounts:     map[string]int{},
		LastPeerChange: map[string]time.Time{},
	}
	if snap.PrometheusAddr != "" {
		in.TUNAddresses[snap.Name] = snap.PrometheusAddr
	}
	if m, ok := snap.Metrics.Get(); ok {
		if v, ok2 := m.Connections.Active.Get(); ok2 {
			in.PeerCounts[snap.Name] = int(v)
		}
	}
	return in
}

// tunAddrOf returns the instance's TUN address if carried on the snapshot
// (via PrometheusAddr placeholder for now — no TUN series exists yet).
func tunAddrOf(snap *collect.InstanceSnapshot) string {
	if snap == nil || snap.PrometheusAddr == "" {
		return ""
	}
	return snap.PrometheusAddr
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
