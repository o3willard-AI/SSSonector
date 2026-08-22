package facade

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"go.uber.org/zap"
)

const (
	// DefaultDirectTimeout is the default timeout for direct connection attempts
	DefaultDirectTimeout = 3 * time.Second
)

// ConnectResult holds the result of a connection attempt.
type ConnectResult struct {
	// Conn is the established network connection
	Conn net.Conn
	// ViaFacade indicates whether the connection was established through the
	// HTTPS facade. If true, the connection is already TLS-encrypted and the
	// tunnel layer should NOT apply its own TLS wrapping.
	ViaFacade bool
}

// Client represents the HTTPS facade client with direct-connect fallback.
// It first attempts a direct TCP connection to the configured tunnel port.
// If that fails (typically because a firewall blocks the port), it falls
// back to establishing the tunnel through the HTTPS facade on port 443.
type Client struct {
	facadeConfig  *types.FacadeConfig
	tunnelConfig  *types.TunnelConfig
	authConfig    *types.AuthConfig
	logger        *zap.Logger
	secret        []byte
	directTimeout time.Duration
}

// NewClient creates a new facade client.
func NewClient(facadeCfg *types.FacadeConfig, tunnelCfg *types.TunnelConfig, authCfg *types.AuthConfig, logger *zap.Logger) (*Client, error) {
	if facadeCfg == nil {
		return nil, fmt.Errorf("facade config is required")
	}
	if tunnelCfg == nil {
		return nil, fmt.Errorf("tunnel config is required")
	}
	if authCfg == nil {
		return nil, fmt.Errorf("auth config is required")
	}

	// Resolve the token secret (must match server; explicit configuration required)
	secret, err := ResolveSecret(facadeCfg.TokenSecret, "")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve token secret: %w", err)
	}

	directTimeout := facadeCfg.DirectTimeout
	if directTimeout <= 0 {
		directTimeout = DefaultDirectTimeout
	}

	return &Client{
		facadeConfig:  facadeCfg,
		tunnelConfig:  tunnelCfg,
		authConfig:    authCfg,
		logger:        logger,
		secret:        secret,
		directTimeout: directTimeout,
	}, nil
}

// Connect attempts to establish a connection to the tunnel server.
// It first tries a direct TCP connection to the configured tunnel port.
// If that fails within the direct timeout, it falls back to the HTTPS facade.
// The returned ConnectResult indicates which method was used, so the caller
// knows whether to apply TLS wrapping.
func (c *Client) Connect(ctx context.Context) (*ConnectResult, error) {
	// Determine the tunnel server address
	serverAddr := fmt.Sprintf("%s:%d", c.tunnelConfig.ServerAddress, c.tunnelConfig.ServerPort)

	// Step 1: Try direct connection
	c.logger.Debug("Attempting direct connection", zap.String("address", serverAddr))

	directCtx, directCancel := context.WithTimeout(ctx, c.directTimeout)
	defer directCancel()

	conn, err := c.connectDirect(directCtx, serverAddr)
	if err == nil {
		c.logger.Info("Direct connection established", zap.String("address", serverAddr))
		return &ConnectResult{
			Conn:      conn,
			ViaFacade: false,
		}, nil
	}

	c.logger.Info("Direct connection failed, falling back to HTTPS facade",
		zap.String("address", serverAddr),
		zap.Error(err),
	)

	// Step 2: Fall back to HTTPS facade
	facadeConn, err := c.connectViaFacade(ctx)
	if err != nil {
		return nil, fmt.Errorf("both direct and facade connections failed: direct=%v, facade=%w", err, err)
	}

	c.logger.Info("Connection established via HTTPS facade",
		zap.String("facade_address", c.facadeAddress()),
		zap.Int("tunnel_port", c.tunnelConfig.ServerPort),
	)

	return &ConnectResult{
		Conn:      facadeConn,
		ViaFacade: true,
	}, nil
}

// connectDirect attempts a direct TCP connection to the tunnel port.
func (c *Client) connectDirect(ctx context.Context, address string) (net.Conn, error) {
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp4", address)
	if err != nil {
		return nil, fmt.Errorf("direct connection to %s failed: %w", address, err)
	}
	return conn, nil
}

