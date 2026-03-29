package tunnel

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"sync"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/adapter"
	"github.com/o3willard-AI/SSSonector/internal/config/interfaces"
	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/o3willard-AI/SSSonector/internal/monitor"
	"go.uber.org/zap"
)

// Tunnel defines the interface for tunnel operations
type Tunnel interface {
	Start() error
	Stop() error
}

// UpdateCertificatePaths updates certificate paths to be absolute
func UpdateCertificatePaths(cfg *types.AppConfig, baseDir string) error {
	resolvePath := func(path string) string {
		if path == "" || filepath.IsAbs(path) {
			return path
		}
		return filepath.Join(baseDir, path)
	}

	cfg.Config.Auth.CertFile = resolvePath(cfg.Config.Auth.CertFile)
	cfg.Config.Auth.KeyFile = resolvePath(cfg.Config.Auth.KeyFile)
	cfg.Config.Auth.CAFile = resolvePath(cfg.Config.Auth.CAFile)

	return nil
}

// Server represents a tunnel server (point-to-point, one client per instance)
type Server struct {
	config     *types.AppConfig
	manager    interfaces.ConfigManager
	logger     *zap.Logger
	ln         net.Listener
	iface      adapter.Interface
	tlsManager *TLSManager
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.Mutex
	activeConn net.Conn
}

// NewServer creates a new tunnel server
func NewServer(cfg *types.AppConfig, manager interfaces.ConfigManager, logger *zap.Logger) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	var tlsManager *TLSManager
	if cfg.Config.Auth.CertFile != "" && cfg.Config.Auth.KeyFile != "" && cfg.Config.Auth.CAFile != "" {
		tlsConfig := &TLSConfig{
			CertFile:      cfg.Config.Auth.CertFile,
			KeyFile:       cfg.Config.Auth.KeyFile,
			CAFile:        cfg.Config.Auth.CAFile,
			SecurityLevel: SecurityModern,
		}
		var err error
		tlsManager, err = NewTLSManager(tlsConfig)
		if err != nil {
			logger.Error("Failed to create TLS manager, running without TLS", zap.Error(err))
		}
	}

	return &Server{
		config:     cfg,
		manager:    manager,
		logger:     logger,
		ctx:        ctx,
		cancel:     cancel,
		tlsManager: tlsManager,
	}
}

// Start starts the tunnel server
func (s *Server) Start() error {
	adapterOpts := adapter.DefaultOptions()
	iface, err := adapter.New(s.config.Config.Network.Name, adapterOpts)
	if err != nil {
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	if err := iface.Configure(&adapter.Config{
		Name:    s.config.Config.Network.Name,
		Address: s.config.Config.Network.Address,
		MTU:     s.config.Config.Network.MTU,
	}); err != nil {
		iface.Close()
		return fmt.Errorf("failed to configure adapter: %w", err)
	}
	s.iface = iface

	listenAddr := fmt.Sprintf("%s:%d", s.config.Config.Tunnel.ListenAddress, s.config.Config.Tunnel.ListenPort)
	s.logger.Info("Starting tunnel server",
		zap.String("address", listenAddr),
		zap.String("tun", s.config.Config.Network.Name),
		zap.String("tun_address", s.config.Config.Network.Address),
	)

	ln, err := net.Listen("tcp4", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	s.ln = ln

	s.wg.Add(1)
	go s.acceptLoop()

	return nil
}

// acceptLoop accepts connections and handles them one at a time
func (s *Server) acceptLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			conn, err := s.ln.Accept()
			if err != nil {
				if s.ctx.Err() == nil {
					s.logger.Error("Failed to accept connection", zap.Error(err))
				}
				continue
			}

			s.handleConnection(conn)
		}
	}
}

