package main

import (
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/cert/generator"
	"github.com/o3willard-AI/SSSonector/internal/provision"
)

// runProvision dispatches `sssonector provision <verb>` subcommands.
// Provisioning is offline-capable: no TUN, no service manager, root only
// required by apply when writing to system directories.
func runProvision(args []string) error {
	if len(args) == 0 {
		fmt.Fprintln(os.Stderr, "usage: sssonector provision create|apply|verify [flags]")
		os.Exit(1)
	}
	verb, rest := args[0], args[1:]
	switch verb {
	case "create":
		return provisionCreate(rest)
	case "apply":
		return provisionApply(rest)
	case "verify":
		return provisionVerify(rest)
	default:
		fmt.Fprintf(os.Stderr, "unknown provision command %q: expected create|apply|verify\n", verb)
		os.Exit(1)
	}
	return nil
}

// provisionCreate implements T1: assemble + encrypt an .ssp bundle and emit
// pairing code + CA fingerprint. Server operators run this on the box that
// owns the CA private key.
func provisionCreate(args []string) error {
	fs := flag.NewFlagSet("provision create", flag.ExitOnError)
	var (
		role       = fs.String("role", "client", "role to enroll: 'client' bundles ca+client cert/key; 'server' bootstrap bundles ca only")
		name       = fs.String("name", "", "human label recorded inside the bundle")
		out        = fs.String("out", "", "output .ssp path (default ./<name|client>-<ts>.ssp)")
		certsDir   = fs.String("certs-dir", provision.DefaultCertsDir(), "directory holding CA material; created with a fresh CA when absent")
		serverAddr = fs.String("server-addr", "", "tunnel server address clients dial (required for role=client)")
		serverPort = fs.Int("server-port", 0, "tunnel server port clients dial (required for role=client)")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() > 0 {
		return fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}
	if *role != "client" && *role != "server" {
		return fmt.Errorf("--role must be client or server")
	}
	if *role == "client" && (*serverAddr == "" || *serverPort <= 0) {
		return fmt.Errorf("--server-addr and --server-port are required for --role client")
	}

	// Ensure CA material exists locally (generate on first use), then mint a
	// unique client certificate per enrollment.
	if _, err := os.Stat(filepath.Join(*certsDir, "ca.crt")); errors.Is(err, os.ErrNotExist) {
		fmt.Fprintf(os.Stderr, "No CA found in %s; generating new CA + server pair...\n", *certsDir)
		if err := generator.GenerateCertificates(*certsDir); err != nil {
			return fmt.Errorf("generate certificates: %w", err)
		}
	}
	if *role == "client" {
		if err := generator.IssueClientCert(*certsDir); err != nil {
			return fmt.Errorf("issue enrollment certificate: %w", err)
		}
	}

	readPEM := func(name string) (string, error) {
		b, err := os.ReadFile(filepath.Join(*certsDir, name))
		if err != nil {
			return "", fmt.Errorf("read %s: %w", name, err)
		}
		return string(b), nil
	}
	caPEM, err := readPEM("ca.crt")
	if err != nil {
		return err
	}

	payload := &provision.PairingPayload{
		Role:             *role,
		ServerAddr:       *serverAddr,
		ServerPort:       *serverPort,
		CreatedAtRFC3339: time.Now().UTC().Format(time.RFC3339),
		Name:             *name,
		CACertPEM:        caPEM,
		FingerprintOfCA:  provision.Fingerprint([]byte(caPEM)),
	}
	if payload.Name == "" {
		payload.Name = strings.ToLower(*role)
	}

	switch *role {
	case "client":
		crt, err := readPEM("client.crt")
		if err != nil {
			return err
		}
		key, err := readPEM("client.key")
		if err != nil {
			return err
		}
		payload.ClientCertPEM = crt
		payload.ClientKeyPEM = key
		// The facade token secret lives in the SERVER config; require it here
		// so both roles share the exact same value from day one.
		secretPath := filepath.Join(*certsDir, "..", "token_secret")
		raw, rerr := os.ReadFile(secretPath)
		switch rerr {
		case nil:
			payload.FacadeTokenSecret = strings.TrimSpace(string(raw))
		default:
			return fmt.Errorf("read %s: %w (create this file with one high-entropy line; it must match the server config's facade.token_secret)", secretPath, rerr)
		}
		if payload.FacadeTokenSecret == "" {
			return fmt.Errorf("%s is empty; provide a high-entropy shared secret", secretPath)
		}
	}

	code, err := provision.GeneratePairingCode()
	if err != nil {
		return err
	}
	bundle, err := provision.Seal(payload, code)
	if err != nil {
		return err
	}

	outPath := *out
	if outPath == "" {
		label := payload.Name
		outPath = fmt.Sprintf("%s-%d.ssp", label, time.Now().Unix())
	}
	// Refuse to clobber an existing bundle: regenerating is cheap, silently
	// replacing an undelivered secret envelope is not.
	if _, err := os.Stat(outPath); err == nil && !overwriteForced() {
		return fmt.Errorf("output %s exists (pass -out elsewhere or set SSSONECTOR_FORCE=1)", outPath)
	}
	if err := os.WriteFile(outPath, bundle, 0o600); err != nil {
		return fmt.Errorf("write bundle: %w", err)
	}

	fmt.Printf("bundle written: %s (%d bytes)\n", outPath, len(bundle))
	fmt.Printf("pairing code:   %s\n", code)
	fmt.Printf("CA fingerprint: %s\n", payload.FingerprintOfCA)
	fmt.Println("Share the CODE out-of-band (voice/SMS). The .ssp file itself may travel by any transport — it is unreadable without the code.")
	return nil
}

func overwriteForced() bool {
	return os.Getenv("SSSONECTOR_FORCE") == "1"
}

// provisionApply implements T2/T3: decrypt, display fingerprint, confirm,
// install certs + config skeleton with restrictive permissions.
func provisionApply(args []string) error {
	fs := flag.NewFlagSet("provision apply", flag.ExitOnError)
	var (
		from     = fs.String("from", "", ".ssp bundle path (required)")
		dir      = fs.String("certs-dir", provision.DefaultCertsDir(), "target certificate directory")
		force    = fs.Bool("force", false, "overwrite existing files")
		skipConf = fs.Bool("yes", false, "skip fingerprint confirmation prompt (automation)")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *from == "" || fs.NArg() > 0 {
		return fmt.Errorf("--from <path> is required")
	}
	bundle, err := os.ReadFile(*from)
	if err != nil {
		return fmt.Errorf("read bundle: %w", err)
	}

	if !provision.IsStdinTerminal() {
		return fmt.Errorf("%w: apply requires interactive confirmation", provision.ErrNotTerminal)
	}
	code, err := provision.PromptHidden("Enter pairing code: ")
	if err != nil {
		return err
	}
	norm, err := provision.NormalizePairingCode(code)
	if err != nil {
		return err
	}

	payload, err := provision.Open(bundle, norm)
	if err != nil {
		// Deliberately generic: never reveal whether code or bytes were wrong.
		return err
	}

	fmt.Printf("\nBundle contents:\n")
	fmt.Printf("  role:            %s\n", payload.Role)
	fmt.Printf("  server:          %s:%d\n", payload.ServerAddr, payload.ServerPort)
	fmt.Printf("  label:           %s\n", payload.Name)
	fmt.Printf("  created:         %s\n", payload.CreatedAtRFC3339)
	fmt.Printf("  CA fingerprint:  %s\n", payload.FingerprintOfCA)
	fmt.Printf("  target dir:      %s\n\n", *dir)

	targets := map[string]string{
		"ca.crt": payload.CACertPEM,
	}
	if payload.Role == "client" {
		targets["client.crt"] = payload.ClientCertPEM
		targets["client.key"] = payload.ClientKeyPEM
	}
	for name, data := range targets {
		if data == "" {
			return fmt.Errorf("bundle field %s missing for role %s", name, payload.Role)
		}
		path := filepath.Join(*dir, name)
		if _, err := os.Stat(path); err == nil && !*force {
			return fmt.Errorf("refusing to overwrite existing %s (use --force after reviewing)", path)
		}
	}

	if !*skipConf {
		ok, cerr := provision.Confirm(fmt.Sprintf("Install these files into %s ?", *dir))
		if cerr != nil {
			return cerr
		}
		if !ok {
			return fmt.Errorf("aborted by operator")
		}
	}

	if err := os.MkdirAll(*dir, 0o750); err != nil {
		return fmt.Errorf("create certs dir: %w", err)
	}
	for name, data := range targets {
		path := filepath.Join(*dir, name)
		mode := os.FileMode(0o644)
		if strings.HasSuffix(name, ".key") {
			mode = 0o600
		}
		if err := os.WriteFile(path, []byte(data), mode); err != nil {
			return fmt.Errorf("write %s: %w", path, err)
		}
		if strings.HasSuffix(name, ".key") {
			if kerr := provision.RestrictKeyFile(path); kerr != nil {
				fmt.Fprintf(os.Stderr, "warning: key ACL hardening: %v\n", kerr)
			}
		}
	}

	skel := renderConfigSkeleton(payload)
	cfgPath := filepath.Join(filepath.Dir(*dir), "config.yaml")
	if _, err := os.Stat(cfgPath); err == nil && !*force {
		fmt.Printf("config skeleton skipped: %s already exists\n", cfgPath)
	} else if werr := os.WriteFile(cfgPath, []byte(skel), 0o600); werr != nil {
		return fmt.Errorf("write config skeleton: %w", werr)
	} else {
		fmt.Printf("config skeleton written: %s\n", cfgPath)
	}

	fmt.Println("Provisioning complete. Review config.yaml, then start the service.")
	return nil
}

func renderConfigSkeleton(p *provision.PairingPayload) string {
	certsRel := "certs"
	var b strings.Builder
	b.WriteString("# Generated by sssonector provision apply - review before production use.\n")
	b.WriteString("metadata:\n  schema_version: \"2.0.0\"\n  environment: qa\n")

	if p.Role == "client" {
		fmt.Fprintf(&b, "type: client\nconfig:\n  mode: client\n")
		b.WriteString("  logging:\n    level: info\n    format: json\n")
		fmt.Fprintf(&b, "  network:\n    name: tun0\n    interface: tun0\n    mtu: 1400\n    address: \"10.77.0.2/24\"\n")
		fmt.Fprintf(&b, "  tunnel:\n    server_address: %q\n    server_port: %d\n", p.ServerAddr, p.ServerPort)
		fmt.Fprintf(&b, "  auth:\n    cert_file: %q\n    key_file: %q\n    ca_file: %q\n",
			certsRel+"/client.crt", certsRel+"/client.key", certsRel+"/ca.crt")
		b.WriteString("  security:\n    allow_plaintext: false\n    tls:\n      min_version: \"1.2\"\n      max_version: \"1.3\"\n")
		fmt.Fprintf(&b, "  facade:\n    enabled: false\n    token_secret: %q\n", p.FacadeTokenSecret)
		return b.String()
	}

	fmt.Fprintf(&b, "type: server\nconfig:\n  mode: server\n")
	b.WriteString("  logging:\n    level: info\n    format: json\n")
	fmt.Fprintf(&b, "  network:\n    name: tun0\n    interface: tun0\n    mtu: 1400\n    address: \"10.77.0.1/24\"\n")
	fmt.Fprintf(&b, "  tunnel:\n    listen_address: \"0.0.0.0\"\n    listen_port: %d\n", p.ServerPort)
	fmt.Fprintf(&b, "  auth:\n    cert_file: %q\n    key_file: %q\n    ca_file: %q\n",
		certsRel+"/server.crt", certsRel+"/server.key", certsRel+"/ca.crt")
	b.WriteString("  security:\n    allow_plaintext: false\n    tls:\n      min_version: \"1.2\"\n      max_version: \"1.3\"\n")
	fmt.Fprintf(&b, "  facade:\n    enabled: true\n    listen_port: 443\n    token_secret: %q\n    tunnel_ports:\n      - %d\n",
		p.FacadeTokenSecret, p.ServerPort)
	b.WriteString("  monitor:\n    enabled: true\n    type: prometheus\n    interval: 10s\n    prometheus:\n      enabled: true\n      port: 9090\n      path: /metrics\n")
	return b.String()
}

// provisionVerify implements T7: expiry / chain / optional expected-fingerprint.
func provisionVerify(args []string) error {
	fs := flag.NewFlagSet("provision verify", flag.ExitOnError)
	var (
		dir      = fs.String("certs-dir", provision.DefaultCertsDir(), "certificate directory to verify")
		expectFP = fs.String("expect-fingerprint", "", "fail unless the CA fingerprint equals this SHA-256 hex digest")
		rotateIn = fs.Duration("rotation-within", 30*24*time.Hour, "warn when expiry falls within this window")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}

	caPEM, err := os.ReadFile(filepath.Join(*dir, "ca.crt"))
	if err != nil {
		return fmt.Errorf("read CA: %w", err)
	}
	fp := provision.Fingerprint(caPEM)
	status := 0

	if *expectFP != "" {
		if provision.CodesMatch(strings.ToLower(*expectFP), fp) {
			fmt.Printf("fingerprint:  MATCH    %s\n", fp)
		} else {
			fmt.Printf("fingerprint:  MISMATCH got=%s want=%s\n", fp, strings.ToLower(*expectFP))
			status = 1
		}
	} else {
		fmt.Printf("fingerprint:  %s\n", fp)
	}

	checkOne := func(file string) {
		data, err := os.ReadFile(filepath.Join(*dir, file))
		if err != nil {
			fmt.Printf("%-12s MISSING (%v)\n", file, err)
			status = 1
			return
		}
		block, _ := pem.Decode(data)
		if block == nil {
			fmt.Printf("%-12s UNPARSEABLE PEM\n", file)
			status = 1
			return
		}
		crt, err := x509ParseBlock(block)
		if err != nil {
			fmt.Printf("%-12s PARSE ERROR: %v\n", file, err)
			status = 1
			return
		}
		days := int(time.Until(crt.NotAfter).Hours() / 24)
		state := "ok"
		if time.Now().After(crt.NotAfter) {
			state = "EXPIRED"
			status = 1
		} else if days <= int((*rotateIn).Hours()/24) {
			state = fmt.Sprintf("ROTATION-DUE (<%dd)", int((*rotateIn).Hours()/24)+1)
		}
		fmt.Printf("%-12s expires %-12s (%3dd) [%s] CN=%s\n", file, crt.NotAfter.UTC().Format("2006-01-02"), days, state, crt.Subject.CommonName)
	}
	checkOne("ca.crt")
	checkOne("server.crt")
	checkOne("client.crt")

	if status != 0 {
		return fmt.Errorf("verification reported problems above")
	}
	fmt.Println("verify: all checks passed")
	return nil
}

// x509ParseBlock decodes the first certificate in a PEM block.
func x509ParseBlock(block *pem.Block) (*x509.Certificate, error) {
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, errors.New("not a CERTIFICATE pem block")
	}
	return x509.ParseCertificate(block.Bytes)
}

var _ = json.Marshal // reserved for Phase 2 redemption payloads
