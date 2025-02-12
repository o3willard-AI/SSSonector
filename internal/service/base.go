package service

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/o3willard-AI/SSSonector/internal/tunnel"
	"go.uber.org/zap"
)

// Base represents a base service implementation
type Base struct {
	cfg    *types.AppConfig
	logger *zap.Logger
	tunnel tunnel.Tunnel
}

// NewBase creates a new base service
func NewBase(cfg *types.AppConfig, logger *zap.Logger) *Base {
	return &Base{
		cfg:    cfg,
		logger: logger,
	}
}

// Start starts the service
func (b *Base) Start() error {
	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start tunnel based on mode
	var err error
	switch b.cfg.Config.Mode {
	case "server":
		b.tunnel = tunnel.NewServer(b.cfg, nil, b.logger)
	case "client":
		b.tunnel = tunnel.NewClient(b.cfg, nil, b.logger)
	default:
		return fmt.Errorf("invalid mode: %s", b.cfg.Config.Mode)
	}

	// Start tunnel
	if err = b.tunnel.Start(); err != nil {
		return fmt.Errorf("failed to start tunnel: %w", err)
	}

	// Wait for signal
	<-sigChan

	// Stop tunnel
	if err = b.tunnel.Stop(); err != nil {
		return fmt.Errorf("failed to stop tunnel: %w", err)
	}

	return nil
}

// Stop stops the service
func (b *Base) Stop() error {
	if b.tunnel != nil {
		return b.tunnel.Stop()
	}
	return nil
}
