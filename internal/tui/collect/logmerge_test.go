package collect

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// fakeMergeRunner serves canned journalctl output per unit via the
// CommandRunner seam — no real journalctl runs from tests.
type fakeMergeRunner struct {
	perUnit map[string]string // unit -> journalctl short-iso output
	err     error             // when set, every read fails
}

func (f *fakeMergeRunner) Run(args ...string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	// journalctl argv: ["journalctl", "-u", unit, ...]
	for i, a := range args {
		if a == "-u" && i+1 < len(args) {
			return f.perUnit[args[i+1]], nil
		}
	}
	return "", nil
}

// TestMergeLogs_InterleavedTimeOrder: two unit streams with interleaved
// timestamps merge into a strictly time-ordered stream (oldest first for
// display; RenderLog prints in slice order), stable across repeated calls.
func TestMergeLogs_InterleavedTimeOrder(t *testing.T) {
	r := &fakeMergeRunner{perUnit: map[string]string{
		"sssonector@a.service": "2026-09-13T02:05:20+00:00 h a[1]: a-first\n" +
			"2026-09-13T02:05:26+00:00 h a[1]: a-last\n",
		"sssonector@b.service": "2026-09-13T02:05:22+00:00 h b[2]: b-mid\n" +
			"2026-09-13T02:05:24+00:00 h b[2]: b-mid2\n",
	}}
	tail := NewLogTail(r)
	got := MergeLogs(tail, []string{"sssonector@a.service", "sssonector@b.service"}, 12)

	wantOrder := []string{"a-first", "b-mid", "b-mid2", "a-last"}
	if len(got) != len(wantOrder) {
		t.Fatalf("merged %d entries, want %d: %+v", len(got), len(wantOrder), got)
	}
	for i, e := range got {
		if !strings.HasSuffix(e.Message, wantOrder[i]) {
			t.Errorf("position %d: got %q, want suffix %q", i, e.Message, wantOrder[i])
		}
	}
	// Stability: a second merge produces the identical slice.
	again := MergeLogs(tail, []string{"sssonector@a.service", "sssonector@b.service"}, 12)
	if len(again) != len(got) {
		t.Fatalf("second merge length differs: %d vs %d", len(again), len(got))
	}
	for i := range got {
		if got[i].Message != again[i].Message || got[i].Instance != again[i].Instance {
			t.Errorf("unstable merge at %d: %+v vs %+v", i, got[i], again[i])
		}
	}
}

// TestMergeLogs_TagsCarried: every merged entry carries its source
// instance tag — a-tags say "a", b-tags say "b".
func TestMergeLogs_TagsCarried(t *testing.T) {
	r := &fakeMergeRunner{perUnit: map[string]string{
		"sssonector@a.service": "2026-09-13T02:05:20+00:00 h a[1]: msg-a1\n",
		"sssonector@b.service": "2026-09-13T02:05:21+00:00 h b[2]: msg-b1\n",
	}}
	got := MergeLogs(NewLogTail(r), []string{"sssonector@a.service", "sssonector@b.service"}, 12)
	if len(got) != 2 {
		t.Fatalf("entries: %d", len(got))
	}
	if got[0].Instance != "a" || got[0].Unit != "a" {
		t.Errorf("first entry tag: instance=%q unit=%q, want a/a", got[0].Instance, got[0].Unit)
	}
	if got[1].Instance != "b" || got[1].Unit != "b" {
		t.Errorf("second entry tag: instance=%q unit=%q, want b/b", got[1].Instance, got[1].Unit)
	}
}

