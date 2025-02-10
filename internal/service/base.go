package service

import (
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/o3willard-AI/SSSonector/internal/tunnel"
	"go.uber.org/zap"
)

// BaseService provides a base implementation of the Service interface
type BaseService struct {
	cfg     *types.AppConfig
	options ServiceOptions
	status  ServiceStatus
	metrics ServiceMetrics
	logger  *zap.Logger
	server  *tunnel.Server
	client  *tunnel.Client
}

// NewBaseService creates a new base service
func NewBaseService(cfg *types.AppConfig, opts ServiceOptions) (*BaseService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	svc := &BaseService{
		cfg:     cfg,
		options: opts,
		logger:  logger,
		status: ServiceStatus{
			Name:      opts.Name,
			State:     "stopped",
			StartTime: time.Time{},
			Mode:      string(cfg.Config.Mode),
			Version:   "1.0.0", // TODO: Get from build info
		},
		metrics: ServiceMetrics{
			Platform: runtime.GOOS,
		},
	}

	return svc, nil
}

// Start starts the service
func (b *BaseService) Start() error {
	if b.status.State == "running" {
		return NewServiceError(ErrAlreadyRunning, "Service is already running")
	}

	b.status.State = "starting"
	b.status.StartTime = time.Now()
	b.status.PID = os.Getpid()

	// Create and start tunnel based on mode
	switch b.cfg.Config.Mode {
	case types.ModeServer:
		b.server = tunnel.NewServer(b.cfg, nil, b.logger)
		if err := b.server.Start(); err != nil {
			b.status.State = "stopped"
			return fmt.Errorf("failed to start server: %w", err)
		}
	case types.ModeClient:
		b.client = tunnel.NewClient(b.cfg, nil, b.logger)
		if err := b.client.Start(); err != nil {
			b.status.State = "stopped"
			return fmt.Errorf("failed to start client: %w", err)
		}
	default:
		b.status.State = "stopped"
		return fmt.Errorf("invalid mode: %s", b.cfg.Config.Mode)
	}

	b.status.State = "running"
	return nil
}

// Stop stops the service
func (b *BaseService) Stop() error {
	if b.status.State != "running" {
		return NewServiceError(ErrNotRunning, "Service is not running")
	}

	b.status.State = "stopping"

	// Stop tunnel based on mode
	switch b.cfg.Config.Mode {
	case types.ModeServer:
		if b.server != nil {
			if err := b.server.Stop(); err != nil {
				b.status.State = "stopped"
				return fmt.Errorf("failed to stop server: %w", err)
			}
		}
	case types.ModeClient:
		if b.client != nil {
			if err := b.client.Stop(); err != nil {
				b.status.State = "stopped"
				return fmt.Errorf("failed to stop client: %w", err)
			}
		}
	}

	b.status.State = "stopped"
	return nil
}

// Reload reloads the service configuration
func (b *BaseService) Reload() error {
	if b.status.State != "running" {
		return NewServiceError(ErrNotRunning, "Service is not running")
	}

	b.status.State = "reloading"
	b.status.LastReload = time.Now()

	// Stop existing tunnel
	if err := b.Stop(); err != nil {
		return fmt.Errorf("failed to stop service for reload: %w", err)
	}

	// Start new tunnel with updated config
	if err := b.Start(); err != nil {
		return fmt.Errorf("failed to restart service after reload: %w", err)
	}

	b.status.State = "running"
	return nil
}

// Status returns the current service status
func (b *BaseService) Status() (*ServiceStatus, error) {
	if b.status.State == "running" {
		b.updateMetrics()
	}
	return &b.status, nil
}

// Metrics returns the current service metrics
func (b *BaseService) Metrics() (*ServiceMetrics, error) {
	if b.status.State != "running" {
		return nil, NewServiceError(ErrNotRunning, "Service is not running")
	}

	b.updateMetrics()
	return &b.metrics, nil
}

// Health performs a health check
func (b *BaseService) Health() error {
	if b.status.State != "running" {
		return NewServiceError(ErrNotRunning, "Service is not running")
	}

	// Check tunnel health based on mode
	switch b.cfg.Config.Mode {
	case types.ModeServer:
		if b.server == nil {
			return fmt.Errorf("server not initialized")
		}
		// TODO: Add server health checks
	case types.ModeClient:
		if b.client == nil {
			return fmt.Errorf("client not initialized")
		}
		// TODO: Add client health checks
	}

	return nil
}

// ExecuteCommand executes a service command
func (b *BaseService) ExecuteCommand(cmd ServiceCommand, args map[string]interface{}) (*ServiceResponse, error) {
	switch cmd {
	case CmdStatus:
		status, err := b.Status()
		if err != nil {
			return nil, err
		}
		return &ServiceResponse{Success: true, Data: status}, nil

	case CmdReload:
		if err := b.Reload(); err != nil {
			return nil, err
		}
		return &ServiceResponse{Success: true, Message: "Configuration reloaded"}, nil

	case CmdMetrics:
		metrics, err := b.Metrics()
		if err != nil {
			return nil, err
		}
		return &ServiceResponse{Success: true, Data: metrics}, nil

	case CmdHealth:
		if err := b.Health(); err != nil {
			return nil, err
		}
		return &ServiceResponse{Success: true, Message: "Service is healthy"}, nil

	default:
		return nil, NewServiceError(ErrInvalidCommand, fmt.Sprintf("Unknown command: %s", cmd))
	}
}

// updateMetrics updates service metrics
func (b *BaseService) updateMetrics() {
	if b.status.State == "running" {
		b.metrics.UptimeSeconds = int64(time.Since(b.status.StartTime).Seconds())
		// TODO: Update other metrics
	}
}
