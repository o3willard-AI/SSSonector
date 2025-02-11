package tunnel

import (
	"fmt"
	"net"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/o3willard-AI/SSSonector/internal/adapter"
	"github.com/o3willard-AI/SSSonector/internal/config/interfaces"
	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"go.uber.org/zap"
)

// State represents the tunnel state
type State int32

const (
	// StateStopped indicates the tunnel is stopped
	StateStopped State = iota
	// StateStarting indicates the tunnel is starting
	StateStarting
	// StateRunning indicates the tunnel is running
	StateRunning
	// StateStopping indicates the tunnel is stopping
	StateStopping
)

// Tunnel represents a network tunnel
type Tunnel interface {
	Start() error
	Stop() error
}

// Server represents a tunnel server
type Server struct {
	config  *types.AppConfig
	manager interfaces.ConfigManager
	logger  *zap.Logger
	state   State

	adapter   adapter.AdapterInterface
	listener  net.Listener
	transfers sync.WaitGroup
	mu        sync.Mutex
}

// NewServer creates a new tunnel server
func NewServer(cfg *types.AppConfig, manager interfaces.ConfigManager, logger *zap.Logger) Tunnel {
	return &Server{
		config:  cfg,
		manager: manager,
		logger:  logger,
		state:   StateStopped,
	}
}

// UpdateCertificatePaths updates certificate paths to be relative to config directory
func UpdateCertificatePaths(cfg *types.AppConfig, configDir string) error {
	if cfg.Config == nil || cfg.Config.Auth.CertFile == "" {
		return fmt.Errorf("invalid configuration: missing certificate paths")
	}

	// Update certificate paths to be relative to config directory
	cfg.Config.Auth.CertFile = filepath.Join(configDir, cfg.Config.Auth.CertFile)
	cfg.Config.Auth.KeyFile = filepath.Join(configDir, cfg.Config.Auth.KeyFile)
	cfg.Config.Auth.CAFile = filepath.Join(configDir, cfg.Config.Auth.CAFile)

	return nil
}

// Start starts the tunnel server
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.getState() != StateStopped {
		return fmt.Errorf("tunnel is not in stopped state")
	}

	s.setState(StateStarting)
	s.logger.Info("Starting tunnel server")

	// Create TUN adapter
	var err error
	s.adapter, err = adapter.FromConfig(s.config)
	if err != nil {
		s.setState(StateStopped)
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	// Create TCP listener
	addr := fmt.Sprintf("%s:%d",
		s.config.Config.Tunnel.ListenAddress,
		s.config.Config.Tunnel.ListenPort,
	)
	s.listener, err = net.Listen("tcp", addr)
	if err != nil {
		s.setState(StateStopped)
		if err := s.adapter.Cleanup(); err != nil {
			s.logger.Error("Failed to cleanup adapter", zap.Error(err))
		}
		return fmt.Errorf("failed to create listener: %w", err)
	}

	s.setState(StateRunning)
	s.logger.Info("Tunnel server started",
		zap.String("address", addr),
	)

	// Accept connections
	go s.acceptConnections()

	return nil
}

// Stop stops the tunnel server
func (s *Server) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.getState() != StateRunning {
		return fmt.Errorf("tunnel is not in running state")
	}

	s.setState(StateStopping)
	s.logger.Info("Stopping tunnel server")

	// Close listener
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			s.logger.Error("Failed to close listener", zap.Error(err))
		}
	}

	// Wait for transfers to complete
	s.transfers.Wait()

	// Cleanup adapter
	if s.adapter != nil {
		if err := s.adapter.Cleanup(); err != nil {
			s.logger.Error("Failed to cleanup adapter", zap.Error(err))
		}
	}

	s.setState(StateStopped)
	s.logger.Info("Tunnel server stopped")

	return nil
}

// setState atomically sets the tunnel state
func (s *Server) setState(state State) {
	atomic.StoreInt32((*int32)(&s.state), int32(state))
}

// getState atomically gets the tunnel state
func (s *Server) getState() State {
	return State(atomic.LoadInt32((*int32)(&s.state)))
}

// acceptConnections accepts incoming connections
func (s *Server) acceptConnections() {
	for s.getState() == StateRunning {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.getState() != StateRunning {
				return
			}
			s.logger.Error("Failed to accept connection", zap.Error(err))
			continue
		}

		s.logger.Debug("Accepted connection",
			zap.String("remote", conn.RemoteAddr().String()),
		)

		s.transfers.Add(1)
		go func() {
			defer s.transfers.Done()
			defer conn.Close()

			transfer := NewTransfer(
				conn,
				NewAdapterWrapper(s.adapter),
				s.config,
				s.logger,
			)
			if err := transfer.Start(); err != nil {
				s.logger.Error("Transfer failed", zap.Error(err))
			}
		}()
	}
}
