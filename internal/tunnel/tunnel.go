package tunnel

import (
	"fmt"
	"net"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/adapter"
	"github.com/o3willard-AI/SSSonector/internal/config/interfaces"
	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/o3willard-AI/SSSonector/internal/startup"
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

func (s State) String() string {
	switch s {
	case StateStopped:
		return "Stopped"
	case StateStarting:
		return "Starting"
	case StateRunning:
		return "Running"
	case StateStopping:
		return "Stopping"
	default:
		return "Unknown"
	}
}

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

	// Create startup logger
	startupLogger := startup.NewStartupLogger(s.logger, s.config.Config.Logging)

	// Pre-startup phase
	startupLogger.SetPhase(types.StartupPhasePreStartup)
	startupLogger.LogCheckpoint("Starting tunnel server", map[string]interface{}{
		"mode": s.config.Config.Mode,
	})

	s.setState(StateStarting)

	// Initialization phase
	startupLogger.SetPhase(types.StartupPhaseInitialization)

	// Create adapter options
	var opts *adapter.Options
	err := startupLogger.LogOperation(types.StartupComponentAdapter, "Create adapter options", func() error {
		opts = adapter.DefaultOptions()
		opts.Name = s.config.Config.Network.Interface
		opts.MTU = s.config.Config.Network.MTU
		opts.Address = s.config.Config.Network.Address

		if s.config.Adapter != nil {
			opts.RetryAttempts = s.config.Adapter.RetryAttempts
			opts.RetryDelay = time.Duration(s.config.Adapter.RetryDelay) * time.Millisecond
			opts.CleanupTimeout = time.Duration(s.config.Adapter.CleanupTimeout) * time.Millisecond
		}
		return nil
	}, map[string]interface{}{
		"interface": s.config.Config.Network.Interface,
		"mtu":       s.config.Config.Network.MTU,
		"address":   s.config.Config.Network.Address,
	})
	if err != nil {
		s.setState(StateStopped)
		return err
	}

	// Create TUN adapter
	err = startupLogger.LogOperation(types.StartupComponentAdapter, "Create TUN adapter", func() error {
		var adapterErr error
		s.adapter, adapterErr = adapter.NewTUNAdapter(opts)
		return adapterErr
	}, nil)
	if err != nil {
		s.setState(StateStopped)
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	// Wait for adapter to be ready
	err = startupLogger.LogOperation(types.StartupComponentAdapter, "Wait for adapter ready", func() error {
		for i := 0; i < 10; i++ {
			status := s.adapter.GetStatus()
			startupLogger.LogResourceState("adapter", map[string]interface{}{
				"state":     status.State.String(),
				"lastError": status.LastError,
				"attempt":   i + 1,
				"interface": opts.Name,
				"address":   opts.Address,
			})

			if status.State == adapter.StateReady {
				return nil
			}

			if status.State == adapter.StateError {
				if cleanupErr := s.adapter.Cleanup(); cleanupErr != nil {
					s.logger.Error("Failed to cleanup adapter", zap.Error(cleanupErr))
				}
				return fmt.Errorf("adapter in error state: %v", status.LastError)
			}

			time.Sleep(time.Second)
		}

		if s.adapter.GetState() != adapter.StateReady {
			if cleanupErr := s.adapter.Cleanup(); cleanupErr != nil {
				s.logger.Error("Failed to cleanup adapter", zap.Error(cleanupErr))
			}
			return fmt.Errorf("adapter failed to reach ready state")
		}
		return nil
	}, nil)
	if err != nil {
		s.setState(StateStopped)
		return err
	}

	// Connection phase
	startupLogger.SetPhase(types.StartupPhaseConnection)

	// Create TCP listener
	addr := fmt.Sprintf("%s:%d",
		s.config.Config.Tunnel.ListenAddress,
		s.config.Config.Tunnel.ListenPort,
	)
	err = startupLogger.LogOperation(types.StartupComponentNetwork, "Create TCP listener", func() error {
		var listenerErr error
		s.listener, listenerErr = net.Listen("tcp", addr)
		if listenerErr != nil {
			if cleanupErr := s.adapter.Cleanup(); cleanupErr != nil {
				s.logger.Error("Failed to cleanup adapter", zap.Error(cleanupErr))
			}
			return fmt.Errorf("failed to create listener: %w", listenerErr)
		}
		return nil
	}, map[string]interface{}{
		"address": addr,
	})
	if err != nil {
		s.setState(StateStopped)
		return err
	}

	s.setState(StateRunning)
	startupLogger.LogCheckpoint("Tunnel server started", map[string]interface{}{
		"state":   StateRunning.String(),
		"address": addr,
	})

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

	// Create startup logger
	startupLogger := startup.NewStartupLogger(s.logger, s.config.Config.Logging)

	startupLogger.SetPhase(types.StartupPhasePreStartup)
	startupLogger.LogCheckpoint("Stopping tunnel server", map[string]interface{}{
		"state": s.getState(),
	})

	s.setState(StateStopping)

	// Close listener
	if s.listener != nil {
		err := startupLogger.LogOperation(types.StartupComponentNetwork, "Close listener", func() error {
			return s.listener.Close()
		}, nil)
		if err != nil {
			s.logger.Error("Failed to close listener", zap.Error(err))
		}
	}

	// Wait for transfers to complete
	startupLogger.LogOperation(types.StartupComponentConnection, "Wait for transfers", func() error {
		s.transfers.Wait()
		return nil
	}, nil)

	// Cleanup adapter
	if s.adapter != nil {
		err := startupLogger.LogOperation(types.StartupComponentAdapter, "Cleanup adapter", func() error {
			return s.adapter.Cleanup()
		}, nil)
		if err != nil {
			s.logger.Error("Failed to cleanup adapter", zap.Error(err))
		}
	}

	s.setState(StateStopped)
	startupLogger.LogCheckpoint("Tunnel server stopped", map[string]interface{}{
		"state": StateStopped,
	})

	return nil
}

// setState atomically sets the tunnel state
func (s *Server) setState(state State) {
	atomic.StoreInt32((*int32)(&s.state), int32(state))
	s.logger.Info(fmt.Sprintf("State: %s", state.String()))
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
