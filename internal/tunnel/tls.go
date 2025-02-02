package tunnel

import (
	"crypto/tls"
	"fmt"
	"net"

	"github.com/o3willard-AI/SSSonector/internal/cert"
)

// TLSConfig holds TLS configuration
type TLSConfig struct {
	CertFile string
	KeyFile  string
	CAFile   string
}

// TLSManager handles TLS operations
type TLSManager struct {
	certManager *cert.Manager
	config      *TLSConfig
}

// NewTLSManager creates a new TLS manager
func NewTLSManager(config *TLSConfig) (*TLSManager, error) {
	manager, err := cert.NewManager(config.CertFile, config.KeyFile, config.CAFile, false)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate manager: %w", err)
	}

	return &TLSManager{
		certManager: manager,
		config:      config,
	}, nil
}

// GetClientConfig returns TLS configuration for client mode
func (t *TLSManager) GetClientConfig() (*tls.Config, error) {
	return t.certManager.GetTLSConfig()
}

// GetServerConfig returns TLS configuration for server mode
func (t *TLSManager) GetServerConfig() (*tls.Config, error) {
	// Create new manager with server mode
	manager, err := cert.NewManager(t.config.CertFile, t.config.KeyFile, t.config.CAFile, true)
	if err != nil {
		return nil, fmt.Errorf("failed to create server certificate manager: %w", err)
	}
	return manager.GetTLSConfig()
}

// WrapConn wraps a net.Conn with TLS
func (t *TLSManager) WrapConn(conn net.Conn, isServer bool) (net.Conn, error) {
	var tlsConfig *tls.Config
	var err error

	if isServer {
		tlsConfig, err = t.GetServerConfig()
	} else {
		tlsConfig, err = t.GetClientConfig()
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get TLS config: %w", err)
	}

	if isServer {
		return tls.Server(conn, tlsConfig), nil
	}
	return tls.Client(conn, tlsConfig), nil
}
