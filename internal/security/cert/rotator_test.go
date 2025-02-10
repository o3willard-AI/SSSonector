package cert

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

type MockCertificateStore struct {
	mock.Mock
}

func (m *MockCertificateStore) LoadCurrent() (*CertPair, error) {
	args := m.Called()
	return args.Get(0).(*CertPair), args.Error(1)
}

func (m *MockCertificateStore) Store(cert *CertPair) error {
	args := m.Called(cert)
	return args.Error(0)
}

func (m *MockCertificateStore) List() ([]*CertPair, error) {
	args := m.Called()
	return args.Get(0).([]*CertPair), args.Error(1)
}

func (m *MockCertificateStore) Delete(serialNumber string) error {
	args := m.Called(serialNumber)
	return args.Error(0)
}

func (m *MockCertificateStore) Load(serialNumber string) (*Certificate, error) {
	args := m.Called(serialNumber)
	return args.Get(0).(*Certificate), args.Error(1)
}

func (m *MockCertificateStore) Update(cert *Certificate) error {
	args := m.Called(cert)
	return args.Error(0)
}

func (m *MockCertificateStore) ListByType(certType CertificateType) ([]*Certificate, error) {
	args := m.Called(certType)
	return args.Get(0).([]*Certificate), args.Error(1)
}

func (m *MockCertificateStore) ListByStatus(status CertificateStatus) ([]*Certificate, error) {
	args := m.Called(status)
	return args.Get(0).([]*Certificate), args.Error(1)
}

func (m *MockCertificateStore) GetByType(certType CertificateType) ([]*Certificate, error) {
	args := m.Called(certType)
	return args.Get(0).([]*Certificate), args.Error(1)
}

func (m *MockCertificateStore) GetByStatus(status CertificateStatus) ([]*Certificate, error) {
	args := m.Called(status)
	return args.Get(0).([]*Certificate), args.Error(1)
}

func (m *MockCertificateStore) UpdateStatus(serialNumber string, status CertificateStatus) error {
	args := m.Called(serialNumber, status)
	return args.Error(0)
}

func (m *MockCertificateStore) GetChain(cert *Certificate) ([]*Certificate, error) {
	args := m.Called(cert)
	return args.Get(0).([]*Certificate), args.Error(1)
}

func (m *MockCertificateStore) ValidateCRL(cert *Certificate, crl *x509.RevocationList) error {
	args := m.Called(cert, crl)
	return args.Error(0)
}

func (m *MockCertificateStore) ValidateOCSP(cert *Certificate) error {
	args := m.Called(cert)
	return args.Error(0)
}

type MockCertificateManager struct {
	mock.Mock
}

func (m *MockCertificateManager) GetCertificateStore() CertificateStore {
	args := m.Called()
	return args.Get(0).(CertificateStore)
}

func (m *MockCertificateManager) CreateCA(req *CertificateRequest) (*Certificate, error) {
	args := m.Called(req)
	return args.Get(0).(*Certificate), args.Error(1)
}

func (m *MockCertificateManager) CreateIntermediate(req *CertificateRequest, parent *Certificate) (*Certificate, error) {
	args := m.Called(req, parent)
	return args.Get(0).(*Certificate), args.Error(1)
}

func (m *MockCertificateManager) CreateServer(req *CertificateRequest, parent *Certificate) (*Certificate, error) {
	args := m.Called(req, parent)
	return args.Get(0).(*Certificate), args.Error(1)
}

func (m *MockCertificateManager) CreateClient(req *CertificateRequest, parent *Certificate) (*Certificate, error) {
	args := m.Called(req, parent)
	return args.Get(0).(*Certificate), args.Error(1)
}

func (m *MockCertificateManager) Revoke(serialNumber string) error {
	args := m.Called(serialNumber)
	return args.Error(0)
}

func (m *MockCertificateManager) Validate(cert *Certificate) error {
	args := m.Called(cert)
	return args.Error(0)
}

func createTestCertificate(t *testing.T, notBefore, notAfter time.Time) (*x509.Certificate, *rsa.PrivateKey) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	assert.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			CommonName: "test.example.com",
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"test.example.com"},
		CRLDistributionPoints: []string{"http://crl.example.com"},
		OCSPServer:            []string{"http://ocsp.example.com"},
	}

	certBytes, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	assert.NoError(t, err)

	cert, err := x509.ParseCertificate(certBytes)
	assert.NoError(t, err)

	return cert, key
}

func TestCertificateRotator_Start(t *testing.T) {
	store := &MockCertificateStore{}
	manager := &MockCertificateManager{}
	logger, _ := zap.NewDevelopment()

	now := time.Now()
	cert, key := createTestCertificate(t, now, now.Add(24*time.Hour))
	certPair := &CertPair{
		Cert: cert,
		Key:  key,
	}

	// Set up expectations for initial certificate loading and validation
	store.On("LoadCurrent").Return(certPair, nil)
	manager.On("GetCertificateStore").Return(store)
	manager.On("Validate", mock.MatchedBy(func(c *Certificate) bool {
		return c.X509.SerialNumber.String() == cert.SerialNumber.String() && c.Type == CertTypeServer
	})).Return(nil)
	store.On("ValidateCRL", mock.MatchedBy(func(c *Certificate) bool {
		return c.X509.SerialNumber.String() == cert.SerialNumber.String() && c.Type == CertTypeServer
	}), mock.Anything).Return(nil)
	store.On("ValidateOCSP", mock.MatchedBy(func(c *Certificate) bool {
		return c.X509.SerialNumber.String() == cert.SerialNumber.String() && c.Type == CertTypeServer
	})).Return(nil)

	config := &RotationConfig{
		RotationInterval: time.Hour,
		RenewalWindow:    time.Hour,
		GracePeriod:      time.Second,
		KeySize:          2048,
	}

	rotator := NewCertificateRotator(config, manager, logger)
	err := rotator.Start(context.Background())
	assert.NoError(t, err)

	current := rotator.GetCurrent()
	assert.NotNil(t, current)
	assert.Equal(t, CertTypeServer, current.Type)
	assert.Equal(t, CertStatusValid, current.Status)

	metrics := rotator.GetMetrics()
	assert.False(t, metrics.LastRotation.IsZero())
	assert.Equal(t, int64(0), metrics.RotationAttempts)
	assert.Equal(t, int64(0), metrics.RotationErrors)
	assert.Equal(t, int64(0), metrics.ValidationErrors)

	store.AssertExpectations(t)
	manager.AssertExpectations(t)
}

