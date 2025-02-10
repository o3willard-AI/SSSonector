package cert

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
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

// GetCertificateStore returns the certificate store
func (m *Manager) GetCertificateStore() CertificateStore {
	return m.store
}

// CreateCA creates a new CA certificate
func (m *Manager) CreateCA(req *CertificateRequest) (*Certificate, error) {
	// Generate key pair
	key, err := rsa.GenerateKey(rand.Reader, req.KeySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %v", err)
	}

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber:          generateSerialNumber(),
		Subject:               req.Subject,
		NotBefore:             req.NotBefore,
		NotAfter:              req.NotAfter,
		KeyUsage:              req.KeyUsage | x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		ExtKeyUsage:           req.ExtKeyUsage,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}

	// Self-sign CA certificate
	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, fmt.Errorf("failed to create CA certificate: %v", err)
	}

	// Parse certificate
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse CA certificate: %v", err)
	}

	// Create Certificate object
	certificate := &Certificate{
		Raw:          certBytes,
		X509:         cert,
		PrivateKey:   key,
		Type:         CertTypeCA,
		Status:       CertStatusValid,
		SerialNumber: cert.SerialNumber.String(),
		KeyUsage:     req.KeyUsage,
		ExtKeyUsage:  req.ExtKeyUsage,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Metadata:     req.Metadata,
	}

	// Store certificate
	if err := m.store.Store(certificate.ToCertPair()); err != nil {
		return nil, fmt.Errorf("failed to store CA certificate: %v", err)
	}

	return certificate, nil
}

// CreateIntermediate creates a new intermediate certificate
func (m *Manager) CreateIntermediate(req *CertificateRequest, parent *Certificate) (*Certificate, error) {
	// Generate key pair
	key, err := rsa.GenerateKey(rand.Reader, req.KeySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %v", err)
	}

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber:          generateSerialNumber(),
		Subject:               req.Subject,
		NotBefore:             req.NotBefore,
		NotAfter:              req.NotAfter,
		KeyUsage:              req.KeyUsage | x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		ExtKeyUsage:           req.ExtKeyUsage,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            0,
		MaxPathLenZero:        true,
	}

	// Sign certificate with parent
	certBytes, err := x509.CreateCertificate(rand.Reader, template, parent.X509, &key.PublicKey, parent.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create intermediate certificate: %v", err)
	}

	// Parse certificate
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse intermediate certificate: %v", err)
	}

	// Create Certificate object
	certificate := &Certificate{
		Raw:          certBytes,
		X509:         cert,
		PrivateKey:   key,
		Type:         CertTypeIntermediate,
		Status:       CertStatusValid,
		SerialNumber: cert.SerialNumber.String(),
		KeyUsage:     req.KeyUsage,
		ExtKeyUsage:  req.ExtKeyUsage,
		IssuerSerial: parent.SerialNumber,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Metadata:     req.Metadata,
	}

	// Store certificate
	if err := m.store.Store(certificate.ToCertPair()); err != nil {
		return nil, fmt.Errorf("failed to store intermediate certificate: %v", err)
	}

	return certificate, nil
}

// CreateServer creates a new server certificate
func (m *Manager) CreateServer(req *CertificateRequest, parent *Certificate) (*Certificate, error) {
	// Generate key pair
	key, err := rsa.GenerateKey(rand.Reader, req.KeySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %v", err)
	}

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber:          generateSerialNumber(),
		Subject:               req.Subject,
		NotBefore:             req.NotBefore,
		NotAfter:              req.NotAfter,
		KeyUsage:              req.KeyUsage | x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           append(req.ExtKeyUsage, x509.ExtKeyUsageServerAuth),
		BasicConstraintsValid: true,
		IsCA:                  false,
		DNSNames:              req.DNSNames,
		IPAddresses:           req.IPAddresses,
	}

	// Sign certificate with parent
	certBytes, err := x509.CreateCertificate(rand.Reader, template, parent.X509, &key.PublicKey, parent.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create server certificate: %v", err)
	}

	// Parse certificate
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse server certificate: %v", err)
	}

	// Create Certificate object
	certificate := &Certificate{
		Raw:          certBytes,
		X509:         cert,
		PrivateKey:   key,
		Type:         CertTypeServer,
		Status:       CertStatusValid,
		SerialNumber: cert.SerialNumber.String(),
		SANs:         append(req.DNSNames, ipAddressesToStrings(req.IPAddresses)...),
		KeyUsage:     req.KeyUsage,
		ExtKeyUsage:  req.ExtKeyUsage,
		IssuerSerial: parent.SerialNumber,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Metadata:     req.Metadata,
	}

	// Store certificate
	if err := m.store.Store(certificate.ToCertPair()); err != nil {
		return nil, fmt.Errorf("failed to store server certificate: %v", err)
	}

	return certificate, nil
}

