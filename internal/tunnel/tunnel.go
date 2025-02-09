package tunnel

import (
	"net"
	"path/filepath"

	"github.com/o3willard-AI/SSSonector/internal/adapter"
	"github.com/o3willard-AI/SSSonector/internal/config"
	"github.com/o3willard-AI/SSSonector/internal/monitor"
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
func UpdateCertificatePaths(cfg *config.AppConfig, baseDir string) error {
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
	config  *config.AppConfig
	manager config.ConfigManager
	logger  *zap.Logger
}

// NewServer creates a new tunnel server
func NewServer(cfg *config.AppConfig, manager config.ConfigManager, logger *zap.Logger) *Server {
	return &Server{
		config:  cfg,
		manager: manager,
		logger:  logger,
	}
}

// Start starts the tunnel server
func (s *Server) Start() error {
	s.logger.Info("Starting tunnel server",
		zap.String("address", s.config.Config.Network.Interface),
		zap.Int("port", 8080),
	)

	// TODO: Implement server start
	return nil
}

// Stop stops the tunnel server
func (s *Server) Stop() error {
	s.logger.Info("Stopping tunnel server")

	// TODO: Implement server stop
	return nil
}

// Client represents a tunnel client
type Client struct {
	config  *config.AppConfig
	manager config.ConfigManager
	logger  *zap.Logger
}

// NewClient creates a new tunnel client
func NewClient(cfg *config.AppConfig, manager config.ConfigManager, logger *zap.Logger) *Client {
	return &Client{
		config:  cfg,
		manager: manager,
		logger:  logger,
	}
}

// Start starts the tunnel client
func (c *Client) Start() error {
	c.logger.Info("Starting tunnel client",
		zap.String("server", c.config.Config.Network.Interface),
		zap.Int("port", 8080),
	)

	// TODO: Implement client start
	return nil
}

// Stop stops the tunnel client
func (c *Client) Stop() error {
	c.logger.Info("Stopping tunnel client")

	// TODO: Implement client stop
	return nil
}

// tunnelImpl represents a tunnel implementation
type tunnelImpl struct {
	conn    net.Conn
	adapter adapter.Interface
	config  *config.AppConfig
	monitor *monitor.Monitor
}

// New creates a new tunnel
func New(conn net.Conn, adapter adapter.Interface, cfg *config.AppConfig, monitor *monitor.Monitor) (Tunnel, error) {
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
