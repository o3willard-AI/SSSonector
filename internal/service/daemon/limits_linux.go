//go:build linux

package daemon

import (
	"syscall"
)

const (
	// Resource limit constants for Linux
	RLIMIT_NPROC = 0x6 // max number of processes
	RLIMIT_AS    = 0x9 // address space limit
	RLIMIT_STACK = 0x3 // max stack size
)

// setResourceLimits sets process resource limits
func setResourceLimits() error {
	var rLimit syscall.Rlimit

	// Set max number of file descriptors
	rLimit.Max = 65536
	rLimit.Cur = 65536
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		return err
	}

	// Set max number of processes
	rLimit.Max = 4096
	rLimit.Cur = 4096
	if err := syscall.Setrlimit(RLIMIT_NPROC, &rLimit); err != nil {
		return err
	}

	// Set max memory size
	rLimit.Max = 1024 * 1024 * 1024 // 1GB
	rLimit.Cur = 1024 * 1024 * 1024
	if err := syscall.Setrlimit(RLIMIT_AS, &rLimit); err != nil {
		return err
	}

	// Set max stack size
	rLimit.Max = 8 * 1024 * 1024 // 8MB
	rLimit.Cur = 8 * 1024 * 1024
	if err := syscall.Setrlimit(RLIMIT_STACK, &rLimit); err != nil {
		return err
	}

	// Disable core dumps
	rLimit.Max = 0
	rLimit.Cur = 0
	if err := syscall.Setrlimit(syscall.RLIMIT_CORE, &rLimit); err != nil {
		return err
	}

	return nil
}
