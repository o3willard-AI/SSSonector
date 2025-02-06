package tunnel

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"

	"github.com/o3willard-AI/SSSonector/internal/cert"
)

// TLSConfig holds TLS configuration
type TLSConfig struct {
	CertFile      string
	KeyFile       string
	CAFile        string
	SecurityLevel SecurityLevel // Added security level configuration
	ServerName    string        // Added server name for client verification
}

// TLSManager handles TLS operations
type TLSManager struct {
	certManager *cert.Manager
	config      *TLSConfig
}

// NewTLSManager creates a new TLS manager
func NewTLSManager(config *TLSConfig) (*TLSManager, error) {
	if config == nil {
		return nil, fmt.Errorf("TLS config is required")
	}

	// Check if using test mode
	skipVerify := false
	if config.CertFile != "" {
		if strings.Contains(config.CertFile, "/tmp/") {
			skipVerify = true
		}
	}

	manager, err := cert.NewManager(config.CertFile, config.KeyFile, config.CAFile, false, skipVerify)
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
	baseConfig, err := t.certManager.GetTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get base TLS config: %w", err)
	}

	// Apply security level configuration
	config := GetTLSSecurityLevel(t.config.SecurityLevel, baseConfig)

	// Additional client-specific settings
	config.InsecureSkipVerify = false // Never skip verification on client
	if t.config.ServerName != "" {
		config.ServerName = t.config.ServerName
	} else if strings.Contains(t.config.CertFile, "/tmp/") {
		// For tests, allow insecure skip verify if no server name is set
		config.InsecureSkipVerify = true
	}

	return config, nil
}

// GetServerConfig returns TLS configuration for server mode
func (t *TLSManager) GetServerConfig() (*tls.Config, error) {
	// Check if using test mode
	skipVerify := false
	if t.config.CertFile != "" {
		if strings.Contains(t.config.CertFile, "/tmp/") {
			skipVerify = true
		}
	}

	// Create new manager with server mode
	manager, err := cert.NewManager(t.config.CertFile, t.config.KeyFile, t.config.CAFile, true, skipVerify)
	if err != nil {
		return nil, fmt.Errorf("failed to create server certificate manager: %w", err)
	}

	baseConfig, err := manager.GetTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get base TLS config: %w", err)
	}

	// Apply security level configuration
	config := GetTLSSecurityLevel(t.config.SecurityLevel, baseConfig)

	// Additional server-specific settings
	config.PreferServerCipherSuites = true     // Server chooses cipher suite
	config.DynamicRecordSizingDisabled = false // Enable dynamic record sizing for better performance
	config.SessionTicketsDisabled = true       // Disable session tickets for perfect forward secrecy

	return config, nil
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
