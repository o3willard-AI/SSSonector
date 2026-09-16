//go:build windows

package collect

import "fmt"

// SignalHUP is unsupported on Windows: there are no POSIX signals to send.
// The TUI apply path is exercised on Linux/macOS rigs; this stub exists so
// the windows release binary compiles (syscall.Kill does not exist there).
// It fails closed — callers surface the error instead of silently assuming
// the daemon reloaded.
func SignalHUP(pid int) error {
	return fmt.Errorf("signal: SIGHUP to pid %d unsupported on windows; restart the service to reload its config", pid)
}