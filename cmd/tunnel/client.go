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

// Client represents a tunnel client
type Client struct {
	config  *types.AppConfig
	manager interfaces.ConfigManager
	logger  *zap.Logger
	tunnel  tunnel.Tunnel
}

// NewClient creates a new tunnel client
func NewClient(cfg *types.AppConfig, manager interfaces.ConfigManager, logger *zap.Logger) (*Client, error) {
	t := tunnel.NewClient(cfg, manager, logger)
	return &Client{
		config:  cfg,
		manager: manager,
		logger:  logger,
		tunnel:  t,
	}, nil
}

// Start starts the tunnel client
func (c *Client) Start() error {
	// Start tunnel
	if err := c.tunnel.Start(); err != nil {
		return fmt.Errorf("failed to start tunnel: %w", err)
	}

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal
	sig := <-sigChan
	c.logger.Info("Received signal", zap.String("signal", sig.String()))

	// Stop tunnel
	if err := c.tunnel.Stop(); err != nil {
		c.logger.Error("Failed to stop tunnel", zap.Error(err))
	}

	return nil
}

// Stop stops the tunnel client
func (c *Client) Stop() error {
	return c.tunnel.Stop()
}

// Run runs the tunnel client
func (c *Client) Run(ctx context.Context) error {
	// Start client
	if err := c.Start(); err != nil {
		return fmt.Errorf("failed to start client: %w", err)
	}

	// Wait for context cancellation
	<-ctx.Done()

	// Stop client
	if err := c.Stop(); err != nil {
		return fmt.Errorf("failed to stop client: %w", err)
	}

	return nil
}
