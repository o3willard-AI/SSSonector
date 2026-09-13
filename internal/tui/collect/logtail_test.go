package collect

import (
	"strings"
	"testing"
	"time"
)

const unit = "sssonector@client-a.service"

func journalLine(ts string, pid int, msg string) string {
	return ts + " qa-host sssonector@client-a[" + itoa(pid) + "]: " + msg
}

// itoa is a tiny local helper to keep the fixtures readable.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	digits := ""
	for i > 0 {
		digits = string(rune('0'+i%10)) + digits
		i /= 10
	}
	return digits
}

func TestLogTail_CursorResume(t *testing.T) {
	const cursor1 = "s=aaa;i=1;b=boot1"
	const cursor2 = "s=aaa;i=9;b=boot1"

	r := newFakeRunner()
	// First read: full tail of 3 + cursor.
	r.outputs["journalctl -u "+unit+" -n 3 --no-pager -o short-iso --show-cursor"] =
		journalLine("2026-09-13T02:05:20+00:00", 100, "listener open") + "\n" +
			journalLine("2026-09-13T02:05:25+00:00", 100, "tls ok") + "\n" +
			journalLine("2026-09-13T02:05:26+00:00", 100, "tunnel up") + "\n" +
			"-- cursor: " + cursor1 + "\n"

	lt := NewLogTail(r)
	lt.BootID = func() (string, error) { return "boot-1", nil }
	first, err := lt.Read(unit, "", "boot-1", 3)
	if err != nil {
		t.Fatalf("first read: %v", err)
	}
	if len(first.Entries) != 3 {
		t.Fatalf("first read entries: %d", len(first.Entries))
	}
	if first.Cursor != cursor1 {
		t.Errorf("first cursor: %q", first.Cursor)
	}
	if first.ReSeeked {
		t.Error("first read must not be marked re-seek")
	}

	// Second read: only NEW lines after the cursor — old lines are not
	// re-sent.
	r.outputs["journalctl -u "+unit+" --after-cursor="+cursor1+" --no-pager -o short-iso --show-cursor"] =
		journalLine("2026-09-13T02:06:00+00:00", 100, "rekeyed") + "\n" +
			"-- cursor: " + cursor2 + "\n"

	second, err := lt.Read(unit, cursor1, "boot-1", 3)
	if err != nil {
		t.Fatalf("second read: %v", err)
	}
	if len(second.Entries) != 1 {
		t.Fatalf("second read entries: %d (%+v) — old lines must be skipped", len(second.Entries), second.Entries)
	}
	if second.Entries[0].Message != "rekeyed" {
		t.Errorf("second read message: %q", second.Entries[0].Message)
	}
	if second.Cursor != cursor2 {
		t.Errorf("second cursor: %q", second.Cursor)
	}
	if second.ReSeeked {
		t.Error("incremental read must not be marked re-seek")
	}

	// Prove the argv actually used --after-cursor (no full re-tail).
	found := false
	for _, call := range r.calls {
		if len(call) > 2 && call[1] == "-u" && strings.Contains(strings.Join(call, " "), "--after-cursor="+cursor1) {
			found = true
		}
	}
	if !found {
		t.Error("incremental read must pass --after-cursor")
	}
}

func TestLogTail_RotationReSeeks(t *testing.T) {
	const oldCursor = "s=old;i=5;b=boot-OLD"

	r := newFakeRunner()
	// Rotation: boot id changed, so the old cursor is stale. The read must
	// fall back to -n 3 (not silently return empty).
	r.bootID = "boot-NEW"
	r.outputs["journalctl -u "+unit+" -n 3 --no-pager -o short-iso --show-cursor"] =
		journalLine("2026-09-14T00:00:01+00:00", 200, "new boot first line") + "\n" +
			"-- cursor: s=new;i=1;b=boot-NEW\n"

	lt := NewLogTail(r)
	lt.BootID = func() (string, error) { return "boot-NEW", nil }
	got, err := lt.Read(unit, oldCursor, "boot-OLD", 3)
	if err != nil {
		t.Fatalf("rotation read: %v", err)
	}
	if !got.ReSeeked {
		t.Error("rotation must be marked ReSeeked")
	}
	if len(got.Entries) != 1 || got.Entries[0].Message != "new boot first line" {
		t.Errorf("re-seek entries: %+v", got.Entries)
	}
	if got.Cursor != "s=new;i=1;b=boot-NEW" {
		t.Errorf("fresh cursor: %q", got.Cursor)
	}

	// Prove the fallback used -n (full tail), not --after-cursor.
	for _, call := range r.calls {
		if strings.Contains(strings.Join(call, " "), "--after-cursor") {
			t.Error("rotation must NOT pass the stale cursor to journalctl")
		}
	}
}