// connectViaFacade establishes a tunnel connection through the HTTPS facade.
// It performs a TLS handshake, sends a WebSocket upgrade request with the
// HMAC token, and on success returns the underlying connection for raw
// bidirectional tunnel data transfer.
func (c *Client) connectViaFacade(ctx context.Context) (net.Conn, error) {
	// Build TLS config for the facade connection
	tlsConfig, err := c.buildTLSConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to build facade TLS config: %w", err)
	}

	// Connect to the facade server
	facadeAddr := c.facadeAddress()
	dialer := &tls.Dialer{
		NetDialer: &net.Dialer{
			Timeout: 10 * time.Second,
		},
		Config: tlsConfig,
	}

	conn, err := dialer.DialContext(ctx, "tcp4", facadeAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to facade at %s: %w", facadeAddr, err)
	}

	// Generate the tunnel token
	tunnelPort := c.tunnelConfig.ServerPort
	token, err := GenerateToken(tunnelPort, c.secret)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to generate tunnel token: %w", err)
	}

	// Generate a random WebSocket key
	wsKey, err := generateWebSocketKey()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to generate WebSocket key: %w", err)
	}

	// Determine the host header
	host := c.facadeConfig.ServerAddress
	if host == "" {
		host = c.tunnelConfig.ServerAddress
	}
	if c.facadeConfig.ServerPort != 443 {
		host = net.JoinHostPort(host, strconv.Itoa(c.facadeConfig.ServerPort))
	}

	// Send the HTTP upgrade request
	request := fmt.Sprintf(
		"GET %s HTTP/1.1\r\n"+
			"Host: %s\r\n"+
			"Upgrade: websocket\r\n"+
			"Connection: Upgrade\r\n"+
			"Sec-WebSocket-Key: %s\r\n"+
			"Sec-WebSocket-Version: 13\r\n"+
			"%s: %s\r\n"+
			"\r\n",
		connectPath,
		host,
		wsKey,
		tunnelTokenHeader,
		token,
	)

	if _, err := conn.Write([]byte(request)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to send upgrade request: %w", err)
	}

	// Read the response
	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, nil)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to read upgrade response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusSwitchingProtocols {
		conn.Close()
		return nil, fmt.Errorf("facade upgrade rejected: HTTP %d %s", resp.StatusCode, resp.Status)
	}

	// Verify the WebSocket accept header
	expectedAccept := computeWebSocketAccept(wsKey)
	if resp.Header.Get("Sec-WebSocket-Accept") != expectedAccept {
		conn.Close()
		return nil, fmt.Errorf("invalid Sec-WebSocket-Accept in upgrade response")
	}

	c.logger.Debug("WebSocket upgrade completed, connection ready for tunnel data",
		zap.String("facade", facadeAddr),
		zap.Int("tunnel_port", tunnelPort),
	)

	// The connection is now hijacked -- return it for raw tunnel use.
	// Note: If there's buffered data in the reader, we need to wrap the connection.
	if reader.Buffered() > 0 {
		return &bufferedConn{Conn: conn, reader: reader}, nil
	}

	return conn, nil
}

// facadeAddress returns the facade server address:port string.
func (c *Client) facadeAddress() string {
	addr := c.facadeConfig.ServerAddress
	if addr == "" {
		addr = c.tunnelConfig.ServerAddress
	}
	port := c.facadeConfig.ServerPort
	if port == 0 {
		port = 443
	}
	return net.JoinHostPort(addr, strconv.Itoa(port))
}

// buildTLSConfig creates the TLS configuration for connecting to the facade.
func (c *Client) buildTLSConfig() (*tls.Config, error) {
	certFile := c.facadeConfig.TLS.CertFile
	if certFile == "" {
		certFile = c.authConfig.CertFile
	}
	keyFile := c.facadeConfig.TLS.KeyFile
	if keyFile == "" {
		keyFile = c.authConfig.KeyFile
	}
	caFile := c.facadeConfig.TLS.CAFile
	if caFile == "" {
		caFile = c.authConfig.CAFile
	}

	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Load client certificate if available (for optional mTLS with facade)
	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate for facade: %w", err)
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
	}

	// Load CA certificate
	if caFile != "" {
		caCert, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}
		tlsConfig.RootCAs = caPool
	}

	// Set ServerName for SNI
	serverName := c.facadeConfig.ServerAddress
	if serverName == "" {
		serverName = c.tunnelConfig.ServerAddress
	}
	tlsConfig.ServerName = serverName

	return tlsConfig, nil
}

// generateWebSocketKey generates a random 16-byte base64-encoded WebSocket key
// as required by RFC 6455 Section 4.1.
func generateWebSocketKey() (string, error) {
	key := make([]byte, 16)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate random key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(key), nil
}

// bufferedConn wraps a net.Conn with a buffered reader to handle any data
// that was buffered during the HTTP response reading.
type bufferedConn struct {
	net.Conn
	reader *bufio.Reader
}

func (bc *bufferedConn) Read(b []byte) (int, error) {
	return bc.reader.Read(b)
}
