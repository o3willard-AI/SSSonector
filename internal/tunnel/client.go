package tunnel

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/adapter"
	"github.com/o3willard-AI/SSSonector/internal/config/interfaces"
	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/o3willard-AI/SSSonector/internal/network"
	"github.com/o3willard-AI/SSSonector/internal/startup"
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

	// Create startup logger
	startupLogger := startup.NewStartupLogger(c.logger, c.config.Config.Logging)

	// Pre-startup phase
	startupLogger.SetPhase(types.StartupPhasePreStartup)
	startupLogger.LogCheckpoint("Starting tunnel client", map[string]interface{}{
		"mode": c.config.Config.Mode,
	})

	c.setState(StateStarting)

	// Initialization phase
	startupLogger.SetPhase(types.StartupPhaseInitialization)

	// Check and enable IP forwarding
	err := startupLogger.LogOperation(types.StartupComponentNetwork, "Check IP forwarding", func() error {
		return network.EnableIPForwarding(c.logger)
	}, nil)
	if err != nil {
		c.logger.Warn("Failed to enable IP forwarding, packet forwarding may not work",
			zap.Error(err),
		)
		// Continue anyway, as this might be a permission issue or non-Linux system
	}

	// Create TUN adapter
	err = startupLogger.LogOperation(types.StartupComponentAdapter, "Create TUN adapter", func() error {
		c.adapter, err = adapter.FromConfig(c.config)
		return err
	}, map[string]interface{}{
		"config": c.config.Adapter,
	})
	if err != nil {
		c.setState(StateStopped)
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	// Connection phase
	startupLogger.SetPhase(types.StartupPhaseConnection)

	// Connect to server using the Server field from config with retries
	addr := c.config.Config.Tunnel.Server
	maxRetries := 5
	retryDelay := types.NewDuration(2 * time.Second)

	err = startupLogger.LogOperation(types.StartupComponentConnection, "Connect to server", func() error {
		var lastErr error
		for i := 0; i < maxRetries; i++ {
			// Create TCP connection
			c.conn, err = net.Dial("tcp", addr)
			if err != nil {
				lastErr = err
				c.logger.Warn("Connection attempt failed",
					zap.Int("attempt", i+1),
					zap.Int("max_attempts", maxRetries),
					zap.Error(err),
				)
				if c.conn != nil {
					c.logger.Debug("Connection details",
						zap.String("local_addr", c.conn.LocalAddr().String()),
						zap.String("remote_addr", c.conn.RemoteAddr().String()),
					)
				}
				if i < maxRetries-1 {
					time.Sleep(retryDelay.Duration)
					continue
				}
				if cleanupErr := c.adapter.Cleanup(); cleanupErr != nil {
					c.logger.Error("Failed to cleanup adapter", zap.Error(cleanupErr))
				}
				return fmt.Errorf("failed to connect to server after %d attempts: %w", maxRetries, lastErr)
			}

			// Authenticate using certificates
			certManager := NewCertManager(c.logger, c.config)
			if err := certManager.VerifyPeerCertificate(c.conn); err != nil {
				c.logger.Error("Authentication failed", zap.Error(err))
				c.conn.Close()
				if i < maxRetries-1 {
					time.Sleep(retryDelay.Duration)
					continue
				}
				if cleanupErr := c.adapter.Cleanup(); cleanupErr != nil {
					c.logger.Error("Failed to cleanup adapter", zap.Error(cleanupErr))
				}
				return fmt.Errorf("authentication failed after %d attempts: %w", maxRetries, err)
			}

			// Successfully connected and authenticated
			return nil
		}
		return lastErr
	}, map[string]interface{}{
		"address":     addr,
		"max_retries": maxRetries,
		"retry_delay": retryDelay.String(),
	})
	if err != nil {
		c.setState(StateStopped)
		return err
	}

	c.setState(StateRunning)
	startupLogger.LogCheckpoint("Tunnel client started", map[string]interface{}{
		"server": addr,
		"state":  StateRunning,
	})

	// Start transfer using TUN interface
	c.transfers.Add(1)
	go func() {
		defer c.transfers.Done()
		defer c.conn.Close()

		transfer := NewTransfer(
			c.conn,                       // Initial TCP connection
			NewAdapterWrapper(c.adapter), // TUN adapter for data
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

	// Create startup logger
	startupLogger := startup.NewStartupLogger(c.logger, c.config.Config.Logging)

	startupLogger.SetPhase(types.StartupPhasePreStartup)
	startupLogger.LogCheckpoint("Stopping tunnel client", map[string]interface{}{
		"state": c.getState(),
	})

	c.setState(StateStopping)

	// Close connection
	if c.conn != nil {
		err := startupLogger.LogOperation(types.StartupComponentConnection, "Close connection", func() error {
			return c.conn.Close()
		}, nil)
		if err != nil {
			c.logger.Error("Failed to close connection", zap.Error(err))
		}
	}

	// Wait for transfers to complete
	startupLogger.LogOperation(types.StartupComponentConnection, "Wait for transfers", func() error {
		c.transfers.Wait()
		return nil
	}, nil)

	// Cleanup adapter
	if c.adapter != nil {
		err := startupLogger.LogOperation(types.StartupComponentAdapter, "Cleanup adapter", func() error {
			return c.adapter.Cleanup()
		}, nil)
		if err != nil {
			c.logger.Error("Failed to cleanup adapter", zap.Error(err))
		}
	}

	c.setState(StateStopped)
	startupLogger.LogCheckpoint("Tunnel client stopped", map[string]interface{}{
		"state": StateStopped,
	})

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