// handleConnection handles a client connection
func (s *Server) handleConnection(conn net.Conn) {
	s.mu.Lock()
	if s.activeConn != nil {
		s.mu.Unlock()
		s.logger.Warn("Rejecting connection - tunnel already active")
		conn.Close()
		return
	}
	s.activeConn = conn
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.activeConn = nil
		s.mu.Unlock()
		conn.Close()
	}()

	remoteAddr := conn.RemoteAddr().String()
	s.logger.Info("Client connected", zap.String("remote", remoteAddr))

	var tunnelConn net.Conn
	var err error

	if s.tlsManager != nil {
		tunnelConn, err = s.tlsManager.WrapConn(conn, true)
		if err != nil {
			s.logger.Error("Failed to wrap connection with TLS", zap.Error(err))
			return
		}
	} else {
		tunnelConn = conn
	}

	adapterConn := NewAdapterWrapper(s.iface)
	transfer := NewTransfer(tunnelConn, adapterConn, s.config, s.logger)

	s.logger.Info("Tunnel established", zap.String("remote", remoteAddr))
	if err := transfer.Start(); err != nil {
		s.logger.Error("Transfer ended", zap.Error(err), zap.String("remote", remoteAddr))
	}
	s.logger.Info("Client disconnected", zap.String("remote", remoteAddr))
}

// Stop stops the tunnel server
func (s *Server) Stop() error {
	s.logger.Info("Stopping tunnel server")

	s.cancel()

	if s.ln != nil {
		s.ln.Close()
	}

	s.mu.Lock()
	if s.activeConn != nil {
		s.activeConn.Close()
	}
	s.mu.Unlock()

	s.wg.Wait()

	if s.iface != nil {
		if err := s.iface.Cleanup(); err != nil {
			s.logger.Error("Failed to cleanup adapter", zap.Error(err))
		}
	}

	return nil
}

// Client represents a tunnel client
type Client struct {
	config       *types.AppConfig
	manager      interfaces.ConfigManager
	logger       *zap.Logger
	iface        adapter.Interface
	tlsManager   *TLSManager
	conn         net.Conn
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	reconnect    bool
	maxRetries   int
	retryDelay   time.Duration
	maxRetryWait time.Duration
}

// NewClient creates a new tunnel client
func NewClient(cfg *types.AppConfig, manager interfaces.ConfigManager, logger *zap.Logger) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	var tlsManager *TLSManager
	if cfg.Config.Auth.CertFile != "" && cfg.Config.Auth.KeyFile != "" && cfg.Config.Auth.CAFile != "" {
		tlsConfig := &TLSConfig{
			CertFile:      cfg.Config.Auth.CertFile,
			KeyFile:       cfg.Config.Auth.KeyFile,
			CAFile:        cfg.Config.Auth.CAFile,
			SecurityLevel: SecurityModern,
			ServerName:    cfg.Config.Tunnel.ServerAddress,
		}
		var err error
		tlsManager, err = NewTLSManager(tlsConfig)
		if err != nil {
			logger.Error("Failed to create TLS manager, running without TLS", zap.Error(err))
		}
	}

	return &Client{
		config:       cfg,
		manager:      manager,
		logger:       logger,
		ctx:          ctx,
		cancel:       cancel,
		tlsManager:   tlsManager,
		reconnect:    true,
		maxRetries:   10,
		retryDelay:   time.Second,
		maxRetryWait: 30 * time.Second,
	}
}

