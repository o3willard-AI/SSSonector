package cert

import (
"crypto/tls"
"crypto/x509"
"fmt"
"io/ioutil"
)

// Manager handles SSL/TLS certificate operations
type Manager struct {
certificate tls.Certificate
caCertPool  *x509.CertPool
}

// NewManager creates a new certificate manager
func NewManager(certFile, keyFile string) (*Manager, error) {
cert, err := tls.LoadX509KeyPair(certFile, keyFile)
if err != nil {
return nil, fmt.Errorf("failed to load certificate: %w", err)
}

return &Manager{
certificate: cert,
caCertPool:  x509.NewCertPool(),
}, nil
}

// LoadCACertificate loads a CA certificate into the certificate pool
func (m *Manager) LoadCACertificate(caFile string) error {
caCert, err := ioutil.ReadFile(caFile)
if err != nil {
return fmt.Errorf("failed to read CA certificate: %w", err)
}

if !m.caCertPool.AppendCertsFromPEM(caCert) {
return fmt.Errorf("failed to append CA certificate")
}

return nil
}

// GetTLSConfig returns a TLS configuration for server or client mode
func (m *Manager) GetTLSConfig(isServer bool) *tls.Config {
config := &tls.Config{
MinVersion: tls.VersionTLS13, // Enforce TLS 1.3
CipherSuites: []uint16{
// EU-exportable cipher suites for TLS 1.3
tls.TLS_AES_128_GCM_SHA256,
tls.TLS_AES_256_GCM_SHA384,
tls.TLS_CHACHA20_POLY1305_SHA256,
},
}

if isServer {
config.Certificates = []tls.Certificate{m.certificate}
config.ClientAuth = tls.RequireAndVerifyClientCert
config.ClientCAs = m.caCertPool
} else {
config.Certificates = []tls.Certificate{m.certificate}
config.RootCAs = m.caCertPool
config.ServerName = "sssonector-server"
config.InsecureSkipVerify = true // For testing only
}

return config
}

// GetCertificate returns the loaded certificate
func (m *Manager) GetCertificate() tls.Certificate {
return m.certificate
}

// GetCACertPool returns the CA certificate pool
func (m *Manager) GetCACertPool() *x509.CertPool {
return m.caCertPool
}
