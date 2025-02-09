package integration

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/service/daemon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestSecurityNamespaces(t *testing.T) {
	// Set up test logger
	logger := zaptest.NewLogger(t)

	// Create temporary PID file
	pidFile := filepath.Join(t.TempDir(), "test.pid")

	// Create daemon
	d, err := daemon.NewLinuxDaemon(logger, pidFile)
	require.NoError(t, err)

	// Configure namespaces
	config := daemon.GetDefaultNamespaceConfig()
	config.Network.Enabled = true
	config.Network.Type = "bridge"
	config.Mount.Enabled = true
	config.PID.Enabled = true
	config.IPC.Enabled = true
	config.UTS.Enabled = true
	config.User.Enabled = false

	err = d.Configure(config, daemon.GetDefaultConfig())
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

	// Stop daemon
	err = d.Stop()
	require.NoError(t, err)
}

func TestSecurityCgroups(t *testing.T) {
	// Set up test logger
	logger := zaptest.NewLogger(t)

	// Create temporary PID file
	pidFile := filepath.Join(t.TempDir(), "test.pid")

	// Create daemon
	d, err := daemon.NewLinuxDaemon(logger, pidFile)
	require.NoError(t, err)

	// Configure cgroups
	config := daemon.GetDefaultConfig()
	config.Memory.Limit = 1024 * 1024 * 100 // 100MB
	config.CPU.Shares = 512
	config.PIDs.Limit = 100

	err = d.Configure(daemon.GetDefaultNamespaceConfig(), config)
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
	assert.NotNil(t, status.Stats)
	assert.Less(t, status.Stats.Memory.Usage, uint64(1024*1024*100))

	// Stop daemon
	err = d.Stop()
	require.NoError(t, err)
}

func TestSecurityLimits(t *testing.T) {
	// Set up test logger
	logger := zaptest.NewLogger(t)

	// Create temporary PID file
	pidFile := filepath.Join(t.TempDir(), "test.pid")

	// Create daemon
	d, err := daemon.NewLinuxDaemon(logger, pidFile)
	require.NoError(t, err)

	// Configure resource limits
	config := daemon.GetDefaultConfig()
	config.Memory.Limit = 1024 * 1024 * 50 // 50MB
	config.CPU.Shares = 256
	config.PIDs.Limit = 50

	err = d.Configure(daemon.GetDefaultNamespaceConfig(), config)
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
	assert.NotNil(t, status.Stats)
	assert.Less(t, status.Stats.Memory.Usage, uint64(1024*1024*50))

	// Stop daemon
	err = d.Stop()
	require.NoError(t, err)
}