// CreateClient creates a new client certificate
func (m *Manager) CreateClient(req *CertificateRequest, parent *Certificate) (*Certificate, error) {
	// Generate key pair
	key, err := rsa.GenerateKey(rand.Reader, req.KeySize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %v", err)
	}

	// Create certificate template
	template := &x509.Certificate{
		SerialNumber:          generateSerialNumber(),
		Subject:               req.Subject,
		NotBefore:             req.NotBefore,
		NotAfter:              req.NotAfter,
		KeyUsage:              req.KeyUsage | x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           append(req.ExtKeyUsage, x509.ExtKeyUsageClientAuth),
		BasicConstraintsValid: true,
		IsCA:                  false,
		DNSNames:              req.DNSNames,
		IPAddresses:           req.IPAddresses,
	}

	// Sign certificate with parent
	certBytes, err := x509.CreateCertificate(rand.Reader, template, parent.X509, &key.PublicKey, parent.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create client certificate: %v", err)
	}

	// Parse certificate
	cert, err := x509.ParseCertificate(certBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse client certificate: %v", err)
	}

	// Create Certificate object
	certificate := &Certificate{
		Raw:          certBytes,
		X509:         cert,
		PrivateKey:   key,
		Type:         CertTypeClient,
		Status:       CertStatusValid,
		SerialNumber: cert.SerialNumber.String(),
		SANs:         append(req.DNSNames, ipAddressesToStrings(req.IPAddresses)...),
		KeyUsage:     req.KeyUsage,
		ExtKeyUsage:  req.ExtKeyUsage,
		IssuerSerial: parent.SerialNumber,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Metadata:     req.Metadata,
	}

	// Store certificate
	if err := m.store.Store(certificate.ToCertPair()); err != nil {
		return nil, fmt.Errorf("failed to store client certificate: %v", err)
	}

	return certificate, nil
}

// Revoke revokes a certificate
func (m *Manager) Revoke(serialNumber string) error {
	return m.store.UpdateStatus(serialNumber, CertStatusRevoked)
}

// Validate validates a certificate
func (m *Manager) Validate(cert *Certificate) error {
	// Check certificate status
	if cert.Status == CertStatusRevoked {
		return fmt.Errorf("certificate is revoked")
	}

	// Check expiration
	if time.Now().After(cert.X509.NotAfter) {
		return fmt.Errorf("certificate has expired")
	}

	// Check certificate chain
	chain, err := m.store.GetChain(cert)
	if err != nil {
		return fmt.Errorf("failed to get certificate chain: %v", err)
	}

	// Build verification pool
	roots := x509.NewCertPool()
	intermediates := x509.NewCertPool()

	for _, c := range chain {
		if c.Type == CertTypeCA {
			roots.AddCert(c.X509)
		} else if c.Type == CertTypeIntermediate {
			intermediates.AddCert(c.X509)
		}
	}

	// Verify certificate
	opts := x509.VerifyOptions{
		Roots:         roots,
		Intermediates: intermediates,
		KeyUsages:     cert.ExtKeyUsage,
	}

	if _, err := cert.X509.Verify(opts); err != nil {
		return fmt.Errorf("certificate verification failed: %v", err)
	}

	return nil
}

// Helper function to generate a random serial number
func generateSerialNumber() *big.Int {
	serialNumberLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serialNumber, _ := rand.Int(rand.Reader, serialNumberLimit)
	return serialNumber
}
