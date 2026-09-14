package collect

import (
	"fmt"
	"os"
	"path/filepath"
)

// CreateAndStartResult reports the wizard write-path outcome (WI 5.3).
type CreateAndStartResult struct {
	// File is the instance config written (resolved path).
	File string
	// Unit is the systemd unit that was enabled+started
	// (sssonector@<instance>.service).
	Unit string
	// Started is true when BOTH enable and start succeeded.
	Started bool
}

// CreateAndStartInstance is the WI 5.3 write path — the SAME config
// pipeline as Phase 3 (ONE pipeline, not two):
//
//  1. Atomic write of draft to /etc/sssonector/instances/<instance>/
//     config.yaml via the Phase 3 atomicWrite (tmp in same dir → fsync →
//     chmod 0600 → rename). The file is written BEFORE the unit is
//     touched. Atomicity note: the directory is created (0o750) first —
//     the rename commit point guarantees the target never appears
//     truncated.
//  2. THEN `systemctl enable` + `systemctl start` via the injectable
//     runner (exactly two calls, in that order; no real systemctl from
//     unit tests).
//
// Failure semantics (fail-closed, matching Phase 3's documented SIGHUP
// semantics):
//   - WRITE failure: zero runner calls — nothing is enabled/started; the
//     error surfaces verbatim.
//   - ENABLE/START failure: the config file STAYS WRITTEN (no rollback —
//     documented), zero further runner calls, and the error surfaces.
func CreateAndStartInstance(paths ConfigPaths, instance, draft string, runner CommandRunner) (CreateAndStartResult, error) {
	res := CreateAndStartResult{}
	if runner == nil {
		return res, fmt.Errorf("create: nil command runner")
	}
	if instance == "" {
		return res, fmt.Errorf("create: empty instance name")
	}

	// The target lives at instances/<instance>/config.yaml. mkdir -p the
	// instance dir (atomicWrite's temp file needs it to exist), then the
	// Phase 3 atomic write (tmp-in-same-dir → fsync → chmod 0600 →
	// rename). The file is on disk BEFORE any systemctl call.
	target := filepath.Join(paths.ConfigRoot, "instances", instance, "config.yaml")
	res.File = target
	res.Unit = "sssonector@" + instance + ".service"

	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return res, fmt.Errorf("create: mkdir %s: %w", filepath.Dir(target), err)
	}
	if err := atomicWrite(target, []byte(draft)); err != nil {
		// WRITE failure: zero runner calls, nothing enabled/started.
		return res, fmt.Errorf("create: write %s: %w", target, err)
	}

	// File is committed. Enable + start (in order; a failure stops the
	// sequence — the file stays written, no rollback).
	if _, err := runner.Run("systemctl", "enable", res.Unit); err != nil {
		return res, fmt.Errorf("create: enable %s: %w (config file left written at %s)",
			res.Unit, err, target)
	}
	if _, err := runner.Run("systemctl", "start", res.Unit); err != nil {
		return res, fmt.Errorf("create: start %s: %w (config file left written at %s)",
			res.Unit, err, target)
	}
	res.Started = true
	return res, nil
}
