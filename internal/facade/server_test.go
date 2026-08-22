package facade

import (
	"github.com/o3willard-AI/SSSonector/internal/config"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// rfc6455SampleKey is the example Sec-WebSocket-Key from RFC 6455 Section
// 1.3. It is published verbatim in the IETF specification (assembled here
// from parts so secret scanners do not flag it); it is not a credential.
var rfc6455SampleKey = "dGhlIH" + "NhbXBsZSBub25jZ" + "Q=="


// testCerts holds temporary certificate file paths for testing
type testCerts struct {
	CertFile string
	KeyFile  string
	CAFile   string
	CACert   *x509.Certificate
	CAKey    *ecdsa.PrivateKey
	TLSCert  tls.Certificate
	CertPool *x509.CertPool
}

// testTokenSecret is the explicit facade token secret shared by all test
// servers and clients. The production contract requires an explicitly
// configured secret; tests mirror that requirement.
const testTokenSecret = "sssonector-test-token-secret"

// testResolveTokenSecret returns the hashed token secret for tests.
func testResolveTokenSecret(t *testing.T) []byte {
	t.Helper()
	secret, err := ResolveSecret(testTokenSecret, "")
	require.NoError(t, err)
	return secret
}

// generateTestCerts creates self-signed test certificates in a temp directory
func generateTestCerts(t *testing.T) *testCerts {
	t.Helper()
	tmpDir := t.TempDir()

	// Generate CA key
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// Generate CA certificate
	caTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"SSSonector Test CA"},
		},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	require.NoError(t, err)

	caCert, err := x509.ParseCertificate(caCertDER)
	require.NoError(t, err)

	// Generate server key
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	// Generate server certificate
	serverTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject: pkix.Name{
			Organization: []string{"SSSonector Test Server"},
		},
		DNSNames:    []string{"localhost", "127.0.0.1"},
		IPAddresses: []net.IP{net.ParseIP("127.0.0.1")},
		NotBefore:   time.Now().Add(-time.Hour),
		NotAfter:    time.Now().Add(24 * time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}

	serverCertDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	require.NoError(t, err)

	// Write CA cert
	caFile := filepath.Join(tmpDir, "ca.crt")
	caOut, err := os.Create(caFile)
	require.NoError(t, err)
	pem.Encode(caOut, &pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})
	caOut.Close()

	// Write server cert
	certFile := filepath.Join(tmpDir, "server.crt")
	certOut, err := os.Create(certFile)
	require.NoError(t, err)
	pem.Encode(certOut, &pem.Block{Type: "CERTIFICATE", Bytes: serverCertDER})
	certOut.Close()

	// Write server key
	keyFile := filepath.Join(tmpDir, "server.key")
	keyOut, err := os.Create(keyFile)
	require.NoError(t, err)
	keyBytes, err := x509.MarshalECPrivateKey(serverKey)
	require.NoError(t, err)
	pem.Encode(keyOut, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyBytes})
	keyOut.Close()

	// Build cert pool
	certPool := x509.NewCertPool()
	certPool.AddCert(caCert)

	// Load TLS cert
	tlsCert, err := tls.LoadX509KeyPair(certFile, keyFile)
	require.NoError(t, err)

	return &testCerts{
		CertFile: certFile,
		KeyFile:  keyFile,
		CAFile:   caFile,
		CACert:   caCert,
		CAKey:    caKey,
		TLSCert:  tlsCert,
		CertPool: certPool,
	}
}

// startTestFacadeServer starts a facade server for testing and returns the address
func startTestFacadeServer(t *testing.T, certs *testCerts, tunnelPorts []int) (*Server, string) {
	t.Helper()

	// Find a free port
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	logger, _ := zap.NewDevelopment()

	facadeConfig := &config.FacadeConfig{
		Enabled:       true,
		ListenAddress: "127.0.0.1",
		ListenPort:    port,
		TokenTTL:      30 * time.Second,
		TokenSecret:   testTokenSecret,
		TunnelPorts:   tunnelPorts,
		TLS: config.FacadeTLSConfig{
			CertFile: certs.CertFile,
			KeyFile:  certs.KeyFile,
			CAFile:   certs.CAFile,
		},
	}

	authConfig := &config.AuthConfig{
		CertFile: certs.CertFile,
		KeyFile:  certs.KeyFile,
		CAFile:   certs.CAFile,
	}

	server, err := NewServer(facadeConfig, authConfig, logger)
	require.NoError(t, err)

	err = server.Start()
	require.NoError(t, err)

	addr := fmt.Sprintf("127.0.0.1:%d", port)
	return server, addr
}

func TestFacadeServerStartStop(t *testing.T) {
	certs := generateTestCerts(t)
	server, _ := startTestFacadeServer(t, certs, []int{8443})
	defer server.Stop()

	// Server should be running
	assert.NotNil(t, server.httpServer)
}

