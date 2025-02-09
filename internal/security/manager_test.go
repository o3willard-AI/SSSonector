package security

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSecurityManagerFeatures(t *testing.T) {
	helper := NewSecurityTestHelper(t)

	t.Run("NewSecurityManager", func(t *testing.T) {
		opts := GetDefaultOptions()
		mgr := helper.CreateManager(opts)
		require.NotNil(t, mgr)
		require.Equal(t, opts, mgr.options)
	})

	t.Run("Apply", func(t *testing.T) {
		if os.Getuid() != 0 {
			t.Skip("Test requires root privileges")
		}

		opts := GetDefaultOptions()
		mgr := helper.CreateManager(opts)

		err := mgr.Apply()
		require.NoError(t, err)

		helper.VerifyProcessSecurity(mgr)
		helper.VerifyMemoryProtections(mgr)
		helper.VerifyResourceLimits(mgr)
		helper.VerifyCapabilities(mgr)
		helper.VerifySeccompFilter(mgr)
	})

	t.Run("SecurityOptions", func(t *testing.T) {
		opts := SecurityOptions{
			NoNewPrivs:     true,
			SecureExec:     true,
			DropPrivileges: true,
			RetainCaps:     []string{"CAP_NET_ADMIN"},
			Denysetuid:     true,
			Mlock:          true,
			StackProtect:   true,
			RandomizeVA:    true,
			SeccompMode:    "filtered",
			AllowedSyscalls: []string{
				"read",
				"write",
				"exit",
			},
			NoCoreDump:   true,
			RlimitNofile: 1024,
			RlimitNproc:  64,
		}

		mgr := helper.CreateManager(opts)
		require.Equal(t, opts, mgr.options)
	})

	t.Run("DefaultOptions", func(t *testing.T) {
		opts := GetDefaultOptions()
		helper.VerifyOptions(opts)
	})

	t.Run("ProcessSecurity", func(t *testing.T) {
		if os.Getuid() != 0 {
			t.Skip("Test requires root privileges")
		}

		opts := GetDefaultOptions()
		mgr := helper.CreateManager(opts)

		err := mgr.setProcessSecurity()
		require.NoError(t, err)

		helper.VerifyProcessSecurity(mgr)
	})

	t.Run("MemoryProtections", func(t *testing.T) {
		if os.Getuid() != 0 {
			t.Skip("Test requires root privileges")
		}

		opts := GetDefaultOptions()
		mgr := helper.CreateManager(opts)

		err := mgr.setMemoryProtections()
		require.NoError(t, err)

		helper.VerifyMemoryProtections(mgr)
	})

	t.Run("ResourceLimits", func(t *testing.T) {
		if os.Getuid() != 0 {
			t.Skip("Test requires root privileges")
		}

		opts := GetDefaultOptions()
		mgr := helper.CreateManager(opts)

		err := mgr.setResourceLimits()
		require.NoError(t, err)

		helper.VerifyResourceLimits(mgr)
	})

	t.Run("Capabilities", func(t *testing.T) {
		if os.Getuid() != 0 {
			t.Skip("Test requires root privileges")
		}

		opts := GetDefaultOptions()
		mgr := helper.CreateManager(opts)

		err := mgr.dropCapabilities()
		require.NoError(t, err)

		helper.VerifyCapabilities(mgr)
	})

	t.Run("Seccomp", func(t *testing.T) {
		opts := GetDefaultOptions()
		mgr := helper.CreateManager(opts)

		err := mgr.initSeccomp()
		require.NoError(t, err)

		helper.VerifySeccompFilter(mgr)
	})
}
