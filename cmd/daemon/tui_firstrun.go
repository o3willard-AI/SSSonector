package main

import (
	"fmt"
	"os"
	"path/filepath"
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

// runWizard launches the wizard for the given mode. The server wizard is
// the real form (WI 5.2); the client wizard stays a stub until WI 5.5.
func runWizard(mode TUIMode) error {
	switch mode {
	case ModeServerWizard:
		form := newWizardForm(
			collect.ValidateDraft,
			collect.NewSSPortProbe(collect.OSCommandRunner{}),
			nil, // fresh host: no existing instances
		)
		// enable/start need root: route through sudo -A when SUDO_ASKPASS
		// is configured (askpass); otherwise the OS runner as before.
		form.runner = sudoRunner{inner: collect.OSCommandRunner{}}
		prog := tea.NewProgram(serverWizardModel{form: form})
		final, err := prog.Run()
		if err != nil {
			return fmt.Errorf("tui: %w", err)
		}
		// On success, land on the dashboard focused on the new instance
		// (docs/tui.md §3.3: create & start … lands on the dashboard).
		if wm, ok := final.(serverWizardModel); ok && wm.form.done {
			return runTUIDashboardFocused(wm.form.createdIns)
		}
		return nil
	case ModeClientWizard:
		// WI 5.5: the real client wizard — bundle prefill + chain
		// pre-flight + create through the WI 5.3 pipeline.
		m := newClientWizard(ModeBundlePath, os.TempDir)
		prog := tea.NewProgram(m)
		final, err := prog.Run()
		if err != nil {
			return fmt.Errorf("tui: %w", err)
		}
		if cm, ok := final.(clientWizardModel); ok && cm.done {
			return runTUIDashboardFocused(cm.instance)
		}
		return nil
	default:
		return fmt.Errorf("tui: not a wizard mode: %v", mode)
	}
}

// serverWizardModel is the tea wrapper around wizardForm.
type serverWizardModel struct {
	form    wizardForm
	lastErr string // verbatim loader error from the last create attempt
}

func (m serverWizardModel) Init() tea.Cmd { return nil }

// Update implements the §3.3 keymap: Tab next field · ↑↓ edit ·
// [Enter] create & start · [Esc] abort.
func (m serverWizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch k.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.form.aborted = true
		return m, tea.Quit
	case "tab":
		m.form.focus = (m.form.focus + 1) % wfCount
		return m, nil
	case "up":
		if m.form.focus > 0 {
			m.form.focus--
		}
		return m, nil
	case "down":
		if m.form.focus < wfCount-1 {
			m.form.focus++
		}
		return m, nil
	case "enter":
		m.handleEnter()
		return m, nil
	case " ", "x":
		m.handleToggle()
		return m, nil
	}
	// Text entry always lands in the focused text field (no separate
	// edit mode — per-keystroke validation sees every rune). The buffer
	// RESETS when the field changes: applyBuf writes the whole buffer,
	// so switching fields must not leak the previous field's text.
	if isTextField(m.form.focus) {
		if m.form.bufField != m.form.focus {
			m.form.bufField = m.form.focus
			m.form.buf = m.form.fieldValue(m.form.focus)
		}
		switch k.String() {
		case "backspace":
			if r := []rune(m.form.buf); len(r) > 0 {
				m.form.buf = string(r[:len(r)-1])
			}
		default:
			if len(k.Runes) > 0 {
				m.form.buf += string(k.Runes)
			}
		}
		m.applyBuf()
	}
	return m, nil
}

// fieldValue returns the current value of a text field (for buffer sync).
func (f wizardForm) fieldValue(field wizardField) string {
	switch field {
	case wfInstance:
		return f.instance
	case wfPort:
		return f.port
	case wfTun:
		return f.tun
	}
	return ""
}

// handleEnter commits the focused radio (mode: Server — the only wired
// choice) or, when every field is ready, runs create (WI 5.3 no-op:
// print the validated draft). NOT ready ⇒ create is a no-op (blocked).
func (m *serverWizardModel) handleEnter() {
	switch m.form.focus {
	case wfMode:
		m.form.modeChosen = true
		m.form.modeClient = false
	default:
		if m.form.ready() && !m.form.done {
			draft, err := m.form.draft()
			if err != nil {
				m.lastErr = err.Error() // verbatim loader/validator error
				return
			}
			// WI 5.3/5.6: CERTS=generate-new must mint the instance CA +
			// server leaf BEFORE enable/start — the daemon refuses to start
			// without TLS. (reuse-host needs the shared store present; that
			// is an operator pre-provision.)
			if m.form.cert == certGenerateNew {
				ins := strings.TrimSpace(m.form.instance)
				certDir := filepath.Join(m.form.paths.ConfigRoot, "instances", ins, "certs")
				if err := os.MkdirAll(certDir, 0o750); err != nil {
					m.form.createErr = "create: mkdir certs: " + err.Error()
					return
				}
				if m.form.genCerts == nil {
					m.form.createErr = "create: no cert generator wired"
					return
				}
				if err := m.form.genCerts(certDir); err != nil {
					m.form.createErr = "create: generate certs: " + err.Error()
					return
				}
			}
			// WI 5.3: the REAL write path — atomic write the config,
			// then enable+start, via the injectable runner (order
			// guaranteed by CreateAndStartInstance: file before unit).
			res, cerr := m.form.create(m.form.paths, strings.TrimSpace(m.form.instance), draft, m.form.runner)
			if cerr != nil {
				m.form.createErr = cerr.Error() // verbatim; file stays written on enable/start failure
				return
			}
			m.form.done = true
			m.form.createdIns = strings.TrimSpace(m.form.instance)
			_ = res
		}
	}
}

