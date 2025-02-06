package tunnel

import (
	"crypto/tls"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/cert/generator"
)

func TestTLSSecurityLevels(t *testing.T) {
	// Create temporary directory for test certificates
	tempDir, err := os.MkdirTemp("", "tls-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Generate test certificates
	if err := generator.GenerateTemporaryCertificates(tempDir); err != nil {
		t.Fatalf("Failed to generate certificates: %v", err)
	}

	tests := []struct {
		name          string
		securityLevel SecurityLevel
		wantMinVer    uint16
		wantCiphers   []uint16
	}{
		{
			name:          "Modern security profile",
			securityLevel: SecurityModern,
			wantMinVer:    tls.VersionTLS12,
			wantCiphers:   modernCipherSuites,
		},
		{
			name:          "Intermediate security profile",
			securityLevel: SecurityIntermediate,
			wantMinVer:    tls.VersionTLS12,
			wantCiphers: append(modernCipherSuites,
				tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
				tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			),
		},
		{
			name:          "Old security profile",
			securityLevel: SecurityOld,
			wantMinVer:    tls.VersionTLS10,
			wantCiphers: append(modernCipherSuites,
				tls.TLS_RSA_WITH_AES_256_CBC_SHA,
				tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &TLSConfig{
				CertFile:      filepath.Join(tempDir, "server.crt"),
				KeyFile:       filepath.Join(tempDir, "server.key"),
				CAFile:        filepath.Join(tempDir, "ca.crt"),
				SecurityLevel: tt.securityLevel,
			}

			manager, err := NewTLSManager(config)
			if err != nil {
				t.Fatalf("Failed to create TLS manager: %v", err)
			}

			// Test server config
			serverConfig, err := manager.GetServerConfig()
			if err != nil {
				t.Fatalf("Failed to get server config: %v", err)
			}

			if serverConfig.MinVersion != tt.wantMinVer {
				t.Errorf("Server MinVersion = %v, want %v", serverConfig.MinVersion, tt.wantMinVer)
			}

			if !containsAllCiphers(serverConfig.CipherSuites, tt.wantCiphers) {
				t.Errorf("Server CipherSuites missing required ciphers")
			}

			// Test client config
			clientConfig, err := manager.GetClientConfig()
			if err != nil {
				t.Fatalf("Failed to get client config: %v", err)
			}

			if clientConfig.MinVersion != tt.wantMinVer {
				t.Errorf("Client MinVersion = %v, want %v", clientConfig.MinVersion, tt.wantMinVer)
			}

			if !containsAllCiphers(clientConfig.CipherSuites, tt.wantCiphers) {
				t.Errorf("Client CipherSuites missing required ciphers")
			}
		})
	}
}

func TestTLSHandshake(t *testing.T) {
	// Create temporary directory for test certificates
	tempDir, err := os.MkdirTemp("", "tls-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Generate test certificates
	if err := generator.GenerateTemporaryCertificates(tempDir); err != nil {
		t.Fatalf("Failed to generate certificates: %v", err)
	}

	// Create server TLS manager
	serverConfig := &TLSConfig{
		CertFile:      filepath.Join(tempDir, "server.crt"),
		KeyFile:       filepath.Join(tempDir, "server.key"),
		CAFile:        filepath.Join(tempDir, "ca.crt"),
		SecurityLevel: SecurityModern,
	}
	serverManager, err := NewTLSManager(serverConfig)
	if err != nil {
		t.Fatalf("Failed to create server TLS manager: %v", err)
	}

	// Create client TLS manager
	clientConfig := &TLSConfig{
		CertFile:      filepath.Join(tempDir, "client.crt"),
		KeyFile:       filepath.Join(tempDir, "client.key"),
		CAFile:        filepath.Join(tempDir, "ca.crt"),
		SecurityLevel: SecurityModern,
	}
	clientManager, err := NewTLSManager(clientConfig)
	if err != nil {
		t.Fatalf("Failed to create client TLS manager: %v", err)
	}

	// Create listener
	listener, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Start server
	serverDone := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			serverDone <- err
			return
		}
		defer conn.Close()

		tlsConn, err := serverManager.WrapConn(conn, true)
		if err != nil {
			serverDone <- err
			return
		}
		defer tlsConn.Close()

		// Write test data
		if _, err := tlsConn.Write([]byte("hello")); err != nil {
			serverDone <- err
			return
		}

		serverDone <- nil
	}()

	// Connect client
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close()

	tlsConn, err := clientManager.WrapConn(conn, false)
	if err != nil {
		t.Fatalf("Failed to wrap client connection: %v", err)
	}
	defer tlsConn.Close()

	// Read test data
	data, err := ioutil.ReadAll(tlsConn)
	if err != nil {
		t.Fatalf("Failed to read data: %v", err)
	}

	if string(data) != "hello" {
		t.Errorf("Data = %q, want %q", string(data), "hello")
	}

	// Wait for server
	select {
	case err := <-serverDone:
		if err != nil {
			t.Fatalf("Server error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Server timeout")
	}
}

func containsAllCiphers(have, want []uint16) bool {
	if len(have) < len(want) {
		return false
	}

	wantMap := make(map[uint16]bool)
	for _, cipher := range want {
		wantMap[cipher] = true
	}

	for _, cipher := range have {
		delete(wantMap, cipher)
	}

	return len(wantMap) == 0
}
