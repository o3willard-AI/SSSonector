package collect

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// makeCertPEM generates a self-signed leaf with the given expiry relative
// to a fixed base time, returning the PEM bytes and the NotAfter.
func makeCertPEM(t *testing.T, base time.Time, ttl time.Duration) ([]byte, time.Time) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	notAfter := base.Add(ttl)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "sssonector-test-leaf"},
		Issuer:       pkix.Name{CommonName: "sssonector-test-ca"},
		NotBefore:    base.Add(-24 * time.Hour),
		NotAfter:     notAfter,
		IsCA:         true,
		KeyUsage:     x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create cert: %v", err)
	}
	p := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	return p, notAfter
}

// writeCertFile writes PEM to a temp file and returns its path.
func writeCertFile(t *testing.T, dir, name string, pemBytes []byte) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, pemBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestReadCertInfo_Valid(t *testing.T) {
	base := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	pemBytes, notAfter := makeCertPEM(t, base, 60*24*time.Hour)
	dir := t.TempDir()
	certPath := writeCertFile(t, dir, "server.crt", pemBytes)

	info, err := ReadCertInfo(CertPaths{CertFile: certPath}, 0, base)
	if err != nil {
		t.Fatalf("ReadCertInfo: %v", err)
	}
	if !strings.Contains(info.Issuer, "sssonector-test-leaf") {
		t.Errorf("issuer: %q", info.Issuer)
	}
	if !info.NotAfter.Equal(notAfter) {
		t.Errorf("notAfter: got %v, want %v", info.NotAfter, notAfter)
	}
	if info.DaysRemaining != 60 {
		t.Errorf("days remaining: got %d, want 60", info.DaysRemaining)
	}
	if info.NeedsRotation {
		t.Error("60 days out must not need rotation (default threshold 30d)")
	}
}

func TestReadCertInfo_ExpiryEdges(t *testing.T) {
	base := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		name         string
		ttl          time.Duration
		wantDays     int
		wantRotation bool
	}{
		{"expires in 31 days (just past default threshold)", 61 * 24 * time.Hour, 61, false},
		{"expires in exactly 30 days", 30 * 24 * time.Hour, 30, true},
		{"expires tomorrow", 1 * 24 * time.Hour, 1, true},
		{"expires today-ish (12h)", 12 * time.Hour, 0, true},
		{"already expired", -5 * 24 * time.Hour, -5, true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			pemBytes, _ := makeCertPEM(t, base, tc.ttl)
			certPath := writeCertFile(t, t.TempDir(), "c.crt", pemBytes)
			info, err := ReadCertInfo(CertPaths{CertFile: certPath}, 0, base)
			if err != nil {
				t.Fatalf("ReadCertInfo: %v", err)
			}
			if info.DaysRemaining != tc.wantDays {
				t.Errorf("days: got %d, want %d", info.DaysRemaining, tc.wantDays)
			}
			if info.NeedsRotation != tc.wantRotation {
				t.Errorf("needsRotation: got %v, want %v (days=%d)", info.NeedsRotation, tc.wantRotation, info.DaysRemaining)
			}
		})
	}
}

func TestReadCertInfo_CustomRotationThreshold(t *testing.T) {
	base := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	// 45 days out: beyond default 30d (no rotation) but within a custom
	// 60-day interval (rotation).
	pemBytes, _ := makeCertPEM(t, base, 45*24*time.Hour)
	certPath := writeCertFile(t, t.TempDir(), "c.crt", pemBytes)

	info, err := ReadCertInfo(CertPaths{CertFile: certPath}, 0, base)
	if err != nil {
		t.Fatalf("default threshold: %v", err)
	}
	if info.NeedsRotation {
		t.Error("45d > 30d default: must not need rotation")
	}

	info, err = ReadCertInfo(CertPaths{CertFile: certPath}, 60*24*time.Hour, base)
	if err != nil {
		t.Fatalf("custom threshold: %v", err)
	}
	if !info.NeedsRotation {
		t.Error("45d <= 60d custom threshold: must need rotation")
	}
}

