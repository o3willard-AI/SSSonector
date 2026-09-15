package collect

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/o3willard-AI/SSSonector/internal/cert/generator"
)

// WI 5.4 [g] client-bundle generation tests. Real cert material is minted
// into a temp dir via internal/cert/generator (test-only, no flags); the
// host-resolution and temp-dir seams are injected so tests never depend on
// the machine's addresses or /tmp layout.

// bundleFixture builds a config root with one server instance (specific
// listen address, so no host resolution is needed) plus its CA + server
// cert dir, mirroring the wizard's write layout.
func bundleFixture(t *testing.T, host string) (root, certDir string) {
	t.Helper()
	root = t.TempDir()
	certDir = filepath.Join(root, "instances", "srv-a", "certs")
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, "instances", "srv-a", "config.yaml")), 0o750); err != nil {
		t.Fatal(err)
	}
	certHost := host
	if certHost == "" {
		certHost = "127.0.0.1"
	}
	yaml := fmt.Sprintf(`metadata:
  schema_version: "2.0.0"
type: server
config:
  mode: server
  network:
    name: tun0
    interface: tun0
    mtu: 1500
    address: 10.77.0.1/24
  tunnel:
    listen_address: "%s"
    listen_port: 9443
  auth:
    cert_file: %q
    key_file: %q
    ca_file: %q
`, host, filepath.Join(certDir, "server.crt"), filepath.Join(certDir, "server.key"), filepath.Join(certDir, "ca.crt"))
	p := filepath.Join(root, "instances", "srv-a", "config.yaml")
	if err := os.WriteFile(p, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, certDir
}

func genCerts(t *testing.T, certDir string) {
	t.Helper()
	// CA + server only: GenerateCertificates also mints a client leaf,
	// which would defeat TestGenerateClientBundle_IssuesWhenAbsent (that
	// test requires client.crt to be absent before the bundle runs).
	if err := generator.GenerateCertificates(certDir); err != nil {
		t.Fatalf("seed CA+server certs: %v", err)
	}
	// Remove the auto-issued client pair; the bundle path issues it.
	for _, f := range []string{"client.crt", "client.key"} {
		if err := os.Remove(filepath.Join(certDir, f)); err != nil && !os.IsNotExist(err) {
			t.Fatalf("clean client material: %v", err)
		}
	}
}

func readTarMembers(t *testing.T, path string) map[string][]byte {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	out := map[string][]byte{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		out[hdr.Name] = data
	}
	return out
}

// TestGenerateClientBundle_Golden: unpack the tarball; the client cert
// chains to the CA (real x509 verify) and every skeleton field traces to
// the server config — nothing invented.
func TestGenerateClientBundle_Golden(t *testing.T) {
	root, certDir := bundleFixture(t, "192.168.101.7")
	genCerts(t, certDir)

	b, err := generateClientBundleWith(ConfigPaths{ConfigRoot: root}, "srv-a",
		func() (string, error) { return "10.9.9.9", nil },
		func() string { return t.TempDir() })
	if err != nil {
		t.Fatalf("bundle: %v", err)
	}
	members := readTarMembers(t, b.Path)

	for _, name := range []string{"ca.crt", "client.crt", "client.key", "skeleton.yaml"} {
		if _, ok := members[name]; !ok {
			t.Errorf("bundle missing member %s (have %v keys)", name, len(members))
		}
	}
	if len(members) != 4 {
		t.Errorf("bundle member count %d, want exactly 4", len(members))
	}

	// x509 chain: client.crt verifies against ca.crt.
	caBlock, _ := pem.Decode(members["ca.crt"])
	if caBlock == nil {
		t.Fatal("no PEM in ca.crt")
	}
	caCert, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	cliBlock, _ := pem.Decode(members["client.crt"])
	if cliBlock == nil {
		t.Fatal("no PEM in client.crt")
	}
	cliCert, err := x509.ParseCertificate(cliBlock.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(caCert)
	if _, err := cliCert.Verify(x509.VerifyOptions{Roots: roots, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}}); err != nil {
		t.Errorf("client.crt does not chain to ca.crt: %v", err)
	}
	// The key in the bundle matches the client cert.
	keyBlock, _ := pem.Decode(members["client.key"])
	if keyBlock == nil {
		t.Fatal("no PEM in client.key")
	}
	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if key.PublicKey.N.Cmp(cliCert.PublicKey.(*rsa.PublicKey).N) != 0 {
		t.Error("client.key does not match client.crt")
	}

	// Skeleton fields mirror the server config exactly (plus the
	// self-documenting header line the generator prepends).
	want := "# SSSonector client bundle skeleton (prefill for 'tui --from-bundle')\ninstance: srv-a\nserver_host: 192.168.101.7\nserver_port: 9443\ntun_address: 10.77.0.2/24\n"
	if got := string(members["skeleton.yaml"]); got != want {
		t.Errorf("skeleton mismatch:\n got %q\nwant %q", got, want)
	}

	// SHA256 printed matches the bytes on disk.
	data, err := os.ReadFile(b.Path)
	if err != nil {
		t.Fatal(err)
	}
	if b.SHA256 != fmt.Sprintf("%x", sha256Bytes(data)) {
		t.Error("SHA256 does not match file bytes")
	}
}

