//go:build !windows

package collect

import "syscall"

// SignalHUP is the production SignalFunc on POSIX: syscall.Kill(pid, SIGHUP).
// Unit tests never call it — they inject fakes (see apply.go).
func SignalHUP(pid int) error {
	return syscall.Kill(pid, syscall.SIGHUP)
}
