package integration

import (
	"os"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/service"
	"github.com/o3willard-AI/SSSonector/internal/service/daemon"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestDaemonIntegration(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("Integration tests require root privileges")
	}

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	t.Run("DaemonLifecycle", func(t *testing.T) {
		// Create daemon
		opts := service.ServiceOptions{
			Name:        "sssonector-test",
			ConfigPath:  "/etc/sssonector/config.yaml",
			PIDFile:     "/var/run/sssonector-test.pid",
			LogFile:     "/var/log/sssonector-test.log",
			User:        "sssonector",
			Group:       "sssonector",
			WorkingDir:  "/var/lib/sssonector",
			Environment: []string{"ENV=test"},
		}

		d, err := daemon.NewLinux(opts, logger)
		require.NoError(t, err)

		// Start daemon
		err = d.Start()
		require.NoError(t, err)

		// Verify daemon is running
		status, err := d.Status()
		require.NoError(t, err)
		require.Equal(t, service.StateRunning, status)

		// Get metrics
		metrics, err := d.GetMetrics()
		require.NoError(t, err)
		require.NotZero(t, metrics.StartTime)
		require.Equal(t, "linux", metrics.Platform)

		// Check health
		err = d.Health()
		require.NoError(t, err)

		// Stop daemon
		err = d.Stop()
		require.NoError(t, err)

		// Verify daemon is stopped
		status, err = d.Status()
		require.NoError(t, err)
		require.Equal(t, service.StateStopped, status)
	})

	t.Run("DaemonReload", func(t *testing.T) {
		// Create daemon
		opts := service.ServiceOptions{
			Name:    "sssonector-test",
			PIDFile: "/var/run/sssonector-test.pid",
			LogFile: "/var/log/sssonector-test.log",
		}

		d, err := daemon.NewLinux(opts, logger)
		require.NoError(t, err)

		// Start daemon
		err = d.Start()
		require.NoError(t, err)
		defer d.Stop()

		// Reload daemon
		err = d.Reload()
		require.NoError(t, err)

		// Verify daemon is running
		status, err := d.Status()
		require.NoError(t, err)
		require.Equal(t, service.StateRunning, status)

		// Get metrics
		metrics, err := d.GetMetrics()
		require.NoError(t, err)
		require.NotZero(t, metrics.LastReload)
	})

	t.Run("DaemonCommand", func(t *testing.T) {
		// Create daemon
		opts := service.ServiceOptions{
			Name:    "sssonector-test",
			PIDFile: "/var/run/sssonector-test.pid",
			LogFile: "/var/log/sssonector-test.log",
		}

		d, err := daemon.NewLinux(opts, logger)
		require.NoError(t, err)

		// Start daemon
		err = d.Start()
		require.NoError(t, err)
		defer d.Stop()

		// Execute status command
		cmd := service.ServiceCommand{
			Command:   service.CmdStatus,
			Type:      "status",
			RequestID: "test-1",
		}

		response, err := d.ExecuteCommand(cmd)
		require.NoError(t, err)
		require.Equal(t, service.StateRunning, response)

		// Execute metrics command
		cmd = service.ServiceCommand{
			Command:   service.CmdMetrics,
			Type:      "metrics",
			RequestID: "test-2",
		}

		response, err = d.ExecuteCommand(cmd)
		require.NoError(t, err)
		require.Contains(t, response, "CPUUsage")
		require.Contains(t, response, "MemoryUsage")
	})

	t.Run("DaemonRecovery", func(t *testing.T) {
		// Create daemon
		opts := service.ServiceOptions{
			Name:    "sssonector-test",
			PIDFile: "/var/run/sssonector-test.pid",
			LogFile: "/var/log/sssonector-test.log",
		}

		d, err := daemon.NewLinux(opts, logger)
		require.NoError(t, err)

		// Start daemon
		err = d.Start()
		require.NoError(t, err)
		defer d.Stop()

		// Simulate failure
		err = os.Remove(opts.PIDFile)
		require.NoError(t, err)

		// Wait for recovery
		time.Sleep(time.Second)

		// Verify daemon is still running
		status, err := d.Status()
		require.NoError(t, err)
		require.Equal(t, service.StateRunning, status)
	})
}
