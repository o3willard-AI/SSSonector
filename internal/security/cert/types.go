package cert

import (
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"net"
	"time"
)

// CertificateType represents the type of certificate
type CertificateType int

const (
	CertTypeUnknown CertificateType = iota
	CertTypeCA
	CertTypeIntermediate
	CertTypeServer
	CertTypeClient
)

// CertificateStatus represents the status of a certificate
type CertificateStatus int

const (
	CertStatusUnknown CertificateStatus = iota
	CertStatusValid
	CertStatusExpired
	CertStatusRevoked
)

// Certificate represents a certificate with additional metadata
type Certificate struct {
	Raw              []byte
	X509             *x509.Certificate
	PrivateKey       *rsa.PrivateKey
	Type             CertificateType
	Status           CertificateStatus
	SerialNumber     string
	SANs             []string
	KeyUsage         x509.KeyUsage
	ExtKeyUsage      []x509.ExtKeyUsage
	IssuerSerial     string
	RevocationReason string
	RevokedAt        *time.Time
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Metadata         map[string]string
}

// CertPair represents a certificate and its private key
type CertPair struct {
	Cert *x509.Certificate
	Key  *rsa.PrivateKey
}

// ToCertPair converts a Certificate to a CertPair
func (c *Certificate) ToCertPair() *CertPair {
	return &CertPair{
		Cert: c.X509,
		Key:  c.PrivateKey,
	}
}

// ToCertificate converts a CertPair to a Certificate
func (p *CertPair) ToCertificate() *Certificate {
	cert := &Certificate{
		Raw:          p.Cert.Raw,
		X509:         p.Cert,
		PrivateKey:   p.Key,
		Type:         CertTypeUnknown,
		Status:       CertStatusValid,
		SerialNumber: p.Cert.SerialNumber.String(),
		SANs:         append(p.Cert.DNSNames, ipAddressesToStrings(p.Cert.IPAddresses)...),
		KeyUsage:     p.Cert.KeyUsage,
		ExtKeyUsage:  p.Cert.ExtKeyUsage,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Metadata:     make(map[string]string),
	}

	if p.Cert.IsCA {
		cert.Type = CertTypeCA
	}

	return cert
}

// Helper function to convert IP addresses to strings
func ipAddressesToStrings(ips []net.IP) []string {
	result := make([]string, len(ips))
	for i, ip := range ips {
		result[i] = ip.String()
	}
	return result
}

// CertificateRequest represents a request to create a new certificate
type CertificateRequest struct {
	Type        CertificateType
	Subject     pkix.Name
	CommonName  string
	DNSNames    []string
	IPAddresses []net.IP
	KeyUsage    x509.KeyUsage
	ExtKeyUsage []x509.ExtKeyUsage
	NotBefore   time.Time
	NotAfter    time.Time
	KeySize     int
	Metadata    map[string]string
}

// RotationConfig represents configuration for certificate rotation
type RotationConfig struct {
	RotationInterval time.Duration
	RenewalWindow    time.Duration
	GracePeriod      time.Duration
	KeySize          int
	OnRotation       func(old, new *x509.Certificate)
}

// DefaultRotationConfig returns the default rotation configuration
func DefaultRotationConfig() *RotationConfig {
	return &RotationConfig{
		RotationInterval: 1 * time.Hour,
		RenewalWindow:    30 * 24 * time.Hour, // 30 days
		GracePeriod:      24 * time.Hour,      // 1 day
		KeySize:          2048,
	}
}

// CertificateStore defines the interface for certificate storage
type CertificateStore interface {
	// Core operations
	LoadCurrent() (*CertPair, error)
	Store(cert *CertPair) error
	List() ([]*CertPair, error)
	Delete(serialNumber string) error

	// Advanced operations
	Load(serialNumber string) (*Certificate, error)
	Update(cert *Certificate) error
	ListByType(certType CertificateType) ([]*Certificate, error)
	ListByStatus(status CertificateStatus) ([]*Certificate, error)
	GetByType(certType CertificateType) ([]*Certificate, error)
	GetByStatus(status CertificateStatus) ([]*Certificate, error)
	UpdateStatus(serialNumber string, status CertificateStatus) error
	GetChain(cert *Certificate) ([]*Certificate, error)

	// Validation operations
	ValidateCRL(cert *Certificate, crl *x509.RevocationList) error
	ValidateOCSP(cert *Certificate) error
}

// CertificateManager defines the interface for certificate management
type CertificateManager interface {
	// Core operations
	GetCertificateStore() CertificateStore
	CreateCA(req *CertificateRequest) (*Certificate, error)
	CreateIntermediate(req *CertificateRequest, parent *Certificate) (*Certificate, error)
	CreateServer(req *CertificateRequest, parent *Certificate) (*Certificate, error)
	CreateClient(req *CertificateRequest, parent *Certificate) (*Certificate, error)
	Revoke(serialNumber string) error
	Validate(cert *Certificate) error
}
