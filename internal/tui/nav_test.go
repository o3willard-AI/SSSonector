package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

// navFixture: model with 3 instances already polled.
func navFixture(t *testing.T) dashboardModel {
	t.Helper()
	m := NewDashboard(func(context.Context) collect.TickResult {
		return fixtureTickResult([]string{"client-a", "client-b", "client-c"})
	}, func() time.Time { return screenNow })
	m.Init()
	return driveTick(m)
}

func key(m dashboardModel, s string) dashboardModel {
	next, _ := m.Update(tea.KeyMsg{Type: keyTypeFor(s), Runes: runesFor(s)})
	return next.(dashboardModel)
}

func keyTypeFor(s string) tea.KeyType {
	switch s {
	case "up":
		return tea.KeyUp
	case "down":
		return tea.KeyDown
	case "pgup":
		return tea.KeyPgUp
	case "pgdown":
		return tea.KeyPgDown
	case "enter":
		return tea.KeyEnter
	case "tab":
		return tea.KeyTab
	case "esc":
		return tea.KeyEsc
	default:
		return tea.KeyRunes
	}
}

func runesFor(s string) []rune {
	switch s {
	case "up", "down", "pgup", "pgdown", "enter", "tab", "esc":
		return nil
	default:
		return []rune(s)
	}
}

func instanceNames(m dashboardModel) []string { return sortedInstanceNames(m.result) }

func TestNav_DownUpClamp(t *testing.T) {
	m := navFixture(t)
	// Documented choice: CLAMP at ends, no wrap.
	if m.selected != 0 {
		t.Fatalf("initial selected: %d", m.selected)
	}
	m = key(m, "up")
	if m.selected != 0 {
		t.Errorf("up at top must clamp: %d", m.selected)
	}
	m = key(m, "down")
	if m.selected != 1 {
		t.Errorf("down: %d, want 1", m.selected)
	}
	m = key(m, "down")
	m = key(m, "down")
	if m.selected != 2 {
		t.Errorf("down: %d, want 2 (last)", m.selected)
	}
	m = key(m, "down")
	if m.selected != 2 {
		t.Errorf("down at bottom must clamp: %d", m.selected)
	}
	m = key(m, "pgup")
	if m.selected != 0 {
		t.Errorf("pgup: %d, want 0", m.selected)
	}
	m = key(m, "pgdown")
	if m.selected != 2 {
		t.Errorf("pgdown: %d, want 2", m.selected)
	}
}

func TestNav_EnterFocusesSelected(t *testing.T) {
	m := navFixture(t)
	if m.focus != "client-a" {
		t.Fatalf("initial focus: %q", m.focus)
	}
	m = key(m, "down")
	m = key(m, "down")
	m = key(m, "enter")
	if m.focus != "client-c" {
		t.Errorf("enter must focus selected: got %q, want client-c", m.focus)
	}
	// Panels render the newly focused instance (client-c's fixture values).
	out := m.View()
	if !strings.Contains(out, "--- focused: client-c ---") {
		t.Errorf("view must show new focus:\n%s", out)
	}
	// Rail highlight follows selection.
	if !strings.Contains(out, "▸ client-c") {
		t.Errorf("rail must highlight selected row:\n%s", out)
	}
}

func TestNav_TabTogglesRegion(t *testing.T) {
	m := navFixture(t)
	if m.region != focusRail {
		t.Fatalf("initial region must be rail")
	}
	m = key(m, "tab")
	if m.region != focusPanels {
		t.Errorf("tab must switch to panels: %v", m.region)
	}
	m = key(m, "tab")
	if m.region != focusRail {
		t.Errorf("tab must toggle back to rail: %v", m.region)
	}
}

func TestNav_PanelsFocus_InertsRailKeys(t *testing.T) {
	m := navFixture(t)
	m = key(m, "down") // selected 1
	m = key(m, "tab")  // panels focused
	for i := 0; i < 5; i++ {
		m = key(m, "down")
	}
	m = key(m, "up")
	m = key(m, "enter")
	if m.selected != 1 {
		t.Errorf("panels-focused rail keys must be inert: selected=%d", m.selected)
	}
	if m.focus != "client-a" {
		t.Errorf("enter while panels focused must not change focus: %q", m.focus)
	}
	if m.region != focusPanels {
		t.Errorf("region must stay panels: %v", m.region)
	}
	// Tab still acts.
	m = key(m, "tab")
	if m.region != focusRail {
		t.Errorf("tab must still toggle: %v", m.region)
	}
}

func TestNav_Quits(t *testing.T) {
	m := navFixture(t)
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("q must return tea.Quit")
	}
	if msg := cmd(); msg != tea.Quit() {
		t.Errorf("q cmd must produce quit, got %T", msg)
	}
	// ctrl+c quits too.
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyCtrlC})
	if cmd == nil {
		t.Error("ctrl+c must return tea.Quit")
	}
}

func TestNav_CIsNoOp(t *testing.T) {
	m := navFixture(t)
	before := m
	m = key(m, "c")
	if m.selected != before.selected || m.focus != before.focus || m.region != before.region {
		t.Errorf("c must be a no-op: %+v -> %+v", before, m)
	}
}

func TestNav_SingleInstance_Inert(t *testing.T) {
	m := NewDashboard(func(context.Context) collect.TickResult {
		return fixtureTickResult([]string{"default"})
	}, func() time.Time { return screenNow })
	m.Init()
	m = driveTick(m)
	// Rail has one row; selection must not move.
	m = key(m, "down")
	if m.selected != 0 {
		t.Errorf("single-row selection must be inert: %d", m.selected)
	}
	m = key(m, "up")
	if m.selected != 0 {
		t.Errorf("single-row up must be inert: %d", m.selected)
	}
	// Enter keeps focus on the only instance.
	m = key(m, "enter")
	if m.focus != "default" {
		t.Errorf("focus must stay default: %q", m.focus)
	}
	if len(instanceNames(m)) != 1 {
		t.Fatalf("fixture should be single-instance")
	}
}

func TestNav_FocusStableAcrossTicks(t *testing.T) {
	m := navFixture(t)
	m = key(m, "down")
	m = key(m, "down")
	m = key(m, "enter") // focus client-c
	m = driveTick(m)    // next tick
	if m.focus != "client-c" {
		t.Errorf("focus must survive ticks: %q", m.focus)
	}
	if m.selected != 2 {
		t.Errorf("selection must survive ticks: %d", m.selected)
	}
}

func TestNav_FooterReadOnly(t *testing.T) {
	m := navFixture(t)
	out := m.View()
	if strings.Contains(out, "[s]top") || strings.Contains(out, "[r]estart") || strings.Contains(out, "[R]eload") {
		t.Errorf("footer must not show destructive actions in WI 2.4:\n%s", out)
	}
	if !strings.Contains(out, "[c]onfig") || !strings.Contains(out, "[q]uit") {
		t.Errorf("footer must show read-only keys:\n%s", out)
	}
}
