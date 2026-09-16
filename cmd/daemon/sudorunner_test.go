package main

import (
	"strings"
	"testing"
)

// WI 5.3/rig: the wizard's enable/start runner must route through
// `sudo -A` when SUDO_ASKPASS is set (rig-verified: plain sudo fails with
// "a terminal is required" even with an askpass helper configured), and
// must pass through unchanged when it is not.
type srRecorder struct {
	calls [][]string
}

func (r *srRecorder) Run(args ...string) (string, error) {
	r.calls = append(r.calls, args)
	return "", nil
}

func TestSudoRunner_AskpassPrefix(t *testing.T) {
	t.Setenv("SUDO_ASKPASS", "/tmp/askpass.sh")
	inner := &srRecorder{}
	sr := sudoRunner{inner: inner}
	if _, err := sr.Run("systemctl", "enable", "sssonector@x.service"); err != nil {
		t.Fatal(err)
	}
	if len(inner.calls) != 1 {
		t.Fatalf("calls: %d", len(inner.calls))
	}
	got := strings.Join(inner.calls[0], " ")
	if got != "sudo -A systemctl enable sssonector@x.service" {
		t.Errorf("argv: %q", got)
	}
}

func TestSudoRunner_PassthroughWithoutAskpass(t *testing.T) {
	t.Setenv("SUDO_ASKPASS", "")
	inner := &srRecorder{}
	sr := sudoRunner{inner: inner}
	if _, err := sr.Run("systemctl", "start", "sssonector@x.service"); err != nil {
		t.Fatal(err)
	}
	got := strings.Join(inner.calls[0], " ")
	if got != "systemctl start sssonector@x.service" {
		t.Errorf("argv must be unchanged without SUDO_ASKPASS: %q", got)
	}
}
