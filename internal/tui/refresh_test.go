package tui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

// TestDashboard_RefreshIsTeaTick guards the no-busy-loop requirement (WI
// 2.5): the re-tick uses tea.Tick at the 1s default, so the message does
// not arrive synchronously (tests still drive tickMsg directly).
func TestDashboard_RefreshIsTeaTick(t *testing.T) {
	m := NewDashboard(func(context.Context) collect.TickResult {
		return fixtureTickResult([]string{"default"})
	}, func() time.Time { return screenNow })
	if m.refresh != time.Second {
		t.Errorf("default refresh: %v, want 1s", m.refresh)
	}
	m.Init()
	m = driveTick(m)
	_, cmd := m.Update(tickMsg{})
	if cmd == nil {
		t.Fatal("re-tick command must exist")
	}
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case msg := <-done:
		t.Fatalf("re-tick fired immediately — busy loop: %T", msg)
	case <-time.After(50 * time.Millisecond):
		// correct: tea.Tick still waiting for the refresh interval
	}
}

func TestDashboard_CustomRefresh(t *testing.T) {
	m := NewDashboardWithRefresh(func(context.Context) collect.TickResult {
		return collect.TickResult{}
	}, func() time.Time { return screenNow }, 2*time.Second)
	if m.refresh != 2*time.Second {
		t.Errorf("custom refresh: %v", m.refresh)
	}
	// Zero/negative falls back to the 1s default.
	m = NewDashboardWithRefresh(func(context.Context) collect.TickResult {
		return collect.TickResult{}
	}, func() time.Time { return screenNow }, 0)
	if m.refresh != time.Second {
		t.Errorf("fallback refresh: %v, want 1s", m.refresh)
	}
}
