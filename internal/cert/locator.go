package cert

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// CertificateLocator handles finding and validating certificates
type CertificateLocator struct {
	searchPaths []string
}

// NewCertificateLocator creates a new certificate locator
func NewCertificateLocator(defaultPath string, additionalPaths ...string) *CertificateLocator {
	paths := append([]string{defaultPath}, additionalPaths...)
	return &CertificateLocator{
		searchPaths: paths,
	}
}

// FindCertificates looks for certificates in the configured search paths
func (l *CertificateLocator) FindCertificates(certFile, keyFile string) (certPath, keyPath string, err error) {
	for _, path := range l.searchPaths {
		certPath = filepath.Join(path, certFile)
		keyPath = filepath.Join(path, keyFile)

		// Check if both files exist in this path
		if _, err := os.Stat(certPath); err == nil {
			if _, err := os.Stat(keyPath); err == nil {
				return certPath, keyPath, nil
			}
		}
	}

	return "", "", fmt.Errorf("certificates not found in any search path")
}

// ValidateCertificates checks the integrity and validity of certificates
func (l *CertificateLocator) ValidateCertificates(certPath, keyPath, caPath string) error {
	// Read and parse certificate
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		return fmt.Errorf("failed to read certificate file: %v", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %v", err)
	}

	// Read and parse private key
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key file: %v", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return fmt.Errorf("failed to decode private key PEM")
	}

	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %v", err)
	}

	// Verify key pair
	if err := verifyKeyPair(cert, key); err != nil {
		return fmt.Errorf("key pair verification failed: %v", err)
	}

	// If CA path is provided, verify certificate against CA
	if caPath != "" {
		if err := verifyAgainstCA(cert, caPath); err != nil {
			return fmt.Errorf("CA verification failed: %v", err)
		}
	}

	// Check certificate validity period
	if err := verifyValidityPeriod(cert); err != nil {
		return fmt.Errorf("certificate validity check failed: %v", err)
	}

	return nil
}

// verifyKeyPair checks if the private key matches the certificate
func verifyKeyPair(cert *x509.Certificate, key *rsa.PrivateKey) error {
	// Get the public key from the certificate
	certPubKey, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("certificate public key is not RSA")
	}

	// Compare the public key components
	if certPubKey.N.Cmp(key.N) != 0 {
		return fmt.Errorf("public key mismatch")
	}

	if certPubKey.E != key.E {
		return fmt.Errorf("public key exponent mismatch")
	}

	return nil
}

// verifyAgainstCA checks if the certificate is signed by the CA
func verifyAgainstCA(cert *x509.Certificate, caPath string) error {
	// Read and parse CA certificate
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

	// Create a certificate pool and add the CA certificate
	roots := x509.NewCertPool()
	roots.AddCert(caCert)

	// Verify the certificate against the CA
	opts := x509.VerifyOptions{
		Roots: roots,
	}

	if _, err := cert.Verify(opts); err != nil {
		return fmt.Errorf("certificate verification failed: %v", err)
	}

	return nil
}

// verifyValidityPeriod checks if the certificate is currently valid
func verifyValidityPeriod(cert *x509.Certificate) error {
	now := time.Now().UTC()
	if now.Before(cert.NotBefore) {
		return fmt.Errorf("certificate is not yet valid")
	}
	if now.After(cert.NotAfter) {
		return fmt.Errorf("certificate has expired")
	}
	return nil
}

// MoveCertificates copies certificates to the target directory if needed
func (l *CertificateLocator) MoveCertificates(sourceCertPath, sourceKeyPath, targetDir string) error {
	// Create target directory if it doesn't exist
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %v", err)
	}

	// Copy certificate
	targetCertPath := filepath.Join(targetDir, filepath.Base(sourceCertPath))
	if err := copyFile(sourceCertPath, targetCertPath); err != nil {
		return fmt.Errorf("failed to copy certificate: %v", err)
	}

	// Copy private key and set restrictive permissions
	targetKeyPath := filepath.Join(targetDir, filepath.Base(sourceKeyPath))
	if err := copyFile(sourceKeyPath, targetKeyPath); err != nil {
		return fmt.Errorf("failed to copy private key: %v", err)
	}
	if err := os.Chmod(targetKeyPath, 0600); err != nil {
		return fmt.Errorf("failed to set private key permissions: %v", err)
	}

	return nil
}

// copyFile copies a file from source to destination
func copyFile(source, dest string) error {
	input, err := os.ReadFile(source)
	if err != nil {
		return fmt.Errorf("failed to read source file: %v", err)
	}

	if err := os.WriteFile(dest, input, 0644); err != nil {
		return fmt.Errorf("failed to write destination file: %v", err)
	}

	return nil
}
