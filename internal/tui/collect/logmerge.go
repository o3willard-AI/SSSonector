package collect

import (
	"sort"
)

// LogMergeSource supplies one instance's log entries (the unit is the
// journalctl unit; production passes snap.Unit). Injectable so the merge
// is testable against fixture entries — no real journalctl runs from
// tests.
type LogMergeSource func(unit string) []LogEntry

// MergeLogs tails EVERY discovered instance and combines their entries
// into ONE list, newest last (RenderLog's display order), time-ordered
// across instances. Ties (equal timestamps) break by instance name, then
// by original per-stream position, so the render is reproducible run to
// run. Entries with zero timestamps (unparseable lines carried verbatim)
// sort before all timestamped entries deterministically.
//
// The tailer reads each unit via the injected CommandRunner seam; a unit
// that fails to read contributes no entries (never fabricates) and never
// blocks the other streams.
func MergeLogs(tail LogTail, units []string, n int) []LogEntry {
	if n <= 0 {
		n = 12
	}
	var merged []LogEntry
	for _, unit := range units {
		read, err := tail.Read(unit, "", "", n)
		if err != nil {
			continue // unit unreadable: contributes nothing, others unaffected
		}
		merged = append(merged, read.Entries...)
	}
	// Stable time order with deterministic tie-breaks: timestamp, then
	// instance name, then original position (sort.SliceStable keeps the
	// per-stream order for entries identical on all keys).
	sort.SliceStable(merged, func(i, j int) bool {
		a, b := merged[i], merged[j]
		switch {
		case a.Timestamp.IsZero() && !b.Timestamp.IsZero():
			return true // unparseable lines first, deterministically
		case !a.Timestamp.IsZero() && b.Timestamp.IsZero():
			return false
		case !a.Timestamp.Equal(b.Timestamp):
			return a.Timestamp.Before(b.Timestamp)
		case a.Instance != b.Instance:
			return a.Instance < b.Instance
		default:
			return false // identical keys: stable order (per-stream position)
		}
	})
	// Keep the newest n (RenderLog also trims, but the merge is the
	// single source of truth for panel content).
	if len(merged) > n {
		merged = merged[len(merged)-n:]
	}
	return merged
}
