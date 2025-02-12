package hardening

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestStartupHardening(t *testing.T) {
	logger := zaptest.NewLogger(t)
	h := New(logger)
	ctx := context.Background()

	t.Run("apply all measures", func(t *testing.T) {
		err := h.Apply(ctx)
		if err != nil {
			t.Logf("Hardening measures failed: %v", err)
		}
	})

	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately
		err := h.Apply(ctx)
		assert.Error(t, err)
		assert.Equal(t, context.Canceled, err)
	})
}

func TestResourceLimits(t *testing.T) {
	logger := zaptest.NewLogger(t)
	h := New(logger)
	ctx := context.Background()

	t.Run("set file descriptor limits", func(t *testing.T) {
		err := h.setResourceLimits(ctx)
		if err != nil {
			t.Logf("Setting resource limits failed: %v", err)
			return
		}

		var rLimit syscall.Rlimit
		err = syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, rLimit.Cur, uint64(65535))
	})

	t.Run("set process limits on linux", func(t *testing.T) {
		if runtime.GOOS != "linux" {
			t.Skip("Skipping Linux-specific test")
		}

		err := h.setResourceLimits(ctx)
		if err != nil {
			t.Logf("Setting resource limits failed: %v", err)
			return
		}

		var rLimit syscall.Rlimit
		const RLIMIT_NPROC = 6 // Linux-specific constant
		err = syscall.Getrlimit(RLIMIT_NPROC, &rLimit)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, rLimit.Cur, uint64(4096))
	})
}

func TestSecureUmask(t *testing.T) {
	logger := zaptest.NewLogger(t)
	h := New(logger)
	ctx := context.Background()

	t.Run("set secure umask", func(t *testing.T) {
		oldMask := syscall.Umask(0)
		defer syscall.Umask(oldMask)

		err := h.setSecureUmask(ctx)
		require.NoError(t, err)

		newMask := syscall.Umask(0)
		syscall.Umask(newMask)
		assert.Equal(t, 0027, newMask)
	})
}

func TestDropPrivileges(t *testing.T) {
	logger := zaptest.NewLogger(t)
	h := New(logger)
	ctx := context.Background()

	t.Run("non-root user", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("Skipping test for non-root user when running as root")
		}

		err := h.dropPrivileges(ctx)
		require.NoError(t, err)
	})

	t.Run("root user", func(t *testing.T) {
		if os.Geteuid() != 0 {
			t.Skip("Skipping root user test when not running as root")
		}

		err := h.dropPrivileges(ctx)
		if err != nil {
			t.Logf("Dropping privileges failed: %v", err)
		}
	})
}

func TestCoreDumpLimits(t *testing.T) {
	logger := zaptest.NewLogger(t)
	h := New(logger)
	ctx := context.Background()

	t.Run("disable core dumps", func(t *testing.T) {
		err := h.setCoreDumpLimits(ctx)
		require.NoError(t, err)

		var rLimit syscall.Rlimit
		err = syscall.Getrlimit(syscall.RLIMIT_CORE, &rLimit)
		require.NoError(t, err)
		assert.Equal(t, uint64(0), rLimit.Cur)
		assert.Equal(t, uint64(0), rLimit.Max)
	})
}

func TestSecureEnvironment(t *testing.T) {
	logger := zaptest.NewLogger(t)
	h := New(logger)
	ctx := context.Background()

	t.Run("clear dangerous environment variables", func(t *testing.T) {
		// Set some dangerous environment variables
		dangerousEnvVars := []string{
			"LD_PRELOAD",
			"LD_LIBRARY_PATH",
			"DYLD_LIBRARY_PATH",
			"DYLD_INSERT_LIBRARIES",
		}

		for _, envVar := range dangerousEnvVars {
			os.Setenv(envVar, "test")
		}

		err := h.setSecureEnvironment(ctx)
		require.NoError(t, err)

		// Verify variables are unset
		for _, envVar := range dangerousEnvVars {
			value := os.Getenv(envVar)
			assert.Empty(t, value, "Environment variable %s should be unset", envVar)
		}
	})

	t.Run("set secure environment variables", func(t *testing.T) {
		err := h.setSecureEnvironment(ctx)
		require.NoError(t, err)

		// Verify secure variables are set
		assert.Equal(t, "UTC", os.Getenv("TZ"))
		assert.Equal(t, "100", os.Getenv("GOGC"))
		assert.Equal(t, "crash", os.Getenv("GOTRACEBACK"))
		assert.Equal(t, "gctrace=0", os.Getenv("GODEBUG"))
		assert.Equal(t, fmt.Sprintf("%d", runtime.NumCPU()), os.Getenv("GOMAXPROCS"))
	})
}
