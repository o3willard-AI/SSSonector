package tunnel

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/o3willard-AI/SSSonector/internal/adapter"
	"github.com/o3willard-AI/SSSonector/internal/config/interfaces"
	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"go.uber.org/zap"
)

// Client represents a tunnel client
type Client struct {
	config  *types.AppConfig
	manager interfaces.ConfigManager
	logger  *zap.Logger
	state   State

	adapter   adapter.AdapterInterface
	conn      net.Conn
	transfers sync.WaitGroup
	mu        sync.Mutex
}

// NewClient creates a new tunnel client
func NewClient(cfg *types.AppConfig, manager interfaces.ConfigManager, logger *zap.Logger) Tunnel {
	return &Client{
		config:  cfg,
		manager: manager,
		logger:  logger,
		state:   StateStopped,
	}
}

// Start starts the tunnel client
func (c *Client) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.getState() != StateStopped {
		return fmt.Errorf("tunnel is not in stopped state")
	}

	c.setState(StateStarting)
	c.logger.Info("Starting tunnel client")

	// Create TUN adapter
	var err error
	c.adapter, err = adapter.FromConfig(c.config)
	if err != nil {
		c.setState(StateStopped)
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	// Connect to server
	addr := fmt.Sprintf("%s:%d",
		c.config.Config.Tunnel.ServerAddress,
		c.config.Config.Tunnel.ServerPort,
	)
	c.conn, err = net.Dial("tcp", addr)
	if err != nil {
		c.setState(StateStopped)
		if err := c.adapter.Cleanup(); err != nil {
			c.logger.Error("Failed to cleanup adapter", zap.Error(err))
		}
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	c.setState(StateRunning)
	c.logger.Info("Tunnel client started",
		zap.String("server", addr),
	)

	// Start transfer
	c.transfers.Add(1)
	go func() {
		defer c.transfers.Done()
		defer c.conn.Close()

		transfer := NewTransfer(
			c.conn,
			NewAdapterWrapper(c.adapter),
			c.config,
			c.logger,
		)
		if err := transfer.Start(); err != nil {
			c.logger.Error("Transfer failed", zap.Error(err))
		}
	}()

	return nil
}

// Stop stops the tunnel client
func (c *Client) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.getState() != StateRunning {
		return fmt.Errorf("tunnel is not in running state")
	}

	c.setState(StateStopping)
	c.logger.Info("Stopping tunnel client")

	// Close connection
	if c.conn != nil {
		if err := c.conn.Close(); err != nil {
			c.logger.Error("Failed to close connection", zap.Error(err))
		}
	}

	// Wait for transfers to complete
	c.transfers.Wait()

	// Cleanup adapter
	if c.adapter != nil {
		if err := c.adapter.Cleanup(); err != nil {
			c.logger.Error("Failed to cleanup adapter", zap.Error(err))
		}
	}

	c.setState(StateStopped)
	c.logger.Info("Tunnel client stopped")

	return nil
}

// setState atomically sets the tunnel state
func (c *Client) setState(state State) {
	atomic.StoreInt32((*int32)(&c.state), int32(state))
}

// getState atomically gets the tunnel state
func (c *Client) getState() State {
	return State(atomic.LoadInt32((*int32)(&c.state)))
}
