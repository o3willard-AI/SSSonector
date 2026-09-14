// Package main - tui subcommand: a submode of the existing daemon binary
// (AGENTS.md single entry point). It never starts the daemon lifecycle;
// like provision, it bypasses service startup entirely.
package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/o3willard-AI/SSSonector/internal/tui"
	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

// tuiFlags holds the parsed tui subcommand flags.
type tuiFlags struct {
	probe      bool
	instance   string
	refresh    time.Duration
	FromBundle string // --from-bundle: routes to the client wizard (WI 5.1; loading is WI 5.5)
}

// parseTUIFlags parses tui flags (unknown flags error, never panic).
// runTUI wraps this with ExitOnError semantics; tests use this seam.
func parseTUIFlags(args []string) (*tuiFlags, error) {
	fs := flag.NewFlagSet("tui", flag.ContinueOnError)
	fs.SetOutput(nil)
	f := &tuiFlags{}
	fs.BoolVar(&f.probe, "probe", false, "assemble and print a one-shot per-instance snapshot, then exit")
	fs.StringVar(&f.instance, "instance", "", "instance to focus (accepted; wired in a later phase)")
	fs.DurationVar(&f.refresh, "refresh", time.Second, "refresh interval (accepted; wired in a later phase)")
	fs.StringVar(&f.FromBundle, "from-bundle", "", "client bundle path (routes to the client setup wizard; bundle loading is WI 5.5)")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() > 0 {
		return nil, fmt.Errorf("tui: unexpected argument %q", fs.Arg(0))
	}
	return f, nil
}

// runTUI dispatches the tui subcommand (WI 5.1 routing): --probe → probe;
// --from-bundle → client wizard; fresh host (no units + zero configs) →
// server wizard; otherwise the dashboard.
func runTUI(args []string) error {
	f, err := parseTUIFlags(args)
	if err != nil {
		// -h/--help: flag package prints usage; exit cleanly.
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if f.probe {
		out, fatal := runTUIProbe()
		fmt.Print(out)
		if fatal {
			os.Exit(1)
		}
		return nil
	}

	// First-run mode selection (WI 5.1): reuses the SAME discovery the
	// dashboard uses (systemd collector + config fallback) — never a
	// second discovery path.
	sys := collect.SystemdCollector{Runner: collect.OSCommandRunner{}, Paths: collect.DefaultSystemdPaths()}
	detect := func() (collect.DiscoveryState, error) {
		return collect.DiscoverForModeSelection(sys, collect.DefaultSystemdPaths().ConfigRoot)
	}
	mode, err := SelectTUIMode(*f, detect)
	if err != nil {
		return err
	}
	switch mode {
	case ModeClientWizard, ModeServerWizard:
		return runWizard(mode)
	default:
		return runTUIDashboard()
	}
}

// runTUIDashboard launches the interactive dashboard (WI 2.5): real
// collectors, real Poller seam, tea.NewProgram with alt-screen. Runs
// views+collect only — never the daemon service lifecycle.
func runTUIDashboard() error {
	return runTUIDashboardFocused("")
}

// runTUIDashboardFocused launches the dashboard with an instance
// pre-focused (WI 5.3: the wizard lands here after create & start, on the
// new instance). Empty focus keeps the default (first sorted instance).
func runTUIDashboardFocused(focus string) error {
	paths := collect.DefaultConfigPaths()
	poller := collect.NewPoller(
		collect.SystemdCollector{Runner: collect.OSCommandRunner{}, Paths: collect.DefaultSystemdPaths()},
		paths,
		&http.Client{Timeout: 3 * time.Second},
	)
	poll := func(context.Context) collect.TickResult { return poller.PollOnce(context.Background()) }

	// Production config-view deps (WI 3.4): the real dump/validate/apply,
	// the real SIGHUP, and a reload reader that tails the daemon log for
	// the reload outcome.
	deps := tui.ConfigDeps{
		Dump:     collect.EffectiveConfigDump,
		Validate: validateDraftErr,
		Apply:    collect.ApplyConfig,
		Paths:    paths,
		Signal:   collect.SignalHUP,
		Read:     collect.LogReloadReader(collect.OSCommandRunner{}, unitForPid, 5*time.Second),
	}

	// Production lifecycle deps (WI 4.1): the real systemctl runner for
	// stop/restart, the real SIGHUP, and the same reload reader for R.
	lc := tui.LifecycleDeps{
		Runner: collect.OSCommandRunner{},
		Signal: collect.SignalHUP,
		Read:   collect.LogReloadReader(collect.OSCommandRunner{}, unitForPid, 5*time.Second),
	}

	model := tui.NewDashboardWithLifecycle(poll, time.Now, 0, deps, lc).WithFocus(focus)
	prog := tea.NewProgram(model, tea.WithAltScreen())
	_, err := prog.Run()
	if err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	return nil
}

// validateDraftErr adapts collect.ValidateDraft to the deps signature.
func validateDraftErr(draft string) error {
	_, err := collect.ValidateDraft(draft)
	return err
}

// unitForPid resolves the systemd unit owning a pid via systemctl status
// (best effort; falls back to the legacy unit name).
func unitForPid(pid int) string {
	out, err := (collect.OSCommandRunner{}).Run("systemctl", "status", fmt.Sprintf("%d", pid), "--no-pager")
	if err != nil {
		return "sssonector.service"
	}
	for _, line := range strings.Split(out, "\n") {
		if i := strings.Index(line, "sssonector@"); i >= 0 {
			rest := line[i:]
			if j := strings.IndexAny(rest, " \t"); j > 0 {
				return rest[:j]
			}
		}
	}
	return "sssonector.service"
}

// runTUIProbe builds the real collectors (systemd defaults, /etc/sssonector
// root, default HTTP client), runs one PollOnce + per-instance log tails,
// and renders via FormatProbe. Returns (text, fatal).
func runTUIProbe() (string, bool) {
	poller := collect.NewPoller(
		collect.SystemdCollector{Runner: collect.OSCommandRunner{}, Paths: collect.DefaultSystemdPaths()},
		collect.DefaultConfigPaths(),
		&http.Client{Timeout: 3 * time.Second},
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := poller.PollOnce(ctx)

	lt := collect.NewLogTail(collect.OSCommandRunner{})
	logs := make(map[string][]collect.LogEntry, len(result.Instances))
	for name, snap := range result.Instances {
		if snap.Unit == "" {
			continue
		}
		read, err := lt.Read(snap.Unit, "", "", 12)
		if err == nil {
			logs[name] = read.Entries
		}
	}
	return collect.FormatProbe(result, logs)
}
