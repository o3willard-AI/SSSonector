package main

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/o3willard-AI/SSSonector/internal/tui/collect"
)

// Server-wizard form (WI 5.2, docs/tui.md §3.3). MODE is ASKED, never
// defaulted: no radio is pre-selected, and create stays disabled until
// every field is explicitly chosen and validates (fail-closed). The
// create action itself is WI 5.3 — for 5.2 it prints the validated draft.

// wizardField identifies the focused form field.
type wizardField int

const (
	wfMode wizardField = iota
	wfInstance
	wfPort
	wfTun
	wfNat
	wfCerts
	wfCount // sentinel
)

// certChoice is the CERTS radio state (neither = fail-closed).
type certChoice int

const (
	certNone certChoice = iota
	certReuseHost
	certGenerateNew
)

// natChoice is the FORWARD NAT checkbox state (default-deny egress CIDR
// is required when enabled; the ACL is edited post-setup, never here).
type natChoice int

const (
	natUnset natChoice = iota
	natDisabled
	natEnabled
)

// wizardForm is the server-wizard state.
type wizardForm struct {
	focus wizardField
	// Mode radio: neither pre-selected (mode is ASKED, never guessed).
	modeChosen bool
	modeClient bool // false = server (only meaningful when modeChosen)
	// Text fields (blank by default — no invented defaults).
	instance string
	port     string
	tun      string
	// NAT checkbox: unset until touched (part of "explicitly chosen").
	nat natChoice
	// CERTS radio: certNone until chosen (fail-closed).
	cert certChoice
	// injectable seams
	validate   collect.ValidateDraftFunc // real loader+validator
	portFree   collect.PortProbeFunc     // ss -tlnp via injectable runner
	create     collect.CreateFunc        // write+enable+start (WI 5.3)
	genCerts   collect.CertGenFunc       // mint instance CA+leaf (CERTS=generate-new)
	paths      collect.ConfigPaths       // config root for the write
	runner     collect.CommandRunner     // systemctl enable/start runner
	existing   []string                  // existing TUN subnets (from discovery)
	editing    bool                      // text field has keyboard focus
	buf        string                    // edit buffer for the focused text field
	bufField   wizardField               // which field the buffer belongs to
	done       bool                      // create pressed (lands on dashboard)
	createErr  string                    // verbatim create/write error
	aborted    bool                      // Esc pressed
	createdIns string                    // instance to focus on the dashboard
}

// newWizardForm builds the form with the real loader + injectable port
// probe + the real create path. existing carries the TUN subnets of
// already-configured instances (empty on a fresh host).
func newWizardForm(validate collect.ValidateDraftFunc, portFree collect.PortProbeFunc, existing []string) wizardForm {
	return wizardForm{
		validate: validate,
		portFree: portFree,
		create:   collect.CreateAndStartInstance,
		genCerts: collect.GenerateInstanceCerts,
		paths:    collect.DefaultConfigPaths(),
		runner:   collect.OSCommandRunner{},
		existing: existing,
		buf:      "",
	}
}

// fieldErrs runs the per-keystroke validation table. Every check fails
// closed: a field with no value yet is an error, not a default.
func (f wizardForm) fieldErrs() []string {
	var errs []string
	if !f.modeChosen {
		errs = append(errs, "MODE: choose Server or Client (never defaulted)")
	}
	if strings.TrimSpace(f.instance) == "" {
		errs = append(errs, "INSTANCE: required")
	}
	if p := strings.TrimSpace(f.port); p == "" {
		errs = append(errs, "LISTEN PORT: required")
	} else if n, err := strconv.Atoi(p); err != nil || n < 1 || n > 65535 {
		errs = append(errs, "LISTEN PORT: must be 1–65535")
	} else if f.portFree != nil && !f.portFree(n) {
		errs = append(errs, fmt.Sprintf("LISTEN PORT: %d is already in use (ss -tlnp)", n))
	}
	if err := f.checkTun(); err != nil {
		errs = append(errs, "TUN ADDRESS: "+err.Error())
	}
	if f.nat == natUnset {
		errs = append(errs, "FORWARD NAT: choose enabled or disabled")
	}
	if f.cert == certNone {
		errs = append(errs, "CERTS: choose reuse-host or generate-new (fail-closed)")
	}
	return errs
}

