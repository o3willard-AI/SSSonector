package cert

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// FileStore implements CertificateStore using file-based storage
type FileStore struct {
	// Base directory for certificate storage
	baseDir string
	// Encryption key for sensitive data
	encryptionKey []byte
	// Logger instance
	logger *zap.Logger
	// Mutex for thread safety
	mu sync.RWMutex
}

// StoredCertificate represents the stored format of a certificate
type StoredCertificate struct {
	// Certificate data
	Raw []byte `json:"raw"`
	// Encrypted private key
	EncryptedPrivateKey []byte `json:"private_key,omitempty"`
	// Initialization vector for private key encryption
	PrivateKeyIV []byte `json:"private_key_iv,omitempty"`
	// Certificate type
	Type CertificateType `json:"type"`
	// Certificate status
	Status CertificateStatus `json:"status"`
	// Creation time
	CreatedAt time.Time `json:"created_at"`
	// Last update time
	UpdatedAt time.Time `json:"updated_at"`
	// Revocation time
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
	// Revocation reason
	RevocationReason string `json:"revocation_reason,omitempty"`
	// Serial number
	SerialNumber string `json:"serial_number"`
	// Subject alternative names
	SANs []string `json:"sans"`
	// Key usage
	KeyUsage int `json:"key_usage"`
	// Extended key usage
	ExtKeyUsage []int `json:"ext_key_usage"`
	// Issuer serial number
	IssuerSerial string `json:"issuer_serial"`
	// Metadata
	Metadata map[string]string `json:"metadata"`
}

// NewFileStore creates a new file-based certificate store
func NewFileStore(baseDir string, encryptionKey []byte, logger *zap.Logger) (*FileStore, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	// Hash encryption key to ensure proper length
	hashedKey := sha256.Sum256(encryptionKey)

	return &FileStore{
		baseDir:       baseDir,
		encryptionKey: hashedKey[:],
		logger:        logger,
	}, nil
}

// Store stores a certificate
func (s *FileStore) Store(cert *Certificate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Create stored certificate
	stored := &StoredCertificate{
		Raw:              cert.Raw,
		Type:             cert.Type,
		Status:           cert.Status,
		CreatedAt:        cert.CreatedAt,
		UpdatedAt:        cert.UpdatedAt,
		RevokedAt:        cert.RevokedAt,
		RevocationReason: cert.RevocationReason,
		SerialNumber:     cert.SerialNumber,
		SANs:             cert.SANs,
		KeyUsage:         int(cert.KeyUsage),
		IssuerSerial:     cert.IssuerSerial,
		Metadata:         cert.Metadata,
	}

	// Convert extended key usage
	for _, usage := range cert.ExtKeyUsage {
		stored.ExtKeyUsage = append(stored.ExtKeyUsage, int(usage))
	}

	// Encrypt private key if present
	if len(cert.PrivateKey) > 0 {
		block, err := aes.NewCipher(s.encryptionKey)
		if err != nil {
			return fmt.Errorf("failed to create cipher: %w", err)
		}

		// Generate random IV
		iv := make([]byte, aes.BlockSize)
		if _, err := io.ReadFull(rand.Reader, iv); err != nil {
			return fmt.Errorf("failed to generate IV: %w", err)
		}

		// Create GCM cipher
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return fmt.Errorf("failed to create GCM: %w", err)
		}

		// Encrypt private key
		stored.EncryptedPrivateKey = gcm.Seal(nil, iv, cert.PrivateKey, nil)
		stored.PrivateKeyIV = iv
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal certificate: %w", err)
	}

	// Write to file
	filename := filepath.Join(s.baseDir, fmt.Sprintf("%s.json", cert.SerialNumber))
	if err := os.WriteFile(filename, data, 0600); err != nil {
		return fmt.Errorf("failed to write certificate file: %w", err)
	}

	return nil
}

// Load loads a certificate by serial number
func (s *FileStore) Load(serialNumber string) (*Certificate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Read certificate file
	filename := filepath.Join(s.baseDir, fmt.Sprintf("%s.json", serialNumber))
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read certificate file: %w", err)
	}

	// Unmarshal stored certificate
	var stored StoredCertificate
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, fmt.Errorf("failed to unmarshal certificate: %w", err)
	}

	// Create certificate
	cert := &Certificate{
		Raw:              stored.Raw,
		Type:             stored.Type,
		Status:           stored.Status,
		CreatedAt:        stored.CreatedAt,
		UpdatedAt:        stored.UpdatedAt,
		RevokedAt:        stored.RevokedAt,
		RevocationReason: stored.RevocationReason,
		SerialNumber:     stored.SerialNumber,
		SANs:             stored.SANs,
		KeyUsage:         x509.KeyUsage(stored.KeyUsage),
		IssuerSerial:     stored.IssuerSerial,
		Metadata:         stored.Metadata,
	}

	// Convert extended key usage
	for _, usage := range stored.ExtKeyUsage {
		cert.ExtKeyUsage = append(cert.ExtKeyUsage, x509.ExtKeyUsage(usage))
	}

	// Parse X.509 certificate
	if len(cert.Raw) > 0 {
		x509Cert, err := x509.ParseCertificate(cert.Raw)
		if err != nil {
			return nil, fmt.Errorf("failed to parse X.509 certificate: %w", err)
		}
		cert.X509 = x509Cert
	}

	// Decrypt private key if present
	if len(stored.EncryptedPrivateKey) > 0 {
		block, err := aes.NewCipher(s.encryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create cipher: %w", err)
		}

		// Create GCM cipher
		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return nil, fmt.Errorf("failed to create GCM: %w", err)
		}

		// Decrypt private key
		privateKey, err := gcm.Open(nil, stored.PrivateKeyIV, stored.EncryptedPrivateKey, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt private key: %w", err)
		}
		cert.PrivateKey = privateKey
	}

	return cert, nil
}

// Delete deletes a certificate by serial number
func (s *FileStore) Delete(serialNumber string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	filename := filepath.Join(s.baseDir, fmt.Sprintf("%s.json", serialNumber))
	if err := os.Remove(filename); err != nil {
		return fmt.Errorf("failed to delete certificate file: %w", err)
	}

	return nil
}

// List lists all certificates
func (s *FileStore) List() ([]*Certificate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var certs []*Certificate

	// Read all certificate files
	files, err := os.ReadDir(s.baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read base directory: %w", err)
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		// Extract serial number from filename
		serialNumber := file.Name()[:len(file.Name())-5]

		// Load certificate
		cert, err := s.Load(serialNumber)
		if err != nil {
			s.logger.Error("Failed to load certificate",
				zap.String("serial_number", serialNumber),
				zap.Error(err))
			continue
		}

		certs = append(certs, cert)
	}

	return certs, nil
}

// ListByType lists certificates by type
func (s *FileStore) ListByType(certType CertificateType) ([]*Certificate, error) {
	certs, err := s.List()
	if err != nil {
		return nil, err
	}

	var filtered []*Certificate
	for _, cert := range certs {
		if cert.Type == certType {
			filtered = append(filtered, cert)
		}
	}

	return filtered, nil
}

// ListByStatus lists certificates by status
func (s *FileStore) ListByStatus(status CertificateStatus) ([]*Certificate, error) {
	certs, err := s.List()
	if err != nil {
		return nil, err
	}

	var filtered []*Certificate
	for _, cert := range certs {
		if cert.Status == status {
			filtered = append(filtered, cert)
		}
	}

	return filtered, nil
}

// Update updates a certificate's metadata
func (s *FileStore) Update(cert *Certificate) error {
	return s.Store(cert)
}
