package tunnel

import (
	"crypto/tls"
	"fmt"
	"net"

	"SSSonector/internal/cert"
	"SSSonector/internal/config"

	"go.uber.org/zap"
)

// TLSManager handles TLS connection establishment
type TLSManager struct {
	logger      *zap.Logger
	config      *config.TLSConfig
	certManager *cert.Manager
}

// NewTLSManager creates a new TLS manager
func NewTLSManager(logger *zap.Logger, cfg *config.TLSConfig) *TLSManager {
	return &TLSManager{
		logger: logger,
		config: cfg,
	}
}

// SetCertManager sets the certificate manager
func (m *TLSManager) SetCertManager(certManager *cert.Manager) {
	m.certManager = certManager
}

// CreateServerConfig creates TLS configuration for server mode
func (m *TLSManager) CreateServerConfig() (*tls.Config, error) {
	if m.certManager == nil {
		return nil, fmt.Errorf("certificate manager not initialized")
	}

	cert, key := m.certManager.GetCertificate()
	if cert == nil || key == nil {
		return nil, fmt.Errorf("no valid certificate available")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{cert.Raw},
				PrivateKey:  key,
				Leaf:        cert,
			},
		},
		MinVersion: tls.VersionTLS13,
		ClientAuth: tls.RequireAndVerifyClientCert,
		CipherSuites: []uint16{
			// TLS 1.3 cipher suites (EU-exportable)
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
		VerifyConnection: m.verifyConnection,
	}, nil
}

// CreateClientConfig creates TLS configuration for client mode
func (m *TLSManager) CreateClientConfig() (*tls.Config, error) {
	if m.certManager == nil {
		return nil, fmt.Errorf("certificate manager not initialized")
	}

	cert, key := m.certManager.GetCertificate()
	if cert == nil || key == nil {
		return nil, fmt.Errorf("no valid certificate available")
	}

	return &tls.Config{
		Certificates: []tls.Certificate{
			{
				Certificate: [][]byte{cert.Raw},
				PrivateKey:  key,
				Leaf:        cert,
			},
		},
		MinVersion:         tls.VersionTLS13,
		InsecureSkipVerify: false,
		CipherSuites: []uint16{
			// TLS 1.3 cipher suites (EU-exportable)
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
		VerifyConnection: m.verifyConnection,
	}, nil
}

// verifyConnection implements additional connection verification
func (m *TLSManager) verifyConnection(cs tls.ConnectionState) error {
	// Verify TLS version
	if cs.Version < tls.VersionTLS13 {
		return fmt.Errorf("TLS version %x not supported, minimum required: TLS 1.3", cs.Version)
	}

	// Verify cipher suite
	validCipher := false
	for _, suite := range []uint16{
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
	} {
		if cs.CipherSuite == suite {
			validCipher = true
			break
		}
	}
	if !validCipher {
		return fmt.Errorf("cipher suite %x not allowed", cs.CipherSuite)
	}

	// Log connection details
	var tlsVersion string
	switch cs.Version {
	case tls.VersionTLS13:
		tlsVersion = "TLS 1.3"
	default:
		tlsVersion = fmt.Sprintf("Unknown (0x%04x)", cs.Version)
	}

	m.logger.Info("TLS connection established",
		zap.String("version", tlsVersion),
		zap.String("cipher_suite", tls.CipherSuiteName(cs.CipherSuite)),
		zap.String("server_name", cs.ServerName),
		zap.Bool("handshake_complete", cs.HandshakeComplete),
	)

	return nil
}

// WrapListener wraps a net.Listener with TLS
func (m *TLSManager) WrapListener(listener net.Listener) (net.Listener, error) {
	config, err := m.CreateServerConfig()
	if err != nil {
		return nil, err
	}
	return tls.NewListener(listener, config), nil
}

// WrapConn wraps a net.Conn with TLS
func (m *TLSManager) WrapConn(conn net.Conn) (*tls.Conn, error) {
	config, err := m.CreateClientConfig()
	if err != nil {
		return nil, err
	}
	return tls.Client(conn, config), nil
}
