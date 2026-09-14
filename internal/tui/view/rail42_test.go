package view

import (
	"strings"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

// WI 4.2: rail-completion goldens — 1/2/4 instance collapses and counts,
// mixed healthz states per row, plus the sorted-order contract asserted
// directly (fails if the sort is dropped from the display path).

// rail42Input builds a RailInput for the given names: index 0 gets ok
// healthz, index 1 error, index 2+ absent. Focus stays on row 0.
func rail42Input(names []string) RailInput {
	in := RailInput{
		FocusIdx:       0,
		TUNAddresses:   map[string]string{},
		ListenPorts:    map[string]int{},
		PeerCounts:     map[string]int{},
		LastPeerChange: map[string]time.Time{},
		HealthStatus:   map[string]collect.SourceStatus{},
	}
	for i, name := range names {
		in.Instances = append(in.Instances, collect.InstanceState{
			Name: name, Unit: "sssonector@" + name + ".service",
			ActiveState: "active", SubState: "running", MainPID: 8100 + i,
		})
		in.TUNAddresses[name] = "10.77.0." + itoa(i+1) + "/24"
		in.ListenPorts[name] = 9443 + i
		in.PeerCounts[name] = 2 + i
		in.LastPeerChange[name] = fixedNow.Add(-time.Duration(41+i) * time.Minute)
		switch i {
		case 0:
			in.HealthStatus[name] = collect.StatusOK
		case 1:
			in.HealthStatus[name] = collect.StatusError
		default:
			in.HealthStatus[name] = collect.StatusAbsent
		}
	}
	return in
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

// TestRail42_CountGoldens: 1 (client collapse — one row, no marker),
// 2 and 4 instance rails.
func TestRail42_CountGoldens(t *testing.T) {
	assertGolden(t, "rail42_one", RenderRail(rail42Input([]string{"default"}), fixedNow))
	assertGolden(t, "rail42_two", RenderRail(rail42Input([]string{"client-a", "client-b"}), fixedNow))
	assertGolden(t, "rail42_four", RenderRail(rail42Input([]string{"client-a", "client-b", "client-c", "client-d"}), fixedNow))
}

// TestRail42_MixedStates_Golden: one up (●), one down (✖ errored healthz
// — unreachable per §7.1), one never-seen (✖ absent) — each row shows its
// correct dot and word.
func TestRail42_MixedStates_Golden(t *testing.T) {
	in := rail42Input([]string{"aaa-up", "bbb-err", "ccc-never"})
	// aaa-up is ● up; bbb-err ✖ down; ccc-never ✖ never (defaults).
	out := RenderRail(in, fixedNow)
	assertGolden(t, "rail42_mixed", out)
	if got := railLineFor2(t, out, "aaa-up"); !strings.Contains(got, "●") || !strings.Contains(got, "up") {
		t.Errorf("up row wrong: %q", got)
	}
	if got := railLineFor2(t, out, "bbb-err"); !strings.Contains(got, "✖") || !strings.Contains(got, "down") {
		t.Errorf("down row wrong: %q", got)
	}
	if got := railLineFor2(t, out, "ccc-never"); !strings.Contains(got, "✖") || !strings.Contains(got, "never") {
		t.Errorf("never row wrong: %q", got)
	}
}

// TestRail42_SortedRows_Direct: the display path must sort rows by NAME.
// Feed instances in reverse order; the rendered order must be forward.
// This is the direct assertion backing the golden contracts (a reverted
// sort fails here even if a golden is regenerated).
func TestRail42_SortedRows_Direct(t *testing.T) {
	in := rail42Input([]string{"zeta", "alpha", "mid"})
	out := RenderRail(in, fixedNow)
	var got []string
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "▸ ") || strings.HasPrefix(line, "  ") {
			f := fields0(strings.TrimPrefix(strings.TrimPrefix(line, "▸ "), "  "))
			if f != "" {
				got = append(got, f)
			}
		}
	}
	want := "alpha,mid,zeta"
	if strings.Join(got, ",") != want {
		t.Errorf("rail must render name-sorted: got %v, want %s\n%s", got, want, out)
	}
}

func fields0(s string) string {
	for i, r := range s {
		if r == ' ' {
			return s[:i]
		}
	}
	return s
}

// railLineFor2 returns the rail line containing the instance name (tui
// package has its own copy; this one is view-local).
func railLineFor2(t *testing.T, rail, name string) string {
	t.Helper()
	for _, line := range strings.Split(rail, "\n") {
		if strings.Contains(line, name) {
			return line
		}
	}
	t.Fatalf("no rail line for %q in:\n%s", name, rail)
	return ""
}
