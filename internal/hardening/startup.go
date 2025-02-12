package hardening

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"syscall"

	"go.uber.org/zap"
)

// StartupHardening represents the startup hardening configuration
type StartupHardening struct {
	logger *zap.Logger
}

// New creates a new StartupHardening instance
func New(logger *zap.Logger) *StartupHardening {
	return &StartupHardening{
		logger: logger,
	}
}

// Apply applies all hardening measures
func (h *StartupHardening) Apply(ctx context.Context) error {
	measures := []func(context.Context) error{
		h.setResourceLimits,
		h.setSecureUmask,
		h.dropPrivileges,
		h.setCoreDumpLimits,
		h.setSecureEnvironment,
	}

	for _, measure := range measures {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := measure(ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

// setResourceLimits sets resource limits for the process
func (h *StartupHardening) setResourceLimits(ctx context.Context) error {
	// Set maximum number of file descriptors
	var rLimit syscall.Rlimit
	rLimit.Max = 65535
	rLimit.Cur = 65535
	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		return fmt.Errorf("failed to set file descriptor limit: %w", err)
	}

	// Set maximum number of processes (Linux-specific)
	if runtime.GOOS == "linux" {
		const RLIMIT_NPROC = 6 // Linux-specific constant
		rLimit.Max = 4096
		rLimit.Cur = 4096
		if err := syscall.Setrlimit(RLIMIT_NPROC, &rLimit); err != nil {
			return fmt.Errorf("failed to set process limit: %w", err)
		}
	}

	h.logger.Info("Resource limits set successfully")
	return nil
}

// setSecureUmask sets a secure umask
func (h *StartupHardening) setSecureUmask(ctx context.Context) error {
	syscall.Umask(0027) // Only owner has full access, group has read/execute
	h.logger.Info("Secure umask set successfully")
	return nil
}

// dropPrivileges drops root privileges if running as root
func (h *StartupHardening) dropPrivileges(ctx context.Context) error {
	if os.Geteuid() == 0 {
		// Get the UID and GID for the sssonector user
		targetUID := uint32(1000) // Example UID for sssonector user
		targetGID := uint32(1000) // Example GID for sssonector group

		// Set the GID first
		if err := syscall.Setgid(int(targetGID)); err != nil {
			return fmt.Errorf("failed to set GID: %w", err)
		}

		// Set supplementary groups
		if err := syscall.Setgroups([]int{int(targetGID)}); err != nil {
			return fmt.Errorf("failed to set supplementary groups: %w", err)
		}

		// Set the UID
		if err := syscall.Setuid(int(targetUID)); err != nil {
			return fmt.Errorf("failed to set UID: %w", err)
		}

		h.logger.Info("Privileges dropped successfully",
			zap.Uint32("uid", targetUID),
			zap.Uint32("gid", targetGID))
	}
	return nil
}

// setCoreDumpLimits disables core dumps
func (h *StartupHardening) setCoreDumpLimits(ctx context.Context) error {
	var rLimit syscall.Rlimit
	rLimit.Max = 0
	rLimit.Cur = 0
	if err := syscall.Setrlimit(syscall.RLIMIT_CORE, &rLimit); err != nil {
		return fmt.Errorf("failed to disable core dumps: %w", err)
	}

	h.logger.Info("Core dumps disabled successfully")
	return nil
}

// setSecureEnvironment sets secure environment variables
func (h *StartupHardening) setSecureEnvironment(ctx context.Context) error {
	// Clear potentially dangerous environment variables
	dangerousEnvVars := []string{
		"LD_PRELOAD",
		"LD_LIBRARY_PATH",
		"DYLD_LIBRARY_PATH",
		"DYLD_INSERT_LIBRARIES",
	}

	for _, envVar := range dangerousEnvVars {
		if err := os.Unsetenv(envVar); err != nil {
			return fmt.Errorf("failed to unset %s: %w", envVar, err)
		}
	}

	// Set secure environment variables
	secureEnv := map[string]string{
		"GOMAXPROCS":   fmt.Sprintf("%d", runtime.NumCPU()),
		"GO_GCPERCENT": "100",
		"GOTRACEBACK":  "crash",
		"GOGC":         "100",
		"GODEBUG":      "gctrace=0",
		"TZ":           "UTC",
	}

	for key, value := range secureEnv {
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("failed to set %s: %w", key, err)
		}
	}

	h.logger.Info("Secure environment set successfully")
	return nil
}
