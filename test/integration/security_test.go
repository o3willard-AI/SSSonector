package integration

import (
	"os"
	"testing"

	"github.com/o3willard-AI/SSSonector/internal/security"
	"github.com/o3willard-AI/SSSonector/internal/service"
	"github.com/o3willard-AI/SSSonector/internal/service/daemon"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSecurityIntegration(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("Integration tests require root privileges")
	}

	logger, err := zap.NewDevelopment()
	require.NoError(t, err)

	t.Run("SecurityManagerWithDaemon", func(t *testing.T) {
		// Create security manager
		securityOpts := security.GetDefaultOptions()
		securityMgr, err := security.NewSecurityManager(securityOpts)
		require.NoError(t, err)

		// Create daemon with security manager
		daemonOpts := service.ServiceOptions{
			Name:        "sssonector-test",
			ConfigPath:  "/etc/sssonector/config.yaml",
			PIDFile:     "/var/run/sssonector-test.pid",
			LogFile:     "/var/log/sssonector-test.log",
			User:        "sssonector",
			Group:       "sssonector",
			WorkingDir:  "/var/lib/sssonector",
			Environment: []string{"ENV=test"},
		}

		d, err := daemon.NewLinux(daemonOpts, logger)
		require.NoError(t, err)

		// Apply security hardening
		err = securityMgr.Apply()
		require.NoError(t, err)

		// Start daemon
		err = d.Start()
		require.NoError(t, err)
		defer d.Stop()

		// Verify security settings
		helper := security.NewSecurityTestHelper(t)
		helper.VerifyProcessSecurity(securityMgr)
		helper.VerifyMemoryProtections(securityMgr)
		helper.VerifyResourceLimits(securityMgr)
		helper.VerifyCapabilities(securityMgr)
		helper.VerifySeccompFilter(securityMgr)

		// Verify daemon status
		status, err := d.Status()
		require.NoError(t, err)
		require.Equal(t, service.StateRunning, status)
	})

	t.Run("SecurityManagerReload", func(t *testing.T) {
		// Create security manager with initial options
		initialOpts := security.GetDefaultOptions()
		initialOpts.RlimitNofile = 1024
		securityMgr, err := security.NewSecurityManager(initialOpts)
		require.NoError(t, err)

		// Apply initial security settings
		err = securityMgr.Apply()
		require.NoError(t, err)

		// Create daemon
		daemonOpts := service.ServiceOptions{
			Name:    "sssonector-test",
			User:    "sssonector",
			Group:   "sssonector",
			LogFile: "/var/log/sssonector-test.log",
		}

		d, err := daemon.NewLinux(daemonOpts, logger)
		require.NoError(t, err)

		// Start daemon
		err = d.Start()
		require.NoError(t, err)
		defer d.Stop()

		// Update security options
		updatedOpts := initialOpts
		updatedOpts.RlimitNofile = 2048
		updatedOpts.RlimitNproc = 128

		// Create new security manager with updated options
		newSecurityMgr, err := security.NewSecurityManager(updatedOpts)
		require.NoError(t, err)

		// Apply updated security settings
		err = newSecurityMgr.Apply()
		require.NoError(t, err)

		// Reload daemon
		err = d.Reload()
		require.NoError(t, err)

		// Verify updated security settings
		helper := security.NewSecurityTestHelper(t)
		helper.VerifyResourceLimits(newSecurityMgr)

		// Verify daemon status
		status, err := d.Status()
		require.NoError(t, err)
		require.Equal(t, service.StateRunning, status)
	})

	t.Run("SecurityManagerFailure", func(t *testing.T) {
		// Create security manager with invalid options
		invalidOpts := security.SecurityOptions{
			SeccompMode: "invalid",
			RetainCaps:  []string{"INVALID_CAP"},
		}

		_, err := security.NewSecurityManager(invalidOpts)
		require.Error(t, err)
	})

	t.Run("SecurityManagerRecovery", func(t *testing.T) {
		// Create security manager
		opts := security.GetDefaultOptions()
		securityMgr, err := security.NewSecurityManager(opts)
		require.NoError(t, err)

		// Create daemon
		daemonOpts := service.ServiceOptions{
			Name:    "sssonector-test",
			User:    "sssonector",
			Group:   "sssonector",
			LogFile: "/var/log/sssonector-test.log",
		}

		d, err := daemon.NewLinux(daemonOpts, logger)
		require.NoError(t, err)

		// Start daemon
		err = d.Start()
		require.NoError(t, err)
		defer d.Stop()

		// Simulate failure by applying invalid security settings
		invalidOpts := opts
		invalidOpts.SeccompMode = "invalid"
		invalidSecurityMgr, err := security.NewSecurityManager(invalidOpts)
		require.NoError(t, err)

		err = invalidSecurityMgr.Apply()
		require.Error(t, err)

		// Verify daemon can recover
		err = d.Reload()
		require.NoError(t, err)

		// Verify original security settings still apply
		helper := security.NewSecurityTestHelper(t)
		helper.VerifyProcessSecurity(securityMgr)
		helper.VerifyMemoryProtections(securityMgr)
		helper.VerifyResourceLimits(securityMgr)
		helper.VerifyCapabilities(securityMgr)
		helper.VerifySeccompFilter(securityMgr)

		// Verify daemon status
		status, err := d.Status()
		require.NoError(t, err)
		require.Equal(t, service.StateRunning, status)
	})
}