func TestReadCertInfo_Errors(t *testing.T) {
	base := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)

	t.Run("missing file", func(t *testing.T) {
		_, err := ReadCertInfo(CertPaths{CertFile: filepath.Join(t.TempDir(), "nope.crt")}, 0, base)
		if err == nil || !strings.Contains(err.Error(), "read") {
			t.Errorf("want explicit read error, got %v", err)
		}
	})

	t.Run("empty cert path", func(t *testing.T) {
		_, err := ReadCertInfo(CertPaths{}, 0, base)
		if err == nil || !strings.Contains(err.Error(), "no cert_file configured") {
			t.Errorf("want no-cert_file error, got %v", err)
		}
	})

	t.Run("unparseable garbage", func(t *testing.T) {
		p := writeCertFile(t, t.TempDir(), "c.crt", []byte("this is not a PEM file at all"))
		_, err := ReadCertInfo(CertPaths{CertFile: p}, 0, base)
		if err == nil || !strings.Contains(err.Error(), "no CERTIFICATE PEM block") {
			t.Errorf("want PEM error, got %v", err)
		}
	})

	t.Run("PEM block but not x509", func(t *testing.T) {
		block := &pem.Block{Type: "CERTIFICATE", Bytes: []byte("not DER")}
		p := writeCertFile(t, t.TempDir(), "c.crt", pem.EncodeToMemory(block))
		_, err := ReadCertInfo(CertPaths{CertFile: p}, 0, base)
		if err == nil || !strings.Contains(err.Error(), "x509 parse") {
			t.Errorf("want x509 error, got %v", err)
		}
	})

	t.Run("wrong PEM type only (private key)", func(t *testing.T) {
		block := &pem.Block{Type: "PRIVATE KEY", Bytes: []byte{1, 2, 3}}
		p := writeCertFile(t, t.TempDir(), "c.pem", pem.EncodeToMemory(block))
		_, err := ReadCertInfo(CertPaths{CertFile: p}, 0, base)
		if err == nil {
			t.Error("non-certificate PEM must error, not panic")
		}
	})
}

func TestPoller_CertSourcePopulated(t *testing.T) {
	base := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	pemBytes, _ := makeCertPEM(t, base, 90*24*time.Hour)
	certPath := writeCertFile(t, t.TempDir(), "server.crt", pemBytes)

	d := newFixtureDaemon(t)
	root := t.TempDir()
	writeInstanceConfig(t, root, "client-a", d.port(t), true)
	// Point the config's cert_file at the fixture PEM.
	p := filepath.Join(root, "instances", "client-a", "config.yaml")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(data), `cert_file: ""`, "cert_file: "+certPath, 1)
	if err := os.WriteFile(p, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}

	r := newFakeRunner()
	r.outputs["systemctl list-units sssonector@* --all --plain --no-legend"] =
		"sssonector@client-a.service loaded active running x\n"
	r.outputs["systemctl show sssonector@client-a.service -p ActiveState -p SubState -p MainPID"] =
		"ActiveState=active\nSubState=running\nMainPID=1\n"
	r.outputs["systemctl show sssonector.service -p ActiveState -p SubState -p MainPID"] =
		"ActiveState=not-found\nSubState=dead\nMainPID=0\n"

	poller := NewPoller(SystemdCollector{Runner: r, Paths: DefaultSystemdPaths()},
		ConfigPaths{ConfigRoot: root}, d.srv.Client())
	res := poller.PollOnce(context.Background())
	s := res.Instances["client-a"]
	if s.Cert.Status() != StatusOK {
		t.Fatalf("cert source: want ok, got %v err=%v", s.Cert.Status(), s.Cert.Err())
	}
	info, _ := s.Cert.Get()
	if info.Issuer == "" || info.DaysRemaining <= 0 {
		t.Errorf("cert info not populated correctly: %+v", info)
	}

	// Now point the config at a missing file: next tick the cert source
	// must be an error — never stale, never fabricated.
	if err := os.WriteFile(p, []byte(strings.Replace(string(data), `cert_file: ""`, `cert_file: /nonexistent/gone.crt`, 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	res2 := poller.PollOnce(context.Background())
	s2 := res2.Instances["client-a"]
	if s2.Cert.Status() != StatusError {
		t.Fatalf("cert after break: want error, got %v", s2.Cert.Status())
	}
	if _, ok := s2.Cert.Get(); ok {
		t.Error("NEVER STALE violated: errored cert source returned a value")
	}
}
