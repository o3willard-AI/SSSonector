package generator

import (
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"
	"path/filepath"
	"testing"
)

// loadCert parses the PEM certificate at path.
func loadCert(t *testing.T, path string) *x509.Certificate {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read cert: %v", err)
	}
	block, _ := pem.Decode(data)
	if block == nil {
		t.Fatalf("no PEM block in %s", path)
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatalf("parse cert: %v", err)
	}
	return cert
}

func hasIP(cert *x509.Certificate, ip string) bool {
	want := net.ParseIP(ip)
	for _, got := range cert.IPAddresses {
		if got.Equal(want) {
			return true
		}
	}
	return false
}

// TestGenerateCertificatesServerSANs verifies the server certificate carries
// the server IPs passed to GenerateCertificates so TLS verification succeeds
// when clients connect by the server's address.
func TestGenerateCertificatesServerSANs(t *testing.T) {
	dir := t.TempDir()

	if err := GenerateCertificates(dir, "192.0.2.10"); err != nil {
		t.Fatalf("GenerateCertificates: %v", err)
	}

	server := loadCert(t, filepath.Join(dir, "server.crt"))
	if !hasIP(server, "127.0.0.1") {
		t.Errorf("server cert missing localhost SAN")
	}
	if !hasIP(server, "192.0.2.10") {
		t.Errorf("server cert missing passed server IP SAN (got %v)", server.IPAddresses)
	}

	client := loadCert(t, filepath.Join(dir, "client.crt"))
	if hasIP(client, "192.0.2.10") {
		t.Errorf("client cert should not carry the server IP SAN")
	}
}

// TestGenerateCertificatesNoServerIPs ensures no-arg callers (existing
// non-provisioning paths) still get a server cert with only localhost.
func TestGenerateCertificatesNoServerIPs(t *testing.T) {
	dir := t.TempDir()

	if err := GenerateCertificates(dir); err != nil {
		t.Fatalf("GenerateCertificates: %v", err)
	}

	server := loadCert(t, filepath.Join(dir, "server.crt"))
	if !hasIP(server, "127.0.0.1") {
		t.Errorf("server cert missing localhost SAN")
	}
	if len(server.IPAddresses) != 1 {
		t.Errorf("server cert should have only localhost SAN when no server IPs given (got %v)", server.IPAddresses)
	}
}
