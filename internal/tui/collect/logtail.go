package collect

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// logtailTailerArgs are the shared journalctl flags (verified against
// journalctl --help): no pager, short-iso timestamps, cursor echo.
const (
	journalTimeFormat = "2006-01-02T15:04:05-07:00"
	cursorPrefix      = "-- cursor: "
	bootIDFile        = "/proc/sys/kernel/random/boot_id"
)

// LogEntry is one parsed journal line, tagged with its instance.
type LogEntry struct {
	// Instance is the unit's instance name (tag for the merged rail).
	Instance string
	// Timestamp is the parsed short-iso timestamp (zero when unparseable —
	// the entry is still carried with Raw set).
	Timestamp time.Time
	// Pid is the process id from "unit[pid]", 0 when absent.
	Pid int
	// Unit is the syslog identifier ("sssonector@client-a").
	Unit string
	// Message is the log text after ": ".
	Message string
	// Raw is the verbatim line (used for malformed/non-log lines).
	Raw string
}

// LogRead is the result of one incremental read.
type LogRead struct {
	// Entries are the new lines for this read, in journal order.
	Entries []LogEntry
	// Cursor is the resume cursor for the next incremental read ("" when
	// the read produced no cursor line).
	Cursor string
	// ReSeeked is true when the read fell back to a full tail because the
	// previous cursor was stale (journal rotated / new boot).
	ReSeeked bool
}

// LogTail reads instance journals incrementally via journalctl. All
// invocations go through the injected CommandRunner — tests supply fake
// output and never exec the real binary.
type LogTail struct {
	Runner CommandRunner
	// BootID returns the current boot id (rotation detection). Defaults to
	// reading /proc/sys/kernel/random/boot_id; injectable in tests.
	BootID func() (string, error)
	// Now overrides time parsing "now" (unused for parsing itself, kept
	// for symmetry with cert reader's injectable clock).
}

// NewLogTail builds a LogTail over the given runner.
func NewLogTail(r CommandRunner) LogTail {
	return LogTail{Runner: r, BootID: defaultBootID}
}

// defaultBootID reads the kernel boot id for rotation detection.
func defaultBootID() (string, error) {
	b, err := os.ReadFile(bootIDFile)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// ReadIncremental returns the log entries after cursor for the given unit.
//
// First read (cursor ""): last N lines + captured cursor.
// Later reads: --after-cursor=<cursor>; only NEW lines are returned, old
// lines are never re-sent.
// Rotation: when the boot id differs from bootID, the cursor is by
// definition stale — the read falls back to a full -n N re-seek with a
// fresh cursor (ReSeeked=true) instead of silently returning empty.
// When the cursor seek itself errors, the same re-seek applies.
func (lt LogTail) Read(unit, cursor, bootID string, n int) (LogRead, error) {
	if n <= 0 {
		n = 12
	}

	// Rotation check: a new boot invalidates any prior cursor.
	if cursor != "" && bootID != "" {
		current, err := lt.BootID()
		if err == nil && current != "" && current != bootID {
			return lt.fullRead(unit, n, true)
		}
	}

	if cursor == "" {
		return lt.fullRead(unit, n, false)
	}
	return lt.incrementalRead(unit, cursor, n)
}

// fullRead tails the last N lines and captures the cursor.
func (lt LogTail) fullRead(unit string, n int, reSeek bool) (LogRead, error) {
	out, err := lt.Runner.Run("journalctl", "-u", unit,
		"-n", strconv.Itoa(n), "--no-pager", "-o", "short-iso", "--show-cursor")
	if err != nil {
		return LogRead{}, fmt.Errorf("journalctl %s: %w", unit, err)
	}
	entries, cursor := parseJournalOutput(out, instanceName(unit))
	return LogRead{Entries: entries, Cursor: cursor, ReSeeked: reSeek}, nil
}

// incrementalRead reads only lines after the cursor. A failed seek (stale
// cursor after rotation not caught by boot id) triggers a re-seek rather
// than a silent empty read.
func (lt LogTail) incrementalRead(unit, cursor string, n int) (LogRead, error) {
	out, err := lt.Runner.Run("journalctl", "-u", unit,
		"--after-cursor="+cursor, "--no-pager", "-o", "short-iso", "--show-cursor")
	if err != nil {
		// Cursor no longer seekable (rotated away): re-seek.
		return lt.fullRead(unit, n, true)
	}
	entries, newCursor := parseJournalOutput(out, instanceName(unit))
	if newCursor == "" {
		// No cursor line at all — treat as a stale-cursor signal and
		// re-seek so the next read has a valid resume point. NEVER return
		// a cursor-less success (that would re-send everything next time).
		return lt.fullRead(unit, n, true)
	}
	return LogRead{Entries: entries, Cursor: newCursor}, nil
}

// instanceName extracts the template instance (%i) from a unit name:
// "sssonector@client-a.service" → "client-a"; non-template units pass
// through unchanged (the legacy unit's tag is its own name).
func instanceName(unit string) string {
	if name, ok := instanceNameFromUnit(unit); ok {
		return name
	}
	return unit
}

// parseJournalOutput splits journalctl --show-cursor output into entries
// and the trailing cursor line.
func parseJournalOutput(out, unit string) ([]LogEntry, string) {
	var entries []LogEntry
	cursor := ""
	for _, raw := range strings.Split(out, "\n") {
		line := strings.TrimRight(raw, "\r")
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "-- cursor:") {
			cursor = strings.TrimSpace(strings.TrimPrefix(line, "-- cursor:"))
			continue
		}
		entries = append(entries, parseShortISO(line, unit))
	}
	return entries, cursor
}

// parseShortISO parses one short-iso journal line:
// "2026-09-13T02:05:26+00:00 host unit[pid]: msg".
// Unparseable lines are carried verbatim (Timestamp zero, Raw set), never
// a panic.
func parseShortISO(line, instance string) LogEntry {
	e := LogEntry{Instance: instance, Raw: line}

	// Timestamp is the first space-separated field.
	sp := strings.IndexByte(line, ' ')
	if sp < 0 {
		return e
	}
	ts, err := time.Parse(journalTimeFormat, line[:sp])
	if err != nil {
		return e // malformed timestamp: carried verbatim
	}
	e.Timestamp = ts
	rest := line[sp+1:]

	// "host unit[pid]: msg" — message starts after the first ": ".
	ci := strings.Index(line, ": ")
	if ci < 0 || ci < sp {
		e.Message = rest
		return e
	}
	prefix := line[sp+1 : ci]
	msg := line[ci+2:]

	// Extract unit[pid] from the prefix's last space-separated token.
	if i := strings.LastIndexByte(prefix, ' '); i >= 0 {
		prefix = prefix[i+1:]
	}
	if br := strings.IndexByte(prefix, '['); br >= 0 && strings.HasSuffix(prefix, "]") {
		e.Unit = prefix[:br]
		fmt.Sscanf(prefix[br+1:len(prefix)-1], "%d", &e.Pid)
	} else {
		e.Unit = prefix
	}
	e.Message = msg
	return e
}
