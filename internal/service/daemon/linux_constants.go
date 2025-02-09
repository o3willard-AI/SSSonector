package daemon

import "syscall"

const (
	// Process control
	PR_SET_NO_NEW_PRIVS = 38
	PR_GET_NO_NEW_PRIVS = 39

	// Mount flags
	MS_RDONLY      = 1
	MS_NOSUID      = 2
	MS_NODEV       = 4
	MS_NOEXEC      = 8
	MS_SYNCHRONOUS = 16
	MS_REMOUNT     = 32
	MS_MANDLOCK    = 64
	MS_DIRSYNC     = 128
	MS_NOATIME     = 1024
	MS_NODIRATIME  = 2048
	MS_BIND        = 4096
	MS_MOVE        = 8192
	MS_REC         = 16384
	MS_SILENT      = 32768
	MS_POSIXACL    = 1 << 16
	MS_UNBINDABLE  = 1 << 17
	MS_PRIVATE     = 1 << 18
	MS_SLAVE       = 1 << 19
	MS_SHARED      = 1 << 20
	MS_RELATIME    = 1 << 21
	MS_KERNMOUNT   = 1 << 22
	MS_I_VERSION   = 1 << 23
	MS_STRICTATIME = 1 << 24
	MS_LAZYTIME    = 1 << 25

	// Namespace flags
	CLONE_NEWNS   = 0x00020000 // Mount namespace
	CLONE_NEWUTS  = 0x04000000 // UTS namespace
	CLONE_NEWIPC  = 0x08000000 // IPC namespace
	CLONE_NEWUSER = 0x10000000 // User namespace
	CLONE_NEWPID  = 0x20000000 // PID namespace
	CLONE_NEWNET  = 0x40000000 // Network namespace
)

// Prctl is a wrapper for the prctl syscall
func Prctl(option int, arg2, arg3, arg4, arg5 uintptr) error {
	_, _, errno := syscall.Syscall6(syscall.SYS_PRCTL, uintptr(option), arg2, arg3, arg4, arg5, 0)
	if errno != 0 {
		return errno
	}
	return nil
}
