package cert

import (
	"crypto/x509"
	"encoding/pem"
	"time"
)

// CertificateType represents the type of certificate
type CertificateType int

const (
	// CertTypeServer represents a server certificate
	CertTypeServer CertificateType = iota
	// CertTypeClient represents a client certificate
	CertTypeClient
	// CertTypeCA represents a Certificate Authority certificate
	CertTypeCA
	// CertTypeIntermediate represents an intermediate CA certificate
	CertTypeIntermediate
)

// CertificateStatus represents the status of a certificate
type CertificateStatus int

const (
	// CertStatusValid indicates the certificate is valid
	CertStatusValid CertificateStatus = iota
	// CertStatusExpired indicates the certificate has expired
	CertStatusExpired
	// CertStatusRevoked indicates the certificate has been revoked
	CertStatusRevoked
	// CertStatusNotYetValid indicates the certificate is not yet valid
	CertStatusNotYetValid
)

// Certificate represents a X.509 certificate with metadata
type Certificate struct {
	// Raw certificate data
	Raw []byte
	// Parsed certificate
	X509 *x509.Certificate
	// Certificate type
	Type CertificateType
	// Certificate status
	Status CertificateStatus
	// Private key (if available)
	PrivateKey []byte
	// Creation time
	CreatedAt time.Time
	// Last update time
	UpdatedAt time.Time
	// Revocation time (if revoked)
	RevokedAt *time.Time
	// Revocation reason (if revoked)
	RevocationReason string
	// Serial number as string
	SerialNumber string
	// Subject alternative names
	SANs []string
	// Key usage
	KeyUsage x509.KeyUsage
	// Extended key usage
	ExtKeyUsage []x509.ExtKeyUsage
	// Issuer serial number
	IssuerSerial string
	// Metadata
	Metadata map[string]string
}

// CertificateRequest represents a certificate signing request
type CertificateRequest struct {
	// CSR data
	Raw []byte
	// Certificate type
	Type CertificateType
	// Subject information
	Subject struct {
		CommonName         string
		Organization       []string
		OrganizationalUnit []string
		Country            []string
		Province           []string
		Locality           []string
	}
	// Subject alternative names
	SANs []string
	// Key usage
	KeyUsage x509.KeyUsage
	// Extended key usage
	ExtKeyUsage []x509.ExtKeyUsage
	// Validity period
	NotBefore time.Time
	NotAfter  time.Time
	// Key size in bits
	KeySize int
	// Metadata
	Metadata map[string]string
}

// CertificateStore defines the interface for certificate storage
type CertificateStore interface {
	// Store stores a certificate
	Store(cert *Certificate) error
	// Load loads a certificate by serial number
	Load(serialNumber string) (*Certificate, error)
	// Delete deletes a certificate by serial number
	Delete(serialNumber string) error
	// List lists all certificates
	List() ([]*Certificate, error)
	// ListByType lists certificates by type
	ListByType(certType CertificateType) ([]*Certificate, error)
	// ListByStatus lists certificates by status
	ListByStatus(status CertificateStatus) ([]*Certificate, error)
	// Update updates a certificate's metadata
	Update(cert *Certificate) error
}

// CertificateManager defines the interface for certificate management
type CertificateManager interface {
	// CreateCA creates a new Certificate Authority
	CreateCA(req *CertificateRequest) (*Certificate, error)
	// CreateIntermediate creates a new intermediate CA certificate
	CreateIntermediate(req *CertificateRequest, parent *Certificate) (*Certificate, error)
	// CreateServer creates a new server certificate
	CreateServer(req *CertificateRequest, parent *Certificate) (*Certificate, error)
	// CreateClient creates a new client certificate
	CreateClient(req *CertificateRequest, parent *Certificate) (*Certificate, error)
	// Revoke revokes a certificate
	Revoke(serialNumber string, reason string) error
	// Verify verifies a certificate chain
	Verify(cert *Certificate) error
	// Export exports a certificate in PEM format
	Export(cert *Certificate) (*pem.Block, error)
	// Import imports a certificate from PEM format
	Import(pemBlock *pem.Block) (*Certificate, error)
	// GetCertificateStore returns the certificate store
	GetCertificateStore() CertificateStore
}

// CertificateValidator defines the interface for certificate validation
type CertificateValidator interface {
	// ValidateCertificate validates a certificate
	ValidateCertificate(cert *Certificate) error
	// ValidateChain validates a certificate chain
	ValidateChain(cert *Certificate, intermediates []*Certificate, root *Certificate) error
	// ValidateCRL validates a certificate against a CRL
	ValidateCRL(cert *Certificate, crl *x509.RevocationList) error
	// ValidateOCSP validates a certificate using OCSP
	ValidateOCSP(cert *Certificate) error
}

// CertificateRotator defines the interface for certificate rotation
type CertificateRotator interface {
	// ShouldRotate determines if a certificate should be rotated
	ShouldRotate(cert *Certificate) bool
	// Rotate rotates a certificate
	Rotate(cert *Certificate) (*Certificate, error)
	// RotateAll rotates all certificates of a given type
	RotateAll(certType CertificateType) error
}
