package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

// TUIMode is where runTUI routes after flag parsing (WI 5.1, docs/tui.md
// §2 first-run detection + §3.3).
type TUIMode int

const (
	// ModeDashboard is the normal multi-instance dashboard.
	ModeDashboard TUIMode = iota
	// ModeServerWizard is the fresh-host first-run setup (WI 5.2).
	ModeServerWizard
	// ModeClientWizard is the --from-bundle client setup (WI 5.5).
	ModeClientWizard
)

// ModeBundlePath carries the --from-bundle value through mode selection
// (the bundle is NOT loaded until WI 5.5).
var ModeBundlePath string

// DetectFunc is the injectable discovery seam for mode selection (tests
// pass a fixture; production wires DiscoverForModeSelection over the real
// collector). No real systemctl runs from unit tests.
type DetectFunc func() (collect.DiscoveryState, error)

// SelectTUIMode picks ONE mode, in this priority order (docs/tui.md §2):
//  1. client-wizard: --from-bundle is set (overrides everything)
//  2. server-wizard: no units AND zero configs (truly fresh host)
//  3. dashboard: otherwise — units exist, OR at least one config exists,
//     OR the host is ambiguous (2+ configs: an explicit error the
//     dashboard renders, NOT a wizard trigger), OR discovery failed with
//     configs present.
//
// Pure and testable: all host probing happens through the injected
// DetectFunc.
func SelectTUIMode(f tuiFlags, detect DetectFunc) (TUIMode, error) {
	// Priority 1: the bundle flag routes to the client wizard regardless
	// of host state. The path is carried through; loading is WI 5.5.
	if f.FromBundle != "" {
		ModeBundlePath = f.FromBundle
		return ModeClientWizard, nil
	}

	st, err := detect()
	if err != nil {
		return ModeDashboard, fmt.Errorf("tui: mode detection: %w", err)
	}

	// Priority 2: truly fresh host — no units discovered AND zero configs
	// under the root. Discovery failure alone does NOT mean fresh (the
	// dashboard renders the discovery error); zero configs while systemd
	// is down does, matching the §7 degradation rule (exactly-one-config
	// fallback errors on zero configs ⇒ nothing to show anyway).
	if len(st.Units) == 0 && st.ConfigCount == 0 && !st.Ambiguous {
		return ModeServerWizard, nil
	}

	// Priority 3: dashboard (units exist, or a config exists, or the host
	// is ambiguous — the dashboard renders the explicit error).
	return ModeDashboard, nil
}

// runWizardPlaceholder renders the minimal WI 5.1 placeholder screen for
// a wizard mode: title + one help line + "press q to quit". The real
// wizard UI is WI 5.2 (server) / 5.5 (client).
func runWizardPlaceholder(mode TUIMode) error {
	var title, help string
	switch mode {
	case ModeServerWizard:
		title = "SSSonector — first-run setup (server)"
		help = "No configured instances found on this host. The setup wizard (WI 5.2) will create a server instance."
	case ModeClientWizard:
		title = "SSSonector — client setup from bundle"
		help = "Bundle: " + ModeBundlePath + " (loading arrives in WI 5.5)."
	default:
		return fmt.Errorf("tui: not a wizard mode: %v", mode)
	}
	body := strings.Join([]string{
		"┌─ " + title + " ─┐",
		"│ " + help,
		"│",
		"└─ press q to quit ─┘",
	}, "\n")
	// Placeholder: print and exit (the real wizard replaces tea Program).
	fmt.Println(body)
	return nil
}

// wizardPlaceholderModel is the minimal tea model so the routing is
// demonstrable inside tea.NewProgram (q quits). WI 5.2/5.5 replace it.
type wizardPlaceholderModel struct {
	title string
	help  string
}

func (m wizardPlaceholderModel) Init() tea.Cmd { return nil }

func (m wizardPlaceholderModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if k, ok := msg.(tea.KeyMsg); ok && (k.String() == "q" || k.String() == "ctrl+c") {
		return m, tea.Quit
	}
	return m, nil
}

func (m wizardPlaceholderModel) View() string {
	var b strings.Builder
	b.WriteString("┌─ " + m.title + " ─┐\n")
	b.WriteString("│ " + m.help + "\n")
	b.WriteString("│\n")
	b.WriteString("└─ press q to quit ─┘\n")
	return b.String()
}
