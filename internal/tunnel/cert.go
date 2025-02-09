package tunnel

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"go.uber.org/zap"
)

// CertManager handles TLS certificate management
type CertManager struct {
	logger *zap.Logger
	config *config.AppConfig
}

// NewCertManager creates a new certificate manager
func NewCertManager(logger *zap.Logger, cfg *config.AppConfig) *CertManager {
	return &CertManager{
		logger: logger,
		config: cfg,
	}
}

// GetServerTLSConfig returns the TLS configuration for server mode
func (m *CertManager) GetServerTLSConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(m.config.Config.Auth.CertFile, m.config.Config.Auth.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load server certificate: %w", err)
	}

	var clientCAs *x509.CertPool
	if m.config.Config.Auth.CAFile != "" {
		clientCAs = x509.NewCertPool()
		caCert, err := os.ReadFile(m.config.Config.Auth.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}
		if !clientCAs.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientCAs:    clientCAs,
		ClientAuth:   tls.RequireAndVerifyClientCert,
		MinVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{
			// EU-exportable cipher suites for TLS 1.3
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
	}, nil
}

// GetClientTLSConfig returns the TLS configuration for client mode
func (m *CertManager) GetClientTLSConfig() (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(m.config.Config.Auth.CertFile, m.config.Config.Auth.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	var rootCAs *x509.CertPool
	if m.config.Config.Auth.CAFile != "" {
		rootCAs = x509.NewCertPool()
		caCert, err := os.ReadFile(m.config.Config.Auth.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}
		if !rootCAs.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
	}

	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      rootCAs,
		MinVersion:   tls.VersionTLS13,
		CipherSuites: []uint16{
			// EU-exportable cipher suites for TLS 1.3
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
		ServerName: m.config.Config.Network.Interface,
	}, nil
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
