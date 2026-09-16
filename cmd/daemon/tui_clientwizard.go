package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

// Client wizard (WI 5.5): --from-bundle prefills from the skeleton, the
// CA-verifies-chain pre-flight blocks Enter on mismatch, and create runs
// through the SAME WI 5.3 write pipeline (validate → atomic write →
// enable → start).

// clientWizardField identifies the focused client-wizard field.
type clientWizardField int

const (
	cwInstance clientWizardField = iota
	cwServer
	cwPort
	cwTun
	cwCount // sentinel
)

// clientWizardModel is the client-setup form (docs/tui.md §3.3).
type clientWizardModel struct {
	focus  clientWizardField
	buf    string
	bufFld clientWizardField
	// loaded bundle (nil when no --from-bundle: fields blank)
	loaded   *collect.LoadedBundle
	loadErr  string
	instance string
	server   string
	port     string
	tun      string
	// seams
	validate collect.ValidateDraftFunc
	create   collect.CreateFunc
	stage    func(string, string) error // cert-staging seam (collect.StageClientCerts)
	paths    collect.ConfigPaths
	runner   collect.CommandRunner
	// chain pre-flight result ("" = ok/verified)
	chainErr string
	// outcome
	done      bool
	createErr string
	aborted   bool
}

// newClientWizard builds the form, loading the bundle when a path is set.
func newClientWizard(bundlePath string, tempDir func() string) clientWizardModel {
	m := clientWizardModel{
		validate: collect.ValidateDraft,
		create:   collect.CreateAndStartInstance,
		stage:    collect.StageClientCerts,
		paths:    collect.DefaultConfigPaths(),
		runner:   sudoRunner{inner: collect.OSCommandRunner{}}, // enable/start need root (sudo -A when askpass)
	}
	if bundlePath != "" {
		lb, err := collect.LoadBundle(bundlePath, tempDir)
		if err != nil {
			m.loadErr = err.Error() // verbatim; fields stay blank (no guessed prefill)
		} else {
			m.loaded = &lb
			m.instance = lb.Prefill.Instance
			m.server = lb.Prefill.ServerHost
			m.port = strconv.Itoa(lb.Prefill.ServerPort)
			m.tun = lb.Prefill.TunAddress
			m.bufFld = cwInstance
			m.buf = m.instance
			// Pre-flight: the CA must verify the client cert chain — a
			// mismatch BLOCKS the wizard (docs/tui.md §3.3).
			if err := collect.VerifyClientChainAgainstCA(lb.CAPath, lb.ClientCertPath); err != nil {
				m.chainErr = err.Error()
			}
		}
	}
	return m
}

func (m clientWizardModel) Init() tea.Cmd { return nil }

func (m clientWizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}
	switch k.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.aborted = true
		return m, tea.Quit
	case "tab":
		m.focus = (m.focus + 1) % cwCount
		return m, nil
	case "up":
		if m.focus > 0 {
			m.focus--
		}
		return m, nil
	case "down":
		if m.focus < cwCount-1 {
			m.focus++
		}
		return m, nil
	case "enter":
		m.handleEnter()
		return m, nil
	}
	if m.focus >= cwInstance && m.focus <= cwTun {
		if m.bufFld != m.focus {
			m.bufFld = m.focus
			m.buf = m.fieldValue(m.focus)
		}
		switch k.String() {
		case "backspace":
			if r := []rune(m.buf); len(r) > 0 {
				m.buf = string(r[:len(r)-1])
			}
		default:
			if len(k.Runes) > 0 {
				m.buf += string(k.Runes)
			}
		}
		m.applyBuf()
	}
	return m, nil
}

func (m clientWizardModel) fieldValue(f clientWizardField) string {
	switch f {
	case cwInstance:
		return m.instance
	case cwServer:
		return m.server
	case cwPort:
		return m.port
	case cwTun:
		return m.tun
	}
	return ""
}

func (m *clientWizardModel) applyBuf() {
	switch m.focus {
	case cwInstance:
		m.instance = m.buf
	case cwServer:
		m.server = m.buf
	case cwPort:
		m.port = m.buf
	case cwTun:
		m.tun = m.buf
	}
}

// fieldErrs runs the client form's validation table (fail-closed: blank
// fields are errors, not defaults). The chain pre-flight error is a
// hard block.
func (m clientWizardModel) fieldErrs() []string {
	var errs []string
	if m.chainErr != "" {
		errs = append(errs, m.chainErr) // verbatim pre-flight mismatch (blocking)
	}
	if m.loadErr != "" {
		errs = append(errs, m.loadErr)
	}
	if strings.TrimSpace(m.instance) == "" {
		errs = append(errs, "INSTANCE: required")
	}
	if strings.TrimSpace(m.server) == "" {
		errs = append(errs, "SERVER: required")
	}
	if p := strings.TrimSpace(m.port); p == "" {
		errs = append(errs, "SERVER PORT: required")
	} else if n, err := strconv.Atoi(p); err != nil || n < 1 || n > 65535 {
		errs = append(errs, "SERVER PORT: must be 1–65535")
	}
	// TUN: valid CIDR, host address (not network), and same subnet as the
	// server (the client must sit inside the server's TUN subnet).
	if err := collect.ValidateCIDR(strings.TrimSpace(m.tun)); err != nil {
		errs = append(errs, "TUN ADDRESS: "+err.Error())
	} else if m.loaded != nil {
		serverSub := subnetOf(m.loaded.Prefill.TunAddress)
		if serverSub != "" && subnetOf(strings.TrimSpace(m.tun)) != serverSub {
			errs = append(errs, "TUN ADDRESS: must be inside the server's subnet "+serverSub)
		}
	}
	return errs
}

