package tunnel

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"time"

	"SSSonector/internal/config"
)

// CertificateGenerator handles certificate generation
type CertificateGenerator struct {
	config *config.TLSConfig
}

// NewCertificateGenerator creates a new certificate generator
func NewCertificateGenerator(cfg *config.TLSConfig) *CertificateGenerator {
	return &CertificateGenerator{
		config: cfg,
	}
}

// GenerateCertificate generates a self-signed certificate and key
func (g *CertificateGenerator) GenerateCertificate() error {
	// Create certificate directory if it doesn't exist
	certDir := filepath.Dir(g.config.CertFile)
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return fmt.Errorf("failed to create certificate directory: %w", err)
	}

	// Generate private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Prepare certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			Organization: []string{"SSL Tunnel"},
			CommonName:   "SSL Tunnel Certificate",
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Duration(g.config.ValidityDays) * 24 * time.Hour),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,

		// Add common IP addresses and hostnames
		IPAddresses: []net.IP{
			net.ParseIP("127.0.0.1"),
			net.ParseIP("::1"),
		},
		DNSNames: []string{
			"localhost",
		},
	}

	// Create certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	// Save certificate
	certOut, err := os.Create(g.config.CertFile)
	if err != nil {
		return fmt.Errorf("failed to create certificate file: %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	// Save private key
	keyOut, err := os.OpenFile(g.config.KeyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %w", err)
	}
	defer keyOut.Close()

	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	return nil
}

// LoadOrGenerateCertificate loads existing certificate or generates a new one
func (g *CertificateGenerator) LoadOrGenerateCertificate() error {
	// Check if certificate files exist
	certExists := fileExists(g.config.CertFile)
	keyExists := fileExists(g.config.KeyFile)

	// If either file is missing and auto-generate is enabled, generate new certificate
	if g.config.AutoGenerate && (!certExists || !keyExists) {
		return g.GenerateCertificate()
	}

	// If files don't exist and auto-generate is disabled, return error
	if !certExists {
		return fmt.Errorf("certificate file not found: %s", g.config.CertFile)
	}
	if !keyExists {
		return fmt.Errorf("private key file not found: %s", g.config.KeyFile)
	}

	return nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
