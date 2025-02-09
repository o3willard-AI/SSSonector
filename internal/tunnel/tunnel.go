package tunnel

import (
	"path/filepath"

	"github.com/o3willard-AI/SSSonector/internal/config"
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
	cfg.Tunnel.CertFile = resolvePath(cfg.Tunnel.CertFile)
	cfg.Tunnel.KeyFile = resolvePath(cfg.Tunnel.KeyFile)
	cfg.Tunnel.CAFile = resolvePath(cfg.Tunnel.CAFile)

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
		zap.String("address", s.config.Tunnel.ListenAddress),
		zap.Int("port", s.config.Tunnel.ListenPort),
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
		zap.String("server", c.config.Tunnel.ServerAddress),
		zap.Int("port", c.config.Tunnel.ServerPort),
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
