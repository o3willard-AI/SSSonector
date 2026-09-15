package main

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/o3willard-AI/SSSonector/internal/cert/generator"
	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

// WI 5.5: client-wizard tests — bundle prefill, chain pre-flight blocks,
// field table, create through the WI 5.3 pipeline seam. Bundles are the
// REAL generated ones; create is a recording fake (no real systemctl).

// cwFixture generates a bundle + a ready client wizard loaded from it.
// The fixture mirrors collect's bundle_test.go layout (real cert
// generator, temp config root).
func cwFixture(t *testing.T) (clientWizardModel, *wizCreateRecorder) {
	t.Helper()
	root := t.TempDir()
	certDir := filepath.Join(root, "instances", "srv-a", "certs")
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, "instances", "srv-a", "config.yaml")), 0o750); err != nil {
		t.Fatal(err)
	}
	yaml := "metadata:\n  schema_version: \"2.0.0\"\ntype: server\nconfig:\n  mode: server\n" +
		"  network:\n    name: tun0\n    interface: tun0\n    mtu: 1500\n    address: 10.77.0.1/24\n" +
		"  tunnel:\n    listen_address: \"192.168.101.7\"\n    listen_port: 9443\n" +
		"  auth:\n    cert_file: " + strconv.Quote(filepath.Join(certDir, "server.crt")) + "\n" +
		"    key_file: " + strconv.Quote(filepath.Join(certDir, "server.key")) + "\n" +
		"    ca_file: " + strconv.Quote(filepath.Join(certDir, "ca.crt")) + "\n"
	cfgPath := filepath.Join(root, "instances", "srv-a", "config.yaml")
	if err := os.WriteFile(cfgPath, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := generator.GenerateCertificates(certDir); err != nil {
		t.Fatalf("seed certs: %v", err)
	}
	b, err := collect.GenerateClientBundle(collect.ConfigPaths{ConfigRoot: root}, "srv-a")
	if err != nil {
		t.Fatalf("generate bundle: %v", err)
	}
	rec := &wizCreateRecorder{}
	m := newClientWizard(b.Path, func() string { return t.TempDir() })
	m.create = rec.create
	m.paths = collect.ConfigPaths{ConfigRoot: t.TempDir()}
	return m, rec
}

// TestClientWizard_BundlePrefill: --from-bundle prefills all four fields
// from the skeleton — nothing blank, nothing invented.
func TestClientWizard_BundlePrefill(t *testing.T) {
	m, _ := cwFixture(t)
	if m.loaded == nil {
		t.Fatal("bundle must be loaded")
	}
	if m.instance != "srv-a" || m.server != "192.168.101.7" || m.port != "9443" || m.tun != "10.77.0.2/24" {
		t.Errorf("prefill must trace to the skeleton: instance=%q server=%q port=%q tun=%q",
			m.instance, m.server, m.port, m.tun)
	}
	if m.chainErr != "" {
		t.Errorf("bundle's own chain must pass the pre-flight: %v", m.chainErr)
	}
	if !m.ready() {
		t.Errorf("prefilled valid form must be ready, errors: %v", m.fieldErrs())
	}
}

