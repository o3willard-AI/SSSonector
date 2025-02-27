package tunnel

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"go.uber.org/zap"
)

// CertManager handles TLS certificate management
type CertManager struct {
	logger *zap.Logger
	config *types.AppConfig
}

// NewCertManager creates a new certificate manager
func NewCertManager(logger *zap.Logger, cfg *types.AppConfig) *CertManager {
	return &CertManager{
		logger: logger,
		config: cfg,
	}
}

// VerifyPeerCertificate verifies peer certificates for authentication
func (m *CertManager) VerifyPeerCertificate(conn net.Conn) error {
	// Load CA certificate
	caCertPool := x509.NewCertPool()
	caCert, err := os.ReadFile(m.config.Config.Auth.CAFile)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %w", err)
	}
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return fmt.Errorf("failed to parse CA certificate")
	}

	// Load server certificate
	cert, err := tls.LoadX509KeyPair(m.config.Config.Auth.CertFile, m.config.Config.Auth.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to load certificate: %w", err)
	}

	// Create TLS config for authentication only
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    caCertPool,
		ClientAuth:   tls.RequireAndVerifyClientCert,
	}

	// Perform TLS handshake for authentication
	tlsConn := tls.Server(conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return fmt.Errorf("TLS handshake failed: %w", err)
	}

	// Verify peer certificate
	certs := tlsConn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		return fmt.Errorf("no peer certificates provided")
	}

	opts := x509.VerifyOptions{
		Roots:         caCertPool,
		CurrentTime:   time.Now(),
		Intermediates: x509.NewCertPool(),
	}

	// Add any intermediate certificates
	for _, cert := range certs[1:] {
		opts.Intermediates.AddCert(cert)
	}

	// Verify the peer's certificate chain
	if _, err := certs[0].Verify(opts); err != nil {
		return fmt.Errorf("certificate verification failed: %w", err)
	}

	return nil
}

// VerifyCertificates verifies that all required certificates exist and are valid
func (m *CertManager) VerifyCertificates() error {
	// Check certificate and key
	if _, err := tls.LoadX509KeyPair(m.config.Config.Auth.CertFile, m.config.Config.Auth.KeyFile); err != nil {
		return fmt.Errorf("invalid certificate/key pair: %w", err)
	}

	// Check CA certificate if specified
	if m.config.Config.Auth.CAFile != "" {
		caCert, err := os.ReadFile(m.config.Config.Auth.CAFile)
		if err != nil {
			return fmt.Errorf("failed to read CA certificate: %w", err)
		}

		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caCert) {
			return fmt.Errorf("failed to parse CA certificate")
		}
	}

	return nil
}
