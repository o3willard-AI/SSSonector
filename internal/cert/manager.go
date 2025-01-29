package cert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Config holds certificate manager configuration
type Config struct {
	CertFile     string
	KeyFile      string
	CAFile       string
	AutoGenerate bool
	ValidityDays int
	CommonName   string
	IPAddresses  []string
	DNSNames     []string
}

// Manager handles certificate operations
type Manager struct {
	logger     *zap.Logger
	config     *Config
	certMutex  sync.RWMutex
	cert       *x509.Certificate
	privateKey *ecdsa.PrivateKey
	stopChan   chan struct{}
}

// NewManager creates a new certificate manager
func NewManager(logger *zap.Logger, config *Config) *Manager {
	return &Manager{
		logger:   logger,
		config:   config,
		stopChan: make(chan struct{}),
	}
}

// Initialize sets up the certificate manager
func (m *Manager) Initialize() error {
	// Create certificate directory if it doesn't exist
	certDir := filepath.Dir(m.config.CertFile)
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return fmt.Errorf("failed to create certificate directory: %w", err)
	}

	// Check if certificates exist
	certExists := fileExists(m.config.CertFile)
	keyExists := fileExists(m.config.KeyFile)

	// Generate or load certificates
	if !certExists || !keyExists {
		if m.config.AutoGenerate {
			m.logger.Info("Generating new certificates")
			if err := m.generateCertificates(); err != nil {
				return fmt.Errorf("failed to generate certificates: %w", err)
			}
		} else {
			return fmt.Errorf("certificates not found and auto-generate is disabled")
		}
	} else {
		m.logger.Info("Loading existing certificates")
		if err := m.loadCertificates(); err != nil {
			return fmt.Errorf("failed to load certificates: %w", err)
		}
	}

	// Start certificate monitoring
	go m.monitorCertificates()

	return nil
}

// generateCertificates creates new certificates
func (m *Manager) generateCertificates() error {
	m.certMutex.Lock()
	defer m.certMutex.Unlock()

	// Generate private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate private key: %w", err)
	}

	// Parse IP addresses
	var ips []net.IP
	for _, ip := range m.config.IPAddresses {
		if parsed := net.ParseIP(ip); parsed != nil {
			ips = append(ips, parsed)
		}
	}

	// Add default IPs if none specified
	if len(ips) == 0 {
		ips = append(ips, net.ParseIP("127.0.0.1"), net.ParseIP("::1"))
	}

	// Add default DNS names if none specified
	dnsNames := m.config.DNSNames
	if len(dnsNames) == 0 {
		dnsNames = append(dnsNames, "localhost")
	}

	// Prepare certificate template
	template := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		Subject: pkix.Name{
			Organization: []string{"SSL Tunnel"},
			CommonName:   m.config.CommonName,
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(time.Duration(m.config.ValidityDays) * 24 * time.Hour),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,

		IPAddresses: ips,
		DNSNames:    dnsNames,
	}

	// Create certificate
	derBytes, err := x509.CreateCertificate(rand.Reader, template, template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return fmt.Errorf("failed to create certificate: %w", err)
	}

	// Save certificate
	certOut, err := os.Create(m.config.CertFile)
	if err != nil {
		return fmt.Errorf("failed to create certificate file: %w", err)
	}
	defer certOut.Close()

	if err := pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes}); err != nil {
		return fmt.Errorf("failed to write certificate: %w", err)
	}

	// Save private key
	keyOut, err := os.OpenFile(m.config.KeyFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create private key file: %w", err)
	}
	defer keyOut.Close()

	privBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}

	if err := pem.Encode(keyOut, &pem.Block{Type: "PRIVATE KEY", Bytes: privBytes}); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	m.cert = template
	m.privateKey = privateKey

	return nil
}

// loadCertificates loads existing certificates
func (m *Manager) loadCertificates() error {
	m.certMutex.Lock()
	defer m.certMutex.Unlock()

	// Load certificate
	certPEM, err := os.ReadFile(m.config.CertFile)
	if err != nil {
		return fmt.Errorf("failed to read certificate file: %w", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return fmt.Errorf("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Load private key
	keyPEM, err := os.ReadFile(m.config.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to read private key file: %w", err)
	}

	block, _ = pem.Decode(keyPEM)
	if block == nil {
		return fmt.Errorf("failed to decode private key PEM")
	}

	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	privateKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return fmt.Errorf("private key is not an ECDSA key")
	}

	m.cert = cert
	m.privateKey = privateKey

	return nil
}

// monitorCertificates checks certificate expiration and triggers renewal
func (m *Manager) monitorCertificates() {
	ticker := time.NewTicker(12 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			m.certMutex.RLock()
			if m.cert == nil {
				m.certMutex.RUnlock()
				continue
			}

			expiresIn := time.Until(m.cert.NotAfter)
			m.certMutex.RUnlock()

			// Warn if certificate expires in less than 48 hours
			if expiresIn < 48*time.Hour {
				m.logger.Warn("Certificate expiring soon",
					zap.Time("expiry", m.cert.NotAfter),
					zap.Duration("expires_in", expiresIn),
				)

				// Auto-renew if enabled
				if m.config.AutoGenerate {
					m.logger.Info("Auto-generating new certificate")
					if err := m.generateCertificates(); err != nil {
						m.logger.Error("Failed to auto-generate certificate", zap.Error(err))
					}
				}
			}
		}
	}
}

// GetCertificate returns the current certificate and private key
func (m *Manager) GetCertificate() (*x509.Certificate, *ecdsa.PrivateKey) {
	m.certMutex.RLock()
	defer m.certMutex.RUnlock()
	return m.cert, m.privateKey
}

// Stop stops the certificate manager
func (m *Manager) Stop() {
	close(m.stopChan)
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
