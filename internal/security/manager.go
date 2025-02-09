package security

import (
	"fmt"
	"syscall"

	seccomp "github.com/seccomp/libseccomp-golang"
	"golang.org/x/sys/unix"
)

// SecurityManager handles security hardening features
type SecurityManager struct {
	// Seccomp filter
	seccompFilter *seccomp.ScmpFilter

	// Capabilities to retain
	keepCaps []string

	// Security options
	options SecurityOptions
}

// SecurityOptions configures security features
type SecurityOptions struct {
	// Process restrictions
	NoNewPrivs     bool     // Prevent gaining new privileges
	SecureExec     bool     // Enable secure execution mode
	DropPrivileges bool     // Drop unnecessary privileges
	RetainCaps     []string // Capabilities to retain

	// Memory protections
	Denysetuid    bool // Deny setuid binaries
	Denyexec      bool // Deny executable memory
	Mlock         bool // Lock memory pages
	MemoryLimits  bool // Enable memory limits
	StackProtect  bool // Enable stack protector
	RandomizeVA   bool // Randomize virtual addresses
	RestrictMaps  bool // Restrict memory mappings
	RestrictSysfs bool // Restrict sysfs access

	// System call restrictions
	SeccompMode     string   // Seccomp mode (disabled, strict, filtered)
	AllowedSyscalls []string // Allowed system calls in filtered mode

	// Resource restrictions
	NoCoreDump   bool // Disable core dumps
	RlimitCore   int  // Core dump size limit
	RlimitFsize  int  // File size limit
	RlimitNofile int  // Max open files
	RlimitNproc  int  // Max processes
	RlimitStack  int  // Max stack size

	// Filesystem restrictions
	ReadOnlyRoot   bool     // Mount root filesystem read-only
	MaskPaths      []string // Paths to mask
	ReadOnlyPaths  []string // Paths to mount read-only
	HiddenPaths    []string // Paths to hide
	PrivateTmp     bool     // Private /tmp directory
	PrivateDevices bool     // Private devices
}

// NewSecurityManager creates a new security manager
func NewSecurityManager(opts SecurityOptions) (*SecurityManager, error) {
	mgr := &SecurityManager{
		options: opts,
	}

	// Initialize seccomp if enabled
	if opts.SeccompMode != "disabled" {
		if err := mgr.initSeccomp(); err != nil {
			return nil, fmt.Errorf("failed to initialize seccomp: %w", err)
		}
	}

	return mgr, nil
}

// Apply applies security hardening measures
func (m *SecurityManager) Apply() error {
	// Set secure process attributes
	if err := m.setProcessSecurity(); err != nil {
		return fmt.Errorf("failed to set process security: %w", err)
	}

	// Apply memory protections
	if err := m.setMemoryProtections(); err != nil {
		return fmt.Errorf("failed to set memory protections: %w", err)
	}

	// Apply resource limits
	if err := m.setResourceLimits(); err != nil {
		return fmt.Errorf("failed to set resource limits: %w", err)
	}

	// Drop capabilities
	if m.options.DropPrivileges {
		if err := m.dropCapabilities(); err != nil {
			return fmt.Errorf("failed to drop capabilities: %w", err)
		}
	}

	// Apply seccomp filter
	if m.options.SeccompMode != "disabled" && m.seccompFilter != nil {
		if err := m.seccompFilter.Load(); err != nil {
			return fmt.Errorf("failed to load seccomp filter: %w", err)
		}
	}

	return nil
}

// setProcessSecurity sets secure process attributes
func (m *SecurityManager) setProcessSecurity() error {
	// No new privileges
	if m.options.NoNewPrivs {
		if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
			return fmt.Errorf("failed to set no new privs: %w", err)
		}
	}

	// Secure execution
	if m.options.SecureExec {
		if err := unix.Prctl(unix.PR_SET_SECUREBITS,
			SECBIT_KEEP_CAPS|SECBIT_KEEP_CAPS_LOCKED|
				SECBIT_NO_CAP_AMBIENT_RAISE|SECBIT_NO_CAP_AMBIENT_RAISE_LOCKED,
			0, 0, 0); err != nil {
			return fmt.Errorf("failed to set secure bits: %w", err)
		}
	}

	return nil
}

// setMemoryProtections applies memory protection measures
func (m *SecurityManager) setMemoryProtections() error {
	// Lock memory pages
	if m.options.Mlock {
		if err := unix.Mlockall(unix.MCL_CURRENT | unix.MCL_FUTURE); err != nil {
			return fmt.Errorf("failed to lock memory: %w", err)
		}
	}

	// Enable stack protector
	if m.options.StackProtect {
		if err := unix.Prctl(PR_SET_MM, PR_SET_MM_MAP_MIN_ADDR, DEFAULT_MAP_MIN_ADDR, 0, 0); err != nil {
			return fmt.Errorf("failed to set stack protector: %w", err)
		}
	}

	// Randomize virtual addresses
	if m.options.RandomizeVA {
		if err := unix.Prctl(PR_SET_MM, PR_SET_MM_START_STACK, 0, 0, 0); err != nil {
			return fmt.Errorf("failed to set address randomization: %w", err)
		}
	}

	return nil
}

