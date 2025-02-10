package cert

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileStore implements CertificateStore using the filesystem
type FileStore struct {
	certPath string
	keyPath  string
	mu       sync.RWMutex
	client   *http.Client
}

// NewFileStore creates a new file-based certificate store
func NewFileStore(certPath, keyPath string) *FileStore {
	return &FileStore{
		certPath: certPath,
		keyPath:  keyPath,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// LoadCurrent implements Store.LoadCurrent
func (s *FileStore) LoadCurrent() (*CertPair, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Read certificate
	certPEM, err := os.ReadFile(s.certPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %v", err)
	}

	certBlock, _ := pem.Decode(certPEM)
	if certBlock == nil {
		return nil, fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(certBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %v", err)
	}

	// Read private key
	keyPEM, err := os.ReadFile(s.keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key file: %v", err)
	}

	keyBlock, _ := pem.Decode(keyPEM)
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode key PEM")
	}

	key, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %v", err)
	}

	return &CertPair{
		Cert: cert,
		Key:  key,
	}, nil
}

// Store implements Store.Store
func (s *FileStore) Store(cert *CertPair) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(s.certPath), 0755); err != nil {
		return fmt.Errorf("failed to create certificate directory: %v", err)
	}

	// Write certificate
	certPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Cert.Raw,
	})

	if err := os.WriteFile(s.certPath, certPEM, 0644); err != nil {
		return fmt.Errorf("failed to write certificate file: %v", err)
	}

	// Write private key
	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(cert.Key),
	})

	if err := os.WriteFile(s.keyPath, keyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write key file: %v", err)
	}

	return nil
}

// List implements Store.List
func (s *FileStore) List() ([]*CertPair, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// For now, just return the current certificate
	cert, err := s.LoadCurrent()
	if err != nil {
		return nil, err
	}

	return []*CertPair{cert}, nil
}

// Delete implements Store.Delete
func (s *FileStore) Delete(serialNumber string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// For now, just delete the current certificate files
	if err := os.Remove(s.certPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete certificate file: %v", err)
	}

	if err := os.Remove(s.keyPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete key file: %v", err)
	}

	return nil
}

// Load implements CertificateStore.Load
func (s *FileStore) Load(serialNumber string) (*Certificate, error) {
	cert, err := s.LoadCurrent()
	if err != nil {
		return nil, err
	}

	if cert.Cert.SerialNumber.String() != serialNumber {
		return nil, fmt.Errorf("certificate not found")
	}

	return cert.ToCertificate(), nil
}

// Update implements CertificateStore.Update
func (s *FileStore) Update(cert *Certificate) error {
	return s.Store(cert.ToCertPair())
}

// ListByType implements CertificateStore.ListByType
func (s *FileStore) ListByType(certType CertificateType) ([]*Certificate, error) {
	cert, err := s.LoadCurrent()
	if err != nil {
		return nil, err
	}

	c := cert.ToCertificate()
	if c.Type == certType {
		return []*Certificate{c}, nil
	}

	return nil, nil
}

// ListByStatus implements CertificateStore.ListByStatus
func (s *FileStore) ListByStatus(status CertificateStatus) ([]*Certificate, error) {
	cert, err := s.LoadCurrent()
	if err != nil {
		return nil, err
	}

	c := cert.ToCertificate()
	if c.Status == status {
		return []*Certificate{c}, nil
	}

	return nil, nil
}

// GetByType implements CertificateStore.GetByType
func (s *FileStore) GetByType(certType CertificateType) ([]*Certificate, error) {
	return s.ListByType(certType)
}

// GetByStatus implements CertificateStore.GetByStatus
func (s *FileStore) GetByStatus(status CertificateStatus) ([]*Certificate, error) {
	return s.ListByStatus(status)
}

// UpdateStatus implements CertificateStore.UpdateStatus
func (s *FileStore) UpdateStatus(serialNumber string, status CertificateStatus) error {
	cert, err := s.Load(serialNumber)
	if err != nil {
		return err
	}

	cert.Status = status
	if status == CertStatusRevoked {
		cert.RevokedAt = &time.Time{}
		*cert.RevokedAt = time.Now()
	}

	return s.Update(cert)
}

// GetChain implements CertificateStore.GetChain
func (s *FileStore) GetChain(cert *Certificate) ([]*Certificate, error) {
	// For now, just return the certificate itself
	return []*Certificate{cert}, nil
}

// ValidateCRL validates a certificate against a CRL
func (s *FileStore) ValidateCRL(cert *Certificate, crl *x509.RevocationList) error {
	if crl == nil {
		return fmt.Errorf("invalid CRL: nil")
	}

	// Check CRL validity period
	now := time.Now()
	if now.Before(crl.ThisUpdate) {
		return fmt.Errorf("CRL is not yet valid")
	}
	if now.After(crl.NextUpdate) {
		return fmt.Errorf("CRL has expired")
	}

	// Check if certificate is revoked
	for _, revokedCert := range crl.RevokedCertificates {
		if revokedCert.SerialNumber.Cmp(cert.X509.SerialNumber) == 0 {
			return fmt.Errorf("certificate is revoked")
		}
	}

	return nil
}

// ValidateOCSP validates a certificate using OCSP
func (s *FileStore) ValidateOCSP(cert *Certificate) error {
	if len(cert.X509.OCSPServer) == 0 {
		return fmt.Errorf("certificate has no OCSP servers")
	}

	// For each OCSP server
	for _, server := range cert.X509.OCSPServer {
		// Create OCSP request
		req, err := http.NewRequest("GET", server, nil)
		if err != nil {
			continue
		}

		// Send request
		resp, err := s.client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		// Read response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			continue
		}

		// Parse OCSP response
		if resp.StatusCode != http.StatusOK {
			continue
		}

		// Check response status
		if len(body) == 0 {
			continue
		}

		// For now, just check if we got a response
		// TODO: Implement full OCSP response parsing and validation
		return nil
	}

	return fmt.Errorf("failed to validate certificate status via OCSP")
}
