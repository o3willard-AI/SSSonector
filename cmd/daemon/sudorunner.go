package main

import (
	"os"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

// sudoRunner wraps the OS runner so systemctl lifecycle calls go through
// `sudo -A` (askpass) when SUDO_ASKPASS is set — the wizard's enable/start
// need root, and without -A sudo demands a terminal even when an askpass
// helper is configured (rig-verified: plain sudo fails with "a terminal is
// required", sudo -A succeeds via SUDO_ASKPASS).
type sudoRunner struct {
	inner collect.CommandRunner
}

// Run prefixes argv with `sudo -A` when SUDO_ASKPASS is set; otherwise it
// passes through unchanged (the normal case: root already, or passwordless
// systemctl).
func (s sudoRunner) Run(args ...string) (string, error) {
	if os.Getenv("SUDO_ASKPASS") != "" {
		full := append([]string{"sudo", "-A"}, args...)
		return s.inner.Run(full...)
	}
	return s.inner.Run(args...)
}
