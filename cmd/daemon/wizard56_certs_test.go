package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

// WI 5.6: CERTS=generate-new must mint the instance CA + server leaf into
// the instance certs dir BEFORE create (the daemon refuses to start without
// TLS). Fails if the mint is dropped (the WI 5.6 rig-flown gap).

// certGenRecorder records the cert dir + server IPs passed to the seam.
type certGenRecorder struct {
	dirs []string
	ips  [][]string
}

func (r *certGenRecorder) gen(certDir string, serverIPs ...string) error {
	r.dirs = append(r.dirs, certDir)
	r.ips = append(r.ips, serverIPs)
	return nil
}

// TestWizard56_GenerateNewMintsCerts: Enter with CERTS=generate-new invokes
// the cert-gen seam on instances/<n>/certs BEFORE the create seam, and the
// draft names cert+key+ca.
func TestWizard56_GenerateNewMintsCerts(t *testing.T) {
	cg := &certGenRecorder{}
	rec := &wizCreateRecorder{}
	root := t.TempDir()
	m := serverWizardModel{form: newWizardForm(okValidate, freePortProbe, nil)}
	m.form.create = rec.create
	m.form.genCerts = cg.gen
	m.form.paths = collect.ConfigPaths{ConfigRoot: root}

	// Drive the full form (generate-new is the second space toggle on CERTS).
	m = wizKey(m, "enter")          // MODE
	m = wizKey(m, "tab")            // INSTANCE
	m = wizType(m, "client-a")      //
	m = wizKey(m, "tab")            // PORT
	m = wizType(m, "9443")          //
	m = wizKey(m, "tab")            // TUN
	m = wizType(m, "10.77.0.1/24")  //
	m = wizKey(m, "tab")            // NAT
	m = wizKey(m, " ")              // enable
	m = wizKey(m, "tab")            // CERTS
	m = wizKey(m, " ")              // -> reuse
	if m.form.cert != certReuseHost {
		t.Fatalf("first toggle must land on reuse-host, got %v", m.form.cert)
	}
	m = wizKey(m, " ") // -> generate-new
	if m.form.cert != certGenerateNew {
		t.Fatalf("second toggle must land on generate-new, got %v", m.form.cert)
	}
	if !m.form.ready() {
		t.Fatalf("form must be ready: %v", m.form.fieldErrs())
	}

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(serverWizardModel)

	if !m.form.done {
		t.Fatalf("create must complete: createErr=%q", m.form.createErr)
	}
	// Cert generator ran on the instance certs dir before create.
	if len(cg.dirs) != 1 || !strings.Contains(cg.dirs[0], "instances/client-a/certs") {
		t.Fatalf("cert-gen must run on instances/client-a/certs, got %v", cg.dirs)
	}
	if len(rec.calls) != 1 {
		t.Fatalf("create must run once after cert-gen: %v", rec.calls)
	}
	// The cert dir exists on disk with the record written.
	if _, err := os.Stat(filepath.Join(root, "instances", "client-a", "certs")); err != nil {
		t.Errorf("certs dir must exist after generate-new: %v", err)
	}
}

// TestWizard56_ReuseHostSkipsMint: CERTS=reuse-host does NOT call the
// cert-gen seam (the operator pre-provisions the shared store).
func TestWizard56_ReuseHostSkipsMint(t *testing.T) {
	cg := &certGenRecorder{}
	rec := &wizCreateRecorder{}
	m := serverWizardModel{form: newWizardForm(okValidate, freePortProbe, nil)}
	m.form.create = rec.create
	m.form.genCerts = cg.gen
	m.form.paths = collect.ConfigPaths{ConfigRoot: t.TempDir()}

	m = wizKey(m, "enter")         // MODE
	m = wizKey(m, "tab")           // INSTANCE
	m = wizType(m, "client-a")     //
	m = wizKey(m, "tab")           // PORT
	m = wizType(m, "9443")         //
	m = wizKey(m, "tab")           // TUN
	m = wizType(m, "10.77.0.1/24") //
	m = wizKey(m, "tab")           // NAT
	m = wizKey(m, " ")             // enable
	m = wizKey(m, "tab")           // CERTS
	m = wizKey(m, " ")             // reuse-host (first toggle)

	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(serverWizardModel)
	if !m.form.done {
		t.Fatalf("reuse-host create must complete: %q", m.form.createErr)
	}
	if len(cg.dirs) != 0 {
		t.Errorf("reuse-host must NOT mint certs: %v", cg.dirs)
	}
	// The draft must point at the shared host store (absolute under the
	// configured root; default /etc/sssonector).
	d, _ := m.form.draft()
	if !strings.Contains(d, m.form.paths.ConfigRoot+"/certs/server.crt") {
		t.Errorf("reuse-host draft must name the shared store:\n%s", d)
	}
}

// TestWizard56_GenerateNewDraftNamesFullPaths: the generate-new draft names
// cert+key+ca (the daemon's TLS manager needs all three).
func TestWizard56_GenerateNewDraftNamesFullPaths(t *testing.T) {
	root := t.TempDir()
	m := serverWizardModel{form: newWizardForm(okValidate, freePortProbe, nil)}
	m.form.cert = certGenerateNew
	m.form.instance = "cli"
	m.form.port = "9443"
	m.form.tun = "10.77.0.1/24"
	m.form.nat = natDisabled
	m.form.modeChosen = true
	m.form.paths = collect.ConfigPaths{ConfigRoot: root}
	d, err := m.form.draft()
	if err != nil {
		t.Fatalf("draft: %v", err)
	}
	for _, want := range []string{
		"cert_file: " + root + "/instances/cli/certs/server.crt",
		"key_file: " + root + "/instances/cli/certs/server.key",
		"ca_file: " + root + "/instances/cli/certs/ca.crt",
	} {
		if !strings.Contains(d, want) {
			t.Errorf("generate-new draft must name %q (absolute, never relative):\n%s", want, d)
		}
	}
}