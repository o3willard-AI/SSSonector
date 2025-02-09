package cert

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"

	"go.uber.org/zap"
)

// Manager implements CertificateManager interface
type Manager struct {
	store  CertificateStore
	logger *zap.Logger
}

// NewManager creates a new certificate manager
func NewManager(store CertificateStore, logger *zap.Logger) *Manager {
	return &Manager{
		store:  store,
		logger: logger,
	}
}

// CreateCA creates a new Certificate Authority
func (m *Manager) CreateCA(req *CertificateRequest) (*Certificate, error) {
	// Generate key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Generate serial number
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:         req.Subject.CommonName,
			Organization:       req.Subject.Organization,
			OrganizationalUnit: req.Subject.OrganizationalUnit,
			Country:            req.Subject.Country,
			Province:           req.Subject.Province,
			Locality:           req.Subject.Locality,
		},
		NotBefore:             req.NotBefore,
		NotAfter:              req.NotAfter,
		KeyUsage:              x509.KeyUsage(req.KeyUsage) | x509.KeyUsageCertSign,
		ExtKeyUsage:           req.ExtKeyUsage,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
		DNSNames:              req.SANs,
	}

	// Create certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode private key
	privKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	// Create certificate object
	cert := &Certificate{
		Raw:          certBytes,
		Type:         CertTypeCA,
		Status:       CertStatusValid,
		PrivateKey:   privKeyBytes,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		SerialNumber: serialNumber.Text(16),
		SANs:         req.SANs,
		KeyUsage:     req.KeyUsage,
		ExtKeyUsage:  req.ExtKeyUsage,
		IssuerSerial: serialNumber.Text(16),
		Metadata:     req.Metadata,
	}

	// Parse X.509 certificate
	cert.X509, err = x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Store certificate
	if err := m.store.Store(cert); err != nil {
		return nil, fmt.Errorf("failed to store certificate: %w", err)
	}

	return cert, nil
}

// CreateIntermediate creates a new intermediate CA certificate
func (m *Manager) CreateIntermediate(req *CertificateRequest, parent *Certificate) (*Certificate, error) {
	// Parse parent private key
	parentKey, err := x509.ParseECPrivateKey(parent.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse parent private key: %w", err)
	}

	// Generate key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Generate serial number
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:         req.Subject.CommonName,
			Organization:       req.Subject.Organization,
			OrganizationalUnit: req.Subject.OrganizationalUnit,
			Country:            req.Subject.Country,
			Province:           req.Subject.Province,
			Locality:           req.Subject.Locality,
		},
		NotBefore:             req.NotBefore,
		NotAfter:              req.NotAfter,
		KeyUsage:              x509.KeyUsage(req.KeyUsage) | x509.KeyUsageCertSign,
		ExtKeyUsage:           req.ExtKeyUsage,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
		DNSNames:              req.SANs,
	}

	// Create certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, template, parent.X509, &privateKey.PublicKey, parentKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode private key
	privKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	// Create certificate object
	cert := &Certificate{
		Raw:          certBytes,
		Type:         CertTypeIntermediate,
		Status:       CertStatusValid,
		PrivateKey:   privKeyBytes,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		SerialNumber: serialNumber.Text(16),
		SANs:         req.SANs,
		KeyUsage:     req.KeyUsage,
		ExtKeyUsage:  req.ExtKeyUsage,
		IssuerSerial: parent.SerialNumber,
		Metadata:     req.Metadata,
	}

	// Parse X.509 certificate
	cert.X509, err = x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Store certificate
	if err := m.store.Store(cert); err != nil {
		return nil, fmt.Errorf("failed to store certificate: %w", err)
	}

	return cert, nil
}

// CreateServer creates a new server certificate
func (m *Manager) CreateServer(req *CertificateRequest, parent *Certificate) (*Certificate, error) {
	// Parse parent private key
	parentKey, err := x509.ParseECPrivateKey(parent.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse parent private key: %w", err)
	}

	// Generate key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Generate serial number
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:         req.Subject.CommonName,
			Organization:       req.Subject.Organization,
			OrganizationalUnit: req.Subject.OrganizationalUnit,
			Country:            req.Subject.Country,
			Province:           req.Subject.Province,
			Locality:           req.Subject.Locality,
		},
		NotBefore:             req.NotBefore,
		NotAfter:              req.NotAfter,
		KeyUsage:              x509.KeyUsage(req.KeyUsage),
		ExtKeyUsage:           append(req.ExtKeyUsage, x509.ExtKeyUsageServerAuth),
		BasicConstraintsValid: true,
		IsCA:                  false,
		DNSNames:              req.SANs,
	}

	// Create certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, template, parent.X509, &privateKey.PublicKey, parentKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode private key
	privKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	// Create certificate object
	cert := &Certificate{
		Raw:          certBytes,
		Type:         CertTypeServer,
		Status:       CertStatusValid,
		PrivateKey:   privKeyBytes,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		SerialNumber: serialNumber.Text(16),
		SANs:         req.SANs,
		KeyUsage:     req.KeyUsage,
		ExtKeyUsage:  append(req.ExtKeyUsage, x509.ExtKeyUsageServerAuth),
		IssuerSerial: parent.SerialNumber,
		Metadata:     req.Metadata,
	}

	// Parse X.509 certificate
	cert.X509, err = x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Store certificate
	if err := m.store.Store(cert); err != nil {
		return nil, fmt.Errorf("failed to store certificate: %w", err)
	}

	return cert, nil
}

