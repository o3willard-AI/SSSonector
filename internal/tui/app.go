// tuiApp is the TUI application entry point (Phase 2).
//
// WI 2.1: minimal import stub so the pinned terminal library
// (charmbracelet/bubbletea v1.3.6) survives go mod tidy. The real
// dashboard model/update/view loop lands in WI 2.2 (panel renderers) and
// WI 2.3 (screen assembly).
package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
)

// bubbleteaVersion is the pinned bubbletea version pinned in go.mod. Kept as a constant so
// the dependency is referenced by package code (not just go.mod), keeping
// `go mod tidy` from dropping it until WI 2.2 replaces this stub.
const bubbleteaVersion = "v1.3.6"

// stubModel is the minimal bubbletea model proving the library links into
// the build. Replaced by the real dashboard in WI 2.2.
type stubModel struct{}

// Init satisfies tea.Model (no commands yet).
func (stubModel) Init() tea.Cmd { return nil }

// Update handles no messages yet.
func (stubModel) Update(tea.Msg) (tea.Model, tea.Cmd) { return stubModel{}, nil }

// View renders a placeholder line.
func (stubModel) View() string {
	return fmt.Sprintf("sssonector tui (stub, bubbletea %s)", bubbleteaVersion)
}

// Ensure the library types are linked (compile-time reference).
var _ tea.Model = stubModel{}
