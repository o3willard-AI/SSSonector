package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

// WI 4.2: rail completion tests — stable name order, healthz-derived dots
// agreeing with the daemon header, per-instance panel re-render on focus
// switch, shared CERT panel, client-mode collapse. All against fixtures —
// no real collectors.

// railModelFixture builds a lifecycle-free dashboard fed by a fixed tick
// over the given instance names (indexes 1+ render "down" like
// fixtureTickResult; index 0 renders up).
func railModelFixture(names []string) dashboardModel {
	m := NewDashboard(func(context.Context) collect.TickResult {
		return fixtureTickResult(names)
	}, func() time.Time { return screenNow })
	m.Init()
	return driveTick(m)
}

// TestRail42_NameSortedRows: the rail renders instances in stable NAME
// order regardless of discovery order. The fixture feeds discovery in a
// non-alphabetical order; the rendered rows must still be alphabetical.
func TestRail42_NameSortedRows(t *testing.T) {
	m := railModelFixture([]string{"zeta", "alpha", "mid"})
	out := m.View()
	order := railRowOrder(t, out)
	want := []string{"alpha", "mid", "zeta"}
	if strings.Join(order, ",") != strings.Join(want, ",") {
		t.Errorf("rail rows must be name-sorted: got %v, want %v\n%s", order, want, out)
	}
}

// railRowOrder extracts the instance names in rendered rail-row order.
func railRowOrder(t *testing.T, screen string) []string {
	t.Helper()
	var order []string
	inRail := false
	for _, line := range strings.Split(screen, "\n") {
		switch {
		case strings.HasPrefix(line, "INSTANCES"):
			inRail = true
			continue
		case inRail && (strings.HasPrefix(line, "▸ ") || strings.HasPrefix(line, "  ")):
			f := strings.Fields(strings.TrimPrefix(strings.TrimPrefix(line, "▸ "), "  "))
			if len(f) > 0 {
				order = append(order, f[0])
			}
		case inRail && line != "":
			// first blank/other line after the rows ends the rail
			if len(order) > 0 {
				inRail = strings.HasPrefix(line, "client")
			}
		}
	}
	return order
}

// TestRail42_DotAgreesWithHeader: the rail dot and the daemon header
// liveness come from the SAME healthz source. For the focused instance, a
// healthz-ok row must pair with a RUNNING header; a healthz-errored row
// with an unreachable header — no systemd-vs-healthz divergence.
func TestRail42_DotAgreesWithHeader(t *testing.T) {
	m := railModelFixture([]string{"aaa-up", "zzz-err"})
	// Focus the up instance (default): dot ● + header RUNNING.
	out := m.View()
	if !strings.Contains(out, "daemon: RUNNING") {
		t.Errorf("healthz-ok focus must render RUNNING header:\n%s", out)
	}
	railLine := railLineFor(t, railSection(t, out), "aaa-up")
	if !strings.Contains(railLine, "●") {
		t.Errorf("healthz-ok row must show ●: %q", railLine)
	}

	// Focus the errored instance: dot ✖ (healthz error ⇒ unreachable per
	// §7.1) + header unreachable — same source, agreeing verdicts.
	m = key(m, "down")
	m = key(m, "enter")
	out = m.View()
	if !strings.Contains(out, "daemon: unreachable") {
		t.Errorf("healthz-error focus must render unreachable header:\n%s", out)
	}
	railLine = railLineFor(t, railSection(t, out), "zzz-err")
	if !strings.Contains(railLine, "✖") {
		t.Errorf("healthz-error row must show ✖ (§7.1): %q", railLine)
	}
}

// railLineFor returns the rail line containing the instance name.
func railLineFor(t *testing.T, rail, name string) string {
	t.Helper()
	for _, line := range strings.Split(rail, "\n") {
		if strings.Contains(line, name) {
			return line
		}
	}
	t.Fatalf("no rail line for %q in:\n%s", name, rail)
	return ""
}