// Start starts the tunnel client
func (c *Client) Start() error {
	adapterOpts := adapter.DefaultOptions()
	iface, err := adapter.New(c.config.Config.Network.Name, adapterOpts)
	if err != nil {
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	if err := iface.Configure(&adapter.Config{
		Name:    c.config.Config.Network.Name,
		Address: c.config.Config.Network.Address,
		MTU:     c.config.Config.Network.MTU,
	}); err != nil {
		iface.Close()
		return fmt.Errorf("failed to configure adapter: %w", err)
	}
	c.iface = iface

	c.logger.Info("Starting tunnel client",
		zap.String("server", fmt.Sprintf("%s:%d", c.config.Config.Tunnel.ServerAddress, c.config.Config.Tunnel.ServerPort)),
		zap.String("tun", c.config.Config.Network.Name),
		zap.String("tun_address", c.config.Config.Network.Address),
	)

	c.wg.Add(1)
	go c.connectLoop()

	return nil
}

// connectLoop manages connection with automatic reconnection
func (c *Client) connectLoop() {
	defer c.wg.Done()

	serverAddr := fmt.Sprintf("%s:%d", c.config.Config.Tunnel.ServerAddress, c.config.Config.Tunnel.ServerPort)
	retryCount := 0
	currentDelay := c.retryDelay

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		conn, err := net.DialTimeout("tcp4", serverAddr, 10*time.Second)
		if err != nil {
			retryCount++
			if retryCount > c.maxRetries {
				c.logger.Error("Max retries exceeded, stopping reconnection",
					zap.Int("attempts", retryCount),
					zap.Error(err))
				return
			}

			c.logger.Warn("Connection failed, retrying",
				zap.Int("attempt", retryCount),
				zap.Duration("delay", currentDelay),
				zap.Error(err))

			select {
			case <-time.After(currentDelay):
			case <-c.ctx.Done():
				return
			}

			currentDelay = time.Duration(float64(currentDelay) * 1.5)
			if currentDelay > c.maxRetryWait {
				currentDelay = c.maxRetryWait
			}
			continue
		}

		retryCount = 0
		currentDelay = c.retryDelay

		var tunnelConn net.Conn
		if c.tlsManager != nil {
			tunnelConn, err = c.tlsManager.WrapConn(conn, false)
			if err != nil {
				conn.Close()
				c.logger.Error("TLS handshake failed", zap.Error(err))
				continue
			}
		} else {
			tunnelConn = conn
		}

		c.conn = conn
		c.logger.Info("Connected to server", zap.String("address", serverAddr))

		adapterConn := NewAdapterWrapper(c.iface)
		transfer := NewTransfer(tunnelConn, adapterConn, c.config, c.logger)

		c.logger.Info("Tunnel established")
		if err := transfer.Start(); err != nil {
			c.logger.Debug("Transfer ended", zap.Error(err))
		}

		c.conn = nil
		conn.Close()
		c.logger.Info("Disconnected from server")

		if !c.reconnect {
			return
		}

		select {
		case <-time.After(time.Second):
		case <-c.ctx.Done():
			return
		}
	}
}

// Stop stops the tunnel client
func (c *Client) Stop() error {
	c.logger.Info("Stopping tunnel client")

	c.cancel()

	if c.conn != nil {
		c.conn.Close()
	}

	c.wg.Wait()

	if c.iface != nil {
		if err := c.iface.Cleanup(); err != nil {
			c.logger.Error("Failed to cleanup adapter", zap.Error(err))
		}
	}

	return nil
}

// tunnelImpl represents a tunnel implementation (for programmatic use)
type tunnelImpl struct {
	conn    net.Conn
	adapter adapter.Interface
	config  *types.AppConfig
	monitor *monitor.Monitor
}

// New creates a new tunnel
func New(conn net.Conn, adapter adapter.Interface, cfg *types.AppConfig, monitor *monitor.Monitor) (Tunnel, error) {
	return &tunnelImpl{
		conn:    conn,
		adapter: adapter,
		config:  cfg,
		monitor: monitor,
	}, nil
}

// Start starts the tunnel
func (t *tunnelImpl) Start() error {
	adapterConn := NewAdapterWrapper(t.adapter)
	transfer := NewTransfer(t.conn, adapterConn, t.config, nil)
	return transfer.Start()
}

// Stop stops the tunnel
func (t *tunnelImpl) Stop() error {
	if t.conn != nil {
		t.conn.Close()
	}
	if t.adapter != nil {
		t.adapter.Close()
	}
	return nil
}