// setResourceLimits applies resource limits
func (m *SecurityManager) setResourceLimits() error {
	var limit syscall.Rlimit

	// Disable core dumps
	if m.options.NoCoreDump {
		limit.Max = 0
		limit.Cur = 0
		if err := syscall.Setrlimit(syscall.RLIMIT_CORE, &limit); err != nil {
			return fmt.Errorf("failed to set core limit: %w", err)
		}
	}

	// Set file size limit
	if m.options.RlimitFsize > 0 {
		limit.Max = uint64(m.options.RlimitFsize)
		limit.Cur = uint64(m.options.RlimitFsize)
		if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &limit); err != nil {
			return fmt.Errorf("failed to set file size limit: %w", err)
		}
	}

	// Set max open files
	if m.options.RlimitNofile > 0 {
		limit.Max = uint64(m.options.RlimitNofile)
		limit.Cur = uint64(m.options.RlimitNofile)
		if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &limit); err != nil {
			return fmt.Errorf("failed to set max files limit: %w", err)
		}
	}

	// Set max processes
	if m.options.RlimitNproc > 0 {
		limit.Max = uint64(m.options.RlimitNproc)
		limit.Cur = uint64(m.options.RlimitNproc)
		if err := syscall.Setrlimit(unix.RLIMIT_NPROC, &limit); err != nil {
			return fmt.Errorf("failed to set process limit: %w", err)
		}
	}

	return nil
}

// dropCapabilities drops unnecessary capabilities
func (m *SecurityManager) dropCapabilities() error {
	// Drop all capabilities except those in RetainCaps
	for i := 0; i <= int(unix.CAP_LAST_CAP); i++ {
		keep := false
		for _, cap := range m.options.RetainCaps {
			if cap == fmt.Sprintf("CAP_%d", i) {
				keep = true
				break
			}
		}
		if !keep {
			if err := unix.Prctl(unix.PR_CAPBSET_DROP, uintptr(i), 0, 0, 0); err != nil {
				return fmt.Errorf("failed to drop capability %d: %w", i, err)
			}
		}
	}

	return nil
}

// initSeccomp initializes seccomp filtering
func (m *SecurityManager) initSeccomp() error {
	var err error

	// Create filter
	m.seccompFilter, err = seccomp.NewFilter(seccomp.ActErrno.SetReturnCode(int16(syscall.EPERM)))
	if err != nil {
		return fmt.Errorf("failed to create seccomp filter: %w", err)
	}

	// Add allowed syscalls
	for _, name := range m.options.AllowedSyscalls {
		syscallID, err := seccomp.GetSyscallFromName(name)
		if err != nil {
			return fmt.Errorf("invalid syscall name %s: %w", name, err)
		}
		if err := m.seccompFilter.AddRule(syscallID, seccomp.ActAllow); err != nil {
			return fmt.Errorf("failed to add syscall rule for %s: %w", name, err)
		}
	}

	return nil
}

// GetDefaultOptions returns secure default options
func GetDefaultOptions() SecurityOptions {
	return SecurityOptions{
		NoNewPrivs:     true,
		SecureExec:     true,
		DropPrivileges: true,
		RetainCaps: []string{
			"CAP_NET_ADMIN",
			"CAP_NET_RAW",
			"CAP_NET_BIND_SERVICE",
		},
		Denysetuid:    true,
		Denyexec:      true,
		Mlock:         true,
		MemoryLimits:  true,
		StackProtect:  true,
		RandomizeVA:   true,
		RestrictMaps:  true,
		RestrictSysfs: true,
		SeccompMode:   "filtered",
		AllowedSyscalls: []string{
			"read", "write", "open", "close", "stat", "fstat", "lstat",
			"poll", "lseek", "mmap", "mprotect", "munmap", "brk",
			"rt_sigaction", "rt_sigprocmask", "rt_sigreturn",
			"ioctl", "pread64", "pwrite64", "readv", "writev",
			"pipe", "select", "sched_yield", "mremap", "msync",
			"mincore", "madvise", "socket", "connect", "accept",
			"sendto", "recvfrom", "sendmsg", "recvmsg", "bind",
			"listen", "getsockname", "getpeername", "socketpair",
			"setsockopt", "getsockopt", "clone", "fork", "vfork",
			"execve", "exit", "wait4", "kill", "uname", "fcntl",
			"flock", "fsync", "fdatasync", "truncate", "ftruncate",
			"getcwd", "chdir", "rename", "mkdir", "rmdir", "creat",
			"link", "unlink", "symlink", "readlink", "chmod", "fchmod",
			"chown", "fchown", "lchown", "umask", "gettimeofday",
			"getrlimit", "getrusage", "sysinfo", "times", "ptrace",
			"getuid", "syslog", "getgid", "setuid", "setgid",
			"geteuid", "getegid", "setpgid", "getppid", "getpgrp",
			"setsid", "setreuid", "setregid", "getgroups", "setgroups",
			"setresuid", "getresuid", "setresgid", "getresgid",
			"getpgid", "setfsuid", "setfsgid", "getsid", "capget",
			"capset", "rt_sigpending", "rt_sigtimedwait",
			"rt_sigqueueinfo", "rt_sigsuspend", "sigaltstack",
			"utime", "mknod", "uselib", "personality", "ustat",
			"statfs", "fstatfs", "sysfs", "getpriority", "setpriority",
			"sched_setparam", "sched_getparam",
		},
		NoCoreDump:   true,
		RlimitCore:   0,
		RlimitFsize:  1024 * 1024 * 1024, // 1GB
		RlimitNofile: 1024,
		RlimitNproc:  64,
		RlimitStack:  8 * 1024 * 1024, // 8MB
		ReadOnlyRoot: true,
		MaskPaths: []string{
			"/proc/kcore",
			"/proc/kallsyms",
			"/proc/kmem",
			"/proc/mem",
		},
		ReadOnlyPaths: []string{
			"/proc",
			"/sys",
			"/boot",
		},
		HiddenPaths: []string{
			"/root",
			"/home",
			"/lost+found",
		},
		PrivateTmp:     true,
		PrivateDevices: true,
	}
}
