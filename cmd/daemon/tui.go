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
	fromBundle string
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
	fs.StringVar(&f.fromBundle, "from-bundle", "", "client bundle path (accepted; wired in a later phase)")
	if err := fs.Parse(args); err != nil {
		return nil, err
	}
	if fs.NArg() > 0 {
		return nil, fmt.Errorf("tui: unexpected argument %q", fs.Arg(0))
	}
	return f, nil
}

// runTUI dispatches the tui subcommand. This WI implements --probe only;
// --instance/--refresh/--from-bundle are accepted for later phases.
func runTUI(args []string) error {
	f, err := parseTUIFlags(args)
	if err != nil {
		// -h/--help: flag package prints usage; exit cleanly.
		if err == flag.ErrHelp {
			return nil
		}
		return err
	}
	if !f.probe {
		return runTUIDashboard()
	}

	out, fatal := runTUIProbe()
	fmt.Print(out)
	if fatal {
		os.Exit(1)
	}
	return nil
}

// runTUIDashboard launches the interactive dashboard (WI 2.5): real
// collectors, real Poller seam, tea.NewProgram with alt-screen. Runs
// views+collect only — never the daemon service lifecycle.
func runTUIDashboard() error {
	poller := collect.NewPoller(
		collect.SystemdCollector{Runner: collect.OSCommandRunner{}, Paths: collect.DefaultSystemdPaths()},
		collect.DefaultConfigPaths(),
		&http.Client{Timeout: 3 * time.Second},
	)
	poll := func(context.Context) collect.TickResult { return poller.PollOnce(context.Background()) }

	model := tui.NewDashboard(poll, time.Now)
	prog := tea.NewProgram(model, tea.WithAltScreen())
	_, err := prog.Run()
	if err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	return nil
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
