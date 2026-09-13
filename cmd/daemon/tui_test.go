package main

import (
	"strings"
	"testing"
)

func TestRunTUI_UsageErrors(t *testing.T) {
	t.Run("no flags => guidance error", func(t *testing.T) {
		err := runTUI([]string{})
		if err == nil || !strings.Contains(err.Error(), "nothing to do") {
			t.Errorf("want guidance error, got %v", err)
		}
	})
	t.Run("unexpected positional arg", func(t *testing.T) {
		err := runTUI([]string{"--probe", "stray"})
		if err == nil || !strings.Contains(err.Error(), "unexpected argument") {
			t.Errorf("want unexpected-argument error, got %v", err)
		}
	})
}

// Flag-parse semantics: unknown flags must be a flag-package error. The
// parse logic is shared via parseTUIFlags so tests avoid flag.ExitOnError's
// os.Exit; runTUI uses it too (one parse implementation).
func TestRunTUI_UnknownFlag(t *testing.T) {
	fs, err := parseTUIFlags([]string{"--bogus"})
	if err == nil {
		t.Errorf("unknown flag must error, got fs=%+v", fs)
	}
	if fs != nil {
		t.Errorf("on error the flagset is nil, got %+v", fs)
	}
}

// Accepted-but-inert flags parse fine.
func TestRunTUI_AcceptedFlags(t *testing.T) {
	fs, err := parseTUIFlags([]string{"--probe", "--instance", "client-a", "--refresh", "2s", "--from-bundle", "/tmp/x.tgz"})
	if err != nil {
		t.Fatalf("accepted flags: %v", err)
	}
	if !fs.probe {
		t.Error("probe flag")
	}
	if fs.instance != "client-a" {
		t.Error("instance flag")
	}
	if fs.refresh.String() != "2s" {
		t.Error("refresh flag")
	}
	if fs.fromBundle != "/tmp/x.tgz" {
		t.Errorf("from-bundle: %q", fs.fromBundle)
	}
}

// TestRunTUI_ProbeNoSystemd exercises the real-defaults wiring on this
// host (no sssonector configs): the probe must take the fatal (exit-1)
// path or degrade — either way, no panic. We call the probe body directly
// (runTUIProbe) and check it does not panic; it returns (out, fatal).
func TestRunTUI_ProbeNoSystemd(t *testing.T) {
	out, fatal := runTUIProbe()
	if fatal {
		if !strings.Contains(out, "discovery: ERROR") && !strings.Contains(out, "no sssonector config found") {
			t.Errorf("fatal output should explain why: %s", out)
		}
	} else {
		if !strings.Contains(out, "probe time:") {
			t.Errorf("probe output missing header: %s", out)
		}
	}
}
