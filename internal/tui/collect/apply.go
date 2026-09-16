package collect

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ReloadOutcome is the result of the post-write SIGHUP.
type ReloadOutcome int

const (
	// ReloadOK: the daemon accepted the new config (fresh in-memory state).
	ReloadOK ReloadOutcome = iota
	// ReloadRejected: the daemon REJECTED the new config (it logged the
	// rejection). Documented semantics: the config file STAYS WRITTEN on
	// disk, the daemon keeps its OLD in-memory config, and nothing is
	// rolled back — matching today's SIGHUP semantics (docs/tui.md §7).
	ReloadRejected
)

// String renders the outcome.
func (o ReloadOutcome) String() string {
	if o == ReloadOK {
		return "ok"
	}
	return "rejected"
}

// SignalFunc sends SIGHUP to a PID. Injectable so tests fake the signal
// (the real implementation lives in signal_unix.go / signal_windows.go;
// never exercised in unit tests).
type SignalFunc func(pid int) error

// ReloadReader reads the reload outcome from the daemon after the signal
// (log tail / healthz generation). Injectable for tests.
type ReloadReader func(pid int) (ReloadOutcome, error)

// ApplyResult reports the apply pipeline outcome.
type ApplyResult struct {
	// File is the config file written (resolved path).
	File string
	// Outcome is the post-signal reload result (meaningful only when
	// Signaled is true).
	Outcome ReloadOutcome
	// Signaled is true when the write succeeded and the signal fired.
	Signaled bool
}

// SignalHUP is the production SignalFunc. The POSIX implementation lives in
// signal_unix.go; the windows stub in signal_windows.go keeps the windows
// release binary compiling (syscall.Kill does not exist there).

// LogReloadReader is the production ReloadReader: tails the daemon's
// journal (via the LogTail runner) for the reload outcome after SIGHUP —
// a reload failure/rejected line => ReloadRejected; an ok line => ReloadOK;
// no outcome line within the bounded window => ReloadOK (success is the
// steady state; the daemon only logs reload failures loudly).
// unitOfPid resolves the systemd unit owning the pid.
func LogReloadReader(runner CommandRunner, unitOfPid func(pid int) string, window time.Duration) ReloadReader {
	return func(pid int) (ReloadOutcome, error) {
		lt := LogTail{Runner: runner, BootID: defaultBootID}
		deadline := time.Now().Add(window)
		unit := unitOfPid(pid)
		var cursor string
		for time.Now().Before(deadline) {
			read, err := lt.Read(unit, cursor, "", 20)
			if err != nil {
				time.Sleep(200 * time.Millisecond)
				continue
			}
			for _, e := range read.Entries {
				msg := strings.ToLower(e.Message)
				if strings.Contains(msg, "reload") &&
					(strings.Contains(msg, "failed") || strings.Contains(msg, "invalid") ||
						strings.Contains(msg, "rejected") || strings.Contains(msg, "error")) {
					return ReloadRejected, nil
				}
				if strings.Contains(msg, "reload") &&
					(strings.Contains(msg, "ok") || strings.Contains(msg, "succeeded") || strings.Contains(msg, "applied")) {
					return ReloadOK, nil
				}
			}
			cursor = read.Cursor
			time.Sleep(250 * time.Millisecond)
		}
		return ReloadOK, nil
	}
}

// Real implementation lives here for the rig path; unit tests inject
// fakes. (golang.org/x/sys/unix or syscall — syscall keeps deps at
// stdlib-only.)
//
// ApplyConfig validates nothing itself — the caller MUST have passed the
// draft through ValidateDraft (WI 3.2) first; this function's contract is:
// write durably, then signal, never the reverse.
//
// Safety invariants (fail-closed):
//  1. Atomic write: temp file in the SAME directory, fsync, os.Rename over
//     the target, chmod 0600. A crash mid-write never leaves a truncated
//     config — the daemon keeps the old file until the rename completes.
//  2. Signal only AFTER a successful write. Write failure → error, NO
//     signal: the daemon is never told to reload a config that was not
//     durably written.
//  3. Failure never kills the daemon: a failed write or a rejected reload
//     leaves the running daemon untouched (it keeps its in-memory config).
//
// Reload semantics (documented, tested): on a SIGHUP the daemon may REJECT
// the new config (logging the rejection). Then: the file stays written on
// disk, the daemon keeps its OLD in-memory config, and the outcome is
// surfaced as ReloadRejected (vs ReloadOK when the reload succeeds).
func ApplyConfig(paths ConfigPaths, name, draft string, pid int, signal SignalFunc, readReload ReloadReader) (ApplyResult, error) {
	res := ApplyResult{}

	if pid <= 0 {
		return res, fmt.Errorf("apply: instance %q: no live daemon pid to signal (pid=%d)", name, pid)
	}
	if signal == nil {
		return res, fmt.Errorf("apply: nil signal func")
	}

	target, err := resolveConfigFile(paths, name)
	if err != nil {
		return res, err
	}
	res.File = target

	if err := atomicWrite(target, []byte(draft)); err != nil {
		// Invariant 2+3: write failed — NO signal, daemon untouched.
		return res, fmt.Errorf("apply: write %s: %w", target, err)
	}

	// Invariant 2: signal only after the durable write.
	if err := signal(pid); err != nil {
		// The file IS written; the signal failed (daemon died between
		// write and signal). Surface it — the next daemon start picks up
		// the new file. Do not roll back the write.
		return res, fmt.Errorf("apply: write ok but SIGHUP to pid %d failed: %w", pid, err)
	}
	res.Signaled = true

	// Reload outcome: ok or rejected (the daemon keeps its old config on
	// rejection — documented, never rolled back).
	if readReload == nil {
		res.Outcome = ReloadOK
		return res, nil
	}
	outcome, err := readReload(pid)
	if err != nil {
		return res, fmt.Errorf("apply: reload outcome read failed: %w", err)
	}
	res.Outcome = outcome
	return res, nil
}

// atomicWrite writes data to target via temp-file-in-same-dir + fsync +
// rename, then chmod 0600. The rename is the commit point.
func atomicWrite(target string, data []byte) error {
	dir := filepath.Dir(target)
	tmp, err := os.CreateTemp(dir, ".sssonector-apply-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp in %s: %w", dir, err)
	}
	tmpName := tmp.Name()
	// Best-effort cleanup on any failure path before the rename.
	committed := false
	defer func() {
		if !committed {
			os.Remove(tmpName) //nolint:errcheck // best-effort cleanup
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		tmp.Close() //nolint:errcheck
		return fmt.Errorf("write temp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close() //nolint:errcheck
		return fmt.Errorf("fsync temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp: %w", err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		return fmt.Errorf("chmod temp: %w", err)
	}
	// Commit point: the target is only ever replaced via rename.
	if err := os.Rename(tmpName, target); err != nil {
		return fmt.Errorf("rename over %s: %w", target, err)
	}
	committed = true
	return nil
}
