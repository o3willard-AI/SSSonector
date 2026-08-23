package cert

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/cert/generator"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// waitForRotation polls the manager until the presented certificate serial
// changes or the deadline elapses. It tolerates transient read errors while
// rotation is in progress.
func waitForRotation(manager *Manager, timeout time.Duration) (*tls.Certificate, error) {
	initial, err := manager.getCertificate(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get initial certificate: %w", err)
	}
	initialCert, err := x509.ParseCertificate(initial.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse initial certificate: %w", err)
	}

	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		cert, err := manager.getCertificate(nil)
		if err == nil {
			parsed, perr := x509.ParseCertificate(cert.Certificate[0])
			if perr == nil && parsed.SerialNumber.Cmp(initialCert.SerialNumber) != 0 {
				return cert, nil
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil, fmt.Errorf("rotation not observed within %v", timeout)
}

func TestCertificateRotation(t *testing.T) {
	// Create temporary directory for test certificates
	tempDir, err := os.MkdirTemp("", "cert-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Generate initial certificates
	if err := generator.GenerateTemporaryCertificates(tempDir); err != nil {
		t.Fatalf("Failed to generate initial certificates: %v", err)
	}

	// Create certificate manager with short check interval
	manager, err := NewManager(
		filepath.Join(tempDir, "server.crt"),
		filepath.Join(tempDir, "server.key"),
		filepath.Join(tempDir, "ca.crt"),
		true,
		false,
		zap.NewNop(),
	)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}
	defer manager.Stop()

	// Configure for testing
	manager.SetCheckInterval(100 * time.Millisecond)
	manager.SetRotationThreshold(10 * time.Second)
	manager.UseTemporaryCerts(true)

	// Get initial TLS config
	config, err := manager.GetTLSConfig()
	if err != nil {
		t.Fatalf("Failed to get TLS config: %v", err)
	}

	// Get initial certificate serial number
	cert, err := manager.getCertificate(nil)
	if err != nil {
		t.Fatalf("Failed to get initial certificate: %v", err)
	}
	initialCert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("Failed to parse initial certificate: %v", err)
	}
	initialSerial := initialCert.SerialNumber

	// Poll for certificate rotation instead of a blind sleep: the rotation
	// threshold is 10s, but scheduler/race-detector load can delay the
	// background check, so allow a generous deadline and finish as soon as
	// rotation is observed.
	newCert, err := waitForRotation(manager, 30*time.Second)
	if err != nil {
		t.Fatalf("Certificate was not rotated in time: %v", err)
	}
	rotatedCert, err := x509.ParseCertificate(newCert.Certificate[0])
	if err != nil {
		t.Fatalf("Failed to parse rotated certificate: %v", err)
	}
	newSerial := rotatedCert.SerialNumber

	// Verify certificate was rotated by comparing serial numbers
	if initialSerial.Cmp(newSerial) == 0 {
		t.Error("Certificate was not rotated (serial numbers match)")
	}

	// Verify TLS config still works
	if _, err := tls.Listen("tcp", "localhost:0", config); err != nil {
		t.Errorf("TLS config invalid after rotation: %v", err)
	}
}

func TestCertificateExpiration(t *testing.T) {
	// Create temporary directory for test certificates
	tempDir, err := os.MkdirTemp("", "cert-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Generate initial certificates with short expiration
	if err := generator.GenerateTemporaryCertificates(tempDir); err != nil {
		t.Fatalf("Failed to generate initial certificates: %v", err)
	}

	// Create certificate manager with short check interval
	manager, err := NewManager(
		filepath.Join(tempDir, "server.crt"),
		filepath.Join(tempDir, "server.key"),
		filepath.Join(tempDir, "ca.crt"),
		true,
		false,
		zap.NewNop(),
	)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}
	defer manager.Stop()

	// Configure for testing
	manager.SetCheckInterval(100 * time.Millisecond)
	// Set rotation threshold to 0 to prevent rotation
	manager.SetRotationThreshold(0)
	manager.UseTemporaryCerts(true)

	// Get initial certificate expiry time
	cert := manager.currentCert
	if cert == nil {
		t.Fatal("No current certificate")
	}
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		t.Fatalf("Failed to parse certificate: %v", err)
	}
	initialExpiry := x509Cert.NotAfter

	// Wait for the certificate to expire (poll with deadline; expiry time
	// comes from the generated certificate, typically ~15s for temp certs).
	timeToExpiry := time.Until(initialExpiry)
	t.Logf("Waiting %v for certificate to expire", timeToExpiry)

	select {
	case <-manager.expireChan:
		// Expected behavior
	case <-time.After(timeToExpiry + 5*time.Second):
		// Get current certificate for debugging
		cert := manager.currentCert
		if cert != nil {
			x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
			if err == nil {
				t.Logf("Current certificate expires in: %v", time.Until(x509Cert.NotAfter))
			}
		}
		t.Error("Certificate expiration not detected")
	}
}

