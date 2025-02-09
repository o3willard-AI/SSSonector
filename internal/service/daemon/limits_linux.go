package daemon

import (
	"fmt"
	"syscall"
)

// Resource limits
const (
	defaultNofile = 1024 * 1024  // 1M open files
	defaultNproc  = 1024 * 32    // 32K processes
	defaultStack  = 1024 * 1024  // 1MB stack size
	defaultCore   = 1024 * 1024  // 1MB core dump size
	defaultCPU    = 60 * 60 * 24 // 24 hours CPU time

	// Linux resource limit constants
	RLIMIT_NPROC = 6 // Max number of processes
)

// setResourceLimits sets Linux resource limits
func setResourceLimits() error {
	// Set open files limit
	if err := setRlimit(syscall.RLIMIT_NOFILE, defaultNofile); err != nil {
		return fmt.Errorf("failed to set RLIMIT_NOFILE: %w", err)
	}

	// Set process limit
	if err := setRlimit(RLIMIT_NPROC, defaultNproc); err != nil {
		return fmt.Errorf("failed to set RLIMIT_NPROC: %w", err)
	}

	// Set stack size limit
	if err := setRlimit(syscall.RLIMIT_STACK, defaultStack); err != nil {
		return fmt.Errorf("failed to set RLIMIT_STACK: %w", err)
	}

	// Set core dump size limit
	if err := setRlimit(syscall.RLIMIT_CORE, defaultCore); err != nil {
		return fmt.Errorf("failed to set RLIMIT_CORE: %w", err)
	}

	// Set CPU time limit
	if err := setRlimit(syscall.RLIMIT_CPU, defaultCPU); err != nil {
		return fmt.Errorf("failed to set RLIMIT_CPU: %w", err)
	}

	return nil
}

// setRlimit sets a resource limit
func setRlimit(resource int, value uint64) error {
	rlimit := &syscall.Rlimit{
		Cur: value,
		Max: value,
	}
	return syscall.Setrlimit(resource, rlimit)
}

// getRlimit gets a resource limit
func getRlimit(resource int) (uint64, error) {
	var rlimit syscall.Rlimit
	if err := syscall.Getrlimit(resource, &rlimit); err != nil {
		return 0, err
	}
	return rlimit.Cur, nil
}

// checkRlimit checks if a resource limit is set correctly
func checkRlimit(resource int, expected uint64) error {
	current, err := getRlimit(resource)
	if err != nil {
		return err
	}
	if current != expected {
		return fmt.Errorf("resource limit mismatch: expected %d, got %d", expected, current)
	}
	return nil
}

// verifyResourceLimits verifies that resource limits are set correctly
func verifyResourceLimits() error {
	// Check open files limit
	if err := checkRlimit(syscall.RLIMIT_NOFILE, defaultNofile); err != nil {
		return fmt.Errorf("RLIMIT_NOFILE verification failed: %w", err)
	}

	// Check process limit
	if err := checkRlimit(RLIMIT_NPROC, defaultNproc); err != nil {
		return fmt.Errorf("RLIMIT_NPROC verification failed: %w", err)
	}

	// Check stack size limit
	if err := checkRlimit(syscall.RLIMIT_STACK, defaultStack); err != nil {
		return fmt.Errorf("RLIMIT_STACK verification failed: %w", err)
	}

	// Check core dump size limit
	if err := checkRlimit(syscall.RLIMIT_CORE, defaultCore); err != nil {
		return fmt.Errorf("RLIMIT_CORE verification failed: %w", err)
	}

	// Check CPU time limit
	if err := checkRlimit(syscall.RLIMIT_CPU, defaultCPU); err != nil {
		return fmt.Errorf("RLIMIT_CPU verification failed: %w", err)
	}

	return nil
}