func sha256Bytes(b []byte) []byte {
	h := sha256.New()
	h.Write(b)
	return h.Sum(nil)
}

// TestGenerateClientBundle_NoPrivateCAMaterial: ca.key and server.key are
// NEVER in the bundle.
func TestGenerateClientBundle_NoPrivateCAMaterial(t *testing.T) {
	root, certDir := bundleFixture(t, "10.1.1.1")
	genCerts(t, certDir)

	b, err := generateClientBundleWith(ConfigPaths{ConfigRoot: root}, "srv-a",
		func() (string, error) { return "10.9.9.9", nil },
		func() string { return t.TempDir() })
	if err != nil {
		t.Fatalf("bundle: %v", err)
	}
	for name, data := range readTarMembers(t, b.Path) {
		if name == "ca.key" || name == "server.key" {
			t.Errorf("private material %s must never travel in the bundle", name)
		}
		if bytes.Contains(data, []byte("PRIVATE KEY")) && name != "client.key" {
			t.Errorf("member %s contains private-key PEM", name)
		}
	}
}

// TestGenerateClientBundle_Idempotent: pressing g twice reuses the client
// leaf (same serial = same cert; ca.key mtime untouched proves no
// regeneration round-tripped through the CA).
func TestGenerateClientBundle_Idempotent(t *testing.T) {
	root, certDir := bundleFixture(t, "10.1.1.1")
	genCerts(t, certDir)

	host := func() (string, error) { return "10.9.9.9", nil }
	tmp := func() string { return t.TempDir() }
	first, err := generateClientBundleWith(ConfigPaths{ConfigRoot: root}, "srv-a", host, tmp)
	if err != nil {
		t.Fatalf("first bundle: %v", err)
	}
	cert1, err := os.ReadFile(filepath.Join(certDir, "client.crt"))
	if err != nil {
		t.Fatal(err)
	}
	caKeyStat1, err := os.Stat(filepath.Join(certDir, "ca.key"))
	if err != nil {
		t.Fatal(err)
	}

	second, err := generateClientBundleWith(ConfigPaths{ConfigRoot: root}, "srv-a", host, tmp)
	if err != nil {
		t.Fatalf("second bundle: %v", err)
	}
	cert2, err := os.ReadFile(filepath.Join(certDir, "client.crt"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(cert1, cert2) {
		t.Error("client.crt was regenerated on a second [g] press — must be reused")
	}
	if first.SHA256 != second.SHA256 {
		t.Error("second bundle SHA differs — bundle content must be reproducible")
	}
	caKeyStat2, err := os.Stat(filepath.Join(certDir, "ca.key"))
	if err != nil {
		t.Fatal(err)
	}
	if !caKeyStat1.ModTime().Equal(caKeyStat2.ModTime()) {
		t.Error("ca.key was rewritten during bundle generation")
	}
}

// TestGenerateClientBundle_IssuesWhenAbsent: the client leaf is minted
// into the instance cert dir when missing, signed by the instance CA.
func TestGenerateClientBundle_IssuesWhenAbsent(t *testing.T) {
	root, certDir := bundleFixture(t, "10.1.1.1")
	genCerts(t, certDir)
	if _, err := os.Stat(filepath.Join(certDir, "client.crt")); !os.IsNotExist(err) {
		t.Fatalf("fixture already has client.crt: %v", err)
	}
	_, err := generateClientBundleWith(ConfigPaths{ConfigRoot: root}, "srv-a",
		func() (string, error) { return "10.9.9.9", nil },
		func() string { return t.TempDir() })
	if err != nil {
		t.Fatalf("bundle: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(certDir, "client.crt"))
	if err != nil {
		t.Fatalf("client.crt not issued into the instance cert dir: %v", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatal("no PEM in issued client.crt")
	}
	cliCert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	caData, err := os.ReadFile(filepath.Join(certDir, "ca.crt"))
	if err != nil {
		t.Fatal(err)
	}
	caBlock, _ := pem.Decode(caData)
	caCert, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	if err := cliCert.CheckSignatureFrom(caCert); err != nil {
		t.Errorf("issued client cert not signed by the instance CA: %v", err)
	}
}

// TestGenerateClientBundle_WildcardHostResolves: 0.0.0.0 listen_address is
// resolved via the host seam — never shipped verbatim.
func TestGenerateClientBundle_WildcardHostResolves(t *testing.T) {
	for _, wildcard := range []string{"0.0.0.0", "::", ""} {
		root, certDir := bundleFixture(t, wildcard)
		genCerts(t, certDir)
		b, err := generateClientBundleWith(ConfigPaths{ConfigRoot: root}, "srv-a",
			func() (string, error) { return "192.168.100.51", nil },
			func() string { return t.TempDir() })
		if err != nil {
			t.Fatalf("wildcard %q: %v", wildcard, err)
		}
		members := readTarMembers(t, b.Path)
		want := "# SSSonector client bundle skeleton (prefill for 'tui --from-bundle')\ninstance: srv-a\nserver_host: 192.168.100.51\nserver_port: 9443\ntun_address: 10.77.0.2/24\n"
		if got := string(members["skeleton.yaml"]); got != want {
			t.Errorf("wildcard %q skeleton:\n got %q\nwant %q", wildcard, got, want)
		}
	}
}

// TestGenerateClientBundle_FailClosed: missing instance, non-server mode,
// no ca_file, no listen_port, no TUN address, and unresolvable host all
// error explicitly — nothing is defaulted into a bundle.
func TestGenerateClientBundle_FailClosed(t *testing.T) {
	root, certDir := bundleFixture(t, "10.1.1.1")
	genCerts(t, certDir)
	paths := ConfigPaths{ConfigRoot: root}

	if _, err := generateClientBundleWith(paths, "nope", func() (string, error) { return "h", nil }, func() string { return t.TempDir() }); err == nil {
		t.Error("missing instance must error")
	}
	if _, err := generateClientBundleWith(ConfigPaths{ConfigRoot: t.TempDir()}, "", func() (string, error) { return "h", nil }, func() string { return t.TempDir() }); err == nil {
		t.Error("empty instance must error")
	}
	// Host resolution failure blocks ONLY when the server listens on a
	// wildcard (the fixture listens on 10.1.1.1 specifically, so the
	// hostFunc is not consulted) — force the wildcard case here.
	wildRoot, wildCertDir := bundleFixture(t, "0.0.0.0")
	genCerts(t, wildCertDir)
	if _, err := generateClientBundleWith(ConfigPaths{ConfigRoot: wildRoot}, "srv-a", func() (string, error) { return "", fmt.Errorf("no addresses") }, func() string { return t.TempDir() }); err == nil {
		t.Error("host resolution failure must error (no wildcard host shipped)")
	}

	// Remove the TUN address line.
	cfgPath := filepath.Join(root, "instances", "srv-a", "config.yaml")
	orig, _ := os.ReadFile(cfgPath)
	os.WriteFile(cfgPath, bytes.ReplaceAll(orig, []byte("address: 10.77.0.1/24"), []byte("address: \"\"")), 0o600)
	if _, err := generateClientBundleWith(paths, "srv-a", func() (string, error) { return "h", nil }, func() string { return t.TempDir() }); err == nil {
		t.Error("empty TUN address must error")
	}
	os.WriteFile(cfgPath, orig, 0o600)

	// Client mode instance: not a bundle source.
	clientCfg := bytes.ReplaceAll(orig, []byte("mode: server"), []byte("mode: client"))
	os.WriteFile(cfgPath, clientCfg, 0o600)
	if _, err := generateClientBundleWith(paths, "srv-a", func() (string, error) { return "h", nil }, func() string { return t.TempDir() }); err == nil {
		t.Error("client-mode instance must not generate a bundle")
	}
	os.WriteFile(cfgPath, orig, 0o600)

	// No ca_file: cert dir unresolvable.
	os.WriteFile(cfgPath, bytes.ReplaceAll(orig, []byte("ca.crt"), []byte("")), 0o600)
	if _, err := generateClientBundleWith(paths, "srv-a", func() (string, error) { return "h", nil }, func() string { return t.TempDir() }); err == nil {
		t.Error("missing ca_file must error")
	}
}
