package tui

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

// viewMode is the dashboard's top-level mode.
type viewMode int

const (
	modeDashboard viewMode = iota
	modeConfig
	modeConfirm // WI 4.1: destructive-action confirm dialog
)

// bannerKind classifies the config-view banner.
type bannerKind int

const (
	bannerNone bannerKind = iota
	bannerValidateOK
	bannerValidateFail
	bannerApplyOK
	bannerApplyRejected
	bannerApplyError
	bannerInfo
	// WI 4.1 lifecycle outcomes (continue the same iota family).
	bannerLifecycleOK       // stop/restart command accepted
	bannerLifecycleError    // systemctl command failed
	bannerReloadOK          // SIGHUP reload accepted
	bannerReloadRejected    // daemon rejected the reload
	bannerReloadError       // signal/outcome-read failure
	bannerLifecycleDisabled // action disabled (e.g. s on stopped instance)
)

// configModel is the config-view state (a sub-model of the dashboard).
// The draft is the full editable text; editing replaces the selected
// value on the selected line.
type configModel struct {
	instance string
	lines    []collect.ConfigLine // the dump (draft rendered from these)
	sel      int                  // selected line index
	editing  bool                 // Enter began an edit
	editBuf  string               // current edit text
	draft    string               // full draft text (rendered for apply)
	banner   bannerKind
	bannerUp string
}

// ConfigDeps are the injectable config-view operations (WI 3.2/3.3 seams).
// Production wires the real functions in cmd/daemon/tui.go; tests inject
// fakes. Exported so the wiring gap found at the Phase 3 rig gate cannot
// regress: the constructor requires them explicitly.
type ConfigDeps struct {
	// Dump renders the effective-config dump (collect.EffectiveConfigDump).
	Dump func(paths collect.ConfigPaths, name string) ([]collect.ConfigLine, error)
	// Validate runs the draft through the daemon loader+validator
	// (collect.ValidateDraft) and returns its error.
	Validate func(draft string) error
	// Apply is the atomic write + SIGHUP pipeline (collect.ApplyConfig).
	Apply func(paths collect.ConfigPaths, name, draft string, pid int, signal collect.SignalFunc, read collect.ReloadReader) (collect.ApplyResult, error)
	// Paths is the config root (collect.DefaultConfigPaths() in production).
	Paths collect.ConfigPaths
	// Signal sends SIGHUP (collect.SignalHUP in production).
	Signal collect.SignalFunc
	// Read reads the reload outcome after the signal.
	Read collect.ReloadReader
}

// openConfig builds the config view for the focused instance from the dump.
func openConfig(deps ConfigDeps, instance string) (configModel, error) {
	if deps.Dump == nil {
		return configModel{}, fmt.Errorf("config: dump function not configured")
	}
	lines, err := deps.Dump(deps.Paths, instance)
	if err != nil {
		return configModel{}, err
	}
	return configModel{
		instance: instance,
		lines:    lines,
		draft:    renderYAMLFromLines(lines),
		banner:   bannerInfo,
		bannerUp: "editing the effective config — Enter edit · v validate · a apply · Esc discard",
	}, nil
}

// draftFromLines re-renders the draft as YAML after an edit (the editable
// surface is the line VALUE on the selected line). ValidateDraft (WI 3.2)
// requires real YAML — the human dump format is display-only.
func (c configModel) draftFromLines() string {
	return renderYAMLFromLines(c.lines)
}

// renderYAMLFromLines reconstructs YAML text from dotted ConfigLine paths.
// Values are quoted as strings EXCEPT true/false/int-ish values, so the
// loader parses types the same way the original file did. Metadata
// timestamps render quoted (they contain colons).
func renderYAMLFromLines(lines []collect.ConfigLine) string {
	type node struct {
		children map[string]*node
		value    string
		leaf     bool
	}
	root := &node{children: map[string]*node{}}
	for _, l := range lines {
		parts := strings.Split(l.Path, ".")
		cur := root
		for _, p := range parts {
			if cur.children[p] == nil {
				cur.children[p] = &node{children: map[string]*node{}}
			}
			cur = cur.children[p]
		}
		cur.value = l.Value
		cur.leaf = true
	}
	var b strings.Builder
	var walk func(n *node, indent int)
	writeLeaf := func(v string) string {
		switch {
		case v == "true" || v == "false":
			return v
		case isNumericLiteral(v):
			return v
		case v == "[]":
			return "[]"
		default:
			return fmt.Sprintf("%q", v)
		}
	}
	walk = func(n *node, indent int) {
		keys := make([]string, 0, len(n.children))
		for k := range n.children {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			child := n.children[k]
			pad := strings.Repeat("  ", indent)
			if child.leaf && len(child.children) == 0 {
				fmt.Fprintf(&b, "%s%s: %s\n", pad, k, writeLeaf(child.value))
			} else {
				fmt.Fprintf(&b, "%s%s:\n", pad, k)
				walk(child, indent+1)
			}
		}
	}
	walk(root, 0)

	// Wrap with the loader-required top-level fields: metadata
	// (schema_version + environment, quoted) and type. Without these the
	// loader rejects the draft outright.
	meta := map[string]string{
		"schema_version": "2.0.0",
		"environment":    "development",
	}
	var out strings.Builder
	out.WriteString("metadata:\n")
	for _, k := range []string{"environment", "schema_version"} {
		fmt.Fprintf(&out, "  %s: %q\n", k, meta[k])
	}
	out.WriteString("type: \"server\"\n")
	out.WriteString("config:\n")
	// Re-indent the config body one level under "config:".
	for _, line := range strings.Split(strings.TrimRight(b.String(), "\n"), "\n") {
		if line == "" {
			out.WriteString("\n")
			continue
		}
		out.WriteString("  " + line + "\n")
	}
	return out.String()
}