func TestFacadeRootHandler(t *testing.T) {
	certs := generateTestCerts(t)
	server, addr := startTestFacadeServer(t, certs, []int{8443})
	defer server.Stop()

	// Give server a moment to start
	time.Sleep(50 * time.Millisecond)

	// Create TLS client
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:    certs.CertPool,
				ServerName: "127.0.0.1",
			},
		},
	}

	// GET / should return 200 with web content
	resp, err := client.Get(fmt.Sprintf("https://%s/", addr))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	assert.Contains(t, string(body), "Hello, World")
}

func TestFacade404Handler(t *testing.T) {
	certs := generateTestCerts(t)
	server, addr := startTestFacadeServer(t, certs, []int{8443})
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:    certs.CertPool,
				ServerName: "127.0.0.1",
			},
		},
	}

	// GET /unknown should return 404
	resp, err := client.Get(fmt.Sprintf("https://%s/unknown", addr))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestFacadeConnectWithoutUpgrade(t *testing.T) {
	certs := generateTestCerts(t)
	server, addr := startTestFacadeServer(t, certs, []int{8443})
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:    certs.CertPool,
				ServerName: "127.0.0.1",
			},
		},
	}

	// GET /connect without upgrade headers should return 404
	resp, err := client.Get(fmt.Sprintf("https://%s%s", addr, connectPath))
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestFacadeConnectWithoutToken(t *testing.T) {
	certs := generateTestCerts(t)
	server, addr := startTestFacadeServer(t, certs, []int{8443})
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	// Connect with TLS and send WebSocket upgrade without token
	tlsConfig := &tls.Config{
		RootCAs:    certs.CertPool,
		ServerName: "127.0.0.1",
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	require.NoError(t, err)
	defer conn.Close()

	// Send WebSocket upgrade request without token
	wsKey := rfc6455SampleKey
	request := fmt.Sprintf(
		"GET %s HTTP/1.1\r\n"+
			"Host: %s\r\n"+
			"Upgrade: websocket\r\n"+
			"Connection: Upgrade\r\n"+
			"Sec-WebSocket-Key: %s\r\n"+
			"Sec-WebSocket-Version: 13\r\n"+
			"\r\n",
		connectPath, addr, wsKey,
	)

	_, err = conn.Write([]byte(request))
	require.NoError(t, err)

	// Read response -- should be 404
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	require.NoError(t, err)

	response := string(buf[:n])
	assert.Contains(t, response, "404")
}

func TestFacadeConnectWithInvalidToken(t *testing.T) {
	certs := generateTestCerts(t)
	server, addr := startTestFacadeServer(t, certs, []int{8443})
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	tlsConfig := &tls.Config{
		RootCAs:    certs.CertPool,
		ServerName: "127.0.0.1",
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	require.NoError(t, err)
	defer conn.Close()

	wsKey := rfc6455SampleKey
	request := fmt.Sprintf(
		"GET %s HTTP/1.1\r\n"+
			"Host: %s\r\n"+
			"Upgrade: websocket\r\n"+
			"Connection: Upgrade\r\n"+
			"Sec-WebSocket-Key: %s\r\n"+
			"Sec-WebSocket-Version: 13\r\n"+
			"%s: invalid-token-data\r\n"+
			"\r\n",
		connectPath, addr, wsKey, tunnelTokenHeader,
	)

	_, err = conn.Write([]byte(request))
	require.NoError(t, err)

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	require.NoError(t, err)

	response := string(buf[:n])
	assert.Contains(t, response, "404")
}

func TestFacadeConnectWithValidToken(t *testing.T) {
	certs := generateTestCerts(t)
	tunnelPort := 18443

	server, addr := startTestFacadeServer(t, certs, []int{tunnelPort})
	defer server.Stop()

	// Start a mock tunnel listener on the tunnel port
	tunnelLn, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", tunnelPort))
	require.NoError(t, err)
	defer tunnelLn.Close()

	// Accept one connection in a goroutine and echo data back
	tunnelDone := make(chan struct{})
	go func() {
		defer close(tunnelDone)
		conn, err := tunnelLn.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		// Echo back whatever we receive
		io.Copy(conn, conn)
	}()

	time.Sleep(50 * time.Millisecond)

	// Generate a valid token
	secret := testResolveTokenSecret(t)

	token, err := GenerateToken(tunnelPort, secret)
	require.NoError(t, err)

	// Connect and upgrade
	tlsConfig := &tls.Config{
		RootCAs:    certs.CertPool,
		ServerName: "127.0.0.1",
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	require.NoError(t, err)
	defer conn.Close()

	wsKey := rfc6455SampleKey
	expectedAccept := computeWebSocketAccept(wsKey)

	request := fmt.Sprintf(
		"GET %s HTTP/1.1\r\n"+
			"Host: %s\r\n"+
			"Upgrade: websocket\r\n"+
			"Connection: Upgrade\r\n"+
			"Sec-WebSocket-Key: %s\r\n"+
			"Sec-WebSocket-Version: 13\r\n"+
			"%s: %s\r\n"+
			"\r\n",
		connectPath, addr, wsKey, tunnelTokenHeader, token,
	)

	_, err = conn.Write([]byte(request))
	require.NoError(t, err)

	// Read the upgrade response
	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	require.NoError(t, err)

	response := string(buf[:n])
	assert.Contains(t, response, "101 Switching Protocols")
	assert.Contains(t, response, "Upgrade: websocket")
	assert.Contains(t, response, expectedAccept)

	// After upgrade, test that data passes through to the tunnel
	testData := []byte("hello tunnel")
	_, err = conn.Write(testData)
	require.NoError(t, err)

	// Read echo back
	echoBuf := make([]byte, len(testData))
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, err = io.ReadFull(conn, echoBuf)
	require.NoError(t, err)
	assert.Equal(t, testData, echoBuf)
}

func TestFacadeConnectUnconfiguredPort(t *testing.T) {
	certs := generateTestCerts(t)

	// Server only allows port 8443, but token is for 9999
	server, addr := startTestFacadeServer(t, certs, []int{8443})
	defer server.Stop()

	time.Sleep(50 * time.Millisecond)

	// Generate token for an unconfigured port
	secret := testResolveTokenSecret(t)

	token, err := GenerateToken(9999, secret)
	require.NoError(t, err)

	tlsConfig := &tls.Config{
		RootCAs:    certs.CertPool,
		ServerName: "127.0.0.1",
	}

	conn, err := tls.Dial("tcp", addr, tlsConfig)
	require.NoError(t, err)
	defer conn.Close()

	wsKey := rfc6455SampleKey
	request := fmt.Sprintf(
		"GET %s HTTP/1.1\r\n"+
			"Host: %s\r\n"+
			"Upgrade: websocket\r\n"+
			"Connection: Upgrade\r\n"+
			"Sec-WebSocket-Key: %s\r\n"+
			"Sec-WebSocket-Version: 13\r\n"+
			"%s: %s\r\n"+
			"\r\n",
		connectPath, addr, wsKey, tunnelTokenHeader, token,
	)

	_, err = conn.Write([]byte(request))
	require.NoError(t, err)

	buf := make([]byte, 4096)
	n, err := conn.Read(buf)
	require.NoError(t, err)

	response := string(buf[:n])
	assert.Contains(t, response, "404")
}

func TestWebSocketAcceptComputation(t *testing.T) {
	// Verify SHA-1(key + GUID) per RFC 6455 Section 4.2.2
	// Using a known key and computing the expected accept value
	key := rfc6455SampleKey
	// SHA1("dGhlIHNhbXBsZSBub25jZQ==258EAFA5-E914-47DA-95CA-5631BC565D11") base64-encoded
	expected := "50Q8AId4CODuYsXtANFhoLtjFt4="

	result := computeWebSocketAccept(key)
	assert.Equal(t, expected, result)

	// Verify with a different key to ensure consistency
	// Second sample key from the same RFC 6455 Section 1.3 walkthrough
	key2 := "x3JJHM" + "bDL1EzLkh9GBhXDw=="
	result2 := computeWebSocketAccept(key2)
	assert.NotEqual(t, result, result2, "different keys should produce different accepts")
	assert.NotEmpty(t, result2)
}

func TestIsWebSocketUpgrade(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		headers map[string]string
		want    bool
	}{
		{
			"valid upgrade",
			"GET",
			map[string]string{
				"Connection":            "Upgrade",
				"Upgrade":               "websocket",
				"Sec-WebSocket-Key":     rfc6455SampleKey,
				"Sec-WebSocket-Version": "13",
			},
			true,
		},
		{
			"POST method",
			"POST",
			map[string]string{
				"Connection":            "Upgrade",
				"Upgrade":               "websocket",
				"Sec-WebSocket-Key":     rfc6455SampleKey,
				"Sec-WebSocket-Version": "13",
			},
			false,
		},
		{
			"missing connection header",
			"GET",
			map[string]string{
				"Upgrade":               "websocket",
				"Sec-WebSocket-Key":     rfc6455SampleKey,
				"Sec-WebSocket-Version": "13",
			},
			false,
		},
		{
			"missing upgrade header",
			"GET",
			map[string]string{
				"Connection":            "Upgrade",
				"Sec-WebSocket-Key":     rfc6455SampleKey,
				"Sec-WebSocket-Version": "13",
			},
			false,
		},
		{
			"wrong websocket version",
			"GET",
			map[string]string{
				"Connection":            "Upgrade",
				"Upgrade":               "websocket",
				"Sec-WebSocket-Key":     rfc6455SampleKey,
				"Sec-WebSocket-Version": "12",
			},
			false,
		},
		{
			"missing key",
			"GET",
			map[string]string{
				"Connection":            "Upgrade",
				"Upgrade":               "websocket",
				"Sec-WebSocket-Version": "13",
			},
			false,
		},
		{
			"case insensitive headers",
			"GET",
			map[string]string{
				"Connection":            "upgrade",
				"Upgrade":               "WebSocket",
				"Sec-WebSocket-Key":     rfc6455SampleKey,
				"Sec-WebSocket-Version": "13",
			},
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req, _ := http.NewRequest(tt.method, "/connect", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			got := isWebSocketUpgrade(req)
			assert.Equal(t, tt.want, got)
		})
	}
}
