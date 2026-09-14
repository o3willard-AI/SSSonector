package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)
// WI 5.3: wizard-level write-path tests — the form's create action goes
// through the CreateFunc seam (recording fake; no real systemctl), errors
// surface verbatim, and success flips to the done state that routes to
// the dashboard.

// wizCreateRecorder records create calls and their file-existence state.
type wizCreateRecorder struct {
	calls        []string
	fileExisted  bool
	failWith     string // non-empty: return this error
	createdPaths []string
}

func (r *wizCreateRecorder) create(paths collect.ConfigPaths, instance, draft string, runner collect.CommandRunner) (collect.CreateAndStartResult, error) {
	target := filepath.Join(paths.ConfigRoot, "instances", instance, "config.yaml")
	r.calls = append(r.calls, instance+"|"+target)
	if r.failWith != "" {
		return collect.CreateAndStartResult{}, errStr(r.failWith)
	}
	// Simulate the real write path (so "lands on dashboard" has a file).
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return collect.CreateAndStartResult{}, err
	}
	if err := os.WriteFile(target, []byte(draft), 0o600); err != nil {
		return collect.CreateAndStartResult{}, err
	}
	r.createdPaths = append(r.createdPaths, target)
	return collect.CreateAndStartResult{File: target, Unit: "sssonector@" + instance + ".service", Started: true}, nil
}

// TestWizard53_CreateThroughSeam: pressing Enter on a ready form invokes
// the injected CreateFunc with the right instance + draft; the form flips
// to done with the created instance carried for the dashboard focus.
func TestWizard53_CreateThroughSeam(t *testing.T) {
	rec := &wizCreateRecorder{}
	m := serverWizardModel{form: newWizardForm(okValidate, freePortProbe, nil)}
	m.form.create = rec.create
	m.form.paths = collect.ConfigPaths{ConfigRoot: t.TempDir()}

	m = wizKey(m, "enter")       // MODE
	m = wizKey(m, "tab")         // INSTANCE
	m = wizType(m, "client-a")   //
	m = wizKey(m, "tab")         // PORT
	m = wizType(m, "9443")       //
	m = wizKey(m, "tab")         // TUN
	m = wizType(m, "10.77.0.1/24") //
	m = wizKey(m, "tab")         // NAT
	m = wizKey(m, " ")           // enable
	m = wizKey(m, "tab")         // CERTS
	m = wizKey(m, " ")           // reuse → toggle to generate
	if m.form.cert == certReuseHost {
		m = wizKey(m, " ")
	}
	if !m.form.ready() {
		t.Fatalf("form must be ready: %v", m.form.fieldErrs())
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(serverWizardModel)

	if !m.form.done {
		t.Fatal("ready + Enter must complete create")
	}
	if len(rec.calls) != 1 || !strings.HasPrefix(rec.calls[0], "client-a|") {
		t.Errorf("CreateFunc must be invoked once for client-a: %v", rec.calls)
	}
	if m.form.createdIns != "client-a" {
		t.Errorf("created instance must be carried for the dashboard: %q", m.form.createdIns)
	}
	out := m.View()
	if !strings.Contains(out, "created & started sssonector@client-a.service") {
		t.Errorf("success screen must name the created unit:\n%s", out)
	}
}

// TestWizard53_CreateErrorSurfaces: a CreateFunc failure surfaces its
// verbatim error on screen; done never flips (no dashboard landing).
func TestWizard53_CreateErrorSurfaces(t *testing.T) {
	rec := &wizCreateRecorder{failWith: "create: enable sssonector@client-a.service: boom (config file left written at /etc/...)"}
	m := serverWizardModel{form: filledForm()}
	m.form.create = rec.create
	m.form.focus = wfInstance // Enter off the MODE radio → create path
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(serverWizardModel)

	if m.form.done {
		t.Fatal("failed create must not flip done (no dashboard landing)")
	}
	if m.form.createErr == "" || !strings.Contains(m.form.createErr, "boom") {
		t.Errorf("verbatim create error must surface: %q", m.form.createErr)
	}
	out := m.View()
	if !strings.Contains(out, "CREATE FAILED: create: enable sssonector@client-a.service: boom") {
		t.Errorf("failure banner must render verbatim:\n%s", out)
	}
}

// TestWizard53_WithFocus_PreFocusesDashboard is in internal/tui
// (withfocus_test.go) — WithFocus is a dashboardModel method. This file
// covers the wizard-side write path.

// TestWizard53_CreateThroughSeam: pressing Enter on a ready form invokes