// TestMergeLogs_TieBreakDeterministic: entries with EQUAL timestamps from
// different instances sort by instance name — same output every run,
// regardless of stream order in the units slice.
func TestMergeLogs_TieBreakDeterministic(t *testing.T) {
	const line = "2026-09-13T02:05:22+00:00 h %[1]s[9]: same-time"
	r := &fakeMergeRunner{perUnit: map[string]string{
		"sssonector@zeta.service":  strings.Replace(line, "%[1]s", "zeta", 1),
		"sssonector@alpha.service": strings.Replace(line, "%[1]s", "alpha", 1),
	}}
	tail := NewLogTail(r)
	first := MergeLogs(tail, []string{"sssonector@zeta.service", "sssonector@alpha.service"}, 12)
	if len(first) != 2 {
		t.Fatalf("entries: %d", len(first))
	}
	if first[0].Instance != "alpha" || first[1].Instance != "zeta" {
		t.Errorf("tie must break by instance name: got %q then %q", first[0].Instance, first[1].Instance)
	}
	// Reversed input order → identical output.
	second := MergeLogs(tail, []string{"sssonector@alpha.service", "sssonector@zeta.service"}, 12)
	if first[0].Instance != second[0].Instance || first[1].Instance != second[1].Instance {
		t.Errorf("tie-break not stable across unit order: %v vs %v", first, second)
	}
}

// TestMergeLogs_FailedUnitContributesNothing: a unit that fails to read is
// skipped (no fabrication), other streams unaffected.
func TestMergeLogs_FailedUnitContributesNothing(t *testing.T) {
	r := &fakeMergeRunner{perUnit: map[string]string{
		"sssonector@a.service": "2026-09-13T02:05:20+00:00 h a[1]: only-a\n",
	}}
	tail := NewLogTail(r) // b's unit is absent from the map → journalctl errors? No: empty output.
	// Force an error for one unit via a failing runner wrapper:
	errTail := LogTail{Runner: failingFor{"sssonector@b.service", r}, BootID: defaultBootID}
	got := MergeLogs(errTail, []string{"sssonector@a.service", "sssonector@b.service"}, 12)
	if len(got) != 1 || got[0].Instance != "a" {
		t.Errorf("only a's entry should survive: %+v", got)
	}
	_ = tail // direct empty-output case
}

type failingFor struct {
	unit string
	next CommandRunner
}

func (f failingFor) Run(args ...string) (string, error) {
	for i, a := range args {
		if a == "-u" && i+1 < len(args) && args[i+1] == f.unit {
			return "", errors.New("journalctl exploded")
		}
	}
	return f.next.Run(args...)
}

// TestMergeLogs_TrimsToN: the merged stream keeps the newest n entries.
func TestMergeLogs_TrimsToN(t *testing.T) {
	var lines string
	for i := 1; i <= 20; i++ {
		lines += "2026-09-13T02:05:" + twoDigits(i) + "+00:00 h a[1]: line-" + twoDigits(i) + "\n"
	}
	r := &fakeMergeRunner{perUnit: map[string]string{"sssonector@a.service": lines}}
	got := MergeLogs(NewLogTail(r), []string{"sssonector@a.service"}, 5)
	if len(got) != 5 {
		t.Fatalf("entries: %d, want 5", len(got))
	}
	// Newest 5 = lines 16..20.
	if got[0].Message != "line-16" || got[4].Message != "line-20" {
		t.Errorf("trim kept wrong window: %q..%q", got[0].Message, got[4].Message)
	}
}

func twoDigits(i int) string {
	if i < 10 {
		return "0" + string(rune('0'+i))
	}
	return string(rune('0'+i/10)) + string(rune('0'+i%10))
}

// TestMergeLogs_TimestampedSanity: the merged entries parse real short-iso
// timestamps (not zero) — guards against a merge that drops parsing.
func TestMergeLogs_TimestampedSanity(t *testing.T) {
	r := &fakeMergeRunner{perUnit: map[string]string{
		"sssonector@a.service": "2026-09-13T02:05:20+00:00 h a[1]: x\n",
	}}
	got := MergeLogs(NewLogTail(r), []string{"sssonector@a.service"}, 12)
	if len(got) != 1 || got[0].Timestamp.IsZero() {
		t.Errorf("timestamp must parse: %+v", got)
	}
	if want := time.Date(2026, 9, 13, 2, 5, 20, 0, time.UTC); !got[0].Timestamp.Equal(want) {
		t.Errorf("timestamp: %v, want %v", got[0].Timestamp, want)
	}
}