// railSection returns the INSTANCES block from a full screen render.
func railSection(t *testing.T, screen string) string {
	t.Helper()
	var b strings.Builder
	inRail := false
	for _, line := range strings.Split(screen, "\n") {
		if strings.HasPrefix(line, "INSTANCES") {
			inRail = true
		} else if inRail && !strings.HasPrefix(line, "▸ ") && !strings.HasPrefix(line, "  ") {
			break
		}
		if inRail {
			b.WriteString(line + "\n")
		}
	}
	if b.Len() == 0 {
		t.Fatalf("no INSTANCES section in:\n%s", screen)
	}
	return b.String()
}

// TestRail42_FocusSwitchRerendersPanels: focus movement re-renders the
// per-instance TUNNEL/NAT/RATE panels with the newly focused instance's
// values.
func TestRail42_FocusSwitchRerendersPanels(t *testing.T) {
	m := railModelFixture([]string{"one", "two"})
	// one: tun 10.77.0.1/24 (fixture index 0)
	out := m.View()
	if !strings.Contains(out, "10.77.0.1/24") {
		t.Errorf("focused panel must show instance one's tun:\n%s", out)
	}
	// Switch focus to two: tun 10.77.0.2/24 must replace it in TUNNEL.
	m = key(m, "down")
	m = key(m, "enter")
	out = m.View()
	if !strings.Contains(out, "--- focused: two ---") {
		t.Fatalf("focus must move to two:\n%s", out)
	}
	if !strings.Contains(out, "10.77.0.2/24") {
		t.Errorf("TUNNEL must re-render with two's tun:\n%s", out)
	}
}

// TestRail42_CertPanelSharedAcrossFocus: the CERT panel is SHARED — it
// shows the host cert and must be byte-identical when focus switches
// between two instances sharing the host cert store. Both instances get
// the same OK cert from the fixture (index 0 renders OK; index 1+ are
// down/errored, which would not exercise the shared-OK path).
func TestRail42_CertPanelSharedAcrossFocus(t *testing.T) {
	res := fixtureTickResult([]string{"one", "two"})
	// Give both instances the identical host-cert source (shared store).
	shared := res.Instances["one"].Cert
	res.Instances["two"].Cert = shared
	m := NewDashboard(func(context.Context) collect.TickResult { return res },
		func() time.Time { return screenNow })
	m.Init()
	m = driveTick(m)
	certOne := certSection(t, m.View())
	m = key(m, "down")
	m = key(m, "enter")
	certTwo := certSection(t, m.View())
	if certOne != certTwo {
		t.Errorf("CERT panel must be byte-identical across focus:\n--- one ---\n%s\n--- two ---\n%s", certOne, certTwo)
	}
}

// certSection returns the CERTIFICATE block from a screen render.
func certSection(t *testing.T, screen string) string {
	t.Helper()
	var b strings.Builder
	inCert := false
	for _, line := range strings.Split(screen, "\n") {
		if strings.HasPrefix(line, "CERTIFICATE") {
			inCert = true
		} else if inCert && strings.TrimSpace(line) != "" && !strings.HasPrefix(line, "  ") {
			break
		}
		if inCert {
			b.WriteString(line + "\n")
		}
	}
	if b.Len() == 0 {
		t.Fatalf("no CERTIFICATE section in:\n%s", screen)
	}
	return b.String()
}

// TestRail42_ClientModeCollapse_Pinned: single-instance (client) mode
// collapses the rail to ONE fixed row and omits the focused-instance
// banner (spec §3.2).
func TestRail42_ClientModeCollapse_Pinned(t *testing.T) {
	m := railModelFixture([]string{"default"})
	out := m.View()
	if strings.Contains(out, "--- focused:") {
		t.Errorf("client mode must omit the focused banner:\n%s", out)
	}
	section := railSection(t, out)
	rows := 0
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, "▸ ") || strings.HasPrefix(line, "  ") && strings.TrimSpace(line) != "" {
			rows++
		}
	}
	if rows != 1 {
		t.Errorf("client rail must collapse to exactly one row, got %d:\n%s", rows, section)
	}
	// No focus marker in collapsed mode.
	if strings.Contains(section, "▸") {
		t.Errorf("collapsed rail must have no focus marker:\n%s", section)
	}
}
