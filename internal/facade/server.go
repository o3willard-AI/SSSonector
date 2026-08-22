package facade

import (
	"github.com/o3willard-AI/SSSonector/internal/config"
	"context"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	// websocketGUID is the magic GUID used in the WebSocket protocol handshake
	// as defined in RFC 6455 Section 4.2.2
	websocketGUID = "258EAFA5-E914-47DA-95CA-5631BC565D11"

	// tunnelTokenHeader is the HTTP header carrying the HMAC-signed tunnel token
	tunnelTokenHeader = "X-Tunnel-Token"

	// defaultWebRoot is the default content returned for GET /
	defaultWebRoot = `<!DOCTYPE html>
<html><head><title>Welcome</title></head>
<body><h1>Hello, World</h1><p>Welcome to our service.</p></body></html>`

	// connectPath is the path used for tunnel negotiation
	connectPath = "/connect"
)

// Server represents the HTTPS facade server.
// It serves a legitimate-looking website on port 443 while also handling
// tunnel connection upgrades disguised as WebSocket connections.
type Server struct {
	config     *config.FacadeConfig
	authConfig *config.AuthConfig
	logger     *zap.Logger
	httpServer *http.Server
	listener   net.Listener
	secret     []byte
	tokenTTL   time.Duration
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup

	// tunnelPorts is the set of valid tunnel ports this facade routes to
	tunnelPorts map[int]bool
}

// NewServer creates a new HTTPS facade server.
func NewServer(cfg *config.FacadeConfig, authCfg *config.AuthConfig, logger *zap.Logger) (*Server, error) {
	if cfg == nil {
		return nil, fmt.Errorf("facade config is required")
	}
	if authCfg == nil {
		return nil, fmt.Errorf("auth config is required")
	}

	// Resolve the token secret (explicit configuration is mandatory)
	secret, err := ResolveSecret(cfg.TokenSecret, "")
	if err != nil {
		return nil, fmt.Errorf("failed to resolve token secret: %w", err)
	}

	// Build tunnel ports set
	tunnelPorts := make(map[int]bool, len(cfg.TunnelPorts))
	for _, port := range cfg.TunnelPorts {
		tunnelPorts[port] = true
	}

	tokenTTL := cfg.TokenTTL
	if tokenTTL <= 0 {
		tokenTTL = DefaultTokenTTL
	}

	ctx, cancel := context.WithCancel(context.Background())

	s := &Server{
		config:      cfg,
		authConfig:  authCfg,
		logger:      logger,
		secret:      secret,
		tokenTTL:    tokenTTL,
		tunnelPorts: tunnelPorts,
		ctx:         ctx,
		cancel:      cancel,
	}

	return s, nil
}

// Start starts the HTTPS facade server.
func (s *Server) Start() error {
	// Build TLS config
	tlsConfig, err := s.buildTLSConfig()
	if err != nil {
		return fmt.Errorf("failed to build TLS config: %w", err)
	}

	// Create HTTP mux
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc(connectPath, s.handleConnect)

	listenAddr := fmt.Sprintf("%s:%d", s.config.ListenAddress, s.config.ListenPort)

	s.httpServer = &http.Server{
		Addr:      listenAddr,
		Handler:   mux,
		TLSConfig: tlsConfig,
		// Timeouts to prevent slowloris attacks
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Create the TLS listener
	ln, err := tls.Listen("tcp", listenAddr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to start facade listener on %s: %w", listenAddr, err)
	}
	s.listener = ln

	s.logger.Info("HTTPS facade started",
		zap.String("address", listenAddr),
		zap.Int("tunnel_ports", len(s.tunnelPorts)),
	)

	// Serve in a goroutine
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := s.httpServer.Serve(ln); err != nil && err != http.ErrServerClosed {
			s.logger.Error("Facade server error", zap.Error(err))
		}
	}()

	return nil
}

// Stop gracefully stops the HTTPS facade server.
func (s *Server) Stop() error {
	s.logger.Info("Stopping HTTPS facade")
	s.cancel()

	// Give active connections time to finish
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("Facade shutdown error", zap.Error(err))
		// Force close
		s.httpServer.Close()
	}

	s.wg.Wait()
	return nil
}

// handleRoot serves a legitimate-looking web page for GET /.
// This makes the server appear as a normal website to casual inspection,
// port scanners, and automated probes.
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	// Only serve the root path exactly -- everything else is 404
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	content := s.config.WebRoot
	if content == "" {
		content = defaultWebRoot
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Server", "nginx") // Blend in with common web servers
	w.WriteHeader(http.StatusOK)
	if r.Method != http.MethodHead {
		fmt.Fprint(w, content)
	}
}