func TestCertificateRotator_CheckRotation(t *testing.T) {
	store := &MockCertificateStore{}
	manager := &MockCertificateManager{}
	logger, _ := zap.NewDevelopment()

	now := time.Now()
	oldCert, oldKey := createTestCertificate(t, now.Add(-23*time.Hour), now.Add(time.Hour))
	newCert, newKey := createTestCertificate(t, now, now.Add(24*time.Hour))

	oldCertPair := &CertPair{
		Cert: oldCert,
		Key:  oldKey,
	}

	// Channel to signal when rotation is complete
	rotationDone := make(chan struct{})

	// Set up expectations for initial certificate loading and validation
	store.On("LoadCurrent").Return(oldCertPair, nil)
	manager.On("GetCertificateStore").Return(store)
	manager.On("Validate", mock.MatchedBy(func(c *Certificate) bool {
		return c.X509.SerialNumber.String() == oldCert.SerialNumber.String() && c.Type == CertTypeServer
	})).Return(nil)
	store.On("ValidateCRL", mock.MatchedBy(func(c *Certificate) bool {
		return c.X509.SerialNumber.String() == oldCert.SerialNumber.String() && c.Type == CertTypeServer
	}), mock.Anything).Return(nil)
	store.On("ValidateOCSP", mock.MatchedBy(func(c *Certificate) bool {
		return c.X509.SerialNumber.String() == oldCert.SerialNumber.String() && c.Type == CertTypeServer
	})).Return(nil)

	// Set up expectations for certificate rotation
	newCertificate := &Certificate{
		Raw:          newCert.Raw,
		X509:         newCert,
		PrivateKey:   newKey,
		Type:         CertTypeServer,
		Status:       CertStatusValid,
		SerialNumber: newCert.SerialNumber.String(),
		SANs:         newCert.DNSNames,
		KeyUsage:     newCert.KeyUsage,
		ExtKeyUsage:  newCert.ExtKeyUsage,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
		Metadata:     make(map[string]string),
	}
	manager.On("CreateServer", mock.Anything, mock.MatchedBy(func(c *Certificate) bool {
		return c.X509.SerialNumber.String() == oldCert.SerialNumber.String() && c.Type == CertTypeServer
	})).Return(newCertificate, nil)

	// Set up expectations for new certificate validation
	manager.On("Validate", mock.MatchedBy(func(c *Certificate) bool {
		return c.X509.SerialNumber.String() == newCert.SerialNumber.String() && c.Type == CertTypeServer
	})).Return(nil)
	store.On("ValidateCRL", mock.MatchedBy(func(c *Certificate) bool {
		return c.X509.SerialNumber.String() == newCert.SerialNumber.String() && c.Type == CertTypeServer
	}), mock.Anything).Return(nil)
	store.On("ValidateOCSP", mock.MatchedBy(func(c *Certificate) bool {
		return c.X509.SerialNumber.String() == newCert.SerialNumber.String() && c.Type == CertTypeServer
	})).Return(nil)

	store.On("Store", mock.MatchedBy(func(cp *CertPair) bool {
		return cp.Cert.SerialNumber.String() == newCert.SerialNumber.String()
	})).Run(func(args mock.Arguments) {
		close(rotationDone)
	}).Return(nil)
	manager.On("Revoke", oldCert.SerialNumber.String()).Return(nil)

	config := &RotationConfig{
		RotationInterval: time.Minute,
		RenewalWindow:    2 * time.Hour,
		GracePeriod:      100 * time.Millisecond, // Short grace period for testing
		KeySize:          2048,
	}

	rotator := NewCertificateRotator(config, manager, logger)
	err := rotator.Start(context.Background())
	assert.NoError(t, err)

	// Wait for rotation to complete
	select {
	case <-rotationDone:
		// Verify new certificate is set
		current := rotator.GetCurrent()
		assert.NotNil(t, current)
		assert.Equal(t, newCert.SerialNumber.String(), current.SerialNumber)

		// Check previous certificate before cleanup
		previous := rotator.GetPrevious()
		assert.NotNil(t, previous)
		assert.Equal(t, oldCert.SerialNumber.String(), previous.SerialNumber)

		// Wait for cleanup
		time.Sleep(config.GracePeriod + 150*time.Millisecond)

		// Verify cleanup
		cleanupPrevious := rotator.GetPrevious()
		assert.Nil(t, cleanupPrevious, "Previous certificate should be nil after cleanup")

		// Verify metrics
		metrics := rotator.GetMetrics()
		assert.Equal(t, int64(1), metrics.RotationAttempts)
		assert.Equal(t, int64(0), metrics.RotationErrors)
		assert.Equal(t, int64(0), metrics.ValidationErrors)
	case <-time.After(5 * time.Second):
		t.Fatal("Timeout waiting for certificate rotation")
	}

	store.AssertExpectations(t)
	manager.AssertExpectations(t)
}
