package cert

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/cert/generator"
	"go.uber.org/zap"
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
	logger            *zap.Logger
	expireChan        chan struct{}
	rotateChan        chan struct{}
	stopChan          chan struct{}
	certDir           string
	currentCert       *tls.Certificate
	certMutex         sync.RWMutex
	settingsMu        sync.RWMutex  // guards the tunable settings below
	checkInterval     time.Duration // Configurable via hot reload/tests
	rotationThreshold time.Duration // Configurable via hot reload/tests
	useTemporaryCerts bool          // Configurable via hot reload/tests
	rotationDone      chan struct{} // Signals when rotation is complete
}

// NewManager creates a new certificate manager
func NewManager(certFile, keyFile, caFile string, isServer bool, skipVerify bool, logger *zap.Logger) (*Manager, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	m := &Manager{
		certFile:          certFile,
		keyFile:           keyFile,
		caFile:            caFile,
		isServer:          isServer,
		skipVerify:        skipVerify,
		logger:            logger,
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

// SetCheckInterval sets the certificate check interval
func (m *Manager) SetCheckInterval(d time.Duration) {
	m.settingsMu.Lock()
	m.checkInterval = d
	m.settingsMu.Unlock()
}

// SetRotationThreshold sets the certificate rotation threshold
func (m *Manager) SetRotationThreshold(d time.Duration) {
	m.settingsMu.Lock()
	m.rotationThreshold = d
	m.settingsMu.Unlock()
}

// UseTemporaryCerts enables temporary certificate generation
func (m *Manager) UseTemporaryCerts(enable bool) {
	m.settingsMu.Lock()
	m.useTemporaryCerts = enable
	m.settingsMu.Unlock()
}

func (m *Manager) getCheckInterval() time.Duration {
	m.settingsMu.RLock()
	defer m.settingsMu.RUnlock()
	return m.checkInterval
}

func (m *Manager) getRotationThreshold() time.Duration {
	m.settingsMu.RLock()
	defer m.settingsMu.RUnlock()
	return m.rotationThreshold
}

// CheckInterval returns the currently configured check interval
func (m *Manager) CheckInterval() time.Duration {
	return m.getCheckInterval()
}

func (m *Manager) getUseTemporaryCerts() bool {
	m.settingsMu.RLock()
	defer m.settingsMu.RUnlock()
	return m.useTemporaryCerts
}

// GetTLSConfig returns a configured TLS configuration
func (m *Manager) GetTLSConfig() (*tls.Config, error) {
	// Load CA certificate
	caCert, err := os.ReadFile(m.caFile)
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

// monitorCertificate periodically checks certificate expiration and triggers rotation.
// The check interval is re-read before every cycle so runtime updates
// (hot reload/tests) take effect deterministically on the next tick.
func (m *Manager) monitorCertificate() {
	timer := time.NewTimer(m.getCheckInterval())
	defer timer.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-timer.C:
			m.checkCertificate()
			timer.Reset(m.getCheckInterval())
		}
	}
}

// checkCertificate performs a single expiration/rotation evaluation
func (m *Manager) checkCertificate() {
	cert := m.currentCert
	if cert == nil {
		m.logger.Warn("No current certificate")
		return
	}

	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		m.logger.Error("Failed to parse current certificate", zap.Error(err))
		return
	}

	timeUntilExpiry := time.Until(x509Cert.NotAfter)
	m.logger.Info("Certificate status",
		zap.Duration("expires_in", timeUntilExpiry),
		zap.String("serial", x509Cert.SerialNumber.String()),
	)

	// Check if certificate has expired
	if timeUntilExpiry <= 0 {
		m.logger.Error("Certificate has expired",
			zap.String("serial", x509Cert.SerialNumber.String()),
		)
		select {
		case <-m.expireChan:
			// Already closed
		default:
			close(m.expireChan)
		}
		return
	}

	// Check if certificate needs rotation
	if timeUntilExpiry < m.getRotationThreshold() {
		m.logger.Info("Certificate needs rotation",
			zap.Duration("time_to_expiry", timeUntilExpiry),
			zap.Duration("threshold", m.getRotationThreshold()),
		)
		select {
		case m.rotateChan <- struct{}{}:
			// Only rotate if we have enough time before expiration
			if timeUntilExpiry > minRotationWindow {
				m.rotateCertificates()
				<-m.rotationDone // Wait for rotation to complete
			} else {
				m.logger.Warn("Skipping rotation due to imminent expiration",
					zap.Duration("time_to_expiry", timeUntilExpiry),
					zap.Duration("min_rotation_window", minRotationWindow),
				)
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
			m.logger.Warn("Rotation already in progress")
		}
	}
}

// rotateCertificates generates new certificates and updates the manager
func (m *Manager) rotateCertificates() {
	m.logger.Info("Starting certificate rotation",
		zap.Bool("use_temporary_certs", m.getUseTemporaryCerts()),
	)

	// Generate new certificates
	var err error
	if m.getUseTemporaryCerts() {
		err = generator.GenerateTemporaryCertificates(m.certDir)
	} else {
		err = generator.GenerateCertificates(m.certDir)
	}
	if err != nil {
		m.logger.Error("Failed to generate new certificates", zap.Error(err))
		m.rotationDone <- struct{}{} // Signal completion even on error
		return
	}

	// Load new certificate
	newCert, err := tls.LoadX509KeyPair(m.certFile, m.keyFile)
	if err != nil {
		m.logger.Error("Failed to load new certificate", zap.Error(err))
		m.rotationDone <- struct{}{} // Signal completion even on error
		return
	}

	// Parse new certificate for logging
	x509Cert, err := x509.ParseCertificate(newCert.Certificate[0])
	if err != nil {
		m.logger.Error("Failed to parse new certificate", zap.Error(err))
		m.rotationDone <- struct{}{} // Signal completion even on error
		return
	}
	m.logger.Info("Generated new certificate",
		zap.String("serial", x509Cert.SerialNumber.String()),
	)

	// Update certificate atomically
	m.certMutex.Lock()
	m.currentCert = &newCert
	m.certMutex.Unlock()

	m.logger.Info("Certificate rotation completed successfully")
	m.rotationDone <- struct{}{} // Signal completion
}

// Stop stops the certificate manager
func (m *Manager) Stop() {
	close(m.stopChan)
}
