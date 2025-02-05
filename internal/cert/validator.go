package cert

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"
)

// CertificateValidator handles certificate validation and testing
type CertificateValidator struct {
	isTestMode bool
}

// NewCertificateValidator creates a new certificate validator
func NewCertificateValidator(isTestMode bool) *CertificateValidator {
	return &CertificateValidator{
		isTestMode: isTestMode,
	}
}

// ValidateCertificateFiles performs comprehensive validation of certificate files
func (v *CertificateValidator) ValidateCertificateFiles(certPath, keyPath, caPath string) error {
	// Read and parse certificate
	cert, key, err := v.loadCertificateAndKey(certPath, keyPath)
	if err != nil {
		return fmt.Errorf("failed to load certificate and key: %v", err)
	}

	// Verify the certificate chain if CA path is provided
	if caPath != "" {
		if err := v.verifyCertificateChain(cert, caPath); err != nil {
			return fmt.Errorf("certificate chain verification failed: %v", err)
		}
	}

	// Verify key pair
	if err := v.verifyKeyPair(cert, key); err != nil {
		return fmt.Errorf("key pair verification failed: %v", err)
	}

	// Check validity period
	if err := v.verifyValidityPeriod(cert); err != nil {
		return fmt.Errorf("validity period verification failed: %v", err)
	}

	// Additional checks for test mode certificates
	if v.isTestMode {
		if err := v.verifyTestCertificate(cert); err != nil {
			return fmt.Errorf("test certificate verification failed: %v", err)
		}
	}

	return nil
}

// loadCertificateAndKey loads and parses a certificate and private key from files
func (v *CertificateValidator) loadCertificateAndKey(certPath, keyPath string) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Read certificate
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read certificate file: %v", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, nil, fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse certificate: %v", err)
	}

	// Read private key
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read private key file: %v", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, nil, fmt.Errorf("failed to decode private key PEM")
	}

	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	return cert, key, nil
}

// verifyCertificateChain verifies the certificate against a CA
func (v *CertificateValidator) verifyCertificateChain(cert *x509.Certificate, caPath string) error {
	// Read CA certificate
	caPEM, err := os.ReadFile(caPath)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %v", err)
	}

	caBlock, _ := pem.Decode(caPEM)
	if caBlock == nil {
		return fmt.Errorf("failed to decode CA certificate PEM")
	}

	caCert, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %v", err)
	}

	// Create certificate pool and add CA
	roots := x509.NewCertPool()
	roots.AddCert(caCert)

	// Verify the certificate
	opts := x509.VerifyOptions{
		Roots: roots,
	}

	if _, err := cert.Verify(opts); err != nil {
		return fmt.Errorf("certificate verification failed: %v", err)
	}

	return nil
}

// verifyKeyPair verifies that a certificate and private key match
func (v *CertificateValidator) verifyKeyPair(cert *x509.Certificate, key *rsa.PrivateKey) error {
	certPubKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("certificate public key is not RSA")
	}

	if certPubKey.N.Cmp(key.N) != 0 {
		return fmt.Errorf("public key modulus mismatch")
	}

	if certPubKey.E != key.E {
		return fmt.Errorf("public key exponent mismatch")
	}

	return nil
}

// verifyValidityPeriod checks if a certificate is currently valid
func (v *CertificateValidator) verifyValidityPeriod(cert *x509.Certificate) error {
	now := time.Now().UTC()

	if now.Before(cert.NotBefore) {
		return fmt.Errorf("certificate is not yet valid (becomes valid at %v)", cert.NotBefore)
	}

	if now.After(cert.NotAfter) {
		return fmt.Errorf("certificate has expired (expired at %v)", cert.NotAfter)
	}

	// For test certificates, ensure they have a short validity period
	if v.isTestMode {
		validityPeriod := cert.NotAfter.Sub(cert.NotBefore)
		if validityPeriod > 15*time.Second {
			return fmt.Errorf("test certificate validity period (%v) exceeds maximum allowed (15s)", validityPeriod)
		}
	}

	return nil
}

// verifyTestCertificate performs additional validation for test certificates
func (v *CertificateValidator) verifyTestCertificate(cert *x509.Certificate) error {
	// Verify the certificate is marked for testing in the CommonName
	if cert.Subject.CommonName != "test-server" && cert.Subject.CommonName != "test-client" {
		return fmt.Errorf("invalid test certificate CommonName: %s", cert.Subject.CommonName)
	}

	// Verify short validity period
	validityPeriod := cert.NotAfter.Sub(cert.NotBefore)
	if validityPeriod > 15*time.Second {
		return fmt.Errorf("test certificate validity period (%v) exceeds maximum allowed (15s)", validityPeriod)
	}

	// Additional test-specific checks can be added here
	return nil
}

// IsTestCertificate checks if a certificate is a test certificate
func (v *CertificateValidator) IsTestCertificate(certPath string) (bool, error) {
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return false, fmt.Errorf("failed to read certificate file: %v", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return false, fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return false, fmt.Errorf("failed to parse certificate: %v", err)
	}

	// Check if it's a test certificate based on CommonName and validity period
	isTest := cert.Subject.CommonName == "test-server" || cert.Subject.CommonName == "test-client"
	if isTest {
		validityPeriod := cert.NotAfter.Sub(cert.NotBefore)
		isTest = validityPeriod <= 15*time.Second
	}

	return isTest, nil
}
