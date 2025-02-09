package daemon

import (
	"golang.org/x/sys/unix"
)

// Linux resource limit constants
const (
	RLIMIT_NPROC      = unix.RLIMIT_NPROC
	RLIMIT_MEMLOCK    = unix.RLIMIT_MEMLOCK
	RLIMIT_NICE       = unix.RLIMIT_NICE
	RLIMIT_RTPRIO     = unix.RLIMIT_RTPRIO
	RLIMIT_SIGPENDING = unix.RLIMIT_SIGPENDING
	RLIMIT_MSGQUEUE   = unix.RLIMIT_MSGQUEUE
)

// Linux namespace constants
const (
	CLONE_NEWNS   = unix.CLONE_NEWNS   // Mount namespace
	CLONE_NEWUTS  = unix.CLONE_NEWUTS  // UTS namespace
	CLONE_NEWIPC  = unix.CLONE_NEWIPC  // IPC namespace
	CLONE_NEWPID  = unix.CLONE_NEWPID  // PID namespace
	CLONE_NEWNET  = unix.CLONE_NEWNET  // Network namespace
	CLONE_NEWUSER = unix.CLONE_NEWUSER // User namespace
)

// Linux mount constants
const (
	MS_RDONLY = unix.MS_RDONLY // Mount read-only
	MS_BIND   = unix.MS_BIND   // Bind mount
)

// Linux process control constants
const (
	PR_SET_NO_NEW_PRIVS = unix.PR_SET_NO_NEW_PRIVS
)

// Prctl sets process control options
func Prctl(option int, arg2, arg3, arg4, arg5 uintptr) error {
	return unix.Prctl(option, arg2, arg3, arg4, arg5)
}
