//go:build !windows

package collect

import (
	"os/exec"
	"syscall"
	"testing"
	"time"
)

// TestSignalHUPKillsChild exercises the production SignalFunc on POSIX:
// the child must die from the delivered SIGHUP (syscall.Kill is the only
// production signal path; apply-pipeline tests inject fakes and never
// reach it, so it is proven directly here).
func TestSignalHUPKillsChild(t *testing.T) {
	if _, err := exec.LookPath("sleep"); err != nil {
		t.Skipf("sleep not available: %v", err)
	}
	child := exec.Command("sleep", "60")
	if err := child.Start(); err != nil {
		t.Fatalf("start child: %v", err)
	}
	defer child.Process.Kill() //nolint:errcheck // best-effort if the test fails early

	if err := SignalHUP(child.Process.Pid); err != nil {
		t.Fatalf("SignalHUP(pid=%d): %v", child.Process.Pid, err)
	}

	done := make(chan error, 1)
	go func() { done <- child.Wait() }()
	select {
	case err := <-done:
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("child wait: %v", err)
		}
		ws, ok := ee.Sys().(syscall.WaitStatus)
		if !ok || !ws.Signaled() || ws.Signal() != syscall.SIGHUP {
			t.Fatalf("child exit = %+v, want death by SIGHUP", ee.Sys())
		}
	case <-time.After(10 * time.Second):
		t.Fatal("child did not exit within 10s of SIGHUP")
	}
}
