package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/service/daemon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestDaemonLifecycle(t *testing.T) {
	// Set up test logger
	logger := zaptest.NewLogger(t)

	// Create temporary PID file
	pidFile := filepath.Join(t.TempDir(), "test.pid")

	// Create daemon
	d, err := daemon.NewLinuxDaemon(logger, pidFile)
	require.NoError(t, err)

	// Start daemon in background
	go func() {
		err := d.Start()
		require.NoError(t, err)
	}()

	// Wait for daemon to start
	time.Sleep(100 * time.Millisecond)

	// Check status
	status, err := d.Status()
	require.NoError(t, err)
	assert.True(t, status.Running)
	assert.Greater(t, status.Pid, 0)

	// Stop daemon
	err = d.Stop()
	require.NoError(t, err)

	// Check status again
	status, err = d.Status()
	require.NoError(t, err)
	assert.False(t, status.Running)
}

func TestDaemonReload(t *testing.T) {
	// Set up test logger
	logger := zaptest.NewLogger(t)

	// Create temporary PID file
	pidFile := filepath.Join(t.TempDir(), "test.pid")

	// Create daemon
	d, err := daemon.NewLinuxDaemon(logger, pidFile)
	require.NoError(t, err)

	// Start daemon in background
	go func() {
		err := d.Start()
		require.NoError(t, err)
	}()

	// Wait for daemon to start
	time.Sleep(100 * time.Millisecond)

	// Check status
	status, err := d.Status()
	require.NoError(t, err)
	assert.True(t, status.Running)

	// Reload daemon
	err = d.Reload()
	require.NoError(t, err)

	// Stop daemon
	err = d.Stop()
	require.NoError(t, err)
}

func TestDaemonPIDFile(t *testing.T) {
	// Set up test logger
	logger := zaptest.NewLogger(t)

	// Create temporary PID file
	pidFile := filepath.Join(t.TempDir(), "test.pid")

	// Create daemon
	d, err := daemon.NewLinuxDaemon(logger, pidFile)
	require.NoError(t, err)

	// Start daemon in background
	go func() {
		err := d.Start()
		require.NoError(t, err)
	}()

	// Wait for daemon to start
	time.Sleep(100 * time.Millisecond)

	// Check PID file
	pidBytes, err := os.ReadFile(pidFile)
	require.NoError(t, err)
	pid := string(pidBytes)
	assert.NotEmpty(t, pid)

	// Stop daemon
	err = d.Stop()
	require.NoError(t, err)

	// Check PID file is removed
	_, err = os.Stat(pidFile)
	assert.True(t, os.IsNotExist(err))
}

func TestDaemonExecuteCommand(t *testing.T) {
	// Set up test logger
	logger := zaptest.NewLogger(t)

	// Create temporary PID file
	pidFile := filepath.Join(t.TempDir(), "test.pid")

	// Create daemon
	d, err := daemon.NewLinuxDaemon(logger, pidFile)
	require.NoError(t, err)

	// Start daemon in background
	go func() {
		err := d.Start()
		require.NoError(t, err)
	}()

	// Wait for daemon to start
	time.Sleep(100 * time.Millisecond)

	// Execute command
	ctx := context.Background()
	output, err := d.ExecuteCommand(ctx, "echo", "hello")
	require.NoError(t, err)
	assert.Contains(t, output, "hello")

	// Stop daemon
	err = d.Stop()
	require.NoError(t, err)
}
