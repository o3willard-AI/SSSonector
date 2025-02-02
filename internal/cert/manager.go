package cert

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
)

// Manager handles TLS certificate operations
type Manager struct {
	certFile string
	keyFile  string
	caFile   string
	isServer bool
}

// NewManager creates a new certificate manager
func NewManager(certFile, keyFile, caFile string, isServer bool) (*Manager, error) {
	return &Manager{
		certFile: certFile,
		keyFile:  keyFile,
		caFile:   caFile,
		isServer: isServer,
	}, nil
}

// GetTLSConfig returns a configured TLS configuration
func (m *Manager) GetTLSConfig() (*tls.Config, error) {
	// Load certificate
	cert, err := tls.LoadX509KeyPair(m.certFile, m.keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

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
		config = &tls.Config{
			Certificates: []tls.Certificate{cert},
			ClientCAs:    caPool,
			ClientAuth:   tls.RequireAndVerifyClientCert,
			MinVersion:   tls.VersionTLS12,
		}
	} else {
		config = &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      caPool,
			MinVersion:   tls.VersionTLS12,
		}
	}

	return config, nil
}

// NewTLSConfig is a convenience function to create a TLS configuration
func NewTLSConfig(certFile, keyFile, caFile string, isServer bool) (*tls.Config, error) {
	manager, err := NewManager(certFile, keyFile, caFile, isServer)
	if err != nil {
		return nil, err
	}
	return manager.GetTLSConfig()
}