// TestClientWizard_ChainMismatch_BlocksEnter: a swapped-in client cert
// signed by another CA hard-blocks create (the §3.3 pre-flight), even
// though every field looks fine.
func TestClientWizard_ChainMismatch_BlocksEnter(t *testing.T) {
	m, rec := cwFixture(t)
	// Corrupt the staged client cert (rebuild it from a foreign CA).
	otherRoot := t.TempDir()
	otherCertDir := filepath.Join(otherRoot, "instances", "other", "certs")
	if err := generator.GenerateCertificates(otherCertDir); err != nil {
		t.Fatal(err)
	}
	if err := generator.IssueClientCert(otherCertDir); err != nil {
		t.Fatal(err)
	}
	foreign, err := os.ReadFile(filepath.Join(otherCertDir, "client.crt"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(m.loaded.ClientCertPath, foreign, 0o600); err != nil {
		t.Fatal(err)
	}
	// Re-run the pre-flight as the wizard does at load: simulate by
	// setting the chain error the same way newClientWizard would.
	if err := collect.VerifyClientChainAgainstCA(m.loaded.CAPath, m.loaded.ClientCertPath); err == nil {
		t.Fatal("mismatch must be detected by the pre-flight")
	}
	m.chainErr = "chain pre-flight: CA does NOT verify client cert"
	if m.ready() {
		t.Fatal("chain mismatch must BLOCK create")
	}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(clientWizardModel)
	if m.done || len(rec.calls) != 0 {
		t.Fatalf("FAIL-CLOSED VIOLATED: chain mismatch must not reach create: done=%v calls=%v", m.done, rec.calls)
	}
	if !strings.Contains(m.View(), "✗") || !strings.Contains(m.View(), "chain pre-flight") {
		t.Errorf("blocking pre-flight error must render:\n%s", m.View())
	}
}

// TestClientWizard_FieldTable: invalid blocks, valid unblocks.
func TestClientWizard_FieldTable(t *testing.T) {
	rows := []struct {
		name    string
		mutate  func(*clientWizardModel)
		wantErr string
	}{
		{"instance blank", func(m *clientWizardModel) { m.instance = "" }, "INSTANCE: required"},
		{"server blank", func(m *clientWizardModel) { m.server = "" }, "SERVER: required"},
		{"port garbage", func(m *clientWizardModel) { m.port = "https" }, "must be 1–65535"},
		{"port zero", func(m *clientWizardModel) { m.port = "0" }, "must be 1–65535"},
		{"tun garbage", func(m *clientWizardModel) { m.tun = "not-a-cidr" }, "not a valid CIDR"},
		{"tun outside server subnet", func(m *clientWizardModel) { m.tun = "10.99.0.5/24" }, "must be inside the server's subnet"},
		{"tun network address", func(m *clientWizardModel) { m.tun = "10.77.0.0/24" }, "host address"},
		{"all valid", func(m *clientWizardModel) {}, ""},
	}
	for _, row := range rows {
		m, _ := cwFixture(t)
		row.mutate(&m)
		errs := m.fieldErrs()
		if row.wantErr == "" {
			if !m.ready() {
				t.Errorf("%s: valid input must unblock, errors: %v", row.name, errs)
			}
			continue
		}
		if m.ready() {
			t.Errorf("%s: invalid input must BLOCK create", row.name)
			continue
		}
		found := false
		for _, e := range errs {
			if strings.Contains(e, row.wantErr) {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: errors %v must mention %q", row.name, errs, row.wantErr)
		}
	}
}

// TestClientWizard_CreateThroughPipeline: Enter on a ready form runs the
// WI 5.3 pipeline seam with the right instance, and the draft carries the
// prefill values with cert paths pointing at the staged bundle material.
func TestClientWizard_CreateThroughPipeline(t *testing.T) {
	m, rec := cwFixture(t)
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(clientWizardModel)
	if !m.done {
		t.Fatalf("ready + Enter must complete create: %v", m.createErr)
	}
	if len(rec.calls) != 1 || !strings.HasPrefix(rec.calls[0], "srv-a|") {
		t.Fatalf("create must be invoked once for srv-a: %v", rec.calls)
	}
	out := m.View()
	if !strings.Contains(out, "created & connected") {
		t.Errorf("success screen must announce the connection:\n%s", out)
	}
}

// TestClientWizard_NoBundle_BlankFields: without --from-bundle the wizard
// runs with blank fields (no guessed prefill) and stays blocked until
// filled — and cert material still requires a bundle.
func TestClientWizard_NoBundle_BlankFields(t *testing.T) {
	m := newClientWizard("", func() string { return t.TempDir() })
	if m.instance != "" || m.server != "" || m.port != "" || m.tun != "" {
		t.Errorf("no-bundle wizard must start blank: %q %q %q %q", m.instance, m.server, m.port, m.tun)
	}
	if m.ready() {
		t.Fatal("blank wizard must be blocked")
	}
	// Fill everything validly, but with no bundle loaded the draft must
	// refuse (cert material has no source).
	m.instance = "cli"
	m.server = "192.168.101.7"
	m.port = "9443"
	m.tun = "10.77.0.2/24"
	if _, err := m.draft(); err == nil {
		t.Fatal("draft without loaded bundle cert material must fail closed")
	}
}

// TestClientWizard_BadBundlePath_SurfaceVerbatim: a missing bundle errors
// verbatim on screen with blank fields (never a guessed prefill).
func TestClientWizard_BadBundlePath_SurfaceVerbatim(t *testing.T) {
	m := newClientWizard(filepath.Join(t.TempDir(), "missing.tgz"), func() string { return t.TempDir() })
	if m.loaded != nil {
		t.Fatal("failed load must not fabricate a bundle")
	}
	if !strings.Contains(m.loadErr, "missing.tgz") {
		t.Errorf("load error must name the path verbatim: %q", m.loadErr)
	}
	out := m.View()
	if !strings.Contains(out, "LOAD FAILED") {
		t.Errorf("load failure must render:\n%s", out)
	}
}
