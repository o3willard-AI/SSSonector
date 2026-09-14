package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

// WI 5.3: WithFocus pre-focuses an instance (the wizard lands on the
// dashboard focused on the newly created instance).
func TestWizard53_WithFocus_PreFocusesDashboard(t *testing.T) {
	m := NewDashboard(func(context.Context) collect.TickResult {
		return fixtureTickResult([]string{"client-a", "client-b"})
	}, func() time.Time { return screenNow }).WithFocus("client-b")
	m.Init()
	m = driveTick(m)
	if m.focus != "client-b" {
		t.Errorf("dashboard must pre-focus client-b, got %q", m.focus)
	}
	if !strings.Contains(m.View(), "--- focused: client-b ---") {
		t.Errorf("panels must render the pre-focused instance:\n%s", m.View())
	}
	// Selection follows the focused row (client-b is index 1).
	if m.selected != 1 {
		t.Errorf("selection must track the pre-focus: %d", m.selected)
	}
}

// TestWizard53_WithFocus_EmptyIsDefault: empty focus keeps the default
// (first sorted instance).
func TestWizard53_WithFocus_EmptyIsDefault(t *testing.T) {
	m := NewDashboard(func(context.Context) collect.TickResult {
		return fixtureTickResult([]string{"client-a", "client-b"})
	}, func() time.Time { return screenNow }).WithFocus("")
	m.Init()
	m = driveTick(m)
	if m.focus != "client-a" {
		t.Errorf("empty focus must keep the default, got %q", m.focus)
	}
}
