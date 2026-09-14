package tui

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

// WI 4.3: dashboard-level merged-log tests. The LOG panel must show the
// merged [inst]-tagged stream across ALL instances (not just the focused
// one), ordered by time, stable across renders. The seam is injected —
// no real journalctl runs.

// logFixture builds a dashboard with the merged-log seam replaced by a
// fixture stream (already in merged form, as MergeLogs would return).
func logFixture(t *testing.T, merged []collect.LogEntry, names []string) dashboardModel {
	t.Helper()
	m := NewDashboard(func(context.Context) collect.TickResult {
		return fixtureTickResult(names)
	}, func() time.Time { return screenNow }).WithLogTail(func(collect.TickResult, int) []collect.LogEntry {
		return merged
	})
	m.Init()
	return driveTick(m)
}

func ts(sec int) time.Time { return screenNow.Add(time.Duration(sec) * time.Second) }

// TestLog43_MergedStreamRenderedAcrossInstances: entries from BOTH
// instances appear in the LOG panel — not just the focused one.
func TestLog43_MergedStreamRenderedAcrossInstances(t *testing.T) {
	merged := []collect.LogEntry{
		{Instance: "client-a", Timestamp: ts(1), Unit: "sssonector@client-a", Pid: 8100, Message: "a-line"},
		{Instance: "client-b", Timestamp: ts(2), Unit: "sssonector@client-b", Pid: 8101, Message: "b-line"},
	}
	m := logFixture(t, merged, []string{"client-a", "client-b"})
	out := m.View()
	if !strings.Contains(out, "[client-a] a-line") {
		t.Errorf("merged LOG must carry client-a's entry:\n%s", logSection(t, out))
	}
	if !strings.Contains(out, "[client-b] b-line") {
		t.Errorf("merged LOG must carry client-b's entry:\n%s", logSection(t, out))
	}
}

// TestLog43_TimeOrderedAcrossInstances: interleaved timestamps render in
// strict time order, regardless of which instance produced them.
func TestLog43_TimeOrderedAcrossInstances(t *testing.T) {
	merged := []collect.LogEntry{
		{Instance: "client-a", Timestamp: ts(1), Unit: "sssonector@client-a", Pid: 8100, Message: "t1-a"},
		{Instance: "client-b", Timestamp: ts(2), Unit: "sssonector@client-b", Pid: 8101, Message: "t2-b"},
		{Instance: "client-a", Timestamp: ts(3), Unit: "sssonector@client-a", Pid: 8100, Message: "t3-a"},
	}
	m := logFixture(t, merged, []string{"client-a", "client-b"})
	section := logSection(t, m.View())
	order := []string{"t1-a", "t2-b", "t3-a"}
	pos := -1
	for _, want := range order {
		i := strings.Index(section, want)
		if i < 0 {
			t.Fatalf("entry %q missing:\n%s", want, section)
		}
		if i < pos {
			t.Errorf("out of order: %q appears before the previous entry:\n%s", want, section)
		}
		pos = i
	}
}

// TestLog43_StableAcrossRenders: two renders of the same state are
// byte-identical (deterministic tie-breaks).
func TestLog43_StableAcrossRenders(t *testing.T) {
	merged := []collect.LogEntry{
		{Instance: "client-b", Timestamp: ts(2), Unit: "sssonector@client-b", Pid: 8101, Message: "tie-b"},
		{Instance: "client-a", Timestamp: ts(2), Unit: "sssonector@client-a", Pid: 8100, Message: "tie-a"},
	}
	m := logFixture(t, merged, []string{"client-a", "client-b"})
	first := logSection(t, m.View())
	second := logSection(t, m.View())
	if first != second {
		t.Errorf("LOG render must be deterministic:\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}
}

// TestLog43_PanelTrimsToBound: the panel keeps the newest n entries
// (panel bound n=12; the fixture returns 20 so the oldest 8 must be
// trimmed).
func TestLog43_PanelTrimsToBound(t *testing.T) {
	var merged []collect.LogEntry
	for i := 1; i <= 20; i++ {
		merged = append(merged, collect.LogEntry{
			Instance: "client-a", Timestamp: ts(i), Unit: "sssonector@client-a",
			Pid: 8100, Message: "line-" + itoa2(i),
		})
	}
	m := logFixture(t, merged, []string{"client-a"})
	section := logSection(t, m.View())
	if strings.Contains(section, "line-8") {
		t.Errorf("oldest entries must be trimmed:\n%s", section)
	}
	for i := 9; i <= 12; i++ {
		if !strings.Contains(section, "line-"+itoa2(i)) {
			t.Errorf("newest entries must survive the trim (line-%d):\n%s", i, section)
		}
	}
}

func itoa2(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return string(rune('0'+i/10)) + string(rune('0'+i%10))
}

// logSection returns the LOG block from a screen render.
func logSection(t *testing.T, screen string) string {
	t.Helper()
	var b strings.Builder
	inLog := false
	for _, line := range strings.Split(screen, "\n") {
		if strings.HasPrefix(line, "LOG") {
			inLog = true
		} else if inLog && !strings.HasPrefix(line, "  ") {
			break
		}
		if inLog {
			b.WriteString(line + "\n")
		}
	}
	if b.Len() == 0 {
		t.Fatalf("no LOG section in:\n%s", screen)
	}
	return b.String()
}
