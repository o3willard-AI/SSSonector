package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/interfaces"
	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/o3willard-AI/SSSonector/internal/tunnel"
	"go.uber.org/zap"
)

const (
	maxStartRetries = 3
	retryDelay      = 5 * time.Second
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

// preStartChecks performs pre-start validation
func (s *Server) preStartChecks() error {
	// Check TUN module
	if _, err := os.Stat("/dev/net/tun"); err != nil {
		return fmt.Errorf("TUN device not available: %w", err)
	}

	// Check certificates
	certFile := s.config.Config.Auth.CertFile
	keyFile := s.config.Config.Auth.KeyFile
	caFile := s.config.Config.Auth.CAFile

	if _, err := os.Stat(certFile); err != nil {
		return fmt.Errorf("certificate file not found: %w", err)
	}
	if _, err := os.Stat(keyFile); err != nil {
		return fmt.Errorf("key file not found: %w", err)
	}
	if _, err := os.Stat(caFile); err != nil {
		return fmt.Errorf("CA certificate file not found: %w", err)
	}

	return nil
}

// Start starts the tunnel server with retry logic
func (s *Server) Start() error {
	s.logger.Info("Starting tunnel server")

	// Perform pre-start checks
	if err := s.preStartChecks(); err != nil {
		return fmt.Errorf("pre-start checks failed: %w", err)
	}

	var lastErr error
	for i := 0; i < maxStartRetries; i++ {
		if i > 0 {
			s.logger.Info("Retrying server start",
				zap.Int("attempt", i+1),
				zap.Int("max_attempts", maxStartRetries),
			)
			time.Sleep(retryDelay)
		}

		if err := s.tunnel.Start(); err != nil {
			lastErr = fmt.Errorf("failed to start tunnel: %w", err)
			s.logger.Error("Start attempt failed",
				zap.Error(err),
				zap.Int("attempt", i+1),
			)
			continue
		}

		s.logger.Info("Server started successfully")
		return nil
	}

	return fmt.Errorf("failed to start server after %d attempts: %w", maxStartRetries, lastErr)
}

// Stop stops the tunnel server
func (s *Server) Stop() error {
	s.logger.Info("Stopping tunnel server")

	// Set up a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Create a channel to signal completion
	done := make(chan error, 1)

	go func() {
		done <- s.tunnel.Stop()
	}()

	// Wait for either completion or timeout
	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("failed to stop tunnel: %w", err)
		}
		s.logger.Info("Server stopped successfully")
		return nil
	case <-ctx.Done():
		return fmt.Errorf("timeout waiting for server to stop")
	}
}

// Run runs the tunnel server
func (s *Server) Run(ctx context.Context) error {
	// Start server
	if err := s.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for either context cancellation or signal
	select {
	case <-ctx.Done():
		s.logger.Info("Context cancelled")
	case sig := <-sigChan:
		s.logger.Info("Received signal", zap.String("signal", sig.String()))
	}

	// Stop server
	if err := s.Stop(); err != nil {
		return fmt.Errorf("failed to stop server: %w", err)
	}

	return nil
}
