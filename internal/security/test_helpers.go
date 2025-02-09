package security

import (
	"fmt"
	"testing"

	seccomp "github.com/seccomp/libseccomp-golang"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

// SecurityTestHelper provides test utilities for security manager
type SecurityTestHelper struct {
	t *testing.T
}

// NewSecurityTestHelper creates a new security test helper
func NewSecurityTestHelper(t *testing.T) *SecurityTestHelper {
	t.Helper()
	return &SecurityTestHelper{t: t}
}

// CreateManager creates a security manager for testing
func (h *SecurityTestHelper) CreateManager(opts SecurityOptions) *SecurityManager {
	mgr, err := NewSecurityManager(opts)
	require.NoError(h.t, err)
	require.NotNil(h.t, mgr)
	return mgr
}

// VerifyOptions verifies security options
func (h *SecurityTestHelper) VerifyOptions(opts SecurityOptions) {
	require.True(h.t, opts.NoNewPrivs, "NoNewPrivs should be enabled")
	require.True(h.t, opts.SecureExec, "SecureExec should be enabled")
	require.True(h.t, opts.DropPrivileges, "DropPrivileges should be enabled")
	require.NotEmpty(h.t, opts.RetainCaps, "RetainCaps should not be empty")
	require.True(h.t, opts.Mlock, "Mlock should be enabled")
	require.True(h.t, opts.StackProtect, "StackProtect should be enabled")
	require.True(h.t, opts.RandomizeVA, "RandomizeVA should be enabled")
	require.NotEmpty(h.t, opts.AllowedSyscalls, "AllowedSyscalls should not be empty")
}

// VerifySeccompFilter verifies seccomp filter configuration
func (h *SecurityTestHelper) VerifySeccompFilter(mgr *SecurityManager) {
	require.NotNil(h.t, mgr.seccompFilter, "Seccomp filter should be initialized")

	// Verify allowed syscalls
	for _, syscallName := range mgr.options.AllowedSyscalls {
		id, err := seccomp.GetSyscallFromName(syscallName)
		require.NoError(h.t, err)
		// Verify syscall is allowed
		err = mgr.seccompFilter.AddRule(id, seccomp.ActKill)
		require.Error(h.t, err, "Should not be able to add conflicting rule")
	}
}

// VerifyCapabilities verifies capability configuration
func (h *SecurityTestHelper) VerifyCapabilities(mgr *SecurityManager) {
	for i := 0; i <= int(unix.CAP_LAST_CAP); i++ {
		keep := false
		for _, cap := range mgr.options.RetainCaps {
			if cap == fmt.Sprintf("CAP_%d", i) {
				keep = true
				break
			}
		}
		if !keep {
			r1, _, errno := unix.Syscall(unix.SYS_PRCTL, unix.PR_CAPBSET_READ, uintptr(i), 0)
			require.Zero(h.t, errno, "Syscall error")
			require.Equal(h.t, uintptr(0), r1)
		}
	}
}

// VerifyResourceLimits verifies resource limit configuration
func (h *SecurityTestHelper) VerifyResourceLimits(mgr *SecurityManager) {
	// Verify file limits
	var nofile unix.Rlimit
	err := unix.Getrlimit(unix.RLIMIT_NOFILE, &nofile)
	require.NoError(h.t, err)
	require.Equal(h.t, uint64(mgr.options.RlimitNofile), nofile.Max)

	// Verify process limits
	var nproc unix.Rlimit
	err = unix.Getrlimit(unix.RLIMIT_NPROC, &nproc)
	require.NoError(h.t, err)
	require.Equal(h.t, uint64(mgr.options.RlimitNproc), nproc.Max)

	// Verify core dump disabled
	var core unix.Rlimit
	err = unix.Getrlimit(unix.RLIMIT_CORE, &core)
	require.NoError(h.t, err)
	require.Equal(h.t, uint64(0), core.Max)
}

// VerifyMemoryProtections verifies memory protection configuration
func (h *SecurityTestHelper) VerifyMemoryProtections(mgr *SecurityManager) {
	// Verify memory locking
	var memlock unix.Rlimit
	err := unix.Getrlimit(unix.RLIMIT_MEMLOCK, &memlock)
	require.NoError(h.t, err)
	require.NotEqual(h.t, ^uint64(0), memlock.Max)

	// Verify address space limit
	var as unix.Rlimit
	err = unix.Getrlimit(unix.RLIMIT_AS, &as)
	require.NoError(h.t, err)
	require.NotEqual(h.t, ^uint64(0), as.Max)
}

// VerifyProcessSecurity verifies process security configuration
func (h *SecurityTestHelper) VerifyProcessSecurity(mgr *SecurityManager) {
	// Verify no new privileges
	r1, _, errno := unix.Syscall(unix.SYS_PRCTL, unix.PR_GET_NO_NEW_PRIVS, 0, 0)
	require.Zero(h.t, errno, "Syscall error")
	require.Equal(h.t, uintptr(1), r1)

	// Verify secure bits
	r1, _, errno = unix.Syscall(unix.SYS_PRCTL, unix.PR_GET_SECUREBITS, 0, 0)
	require.Zero(h.t, errno, "Syscall error")
	require.NotEqual(h.t, uintptr(0), r1&uintptr(SECBIT_KEEP_CAPS))
	require.NotEqual(h.t, uintptr(0), r1&uintptr(SECBIT_KEEP_CAPS_LOCKED))
}
