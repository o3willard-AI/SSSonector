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
	"github.com/o3willard-AI/SSSonector/internal/pool"
	"go.uber.org/zap"
)

// Tunnel defines the interface for tunnel operations
type Tunnel interface {
	// Start starts the tunnel
	Start() error
	// Stop stops the tunnel
	Stop() error
}

// UpdateCertificatePaths updates certificate paths to be absolute
func UpdateCertificatePaths(cfg *types.AppConfig, baseDir string) error {
	// Helper function to resolve path
	resolvePath := func(path string) string {
		if path == "" || filepath.IsAbs(path) {
			return path
		}
		return filepath.Join(baseDir, path)
	}

	// Update certificate paths
	cfg.Config.Auth.CertFile = resolvePath(cfg.Config.Auth.CertFile)
	cfg.Config.Auth.KeyFile = resolvePath(cfg.Config.Auth.KeyFile)
	cfg.Config.Auth.CAFile = resolvePath(cfg.Config.Auth.CAFile)

	return nil
}

// Server represents a tunnel server
type Server struct {
	config  *types.AppConfig
	manager interfaces.ConfigManager
	logger  *zap.Logger
	pool    *pool.Pool
	ln      net.Listener
	wg      sync.WaitGroup
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewServer creates a new tunnel server
func NewServer(cfg *types.AppConfig, manager interfaces.ConfigManager, logger *zap.Logger) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	// Create connection pool
	poolConfig := &pool.Config{
		IdleTimeout:   time.Minute * 5,
		MaxIdle:       100,
		MaxActive:     1000,
		RetryInterval: time.Second * 5,
		MaxRetries:    3,
	}

	// Connection factory for the pool
	factory := func(ctx context.Context) (net.Conn, error) {
		// Create new connection
		conn, err := net.Dial("tcp", cfg.Config.Network.Name)
		if err != nil {
			return nil, err
		}
		return conn, nil
	}

	return &Server{
		config:  cfg,
		manager: manager,
		logger:  logger,
		pool:    pool.NewPool(factory, poolConfig, logger),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start starts the tunnel server
func (s *Server) Start() error {
	// Create adapter first
	adapterOpts := adapter.DefaultOptions()
	iface, err := adapter.New(s.config.Config.Network.Name, adapterOpts)
	if err != nil {
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	// Configure adapter
	if err := iface.Configure(&adapter.Config{
		Name:    s.config.Config.Network.Name,
		Address: s.config.Config.Network.Address,
		MTU:     s.config.Config.Network.MTU,
	}); err != nil {
		return fmt.Errorf("failed to configure adapter: %w", err)
	}

	// Start listener
	listenAddr := fmt.Sprintf("%s:%d", s.config.Config.Tunnel.ListenAddress, s.config.Config.Tunnel.ListenPort)
	s.logger.Info("Starting tunnel server",
		zap.String("address", listenAddr),
	)

	ln, err := net.Listen("tcp4", listenAddr) // Force IPv4
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	s.ln = ln

	// Accept connections
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		for {
			select {
			case <-s.ctx.Done():
				return
			default:
				conn, err := ln.Accept()
				if err != nil {
					if s.ctx.Err() == nil {
						s.logger.Error("Failed to accept connection", zap.Error(err))
					}
					continue
				}

				s.wg.Add(1)
				go func() {
					defer s.wg.Done()
					s.handleConnection(conn)
				}()
			}
		}
	}()

	return nil
}

// Stop stops the tunnel server
func (s *Server) Stop() error {
	s.logger.Info("Stopping tunnel server")

	// Stop accepting new connections
	s.cancel()
	if s.ln != nil {
		s.ln.Close()
	}

	// Close connection pool
	s.pool.Close()

	// Wait for all connections to finish
	s.wg.Wait()

	return nil
}

// handleConnection handles a client connection
func (s *Server) handleConnection(clientConn net.Conn) {
	defer clientConn.Close()

	// Get connection from pool
	conn, err := s.pool.Get(s.ctx)
	if err != nil {
		s.logger.Error("Failed to get connection from pool", zap.Error(err))
		return
	}
	defer s.pool.Put(conn)

	// Create transfer
	transfer := NewTransfer(clientConn, conn, s.config, s.logger)
	if err := transfer.Start(); err != nil {
		s.logger.Error("Transfer failed", zap.Error(err))
	}
}

// Client represents a tunnel client
type Client struct {
	config  *types.AppConfig
	manager interfaces.ConfigManager
	logger  *zap.Logger
	pool    *pool.Pool
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewClient creates a new tunnel client
func NewClient(cfg *types.AppConfig, manager interfaces.ConfigManager, logger *zap.Logger) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	// Create connection pool
	poolConfig := &pool.Config{
		IdleTimeout:   time.Minute * 5,
		MaxIdle:       100,
		MaxActive:     1000,
		RetryInterval: time.Second * 5,
		MaxRetries:    3,
	}

	// Connection factory for the pool
	factory := func(ctx context.Context) (net.Conn, error) {
		// Create new connection to server
		serverAddr := fmt.Sprintf("%s:%d", cfg.Config.Tunnel.ServerAddress, cfg.Config.Tunnel.ServerPort)
		conn, err := net.Dial("tcp4", serverAddr) // Force IPv4
		if err != nil {
			return nil, fmt.Errorf("failed to connect to server: %w", err)
		}
		return conn, nil
	}

	return &Client{
		config:  cfg,
		manager: manager,
		logger:  logger,
		pool:    pool.NewPool(factory, poolConfig, logger),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Start starts the tunnel client
func (c *Client) Start() error {
	// Create adapter with default options
	adapterOpts := adapter.DefaultOptions()
	iface, err := adapter.New(c.config.Config.Network.Name, adapterOpts)
	if err != nil {
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	// Configure adapter
	if err := iface.Configure(&adapter.Config{
		Name:    c.config.Config.Network.Name,
		Address: c.config.Config.Network.Address,
		MTU:     c.config.Config.Network.MTU,
	}); err != nil {
		return fmt.Errorf("failed to configure adapter: %w", err)
	}

	// Get connection from pool
	conn, err := c.pool.Get(c.ctx)
	if err != nil {
		return err
	}
	defer c.pool.Put(conn)

	// Create tunnel
	tunnel, err := New(conn, iface, c.config, nil)
	if err != nil {
		return fmt.Errorf("failed to create tunnel: %w", err)
	}

	return tunnel.Start()
}

// Stop stops the tunnel client
func (c *Client) Stop() error {
	c.logger.Info("Stopping tunnel client")

	// Cancel context
	c.cancel()

	// Close connection pool
	c.pool.Close()

	return nil
}

// tunnelImpl represents a tunnel implementation
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
	// Wrap adapter in net.Conn interface
	adapterConn := NewAdapterWrapper(t.adapter)

	// Create transfer to handle data between connection and adapter
	transfer := NewTransfer(t.conn, adapterConn, t.config, nil)
	return transfer.Start()
}

// Stop stops the tunnel
func (t *tunnelImpl) Stop() error {
	if err := t.conn.Close(); err != nil {
		return err
	}
	return t.adapter.Close()
}