// handleToggle flips the checkbox/radio state of the focused field with
// explicit keys (space/x). Nothing is ever pre-selected.
func (m *serverWizardModel) handleToggle() {
	switch m.form.focus {
	case wfNat:
		switch m.form.nat {
		case natUnset, natDisabled:
			m.form.nat = natEnabled
		case natEnabled:
			m.form.nat = natDisabled
		}
	case wfCerts:
		switch m.form.cert {
		case certNone, certGenerateNew:
			m.form.cert = certReuseHost
		case certReuseHost:
			m.form.cert = certGenerateNew
		}
	}
}

// applyBuf copies the edit buffer into the focused field's value.
func (m *serverWizardModel) applyBuf() {
	switch m.form.focus {
	case wfInstance:
		m.form.instance = m.form.buf
	case wfPort:
		m.form.port = m.form.buf
	case wfTun:
		m.form.tun = m.form.buf
	}
}

func isTextField(f wizardField) bool {
	return f == wfInstance || f == wfPort || f == wfTun
}

// View renders the §3.3 form with the live per-keystroke validation line.
func (m serverWizardModel) View() string {
	f := m.form
	var b strings.Builder
	b.WriteString("┌─ SSSonector — first-run setup ─┐\n")
	b.WriteString("│ No configured instances found on this host.\n")
	b.WriteString("│\n")
	b.WriteString("│ MODE        ")
	if !f.modeChosen {
		b.WriteString("[ ] Server   [ ] Client   (choose — never defaulted)")
	} else if f.modeClient {
		b.WriteString("[ ] Server   [x] Client   (stub — WI 5.5)")
	} else {
		b.WriteString("[x] Server   [ ] Client")
	}
	b.WriteString("\n")
	b.WriteString("│ INSTANCE    " + f.instance + "\n")
	b.WriteString("│ LISTEN PORT " + f.port + "\n")
	b.WriteString("│ TUN ADDRESS " + f.tun + "\n")
	switch f.nat {
	case natEnabled:
		b.WriteString("│ FORWARD NAT [x] enabled   (egress 192.168.100.0/24 — default-deny, edit ACL after setup)\n")
	case natDisabled:
		b.WriteString("│ FORWARD NAT [ ] enabled\n")
	default:
		b.WriteString("│ FORWARD NAT [ ] enabled   (unset — choose explicitly)\n")
	}
	switch f.cert {
	case certReuseHost:
		b.WriteString("│ CERTS       [x] reuse host CA & certs (/etc/sssonector/certs)\n")
		b.WriteString("│             [ ] generate new instance CA + leaf now\n")
	case certGenerateNew:
		b.WriteString("│ CERTS       [ ] reuse host CA & certs (/etc/sssonector/certs)\n")
		b.WriteString("│             [x] generate new instance CA + leaf now\n")
	default:
		b.WriteString("│ CERTS       [ ] reuse host CA & certs\n")
		b.WriteString("│             [ ] generate new instance CA + leaf now   (fail-closed: choose one)\n")
	}
	b.WriteString("│\n")
	if f.done {
		b.WriteString("│ ✓ created & started sssonector@" + f.createdIns + ".service — opening the dashboard…\n")
	} else if f.createErr != "" {
		b.WriteString("│ ✗ CREATE FAILED: " + f.createErr + "\n")
	} else if errs := f.fieldErrs(); len(errs) > 0 {
		b.WriteString("│ validation: (create DISABLED)\n")
		for _, e := range errs {
			b.WriteString("│   ✗ " + e + "\n")
		}
	} else {
		b.WriteString("│ validation: ✓ all fields valid — [Enter] create & start\n")
	}
	if m.lastErr != "" {
		b.WriteString("│ " + m.lastErr + "\n")
	}
	b.WriteString("└─ Tab next field · ↑↓ move · space toggle · type to edit · Enter create · Esc abort ─┘\n")
	return b.String()
}