// isNumericLiteral reports whether v parses as a bare YAML number.
func isNumericLiteral(v string) bool {
	if v == "" {
		return false
	}
	for _, r := range v {
		if (r < '0' || r > '9') && r != '.' {
			return false
		}
	}
	return true
}

// applyEdit sets the selected line's value from the edit buffer.
func (c *configModel) applyEdit() {
	if c.sel >= 0 && c.sel < len(c.lines) {
		c.lines[c.sel].Value = c.editBuf
		c.lines[c.sel].Origin = collect.OriginUser // an edit is by definition user-set
	}
	c.draft = c.draftFromLines()
	c.editing = false
}

// render renders the config overlay (deterministic text).
func (c configModel) render(focus string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "CONFIG — instance %s\n", focus)
	b.WriteString(strings.Repeat("─", 78) + "\n")
	b.WriteString(c.draft)
	b.WriteString(strings.Repeat("─", 78) + "\n")
	if c.editing {
		fmt.Fprintf(&b, "editing [%s] = %s (Enter commit, Esc cancel line)\n",
			c.lines[c.sel].Path, c.editBuf)
	}
	switch c.banner {
	case bannerValidateOK:
		b.WriteString("banner: ✓ draft valid\n")
	case bannerValidateFail:
		fmt.Fprintf(&b, "banner: ✗ VALIDATION FAILED: %s\n", c.bannerUp)
	case bannerApplyOK:
		b.WriteString("banner: ✓ applied — daemon reloaded the new config\n")
	case bannerApplyRejected:
		b.WriteString("banner: ⚠ applied but daemon REJECTED the reload — file written, daemon keeps its old config\n")
	case bannerApplyError:
		fmt.Fprintf(&b, "banner: ✗ APPLY ERROR: %s\n", c.bannerUp)
	default:
		if c.bannerUp != "" {
			fmt.Fprintf(&b, "banner: %s\n", c.bannerUp)
		}
	}
	b.WriteString("footer: Enter edit · v validate · a apply · Esc discard\n")
	return b.String()
}

// handleConfigKey processes keys while modeConfig is active. Returns
// (model, cmd, stayInConfig). All operations use the injected deps — no
// real signal is ever sent when the deps are fakes.
func (m dashboardModel) handleConfigKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	c := m.config
	switch msg.String() {
	case "esc":
		if c.editing {
			// Cancel the line edit only.
			c.editing = false
			c.editBuf = ""
			m.config = c
			return m, nil, true
		}
		// Discard draft, return to dashboard: no write, no signal.
		m.mode = modeDashboard
		m.config = configModel{}
		return m, nil, false
	case "up", "k":
		if !c.editing && c.sel > 0 {
			c.sel--
		}
	case "down", "j":
		if !c.editing && c.sel < len(c.lines)-1 {
			c.sel++
		}
	case "enter":
		if !c.editing && c.sel < len(c.lines) {
			c.editing = true
			c.editBuf = c.lines[c.sel].Value
		} else if c.editing {
			c.applyEdit()
		}
	case "v":
		if c.editing {
			c.applyEdit()
		}
		if m.deps.Validate == nil {
			c.banner = bannerValidateFail
			c.bannerUp = "validate function not configured"
			break
		}
		if err := m.deps.Validate(c.draft); err != nil {
			c.banner = bannerValidateFail
			c.bannerUp = err.Error() // verbatim (carries "line N")
		} else {
			c.banner = bannerValidateOK
			c.bannerUp = ""
		}
	case "a":
		// Fail-closed: validate FIRST; an invalid draft never reaches apply.
		if err := m.deps.Validate(c.draft); err != nil {
			c.banner = bannerValidateFail
			c.bannerUp = err.Error()
			m.config = c
			return m, nil, true
		}
		pid := 0
		if snap := m.focusedSnapshot(); snap != nil {
			pid = snap.MainPID
		}
		res, err := m.deps.Apply(m.deps.Paths, c.instance, c.draft, pid, m.deps.Signal, m.deps.Read)
		switch {
		case err != nil:
			c.banner = bannerApplyError
			c.bannerUp = err.Error()
		case res.Outcome == collect.ReloadOK:
			c.banner = bannerApplyOK
			c.bannerUp = ""
		default:
			c.banner = bannerApplyRejected
			c.bannerUp = ""
		}
	}
	m.config = c
	return m, nil, true
}

// editLineText is a test/keyboard helper: replace the edit buffer's text
// with literal runes (simulating typing a new value).
func (c *configModel) editLineText(s string) {
	c.editBuf = s
}
