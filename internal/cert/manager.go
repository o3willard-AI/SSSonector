package cert

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"time"
)

// Manager handles TLS certificate operations
type Manager struct {
	certFile   string
	keyFile    string
	caFile     string
	isServer   bool
	skipVerify bool
	expireChan chan struct{}
}

// NewManager creates a new certificate manager
func NewManager(certFile, keyFile, caFile string, isServer bool, skipVerify bool) (*Manager, error) {
	m := &Manager{
		certFile:   certFile,
		keyFile:    keyFile,
		caFile:     caFile,
		isServer:   isServer,
		skipVerify: skipVerify,
		expireChan: make(chan struct{}),
	}

	// Start certificate expiration monitor
	go m.monitorExpiration()

	return m, nil
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
		clientAuth := tls.RequireAndVerifyClientCert
		if m.skipVerify {
			clientAuth = tls.NoClientCert
		}
		config = &tls.Config{
			Certificates:       []tls.Certificate{cert},
			ClientCAs:          caPool,
			ClientAuth:         clientAuth,
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: m.skipVerify,
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
			Certificates:       []tls.Certificate{cert},
			RootCAs:            caPool,
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: m.skipVerify,
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

// monitorExpiration periodically checks certificate expiration
func (m *Manager) monitorExpiration() {
	for {
		// Load and parse certificate
		certData, err := ioutil.ReadFile(m.certFile)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		block, _ := pem.Decode(certData)
		if block == nil {
			time.Sleep(time.Second)
			continue
		}

		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}

		// Check if certificate has expired
		if time.Now().After(cert.NotAfter) {
			close(m.expireChan)
			return
		}

		// Sleep for a second before checking again
		time.Sleep(time.Second)
	}
}

// NewTLSConfig is a convenience function to create a TLS configuration
func NewTLSConfig(certFile, keyFile, caFile string, isServer bool, skipVerify bool) (*tls.Config, error) {
	manager, err := NewManager(certFile, keyFile, caFile, isServer, skipVerify)
	if err != nil {
		return nil, err
	}
	return manager.GetTLSConfig()
}
