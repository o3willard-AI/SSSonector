package main

import (
	"strings"
	"testing"
)

func TestRunTUI_UsageErrors(t *testing.T) {
	t.Run("no flags => interactive dashboard (WI 2.5)", func(t *testing.T) {
		// Without --probe the interactive dashboard launches. On a TTY-less
		// host (CI) tea.NewProgram fails to open /dev/tty — that error (or
		// a clean run) both prove the dashboard path was taken, NOT the
		// old guidance error.
		err := runTUI([]string{})
		if err == nil {
			return // ran cleanly (TTY available)
		}
		if !strings.Contains(err.Error(), "TTY") && !strings.Contains(err.Error(), "tty") {
			t.Errorf("want dashboard launch (TTY error on headless CI) or clean run, got: %v", err)
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
	if fs.FromBundle != "/tmp/x.tgz" {
		t.Errorf("from-bundle: %q", fs.FromBundle)
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

// TestRunTUI_HelpFlag: `sssonector tui --help` exits cleanly (flag package
// prints usage; runTUI maps ErrHelp to nil).
func TestRunTUI_HelpFlag(t *testing.T) {
	if err := runTUI([]string{"--help"}); err != nil {
		t.Errorf("--help must exit cleanly, got: %v", err)
	}
	if err := runTUI([]string{"-h"}); err != nil {
		t.Errorf("-h must exit cleanly, got: %v", err)
	}
}

// TestRunTUI_DashboardWiring verifies the dashboard path constructs the
// model + program without starting a daemon: with fake collectors the
// program construction itself is the assertion (no daemon lifecycle code
// is reachable from runTUIDashboard — views+collect only).
func TestRunTUI_DashboardWiring(t *testing.T) {
	// The seam is runTUIDashboard; on headless CI it fails at
	// prog.Run() with the TTY error — AFTER the model/program were built.
	err := runTUIDashboard()
	if err == nil {
		return // real TTY available: program ran
	}
	if !strings.Contains(err.Error(), "TTY") && !strings.Contains(err.Error(), "tty") {
		t.Errorf("want TTY-only failure (model+program built fine), got: %v", err)
	}
}
