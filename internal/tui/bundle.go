package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

// WI 5.4 [g]: generate a client bundle for the focused SERVER instance
// (docs/tui.md §3.3, §4 keybindings). Non-destructive — no confirm dialog.
// The heavy lifting lives in collect.GenerateClientBundle behind the
// injectable BundleGenFunc seam (production: the real generator; tests:
// recording fakes — no real cert minting from the tui package tests).

// bundleState holds the WI 5.4 bundle seam + outcome banner (separate
// field so it never fights the lifecycle/config banners).
type bundleState struct {
	gen    collect.BundleGenFunc
	banner lifecycleBanner
}

// WithBundleGen wires the [g] seam (WI 5.4). Nil keeps the key inert
// (fail-closed: an explicit "not configured" banner, never a panic).
func (m dashboardModel) WithBundleGen(f collect.BundleGenFunc) dashboardModel {
	m.bundle.gen = f
	return m
}

// handleBundleKey processes `g` while the dashboard is active. Returns
// (model, handled). Runs synchronously (cert mint is seconds-scale, RSA
// 2048) and surfaces the outcome in a dedicated banner.
func (m dashboardModel) handleBundleKey(msg tea.KeyMsg) (tea.Model, bool) {
	if msg.String() != "g" {
		return m, false
	}
	snap := m.focusedSnapshot()
	if snap == nil {
		return m, true // nothing focused: no action possible
	}
	if m.bundle.gen == nil {
		m.bundle.banner = lifecycleBanner{kind: bannerBundleError, text: "bundle generator not configured"}
		return m, true
	}
	b, err := m.bundle.gen(m.deps.Paths, snap.Name)
	if err != nil {
		m.bundle.banner = lifecycleBanner{kind: bannerBundleError, text: err.Error()}
		return m, true
	}
	m.bundle.banner = lifecycleBanner{
		kind: bannerBundleOK,
		text: fmt.Sprintf("client bundle: %s (sha256 %s)", b.Path, b.SHA256),
	}
	return m, true
}

// renderBundleBanner renders the [g] outcome banner (or "").
func (m dashboardModel) renderBundleBanner() string {
	switch m.bundle.banner.kind {
	case bannerBundleOK:
		return "banner: ✓ " + m.bundle.banner.text
	case bannerBundleError:
		return "banner: ✗ " + m.bundle.banner.text
	}
	return ""
}