// subnetOf returns the network address of a CIDR ("" when unparseable).
func subnetOf(cidr string) string {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return ""
	}
	return ipnet.String()
}

// ready reports whether create is enabled.
func (m clientWizardModel) ready() bool { return len(m.fieldErrs()) == 0 }

// draft assembles the CLIENT config from the form and validates it
// through the REAL loader chain. Cert paths point at the STAGED bundle
// material (relocated by WI 5.3's write into the instance dir).
func (m clientWizardModel) draft() (string, error) {
	if m.loaded == nil {
		return "", fmt.Errorf("client wizard: no bundle loaded — cert material must come from --from-bundle")
	}
	port := strings.TrimSpace(m.port)
	tun := strings.TrimSpace(m.tun)
	instance := strings.TrimSpace(m.instance)
	// Cert paths point at the INSTANCE's persistent certs dir (staged
	// from the bundle before create), never the volatile extraction dir.
	if m.paths.ConfigRoot == "" {
		m.paths = collect.DefaultConfigPaths()
	}
	certDir := filepath.Join(m.paths.ConfigRoot, "instances", instance, "certs")
	var b strings.Builder
	b.WriteString("metadata:\n")
	b.WriteString("  schema_version: \"2.0.0\"\n")
	b.WriteString("  environment: development\n")
	b.WriteString("type: client\n")
	b.WriteString("config:\n")
	b.WriteString("  mode: client\n")
	b.WriteString("  logging:\n")
	b.WriteString("    level: info\n")
	b.WriteString("  network:\n")
	b.WriteString("    name: tun0\n")
	b.WriteString("    interface: tun0\n")
	b.WriteString("    mtu: 1500\n")
	b.WriteString("    address: " + tun + "\n")
	b.WriteString("  tunnel:\n")
	b.WriteString("    server_address: " + strings.TrimSpace(m.server) + "\n")
	b.WriteString("    server_port: " + port + "\n")
	b.WriteString("  monitor:\n")
	b.WriteString("    enabled: true\n")
	b.WriteString("    type: prometheus\n")
	b.WriteString("    interval: 10s\n")
	b.WriteString("    prometheus:\n")
	b.WriteString("      enabled: true\n")
	b.WriteString("      port: " + promPortFor(port) + "\n")
	b.WriteString("      path: /metrics\n")
	b.WriteString("  security:\n")
	b.WriteString("    tls:\n")
	b.WriteString("      min_version: \"1.2\"\n")
	b.WriteString("      max_version: \"1.3\"\n")
	b.WriteString("  auth:\n")
	b.WriteString("    ca_file: " + filepath.Join(certDir, "ca.crt") + "\n")
	b.WriteString("    cert_file: " + filepath.Join(certDir, "client.crt") + "\n")
	b.WriteString("    key_file: " + filepath.Join(certDir, "client.key") + "\n")
	if m.validate == nil {
		return "", fmt.Errorf("client wizard: no validator wired")
	}
	if _, err := m.validate(b.String()); err != nil {
		return "", err // verbatim loader/validator error
	}
	return b.String(), nil
}

// handleEnter runs create when ready (fail-closed otherwise).
func (m *clientWizardModel) handleEnter() {
	if !m.ready() || m.done {
		return
	}
	draft, err := m.draft()
	if err != nil {
		m.createErr = err.Error()
		return
	}
	// Stage the bundle certs into the instance's PERSISTENT certs dir
	// (never the volatile extraction dir) before enable/start. create
	// mkdirs the instance dir; we mkdir -p the certs/ subdir here.
	instance := strings.TrimSpace(m.instance)
	certDir := filepath.Join(m.paths.ConfigRoot, "instances", instance, "certs")
	if err := os.MkdirAll(certDir, 0o750); err != nil {
		m.createErr = fmt.Errorf("client cert stage: mkdir %s: %w", certDir, err).Error()
		return
	}
	if err := m.stage(m.loaded.StagingDir, certDir); err != nil {
		m.createErr = err.Error()
		return
	}
	res, cerr := m.create(m.paths, instance, draft, m.runner)
	if cerr != nil {
		m.createErr = cerr.Error()
		return
	}
	m.done = true
	_ = res
}

// View renders the §3.3 client form with live validation.
func (m clientWizardModel) View() string {
	var b strings.Builder
	b.WriteString("┌─ SSSonector — client setup ─┐\n")
	if m.loaded != nil {
		b.WriteString("│ CLIENT CERT ▸ bundle loaded (ca.crt, client.crt, client.key)\n")
	} else if m.loadErr != "" {
		b.WriteString("│ CLIENT CERT ▸ LOAD FAILED: " + m.loadErr + "\n")
	} else {
		b.WriteString("│ CLIENT CERT ▸ (no bundle — fields blank; cert material required)\n")
	}
	b.WriteString("│ INSTANCE    " + m.instance + "\n")
	b.WriteString("│ SERVER      " + m.server + "\n")
	b.WriteString("│ SERVER PORT " + m.port + "\n")
	b.WriteString("│ TUN ADDRESS " + m.tun + "\n")
	b.WriteString("│\n")
	switch {
	case m.done:
		b.WriteString("│ ✓ created & connected — opening the dashboard…\n")
	case m.createErr != "":
		b.WriteString("│ ✗ CREATE FAILED: " + m.createErr + "\n")
	case len(m.fieldErrs()) > 0:
		b.WriteString("│ validation: (create DISABLED)\n")
		for _, e := range m.fieldErrs() {
			b.WriteString("│   ✗ " + e + "\n")
		}
	default:
		b.WriteString("│ validation: ✓ schema v2.0.0 · CA verifies client cert chain · TUN subnet ok — [Enter] create & connect\n")
	}
	b.WriteString("└─ Tab next field · type to edit · Enter create · Esc abort ─┘\n")
	return b.String()
}