// checkTun validates the TUN CIDR text and its overlap against existing
// instances' subnets (the REAL loader re-checks the schema in draft()).
func (f wizardForm) checkTun() error {
	c := strings.TrimSpace(f.tun)
	if c == "" {
		return fmt.Errorf("required")
	}
	if err := collect.ValidateCIDR(c); err != nil {
		return err
	}
	if o := collect.OverlapWith(c, f.existing); o != "" {
		return fmt.Errorf("overlaps existing instance subnet %s", o)
	}
	return nil
}

// ready reports whether create is enabled: zero field errors.
func (f wizardForm) ready() bool { return len(f.fieldErrs()) == 0 }

// draft assembles the config YAML from the form and validates it through
// the REAL loader+validator chain (schema v2.0.0). Returns the validated
// text and error. Mode here is always server (the client stub is WI 5.5;
// choosing Client shows that stub and disables create).
func (f wizardForm) draft() (string, error) {
	if !f.modeChosen || f.modeClient {
		return "", fmt.Errorf("client wizard arrives in WI 5.5")
	}
	port := strings.TrimSpace(f.port)
	tun := strings.TrimSpace(f.tun)
	var b strings.Builder
	b.WriteString("metadata:\n")
	b.WriteString("  schema_version: \"2.0.0\"\n")
	b.WriteString("  environment: development\n")
	b.WriteString("type: server\n")
	b.WriteString("config:\n")
	b.WriteString("  mode: server\n")
	b.WriteString("  logging:\n")
	b.WriteString("    level: info\n")
	b.WriteString("  network:\n")
	b.WriteString("    name: tun0\n")
	b.WriteString("    interface: tun0\n")
	b.WriteString("    mtu: 1500\n")
	b.WriteString("    address: " + tun + "\n")
	b.WriteString("  tunnel:\n")
	b.WriteString("    listen_address: 0.0.0.0\n")
	b.WriteString("    listen_port: " + port + "\n")
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
	// CERTS: fail-closed — the draft only names TLS paths when a cert
	// choice was made, and the paths are ABSOLUTE (the daemon resolves
	// relative cert paths against both its config dir AND working dir —
	// a relative `certs/...` doubles to instances/<n>/instances/<n>/certs
	// on the rig, so relative paths are never written). generate-new uses
	// the instance's own certs/ dir (minted by the create path);
	// reuse-host points at the shared host store.
	if f.paths.ConfigRoot == "" {
		f.paths = collect.DefaultConfigPaths()
	}
	certBase := f.paths.ConfigRoot
	b.WriteString("  auth:\n")
	switch f.cert {
	case certGenerateNew:
		b.WriteString("    cert_file: " + certBase + "/instances/" + strings.TrimSpace(f.instance) + "/certs/server.crt\n")
		b.WriteString("    key_file: " + certBase + "/instances/" + strings.TrimSpace(f.instance) + "/certs/server.key\n")
		b.WriteString("    ca_file: " + certBase + "/instances/" + strings.TrimSpace(f.instance) + "/certs/ca.crt\n")
	case certReuseHost:
		b.WriteString("    cert_file: " + certBase + "/certs/server.crt\n")
		b.WriteString("    key_file: " + certBase + "/certs/server.key\n")
		b.WriteString("    ca_file: " + certBase + "/certs/ca.crt\n")
	}
	// NAT: enabling writes the DEFAULT-DENY forward-NAT config with an
	// explicit egress CIDR; the ACL is edited post-setup (never loosened
	// here). Disabling writes no NAT block (fail-closed absence).
	if f.nat == natEnabled {
		b.WriteString("  nat:\n")
		b.WriteString("    enabled: true\n")
		b.WriteString("    mode: full\n")
		b.WriteString("    default_deny: true\n")
		b.WriteString("    egress_cidr: 192.168.100.0/24\n")
	}
	draft := b.String()
	if f.validate == nil {
		return "", fmt.Errorf("wizard: no validator wired")
	}
	if _, err := f.validate(draft); err != nil {
		return "", err // verbatim loader/validator error
	}
	return draft, nil
}

// promPortFor picks a metrics port derived from the listen port (never a
// collision with it): listen+1000, wrapped into the dynamic range.
func promPortFor(listen string) string {
	n, err := strconv.Atoi(strings.TrimSpace(listen))
	if err != nil {
		return "9090"
	}
	p := n + 1000
	if p > 65535 {
		p = n - 1000
	}
	return strconv.Itoa(p)
}