// CreateClient creates a new client certificate
func (m *Manager) CreateClient(req *CertificateRequest, parent *Certificate) (*Certificate, error) {
	// Parse parent private key
	parentKey, err := x509.ParseECPrivateKey(parent.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse parent private key: %w", err)
	}

	// Generate key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Generate serial number
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:         req.Subject.CommonName,
			Organization:       req.Subject.Organization,
			OrganizationalUnit: req.Subject.OrganizationalUnit,
			Country:            req.Subject.Country,
			Province:           req.Subject.Province,
			Locality:           req.Subject.Locality,
		},
		NotBefore:             req.NotBefore,
		NotAfter:              req.NotAfter,
		KeyUsage:              x509.KeyUsage(req.KeyUsage),
		ExtKeyUsage:           append(req.ExtKeyUsage, x509.ExtKeyUsageClientAuth),
		BasicConstraintsValid: true,
		IsCA:                  false,
		DNSNames:              req.SANs,
	}

	// Create certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, template, parent.X509, &privateKey.PublicKey, parentKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode private key
	privKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	// Create certificate object
	cert := &Certificate{
		Raw:          certBytes,
		Type:         CertTypeClient,
		Status:       CertStatusValid,
		PrivateKey:   privKeyBytes,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		SerialNumber: serialNumber.Text(16),
		SANs:         req.SANs,
		KeyUsage:     req.KeyUsage,
		ExtKeyUsage:  append(req.ExtKeyUsage, x509.ExtKeyUsageClientAuth),
		IssuerSerial: parent.SerialNumber,
		Metadata:     req.Metadata,
	}

	// Parse X.509 certificate
	cert.X509, err = x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Store certificate
	if err := m.store.Store(cert); err != nil {
		return nil, fmt.Errorf("failed to store certificate: %w", err)
	}

	return cert, nil
}

// Revoke revokes a certificate
func (m *Manager) Revoke(serialNumber string, reason string) error {
	// Load certificate
	cert, err := m.store.Load(serialNumber)
	if err != nil {
		return fmt.Errorf("failed to load certificate: %w", err)
	}

	// Update certificate status
	cert.Status = CertStatusRevoked
	cert.RevokedAt = &time.Time{}
	*cert.RevokedAt = time.Now()
	cert.RevocationReason = reason
	cert.UpdatedAt = time.Now()

	// Store updated certificate
	if err := m.store.Update(cert); err != nil {
		return fmt.Errorf("failed to update certificate: %w", err)
	}

	return nil
}

// Verify verifies a certificate chain
func (m *Manager) Verify(cert *Certificate) error {
	// Create certificate pool
	roots := x509.NewCertPool()
	intermediates := x509.NewCertPool()

	// Load all certificates
	certs, err := m.store.List()
	if err != nil {
		return fmt.Errorf("failed to list certificates: %w", err)
	}

	// Add certificates to pools
	for _, c := range certs {
		switch c.Type {
		case CertTypeCA:
			roots.AddCert(c.X509)
		case CertTypeIntermediate:
			intermediates.AddCert(c.X509)
		}
	}

	// Verify certificate
	opts := x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		CurrentTime:   time.Now(),
		KeyUsages:     cert.ExtKeyUsage,
	}

	if _, err := cert.X509.Verify(opts); err != nil {
		return fmt.Errorf("certificate verification failed: %w", err)
	}

	return nil
}

// Export exports a certificate in PEM format
func (m *Manager) Export(cert *Certificate) (*pem.Block, error) {
	// Create certificate PEM block
	return &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}, nil
}

// Import imports a certificate from PEM format
func (m *Manager) Import(pemBlock *pem.Block) (*Certificate, error) {
	if pemBlock.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("invalid PEM block type: %s", pemBlock.Type)
	}

	// Parse certificate
	x509Cert, err := x509.ParseCertificate(pemBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Create certificate object
	cert := &Certificate{
		Raw:          pemBlock.Bytes,
		X509:         x509Cert,
		Type:         CertTypeServer, // Default to server type
		Status:       CertStatusValid,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		SerialNumber: x509Cert.SerialNumber.Text(16),
		SANs:         x509Cert.DNSNames,
		KeyUsage:     x509Cert.KeyUsage,
		ExtKeyUsage:  x509Cert.ExtKeyUsage,
		Metadata:     make(map[string]string),
	}

	// Determine certificate type
	if x509Cert.IsCA {
		if bytes.Equal(x509Cert.RawIssuer, x509Cert.RawSubject) {
			cert.Type = CertTypeCA
		} else {
			cert.Type = CertTypeIntermediate
		}
	}

	// Store certificate
	if err := m.store.Store(cert); err != nil {
		return nil, fmt.Errorf("failed to store certificate: %w", err)
	}

	return cert, nil
}

// GetCertificateStore returns the certificate store
func (m *Manager) GetCertificateStore() CertificateStore {
	return m.store
}
