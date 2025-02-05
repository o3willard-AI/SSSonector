package generator

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

const (
	defaultKeySize      = 2048
	defaultCADuration   = 365 * 24 * time.Hour // 1 year
	defaultCertDuration = 180 * 24 * time.Hour // 6 months
	tempCertDuration    = 15 * time.Second     // 15 seconds for test mode
)

// GenerateCertificates generates a complete set of certificates in the specified directory
func GenerateCertificates(certDir string) error {
	// Generate CA key pair
	caKey, err := rsa.GenerateKey(rand.Reader, defaultKeySize)
	if err != nil {
		return fmt.Errorf("failed to generate CA private key: %v", err)
	}

	// Create CA certificate
	ca := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"SSSonector CA"},
			CommonName:   "SSSonector Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(defaultCADuration),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	// Create CA certificate
	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("failed to create CA certificate: %v", err)
	}

	// Save CA certificate and private key
	if err := saveCertAndKey(certDir, "ca", caBytes, caKey); err != nil {
		return fmt.Errorf("failed to save CA certificate and key: %v", err)
	}

	// Generate server certificates
	if err := generateEndEntityCert(certDir, "server", ca, caKey, defaultCertDuration); err != nil {
		return fmt.Errorf("failed to generate server certificate: %v", err)
	}

	// Generate client certificates
	if err := generateEndEntityCert(certDir, "client", ca, caKey, defaultCertDuration); err != nil {
		return fmt.Errorf("failed to generate client certificate: %v", err)
	}

	return nil
}

// GenerateTemporaryCertificates generates short-lived certificates for testing
func GenerateTemporaryCertificates(certDir string) error {
	// Generate CA and certificates with short duration
	caKey, err := rsa.GenerateKey(rand.Reader, defaultKeySize)
	if err != nil {
		return fmt.Errorf("failed to generate temporary CA private key: %v", err)
	}

	ca := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"SSSonector Temporary CA"},
			CommonName:   "SSSonector Temporary Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(tempCertDuration),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caBytes, err := x509.CreateCertificate(rand.Reader, ca, ca, &caKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("failed to create temporary CA certificate: %v", err)
	}

	if err := saveCertAndKey(certDir, "ca", caBytes, caKey); err != nil {
		return fmt.Errorf("failed to save temporary CA certificate and key: %v", err)
	}

	// Generate temporary server and client certificates
	if err := generateEndEntityCert(certDir, "server", ca, caKey, tempCertDuration); err != nil {
		return fmt.Errorf("failed to generate temporary server certificate: %v", err)
	}

	if err := generateEndEntityCert(certDir, "client", ca, caKey, tempCertDuration); err != nil {
		return fmt.Errorf("failed to generate temporary client certificate: %v", err)
	}

	return nil
}

// generateEndEntityCert generates a certificate for either server or client
func generateEndEntityCert(certDir, name string, ca *x509.Certificate, caKey *rsa.PrivateKey, duration time.Duration) error {
	key, err := rsa.GenerateKey(rand.Reader, defaultKeySize)
	if err != nil {
		return fmt.Errorf("failed to generate %s private key: %v", name, err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			Organization: []string{"SSSonector"},
			CommonName:   fmt.Sprintf("SSSonector %s", name),
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(duration),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, ca, &key.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("failed to create %s certificate: %v", name, err)
	}

	return saveCertAndKey(certDir, name, certBytes, key)
}

// saveCertAndKey saves a certificate and private key to disk
func saveCertAndKey(certDir, name string, certBytes []byte, key *rsa.PrivateKey) error {
	// Save certificate
	certFile := filepath.Join(certDir, name+".crt")
	certOut, err := os.OpenFile(certFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open %s for writing: %v", certFile, err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: certBytes}); err != nil {
		return fmt.Errorf("failed to write certificate to %s: %v", certFile, err)
	}

	// Save private key
	keyFile := filepath.Join(certDir, name+".key")
	keyOut, err := os.OpenFile(keyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open %s for writing: %v", keyFile, err)
	}
	defer keyOut.Close()

	privBytes := x509.MarshalPKCS1PrivateKey(key)
	if err := pem.Encode(keyOut, &pem.Block{Type: "RSA PRIVATE KEY", Bytes: privBytes}); err != nil {
		return fmt.Errorf("failed to write private key to %s: %v", keyFile, err)
	}

	return nil
}