// handleConnect handles tunnel connection upgrade requests.
// It validates the WebSocket upgrade headers and HMAC token, then hijacks
// the connection and proxies it to the appropriate local tunnel port.
func (s *Server) handleConnect(w http.ResponseWriter, r *http.Request) {
	// Verify this is a WebSocket upgrade request
	if !isWebSocketUpgrade(r) {
		// Not a WebSocket upgrade -- return 404 to look like a normal server
		http.NotFound(w, r)
		return
	}

	// Extract and validate the tunnel token
	tokenStr := r.Header.Get(tunnelTokenHeader)
	if tokenStr == "" {
		s.logger.Debug("Upgrade request missing tunnel token",
			zap.String("remote", r.RemoteAddr),
		)
		http.NotFound(w, r)
		return
	}

	port, err := ValidateToken(tokenStr, s.secret, s.tokenTTL)
	if err != nil {
		s.logger.Debug("Invalid tunnel token",
			zap.String("remote", r.RemoteAddr),
			zap.Error(err),
		)
		// Return 404 instead of 403 to avoid leaking information
		http.NotFound(w, r)
		return
	}

	// Verify the port is in our allowed list
	if !s.tunnelPorts[port] {
		s.logger.Warn("Token for unconfigured tunnel port",
			zap.String("remote", r.RemoteAddr),
			zap.Int("port", port),
		)
		http.NotFound(w, r)
		return
	}

	// Compute the WebSocket accept value
	wsKey := r.Header.Get("Sec-WebSocket-Key")
	if wsKey == "" {
		http.NotFound(w, r)
		return
	}
	wsAccept := computeWebSocketAccept(wsKey)

	// Hijack the connection
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		s.logger.Error("HTTP server does not support hijacking")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	conn, buf, err := hijacker.Hijack()
	if err != nil {
		s.logger.Error("Failed to hijack connection",
			zap.String("remote", r.RemoteAddr),
			zap.Error(err),
		)
		return
	}

	// Send the 101 Switching Protocols response manually
	response := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + wsAccept + "\r\n" +
		"\r\n"

	if _, err := buf.WriteString(response); err != nil {
		s.logger.Error("Failed to write upgrade response",
			zap.String("remote", r.RemoteAddr),
			zap.Error(err),
		)
		conn.Close()
		return
	}
	if err := buf.Flush(); err != nil {
		s.logger.Error("Failed to flush upgrade response",
			zap.String("remote", r.RemoteAddr),
			zap.Error(err),
		)
		conn.Close()
		return
	}

	s.logger.Info("Tunnel upgrade accepted",
		zap.String("remote", r.RemoteAddr),
		zap.Int("tunnel_port", port),
	)

	// Proxy the hijacked connection to the local tunnel port
	tunnelAddr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		if err := Proxy(s.ctx, conn, tunnelAddr, s.logger); err != nil {
			s.logger.Debug("Proxy ended",
				zap.String("remote", r.RemoteAddr),
				zap.Int("tunnel_port", port),
				zap.Error(err),
			)
		}
	}()
}

// buildTLSConfig creates the TLS configuration for the facade server.
func (s *Server) buildTLSConfig() (*tls.Config, error) {
	// Resolve certificate paths -- facade TLS config takes priority, then auth config
	certFile := s.config.TLS.CertFile
	if certFile == "" {
		certFile = s.authConfig.CertFile
	}
	keyFile := s.config.TLS.KeyFile
	if keyFile == "" {
		keyFile = s.authConfig.KeyFile
	}
	caFile := s.config.TLS.CAFile
	if caFile == "" {
		caFile = s.authConfig.CAFile
	}

	if certFile == "" || keyFile == "" {
		return nil, fmt.Errorf("certificate and key files are required for the facade")
	}

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load facade certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
		// Use standard cipher suites that match typical web servers
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_AES_128_GCM_SHA256,
			tls.TLS_AES_256_GCM_SHA384,
			tls.TLS_CHACHA20_POLY1305_SHA256,
		},
		PreferServerCipherSuites: true,
	}

	// Load CA for optional client certificate verification
	// Note: The facade does NOT require mTLS -- authentication is via HMAC tokens.
	// However, if a CA is configured, we can use it to verify client certs as an
	// additional layer of security.
	if caFile != "" {
		caCert, err := os.ReadFile(caFile)
		if err != nil {
			s.logger.Warn("Failed to read CA file for facade, continuing without client cert verification",
				zap.Error(err),
			)
		} else {
			caPool := x509.NewCertPool()
			if caPool.AppendCertsFromPEM(caCert) {
				tlsConfig.ClientCAs = caPool
				// VerifyClientCertIfGiven allows both browser clients (no cert)
				// and tunnel clients (with cert) to connect
				tlsConfig.ClientAuth = tls.VerifyClientCertIfGiven
			}
		}
	}

	return tlsConfig, nil
}

// isWebSocketUpgrade checks if the request is a valid WebSocket upgrade request.
func isWebSocketUpgrade(r *http.Request) bool {
	if r.Method != http.MethodGet {
		return false
	}

	// Check Connection: Upgrade header (case-insensitive)
	connection := r.Header.Get("Connection")
	hasUpgrade := false
	for _, v := range strings.Split(connection, ",") {
		if strings.TrimSpace(strings.ToLower(v)) == "upgrade" {
			hasUpgrade = true
			break
		}
	}
	if !hasUpgrade {
		return false
	}

	// Check Upgrade: websocket header (case-insensitive)
	upgrade := r.Header.Get("Upgrade")
	if strings.ToLower(strings.TrimSpace(upgrade)) != "websocket" {
		return false
	}

	// Check Sec-WebSocket-Version: 13
	if r.Header.Get("Sec-WebSocket-Version") != "13" {
		return false
	}

	// Check Sec-WebSocket-Key is present
	if r.Header.Get("Sec-WebSocket-Key") == "" {
		return false
	}

	return true
}

// computeWebSocketAccept computes the Sec-WebSocket-Accept value per RFC 6455.
func computeWebSocketAccept(key string) string {
	h := sha1.New()
	h.Write([]byte(key + websocketGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