func TestCertificateValidation(t *testing.T) {
	// Create temporary directory for test certificates
	tempDir, err := os.MkdirTemp("", "cert-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Generate certificates
	if err := generator.GenerateCertificates(tempDir); err != nil {
		t.Fatalf("Failed to generate certificates: %v", err)
	}

	tests := []struct {
		name      string
		certFile  string
		keyFile   string
		caFile    string
		isServer  bool
		wantError bool
	}{
		{
			name:      "Valid server certificate",
			certFile:  filepath.Join(tempDir, "server.crt"),
			keyFile:   filepath.Join(tempDir, "server.key"),
			caFile:    filepath.Join(tempDir, "ca.crt"),
			isServer:  true,
			wantError: false,
		},
		{
			name:      "Valid client certificate",
			certFile:  filepath.Join(tempDir, "client.crt"),
			keyFile:   filepath.Join(tempDir, "client.key"),
			caFile:    filepath.Join(tempDir, "ca.crt"),
			isServer:  false,
			wantError: false,
		},
		{
			name:      "Invalid certificate path",
			certFile:  filepath.Join(tempDir, "nonexistent.crt"),
			keyFile:   filepath.Join(tempDir, "server.key"),
			caFile:    filepath.Join(tempDir, "ca.crt"),
			isServer:  true,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewManager(tt.certFile, tt.keyFile, tt.caFile, tt.isServer, false, zap.NewNop())
			if (err != nil) != tt.wantError {
				t.Errorf("NewManager() error = %v, wantError %v", err, tt.wantError)
			}
		})
	}
}

func TestNewManagerRequiresLogger(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cert-nologger-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	if err := generator.GenerateTemporaryCertificates(tempDir); err != nil {
		t.Fatalf("Failed to generate certificates: %v", err)
	}

	_, err = NewManager(
		filepath.Join(tempDir, "server.crt"),
		filepath.Join(tempDir, "server.key"),
		filepath.Join(tempDir, "ca.crt"),
		true,
		false,
		nil,
	)
	if err == nil {
		t.Fatal("NewManager() with nil logger should fail, got nil error")
	}
}

func TestManagerEmitsStructuredCertLogs(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "cert-logs-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	if err := generator.GenerateTemporaryCertificates(tempDir); err != nil {
		t.Fatalf("Failed to generate certificates: %v", err)
	}

	core, observed := observer.New(zapcore.InfoLevel)
	manager, err := NewManager(
		filepath.Join(tempDir, "server.crt"),
		filepath.Join(tempDir, "server.key"),
		filepath.Join(tempDir, "ca.crt"),
		true,
		false,
		zap.New(core),
	)
	if err != nil {
		t.Fatalf("Failed to create certificate manager: %v", err)
	}
	defer manager.Stop()

	// Threshold 0 keeps this evaluation on the status-report path; the
	// drainer guards against the rotation path blocking on rotationDone
	// if the temporary certificate expires mid-test.
	manager.SetRotationThreshold(0)
	drained := make(chan struct{})
	defer close(drained)
	go func() {
		for {
			select {
			case <-manager.rotationDone:
			case <-drained:
				return
			}
		}
	}()

	manager.checkCertificate()

	entries := observed.All()
	if len(entries) == 0 {
		t.Fatal("Expected at least one log entry from checkCertificate, got none")
	}

	foundStatus := false
	for _, e := range entries {
		if e.Message != "Certificate status" {
			continue
		}
		foundStatus = true

		var expiresIn zapcore.Field
		for _, f := range e.Context {
			if f.Key == "expires_in" {
				expiresIn = f
			}
		}
		if expiresIn.Key != "expires_in" {
			t.Error("Certificate status entry missing expires_in field")
		} else if expiresIn.Type != zapcore.DurationType {
			t.Errorf("expires_in field should be a duration, got type %v", expiresIn.Type)
		}

		serialOK := false
		for _, f := range e.Context {
			if f.Key == "serial" && f.String != "" {
				serialOK = true
			}
		}
		if !serialOK {
			t.Error("Certificate status entry missing non-empty serial field")
		}
	}

	if !foundStatus {
		t.Fatalf("Expected a 'Certificate status' entry, got: %+v", entries)
	}
}
