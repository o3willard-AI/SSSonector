package validator

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
)

// ValidateCertificates checks if all required certificates exist and are valid
func ValidateCertificates(certDir string) error {
	// Required files
	files := []struct {
		name     string
		isKey    bool
		required bool
	}{
		{"ca.crt", false, true},
		{"ca.key", true, true},
		{"server.crt", false, true},
		{"server.key", true, true},
		{"client.crt", false, true},
		{"client.key", true, true},
	}

	// Check each file
	for _, file := range files {
		path := filepath.Join(certDir, file.name)

		// Check if file exists
		info, err := os.Stat(path)
		if err != nil {
			if os.IsNotExist(err) && file.required {
				return fmt.Errorf("required certificate file not found: %s", file.name)
			}
			return fmt.Errorf("error checking file %s: %v", file.name, err)
		}

		// Check permissions
		mode := info.Mode()
		if file.isKey {
			if mode&0077 != 0 {
				return fmt.Errorf("private key %s has incorrect permissions: %v (should be 0600)", file.name, mode)
			}
		} else {
			if mode&0044 == 0 {
				return fmt.Errorf("certificate %s is not readable: %v (should be 0644)", file.name, mode)
			}
		}

		// Validate certificate format
		if !file.isKey {
			if err := validateCertificateFormat(path); err != nil {
				return fmt.Errorf("invalid certificate %s: %v", file.name, err)
			}
		}
	}

	// Validate certificate chain
	if err := validateCertificateChain(certDir); err != nil {
		return fmt.Errorf("certificate chain validation failed: %v", err)
	}

	return nil
}

// validateCertificateFormat checks if a file contains a valid PEM-encoded certificate
func validateCertificateFormat(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read certificate: %v", err)
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return fmt.Errorf("failed to decode PEM block")
	}

	if block.Type != "CERTIFICATE" {
		return fmt.Errorf("invalid PEM block type: %s", block.Type)
	}

	if _, err := x509.ParseCertificate(block.Bytes); err != nil {
		return fmt.Errorf("failed to parse certificate: %v", err)
	}

	return nil
}

// validateCertificateChain verifies that certificates form a valid chain
func validateCertificateChain(certDir string) error {
	// Read CA certificate
	caCertPath := filepath.Join(certDir, "ca.crt")
	caCertData, err := os.ReadFile(caCertPath)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %v", err)
	}

	caBlock, _ := pem.Decode(caCertData)
	if caBlock == nil {
		return fmt.Errorf("failed to decode CA certificate PEM")
	}

	caCert, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %v", err)
	}

	// Create CA cert pool
	roots := x509.NewCertPool()
	roots.AddCert(caCert)

	// Validate server and client certificates
	for _, name := range []string{"server", "client"} {
		certPath := filepath.Join(certDir, name+".crt")
		certData, err := os.ReadFile(certPath)
		if err != nil {
			return fmt.Errorf("failed to read %s certificate: %v", name, err)
		}

		block, _ := pem.Decode(certData)
		if block == nil {
			return fmt.Errorf("failed to decode %s certificate PEM", name)
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return fmt.Errorf("failed to parse %s certificate: %v", name, err)
		}

		opts := x509.VerifyOptions{
			Roots: roots,
		}

		if _, err := cert.Verify(opts); err != nil {
			return fmt.Errorf("%s certificate verification failed: %v", name, err)
		}
	}

	return nil
}
