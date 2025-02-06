package cert

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"sync"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/cert/generator"
)

const (
	// Start rotation when certificate has this much time left
	defaultRotationThreshold = 30 * 24 * time.Hour // 30 days
	// Check certificate expiration every interval
	defaultCheckInterval = time.Hour
	// Minimum time before expiration to attempt rotation
	minRotationWindow = 5 * time.Second
)

// Manager handles TLS certificate operations
type Manager struct {
	certFile          string
	keyFile           string
	caFile            string
	isServer          bool
	skipVerify        bool
	expireChan        chan struct{}
	rotateChan        chan struct{}
	stopChan          chan struct{}
	certDir           string
	currentCert       *tls.Certificate
	certMutex         sync.RWMutex
	checkInterval     time.Duration // Configurable for testing
	rotationThreshold time.Duration // Configurable for testing
	useTemporaryCerts bool          // For testing
	rotationDone      chan struct{} // Signals when rotation is complete
}

// NewManager creates a new certificate manager
func NewManager(certFile, keyFile, caFile string, isServer bool, skipVerify bool) (*Manager, error) {
	m := &Manager{
		certFile:          certFile,
		keyFile:           keyFile,
		caFile:            caFile,
		isServer:          isServer,
		skipVerify:        skipVerify,
		expireChan:        make(chan struct{}),
		rotateChan:        make(chan struct{}, 1), // Buffer size 1
		stopChan:          make(chan struct{}),
		certDir:           filepath.Dir(certFile),
		checkInterval:     defaultCheckInterval,
		rotationThreshold: defaultRotationThreshold,
		useTemporaryCerts: false,
		rotationDone:      make(chan struct{}),
	}

	// Load initial certificate
	cert, err := tls.LoadX509KeyPair(m.certFile, m.keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load initial certificate: %w", err)
	}
	m.currentCert = &cert

	// Start certificate monitoring and rotation
	go m.monitorCertificate()

	return m, nil
}

// SetCheckInterval sets the certificate check interval (for testing)
func (m *Manager) SetCheckInterval(d time.Duration) {
	m.checkInterval = d
}

// SetRotationThreshold sets the certificate rotation threshold (for testing)
func (m *Manager) SetRotationThreshold(d time.Duration) {
	m.rotationThreshold = d
}

// UseTemporaryCerts enables temporary certificate generation (for testing)
func (m *Manager) UseTemporaryCerts(enable bool) {
	m.useTemporaryCerts = enable
}

// GetTLSConfig returns a configured TLS configuration
func (m *Manager) GetTLSConfig() (*tls.Config, error) {
	// Load CA certificate
	caCert, err := ioutil.ReadFile(m.caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	var config *tls.Config
	if m.isServer {
		clientAuth := tls.RequireAndVerifyClientCert
		if m.skipVerify {
			clientAuth = tls.NoClientCert
		}
		config = &tls.Config{
			GetCertificate: m.getCertificate,
			ClientCAs:      caPool,
			ClientAuth:     clientAuth,
			MinVersion:     tls.VersionTLS12,
			VerifyConnection: func(cs tls.ConnectionState) error {
				select {
				case <-m.expireChan:
					return fmt.Errorf("certificate expired")
				default:
					return nil
				}
			},
		}
	} else {
		config = &tls.Config{
			GetClientCertificate: m.getClientCertificate,
			RootCAs:              caPool,
			MinVersion:           tls.VersionTLS12,
			VerifyConnection: func(cs tls.ConnectionState) error {
				select {
				case <-m.expireChan:
					return fmt.Errorf("certificate expired")
				default:
					return nil
				}
			},
		}
	}

	return config, nil
}

// getCertificate returns the current certificate for server TLS
func (m *Manager) getCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	m.certMutex.RLock()
	defer m.certMutex.RUnlock()
	return m.currentCert, nil
}

// getClientCertificate returns the current certificate for client TLS
func (m *Manager) getClientCertificate(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
	m.certMutex.RLock()
	defer m.certMutex.RUnlock()
	return m.currentCert, nil
}

// monitorCertificate periodically checks certificate expiration and triggers rotation
func (m *Manager) monitorCertificate() {
	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			cert := m.currentCert
			if cert == nil {
				log.Printf("No current certificate")
				continue
			}

			x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
			if err != nil {
				log.Printf("Error parsing certificate: %v", err)
				continue
			}

			timeUntilExpiry := time.Until(x509Cert.NotAfter)
			log.Printf("Certificate expires in %v (serial: %v)", timeUntilExpiry, x509Cert.SerialNumber)

			// Check if certificate has expired
			if timeUntilExpiry <= 0 {
				log.Printf("Certificate has expired")
				select {
				case <-m.expireChan:
					// Already closed
				default:
					close(m.expireChan)
				}
				return
			}

			// Check if certificate needs rotation
			if timeUntilExpiry < m.rotationThreshold {
				log.Printf("Certificate needs rotation (threshold: %v)", m.rotationThreshold)
				select {
				case m.rotateChan <- struct{}{}:
					// Only rotate if we have enough time before expiration
					if timeUntilExpiry > minRotationWindow {
						m.rotateCertificates()
						<-m.rotationDone // Wait for rotation to complete
					} else {
						log.Printf("Skipping rotation due to imminent expiration")
						<-m.rotateChan // Clear the rotation signal
						// Close expiration channel since we're too close to expiry
						select {
						case <-m.expireChan:
							// Already closed
						default:
							close(m.expireChan)
						}
					}
				default:
					log.Printf("Rotation already in progress")
				}
			}
		}
	}
}

// rotateCertificates generates new certificates and updates the manager
func (m *Manager) rotateCertificates() {
	log.Printf("Starting certificate rotation (useTemporaryCerts: %v)", m.useTemporaryCerts)

	// Generate new certificates
	var err error
	if m.useTemporaryCerts {
		err = generator.GenerateTemporaryCertificates(m.certDir)
	} else {
		err = generator.GenerateCertificates(m.certDir)
	}
	if err != nil {
		log.Printf("Failed to generate new certificates: %v", err)
		m.rotationDone <- struct{}{} // Signal completion even on error
		return
	}

	// Load new certificate
	newCert, err := tls.LoadX509KeyPair(m.certFile, m.keyFile)
	if err != nil {
		log.Printf("Failed to load new certificate: %v", err)
		m.rotationDone <- struct{}{} // Signal completion even on error
		return
	}

	// Parse new certificate for logging
	x509Cert, err := x509.ParseCertificate(newCert.Certificate[0])
	if err != nil {
		log.Printf("Failed to parse new certificate: %v", err)
		m.rotationDone <- struct{}{} // Signal completion even on error
		return
	}
	log.Printf("Generated new certificate with serial: %v", x509Cert.SerialNumber)

	// Update certificate atomically
	m.certMutex.Lock()
	m.currentCert = &newCert
	m.certMutex.Unlock()

	log.Printf("Certificate rotation completed successfully")
	m.rotationDone <- struct{}{} // Signal completion
}

// Stop stops the certificate manager
func (m *Manager) Stop() {
	close(m.stopChan)
}

// NewTLSConfig is a convenience function to create a TLS configuration
func NewTLSConfig(certFile, keyFile, caFile string, isServer bool, skipVerify bool) (*tls.Config, error) {
	manager, err := NewManager(certFile, keyFile, caFile, isServer, skipVerify)
	if err != nil {
		return nil, err
	}
	return manager.GetTLSConfig()
}
