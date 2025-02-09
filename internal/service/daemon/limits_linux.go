package daemon

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// setResourceLimits sets Linux-specific resource limits
func setResourceLimits() error {
	// Set maximum number of file descriptors
	if err := setRLimit(unix.RLIMIT_NOFILE, 65535); err != nil {
		return fmt.Errorf("failed to set RLIMIT_NOFILE: %w", err)
	}

	// Set maximum number of processes
	if err := setRLimit(RLIMIT_NPROC, 32768); err != nil {
		return fmt.Errorf("failed to set RLIMIT_NPROC: %w", err)
	}

	// Set maximum locked memory
	if err := setRLimit(RLIMIT_MEMLOCK, 16<<30); err != nil { // 16GB
		return fmt.Errorf("failed to set RLIMIT_MEMLOCK: %w", err)
	}

	// Set maximum core dump size
	if err := setRLimit(unix.RLIMIT_CORE, 0); err != nil {
		return fmt.Errorf("failed to set RLIMIT_CORE: %w", err)
	}

	// Set maximum stack size
	if err := setRLimit(unix.RLIMIT_STACK, 8<<20); err != nil { // 8MB
		return fmt.Errorf("failed to set RLIMIT_STACK: %w", err)
	}

	// Set maximum virtual memory
	if err := setRLimit(unix.RLIMIT_AS, 32<<30); err != nil { // 32GB
		return fmt.Errorf("failed to set RLIMIT_AS: %w", err)
	}

	// Set maximum CPU time
	if err := setRLimit(unix.RLIMIT_CPU, 3600); err != nil { // 1 hour
		return fmt.Errorf("failed to set RLIMIT_CPU: %w", err)
	}

	// Set maximum nice priority
	if err := setRLimit(RLIMIT_NICE, 40); err != nil {
		return fmt.Errorf("failed to set RLIMIT_NICE: %w", err)
	}

	// Set maximum realtime priority
	if err := setRLimit(RLIMIT_RTPRIO, 99); err != nil {
		return fmt.Errorf("failed to set RLIMIT_RTPRIO: %w", err)
	}

	// Set maximum pending signals
	if err := setRLimit(RLIMIT_SIGPENDING, 16384); err != nil {
		return fmt.Errorf("failed to set RLIMIT_SIGPENDING: %w", err)
	}

	// Set maximum message queue size
	if err := setRLimit(RLIMIT_MSGQUEUE, 819200); err != nil { // 800KB
		return fmt.Errorf("failed to set RLIMIT_MSGQUEUE: %w", err)
	}

	return nil
}

// setRLimit sets a specific resource limit
func setRLimit(resource int, value uint64) error {
	limit := unix.Rlimit{
		Cur: value,
		Max: value,
	}
	return unix.Setrlimit(resource, &limit)
}

// getRLimit gets a specific resource limit
func getRLimit(resource int) (*unix.Rlimit, error) {
	var limit unix.Rlimit
	if err := unix.Getrlimit(resource, &limit); err != nil {
		return nil, err
	}
	return &limit, nil
}

// checkRLimit checks if a specific resource limit is set correctly
func checkRLimit(resource int, expected uint64) error {
	limit, err := getRLimit(resource)
	if err != nil {
		return err
	}

	if limit.Cur != expected {
		return fmt.Errorf("resource limit mismatch: expected %d, got %d", expected, limit.Cur)
	}

	return nil
}

// verifyResourceLimits verifies all resource limits are set correctly
func verifyResourceLimits() error {
	// Verify file descriptors limit
	if err := checkRLimit(unix.RLIMIT_NOFILE, 65535); err != nil {
		return fmt.Errorf("RLIMIT_NOFILE verification failed: %w", err)
	}

	// Verify processes limit
	if err := checkRLimit(RLIMIT_NPROC, 32768); err != nil {
		return fmt.Errorf("RLIMIT_NPROC verification failed: %w", err)
	}

	// Verify locked memory limit
	if err := checkRLimit(RLIMIT_MEMLOCK, 16<<30); err != nil {
		return fmt.Errorf("RLIMIT_MEMLOCK verification failed: %w", err)
	}

	// Verify core dump limit
	if err := checkRLimit(unix.RLIMIT_CORE, 0); err != nil {
		return fmt.Errorf("RLIMIT_CORE verification failed: %w", err)
	}

	// Verify stack size limit
	if err := checkRLimit(unix.RLIMIT_STACK, 8<<20); err != nil {
		return fmt.Errorf("RLIMIT_STACK verification failed: %w", err)
	}

	// Verify virtual memory limit
	if err := checkRLimit(unix.RLIMIT_AS, 32<<30); err != nil {
		return fmt.Errorf("RLIMIT_AS verification failed: %w", err)
	}

	// Verify CPU time limit
	if err := checkRLimit(unix.RLIMIT_CPU, 3600); err != nil {
		return fmt.Errorf("RLIMIT_CPU verification failed: %w", err)
	}

	// Verify nice priority limit
	if err := checkRLimit(RLIMIT_NICE, 40); err != nil {
		return fmt.Errorf("RLIMIT_NICE verification failed: %w", err)
	}

	// Verify realtime priority limit
	if err := checkRLimit(RLIMIT_RTPRIO, 99); err != nil {
		return fmt.Errorf("RLIMIT_RTPRIO verification failed: %w", err)
	}

	// Verify pending signals limit
	if err := checkRLimit(RLIMIT_SIGPENDING, 16384); err != nil {
		return fmt.Errorf("RLIMIT_SIGPENDING verification failed: %w", err)
	}

	// Verify message queue limit
	if err := checkRLimit(RLIMIT_MSGQUEUE, 819200); err != nil {
		return fmt.Errorf("RLIMIT_MSGQUEUE verification failed: %w", err)
	}

	return nil
}
