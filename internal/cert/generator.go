package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// CertificateGenerator handles the generation of SSL certificates
type CertificateGenerator struct {
	outputDir string
}

// NewCertificateGenerator creates a new certificate generator
func NewCertificateGenerator(outputDir string) *CertificateGenerator {
	return &CertificateGenerator{
		outputDir: outputDir,
	}
}

// GenerateCA generates a new CA certificate and private key
func (g *CertificateGenerator) GenerateCA() error {
	// Generate CA private key
	caKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed to generate CA private key: %v", err)
	}

	// Create CA certificate template
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName: "SSSonector CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 10 years validity
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create CA certificate
	caCertBytes, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("failed to create CA certificate: %v", err)
	}

	// Save CA certificate
	caCertFile := filepath.Join(g.outputDir, "ca.crt")
	certOut, err := os.Create(caCertFile)
	if err != nil {
		return fmt.Errorf("failed to create CA certificate file: %v", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: caCertBytes}); err != nil {
		return fmt.Errorf("failed to write CA certificate: %v", err)
	}

	// Save CA private key
	caKeyFile := filepath.Join(g.outputDir, "ca.key")
	keyOut, err := os.OpenFile(caKeyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create CA private key file: %v", err)
	}
	defer keyOut.Close()

	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(caKey)}); err != nil {
		return fmt.Errorf("failed to write CA private key: %v", err)
	}

	return nil
}

// generateCertificate generates a new certificate signed by the CA
func (g *CertificateGenerator) generateCertificate(name, certFile, keyFile string, isServer bool) error {
	// Load CA certificate and private key
	caCertBytes, err := os.ReadFile(filepath.Join(g.outputDir, "ca.crt"))
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %v", err)
	}

	caCertBlock, _ := pem.Decode(caCertBytes)
	if caCertBlock == nil {
		return fmt.Errorf("failed to decode CA certificate")
	}

	caCert, err := x509.ParseCertificate(caCertBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %v", err)
	}

	caKeyBytes, err := os.ReadFile(filepath.Join(g.outputDir, "ca.key"))
	if err != nil {
		return fmt.Errorf("failed to read CA private key: %v", err)
	}

	caKeyBlock, _ := pem.Decode(caKeyBytes)
	if caKeyBlock == nil {
		return fmt.Errorf("failed to decode CA private key")
	}

	caKey, err := x509.ParsePKCS1PrivateKey(caKeyBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA private key: %v", err)
	}

	// Generate certificate private key
	certKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed to generate certificate private key: %v", err)
	}

	// Create certificate template
	certTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName: name,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(1, 0, 0), // 1 year validity
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	if isServer {
		certTemplate.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	}

	// Create certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, certTemplate, caCert, &certKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %v", err)
	}

	// Save certificate
	certOut, err := os.Create(filepath.Join(g.outputDir, certFile))
	if err != nil {
		return fmt.Errorf("failed to create certificate file: %v", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		return fmt.Errorf("failed to write certificate: %v", err)
	}

	// Save private key
	keyOut, err := os.OpenFile(filepath.Join(g.outputDir, keyFile), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %v", err)
	}
	defer keyOut.Close()

	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(certKey)}); err != nil {
		return fmt.Errorf("failed to write private key: %v", err)
	}

	return nil
}

// GenerateServerCert generates a new server certificate signed by the CA
func (g *CertificateGenerator) GenerateServerCert() error {
	return g.generateCertificate("sssonector-server", "server.crt", "server.key", true)
}

// GenerateClientCert generates a new client certificate signed by the CA
func (g *CertificateGenerator) GenerateClientCert() error {
	return g.generateCertificate("sssonector-client", "client.crt", "client.key", false)
}

// GenerateTestCerts generates temporary certificates for testing
func (g *CertificateGenerator) GenerateTestCerts() error {
	// Generate test CA private key
	caKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed to generate test CA private key: %v", err)
	}

	// Create test CA certificate template with 15-second validity
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName: "SSSonector Test CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(15 * time.Second),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create test CA certificate
	caCertBytes, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("failed to create test CA certificate: %v", err)
	}

	// Save test certificates with "test_" prefix
	testCACertFile := filepath.Join(g.outputDir, "test_ca.crt")
	certOut, err := os.Create(testCACertFile)
	if err != nil {
		return fmt.Errorf("failed to create test CA certificate file: %v", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: caCertBytes}); err != nil {
		return fmt.Errorf("failed to write test CA certificate: %v", err)
	}

	// Save test CA private key
	testCAKeyFile := filepath.Join(g.outputDir, "test_ca.key")
	keyOut, err := os.OpenFile(testCAKeyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create test CA private key file: %v", err)
	}
	defer keyOut.Close()

	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(caKey)}); err != nil {
		return fmt.Errorf("failed to write test CA private key: %v", err)
	}

	// Generate test server and client certificates
	if err := g.generateTestCertificate("test-server", "test_server.crt", "test_server.key", true, caTemplate, caKey); err != nil {
		return fmt.Errorf("failed to generate test server certificate: %v", err)
	}

	if err := g.generateTestCertificate("test-client", "test_client.crt", "test_client.key", false, caTemplate, caKey); err != nil {
		return fmt.Errorf("failed to generate test client certificate: %v", err)
	}

	return nil
}

// generateTestCertificate generates a test certificate with 15-second validity
func (g *CertificateGenerator) generateTestCertificate(name, certFile, keyFile string, isServer bool, caTemplate *x509.Certificate, caKey *rsa.PrivateKey) error {
	certKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed to generate test certificate private key: %v", err)
	}

	certTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			CommonName: name,
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(15 * time.Second),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
	}

	if isServer {
		certTemplate.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, certTemplate, caTemplate, &certKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("failed to create test certificate: %v", err)
	}

	certOut, err := os.Create(filepath.Join(g.outputDir, certFile))
	if err != nil {
		return fmt.Errorf("failed to create test certificate file: %v", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		return fmt.Errorf("failed to write test certificate: %v", err)
	}

	keyOut, err := os.OpenFile(filepath.Join(g.outputDir, keyFile), os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create test private key file: %v", err)
	}
	defer keyOut.Close()

	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(certKey)}); err != nil {
		return fmt.Errorf("failed to write test private key: %v", err)
	}

	return nil
}
