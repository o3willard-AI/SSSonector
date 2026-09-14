package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

// WI 5.1: first-run detection matrix — one test per docs/tui.md §2/§3.3
// row. All tests inject the discovery seam (fake runner); no real
// systemctl runs.

// fakeFirstRunRunner serves canned list-units AND per-unit show output.
type fakeFirstRunRunner struct {
	listUnitsOut string
	err          error
}

func (f *fakeFirstRunRunner) Run(args ...string) (string, error) {
	// `systemctl show <unit> -p ...` — canned state. The legacy
	// sssonector.service reports not-found unless the fixture overrides.
	if len(args) >= 2 && args[0] == "systemctl" && args[1] == "show" {
		if len(args) >= 3 && args[2] == "sssonector.service" && f.listUnitsOut == "" {
			return "ActiveState=not-found\nSubState=dead\nMainPID=0\n", nil
		}
		return "ActiveState=active\nSubState=running\nMainPID=100\n", nil
	}
	return f.listUnitsOut, f.err
}

// firstRunFixture builds a discovery seam over a temp config root and the
// given list-units output ("" + err = systemctl unavailable).
func firstRunFixture(t *testing.T, listOut string, listErr error) func() (collect.DiscoveryState, error) {
	t.Helper()
	root := t.TempDir()
	collect.SetDefaultSystemdPathsForTest(collect.SystemdPaths{ConfigRoot: root})
	t.Cleanup(func() { collect.ResetDefaultSystemdPathsForTest() })
	sys := collect.SystemdCollector{Runner: &fakeFirstRunRunner{listUnitsOut: listOut, err: listErr}}
	return func() (collect.DiscoveryState, error) {
		return collect.DiscoverForModeSelection(sys, collect.DefaultSystemdPathsForTest())
	}
}

func TestModeSelection_FreshHost_ServerWizard(t *testing.T) {
	// No units (systemctl works, returns nothing) + zero configs.
	detect := firstRunFixture(t, "", nil)
	mode, err := SelectTUIMode(tuiFlags{}, detect)
	if err != nil {
		t.Fatalf("fresh host must not error: %v", err)
	}
	if mode != ModeServerWizard {
		t.Errorf("fresh host must route to server-wizard, got %v", mode)
	}
}

func TestModeSelection_ExistingConfig_Dashboard(t *testing.T) {
	// Config exists (exactly one instance config), no units → dashboard.
	detect := firstRunFixture(t, "", nil)
	root := defaultConfigRootForTest()
	if err := os.MkdirAll(filepath.Join(root, "instances", "client-a"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "instances", "client-a", "config.yaml"), []byte("x: y\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	mode, err := SelectTUIMode(tuiFlags{}, detect)
	if err != nil {
		t.Fatalf("existing config must not error: %v", err)
	}
	if mode != ModeDashboard {
		t.Errorf("existing config (no units) must route to dashboard, got %v", mode)
	}
}

func TestModeSelection_UnitsOnly_Dashboard(t *testing.T) {
	// Units exist, no configs → dashboard.
	detect := firstRunFixture(t,
		"sssonector@client-a.service loaded active running x\n", nil)
	mode, err := SelectTUIMode(tuiFlags{}, detect)
	if err != nil {
		t.Fatalf("units-only must not error: %v", err)
	}
	if mode != ModeDashboard {
		t.Errorf("units-only must route to dashboard, got %v", mode)
	}
}

func TestModeSelection_BundleFlag_ClientWizard(t *testing.T) {
	// --from-bundle overrides everything — even a fresh host.
	detect := firstRunFixture(t, "", nil)
	mode, err := SelectTUIMode(tuiFlags{FromBundle: "/tmp/bundle.tgz"}, detect)
	if err != nil {
		t.Fatalf("bundle flag must not error: %v", err)
	}
	if mode != ModeClientWizard {
		t.Errorf("--from-bundle must route to client-wizard, got %v", mode)
	}
	if ModeBundlePath != "/tmp/bundle.tgz" {
		t.Errorf("bundle path must be carried through: %q", ModeBundlePath)
	}
}

func TestModeSelection_TwoConfigs_DashboardNotWizard(t *testing.T) {
	// 2+ instance configs is an explicit error rendered by the dashboard —
	// NOT a wizard trigger.
	detect := firstRunFixture(t, "", nil)
	root := defaultConfigRootForTest()
	for _, n := range []string{"client-a", "client-b"} {
		if err := os.MkdirAll(filepath.Join(root, "instances", n), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "instances", n, "config.yaml"), []byte("x: y\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	mode, err := SelectTUIMode(tuiFlags{}, detect)
	if err != nil {
		t.Fatalf("2+ configs must route (not error) to dashboard: %v", err)
	}
	if mode != ModeDashboard {
		t.Errorf("2+ configs must route to dashboard (error rendered there), got %v", mode)
	}
}

func TestModeSelection_ProbeAndUnknownFlags(t *testing.T) {
	// --probe routes to the probe before mode selection (unchanged).
	f, err := parseTUIFlags([]string{"--probe"})
	if err != nil || !f.probe {
		t.Fatalf("--probe must parse: %v", err)
	}
	// Unknown flag errors (never panics).
	if _, err := parseTUIFlags([]string{"--bogus"}); err == nil {
		t.Error("unknown flag must error")
	}
}

func TestModeSelection_SystemctlUnavailable(t *testing.T) {
	// systemctl fails + zero configs → server-wizard (fresh host).
	detect := firstRunFixture(t, "", errors.New("no systemd"))
	mode, err := SelectTUIMode(tuiFlags{}, detect)
	if err != nil {
		t.Fatalf("systemctl-unavailable fresh host must not error: %v", err)
	}
	if mode != ModeServerWizard {
		t.Errorf("systemctl-unavailable + zero configs must be server-wizard, got %v", mode)
	}
}

func TestModeSelection_DashboardStillRendersDiscoveryError(t *testing.T) {
	// The dashboard must still see the raw discovery error for ambiguous
	// hosts (2+ configs with systemctl down): the wizard detection must
	// agree with what the dashboard would discover — the mode function
	// must NOT swallow the ambiguity as wizard.
	detect := firstRunFixture(t, "", errors.New("no systemd"))
	root := defaultConfigRootForTest()
	for _, n := range []string{"client-a", "client-b"} {
		if err := os.MkdirAll(filepath.Join(root, "instances", n), 0o750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "instances", n, "config.yaml"), []byte("x: y\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	mode, err := SelectTUIMode(tuiFlags{}, detect)
	if err != nil {
		t.Fatalf("ambiguous host must route to dashboard (which renders the error): %v", err)
	}
	if mode != ModeDashboard {
		t.Errorf("ambiguous host must be dashboard, got %v", mode)
	}
}

func defaultConfigRootForTest() string {
	// SetDefaultSystemdPathsForTest stored the root; resolve it via the
	// discovery seam's paths (the fixture set ConfigRoot to a temp dir).
	return collect.DefaultSystemdPathsForTest()
}
