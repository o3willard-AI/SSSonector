package tunnel

import (
	"crypto/tls"
	"net"

	"github.com/o3willard-AI/SSSonector/internal/cert"
	"github.com/o3willard-AI/SSSonector/internal/config"
	"go.uber.org/zap"
)

// TLSManager handles TLS configuration and connections
type TLSManager struct {
	logger      *zap.Logger
	config      *config.TunnelConfig
	certManager *cert.Manager
}

// NewTLSManager creates a new TLS manager
func NewTLSManager(logger *zap.Logger, cfg *config.TunnelConfig) *TLSManager {
	return &TLSManager{
		logger: logger,
		config: cfg,
	}
}

// SetCertManager sets the certificate manager
func (t *TLSManager) SetCertManager(manager *cert.Manager) {
	t.certManager = manager
}

// WrapListener wraps a net.Listener with TLS
func (t *TLSManager) WrapListener(listener net.Listener) (net.Listener, error) {
	if t.certManager == nil {
		return nil, ErrCertManagerNotInitialized
	}

	tlsConfig := t.certManager.GetTLSConfig(true)
	return tls.NewListener(listener, tlsConfig), nil
}

// WrapConn wraps a net.Conn with TLS
func (t *TLSManager) WrapConn(conn net.Conn) (net.Conn, error) {
	if t.certManager == nil {
		return nil, ErrCertManagerNotInitialized
	}

	tlsConfig := t.certManager.GetTLSConfig(false)
	return tls.Client(conn, tlsConfig), nil
}

// GetTLSConfig returns the TLS configuration
func (t *TLSManager) GetTLSConfig(isServer bool) (*tls.Config, error) {
	if t.certManager == nil {
		return nil, ErrCertManagerNotInitialized
	}
	return t.certManager.GetTLSConfig(isServer), nil
}

// LoadCACertificate loads a CA certificate
func (t *TLSManager) LoadCACertificate(caFile string) error {
	if t.certManager == nil {
		return ErrCertManagerNotInitialized
	}
	return t.certManager.LoadCACertificate(caFile)
}
