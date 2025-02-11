package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/o3willard-AI/SSSonector/internal/config/interfaces"
	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/o3willard-AI/SSSonector/internal/tunnel"
	"go.uber.org/zap"
)

// Server represents a tunnel server
type Server struct {
	config  *types.AppConfig
	manager interfaces.ConfigManager
	logger  *zap.Logger
	tunnel  tunnel.Tunnel
}

// NewServer creates a new tunnel server
func NewServer(cfg *types.AppConfig, manager interfaces.ConfigManager, logger *zap.Logger) (*Server, error) {
	t := tunnel.NewServer(cfg, manager, logger)
	return &Server{
		config:  cfg,
		manager: manager,
		logger:  logger,
		tunnel:  t,
	}, nil
}

// Start starts the tunnel server
func (s *Server) Start() error {
	// Start tunnel
	if err := s.tunnel.Start(); err != nil {
		return fmt.Errorf("failed to start tunnel: %w", err)
	}

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal
	sig := <-sigChan
	s.logger.Info("Received signal", zap.String("signal", sig.String()))

	// Stop tunnel
	if err := s.tunnel.Stop(); err != nil {
		s.logger.Error("Failed to stop tunnel", zap.Error(err))
	}

	return nil
}

// Stop stops the tunnel server
func (s *Server) Stop() error {
	return s.tunnel.Stop()
}

// Run runs the tunnel server
func (s *Server) Run(ctx context.Context) error {
	// Start server
	if err := s.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	// Wait for context cancellation
	<-ctx.Done()

	// Stop server
	if err := s.Stop(); err != nil {
		return fmt.Errorf("failed to stop server: %w", err)
	}

	return nil
}