func TestLogTail_StaleCursorWithoutBootChange_ReSeeks(t *testing.T) {
	// Journal rotated away the cursor without a boot change: the seek
	// errors — must re-seek, never silently return empty.
	r := newFakeRunner()
	r.errs["journalctl -u "+unit+" --after-cursor=s=gone;i=1;b=boot-1 --no-pager -o short-iso --show-cursor"] =
		errRunnerErr
	r.outputs["journalctl -u "+unit+" -n 3 --no-pager -o short-iso --show-cursor"] =
		journalLine("2026-09-13T03:00:00+00:00", 100, "after rotation") + "\n" +
			"-- cursor: s=fresh;i=1;b=boot-1\n"

	lt := NewLogTail(r)
	lt.BootID = func() (string, error) { return "boot-1", nil }
	got, err := lt.Read(unit, "s=gone;i=1;b=boot-1", "boot-1", 3)
	if err != nil {
		t.Fatalf("stale-cursor read: %v", err)
	}
	if !got.ReSeeked {
		t.Error("failed seek must re-seek")
	}
	if len(got.Entries) != 1 {
		t.Errorf("re-seek entries: %+v", got.Entries)
	}
}

// errRunnerErr is a canned error for fake runner maps.
var errRunnerErr = &cannedError{}

type cannedError struct{}

func (e *cannedError) Error() string { return "cursor not seekable" }

func TestLogTail_LineParsing(t *testing.T) {
	ts := "2026-09-13T02:05:26+00:00"
	e := parseShortISO(ts+" qa-host sssonector@client-a[8123]: tunnel rekeyed peer 192.168.100.51", "client-a")
	want := time.Date(2026, 9, 13, 2, 5, 26, 0, time.UTC)
	if !e.Timestamp.Equal(want) {
		t.Errorf("timestamp: got %v want %v", e.Timestamp, want)
	}
	if e.Unit != "sssonector@client-a" {
		t.Errorf("unit: %q", e.Unit)
	}
	if e.Pid != 8123 {
		t.Errorf("pid: %d", e.Pid)
	}
	if e.Message != "tunnel rekeyed peer 192.168.100.51" {
		t.Errorf("message: %q", e.Message)
	}
	if e.Instance != "client-a" {
		t.Errorf("instance tag: %q", e.Instance)
	}

	// Timezone offset preserved: +02:00 parses to +02:00 location.
	e2 := parseShortISO("2026-09-13T04:05:26+02:00 h u[1]: msg", "x")
	if e2.Timestamp.UTC().Hour() != 2 {
		t.Errorf("tz handling: got UTC hour %d", e2.Timestamp.UTC().Hour())
	}
}

func TestLogTail_MalformedLines_NoPanic(t *testing.T) {
	weird := []string{
		"",
		"   ",
		"not-a-timestamp host unit[1]: msg",
		"2026-13-99T99:99:99+00:00 host unit[1]: impossible date",
		"2026-09-13T02:05:26+00:00",
		"2026-09-13T02:05:26+00:00 no-colon-message",
		"2026-09-13T02:05:26+00:00 h u: ",
		"-- cursor: with no space",
		"\x00\xff\xfe garbage bytes",
	}
	for _, line := range weird {
		// Must not panic on any input.
		parseShortISO(line, "x")
	}

	// Full-output parse with garbage interleaved: valid lines survive,
	// garbage carried verbatim, cursor still captured.
	out := journalLine("2026-09-13T02:05:26+00:00", 1, "ok") + "\n" +
		"garbage line\n" +
		"-- cursor: s=1;b=b\n"
	entries, cursor := parseJournalOutput(out, unit)
	if cursor != "s=1;b=b" {
		t.Errorf("cursor: %q", cursor)
	}
	if len(entries) != 2 {
		t.Fatalf("entries: %d", len(entries))
	}
	if entries[0].Message != "ok" {
		t.Errorf("valid line lost: %+v", entries[0])
	}
	if entries[1].Raw != "garbage line" {
		t.Errorf("malformed line must be carried verbatim: %+v", entries[1])
	}
}

func TestLogTail_ZeroN_DefaultsTo12(t *testing.T) {
	r := newFakeRunner()
	r.outputs["journalctl -u "+unit+" -n 12 --no-pager -o short-iso --show-cursor"] = "-- cursor: s=c\n"
	lt := NewLogTail(r)
	got, err := lt.Read(unit, "", "b", 0)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.Cursor != "s=c" {
		t.Errorf("cursor: %q", got.Cursor)
	}
}
