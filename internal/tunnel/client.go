package tunnel

import (
	"context"
	"fmt"
	"github.com/o3willard-AI/SSSonector/internal/config"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/adapter"
	"github.com/o3willard-AI/SSSonector/internal/facade"
	"github.com/o3willard-AI/SSSonector/internal/monitor"
	"go.uber.org/zap"
)

// Client represents a tunnel client
type Client struct {
	config       *config.AppConfig
	manager      config.ConfigManager
	logger       *zap.Logger
	monitor      *monitor.Monitor
	iface        adapter.Interface
	tlsManager   *TLSManager
	facadeClient *facade.Client
	conn         net.Conn
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	reconnect    bool
	rng          *rand.Rand

	mu              sync.Mutex
	currentTransfer *Transfer

	// AdapterNew creates the TUN interface; overridable in tests.
	AdapterNew func(name string, opts *adapter.Options) (adapter.Interface, error)

	throttleHitsIn  atomic.Uint64
	throttleHitsOut atomic.Uint64
	lastSeenHitsIn  uint64
	lastSeenHitsOut uint64

	bytesIn     atomic.Int64
	bytesOut    atomic.Int64
	errorsTotal atomic.Int64
	activeConns atomic.Int32
}

// NewClient creates a new tunnel client. The monitor may be nil to run
// without metrics collection.
func NewClient(cfg *config.AppConfig, manager config.ConfigManager, logger *zap.Logger, mon *monitor.Monitor) *Client {
	ctx, cancel := context.WithCancel(context.Background())

	var tlsManager *TLSManager
	if cfg.Config.Auth.CertFile != "" && cfg.Config.Auth.KeyFile != "" && cfg.Config.Auth.CAFile != "" {
		tlsConfig := &TLSConfig{
			CertFile:      cfg.Config.Auth.CertFile,
			KeyFile:       cfg.Config.Auth.KeyFile,
			CAFile:        cfg.Config.Auth.CAFile,
			SecurityLevel: SecurityModern,
			ServerName:    cfg.Config.Tunnel.ServerAddress,
		}
		var err error
		tlsManager, err = NewTLSManager(tlsConfig, logger)
		if err != nil {
			logger.Error("Failed to create TLS manager, running without TLS", zap.Error(err))
		}
	}

	client := &Client{
		config:     cfg,
		manager:    manager,
		logger:     logger,
		monitor:    mon,
		ctx:        ctx,
		cancel:     cancel,
		tlsManager: tlsManager,
		reconnect:  true,
		// #nosec G404 -- jitter timing is not security-sensitive
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	// Initialize facade client if enabled
	if cfg.Config.Facade.Enabled {
		facadeClient, err := facade.NewClient(&cfg.Config.Facade, &cfg.Config.Tunnel, &cfg.Config.Auth, logger)
		if err != nil {
			logger.Error("Failed to create facade client, will use direct connections only", zap.Error(err))
		} else {
			client.facadeClient = facadeClient
			logger.Info("HTTPS facade fallback enabled",
				zap.Int("tunnel_port", cfg.Config.Tunnel.ServerPort),
			)
		}
	}

	return client
}

// Start starts the tunnel client
func (c *Client) Start() error {
	if c.tlsManager == nil && !c.activeConfig().Config.Security.AllowPlaintext {
		return fmt.Errorf(
			"refusing to start without TLS: certificates are missing or unusable " +
				"and security.allow_plaintext is not enabled")
	}

	adapterOpts := adapter.DefaultOptions()
	adapterOpts.Logger = c.logger.Named("adapter")
	newAdapter := c.AdapterNew
	if newAdapter == nil {
		newAdapter = adapter.New
	}
	iface, err := newAdapter(c.activeConfig().Config.Network.Name, adapterOpts)
	if err != nil {
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	if err := iface.Configure(&adapter.Config{
		Name:    c.activeConfig().Config.Network.Name,
		Address: c.activeConfig().Config.Network.Address,
		MTU:     c.activeConfig().Config.Network.MTU,
	}); err != nil {
		iface.Close()
		return fmt.Errorf("failed to configure adapter: %w", err)
	}
	c.iface = iface

	serverAddrForLog := fmt.Sprintf("%s:%d", c.activeConfig().Config.Tunnel.ServerAddress, c.activeConfig().Config.Tunnel.ServerPort)
	c.logger.Info("Starting tunnel client",
		zap.String("server", serverAddrForLog),
		zap.String("tun", c.config.Config.Network.Name),
		zap.String("tun_address", c.config.Config.Network.Address),
	)

	c.wg.Add(1)
	go c.connectLoop()

	c.startMetricsSampler()

	return nil
}

// startMetricsSampler periodically publishes tunnel counters to the monitor
func (c *Client) startMetricsSampler() {
	if c.monitor == nil || !c.config.Config.Metrics.Enabled {
		return
	}

	interval := c.config.Config.Metrics.Interval
	if interval < time.Second {
		interval = time.Second
	}

	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-c.ctx.Done():
				return
			case <-ticker.C:
				c.monitor.UpdateMetrics(
					c.bytesIn.Load(),
					c.bytesOut.Load(),
					0, 0,
					c.errorsTotal.Load(),
					int(c.activeConns.Load()),
				)
				inH, outH, rate, burst := sampleThrottle(
					c.currentTransferSnapshotLocked(),
					&c.throttleHitsIn, &c.throttleHitsOut,
					&c.lastSeenHitsIn, &c.lastSeenHitsOut,
					&c.mu)
				c.monitor.UpdateThrottleMetrics(inH, outH, rate, burst)

				state := "connecting"
				if c.activeConns.Load() > 0 {
					state = "connected"
				}
				c.monitor.SetHealth("client", state)
			}
		}
	}()
}

// connectLoop manages connection with automatic reconnection
func (c *Client) connectLoop() {
	defer c.wg.Done()

	serverAddr := net.JoinHostPort(c.config.Config.Tunnel.ServerAddress, strconv.Itoa(c.config.Config.Tunnel.ServerPort))
	retryCount := 0

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
		}

		var conn net.Conn
		var tunnelConn net.Conn
		var err error
		viaFacade := false

		if c.facadeClient != nil {
			// Use facade client: tries direct first, then falls back to HTTPS facade
			result, connectErr := c.facadeClient.Connect(c.ctx)
			if connectErr != nil {
				err = connectErr
			} else {
				conn = result.Conn
				viaFacade = result.ViaFacade
			}
		} else {
			// Direct connection only (original behavior)
			conn, err = net.DialTimeout("tcp4", serverAddr, 10*time.Second)
		}

		if err != nil {
			c.errorsTotal.Add(1)
			retryCount++
			rc := c.activeConfig().Config.Tunnel.Reconnect.Normalized()
			if retryCount > rc.MaxAttempts {
				c.logger.Error("Max retries exceeded, stopping reconnection",
					zap.Int("attempts", retryCount),
					zap.Error(err))
				return
			}

			delay := computeBackoff(retryCount, rc.InitialDelay, rc.MaxDelay, rc.Jitter, c.rng)

			c.logger.Warn("Connection failed, retrying",
				zap.Int("attempt", retryCount),
				zap.Duration("delay", delay),
				zap.Error(err))

			select {
			case <-time.After(delay):
			case <-c.ctx.Done():
				return
			}

			continue
		}

		retryCount = 0

		if viaFacade {
			// Connection via HTTPS facade is already TLS-encrypted.
			// Skip the tunnel's TLS wrapping to avoid double encryption.
			tunnelConn = conn
			c.logger.Info("Connected to server via HTTPS facade",
				zap.String("address", serverAddr),
			)
		} else if c.tlsManager != nil {
			tunnelConn, err = c.tlsManager.WrapConn(conn, false)
			if err != nil {
				conn.Close()
				c.errorsTotal.Add(1)
				c.logger.Error("TLS handshake failed", zap.Error(err))
				continue
			}
			c.logger.Info("Connected to server", zap.String("address", serverAddr))
		} else {
			tunnelConn = conn
			c.logger.Info("Connected to server", zap.String("address", serverAddr))
		}

		c.conn = conn
		c.activeConns.Add(1)

		tunnelConn = newCountingConn(tunnelConn, &c.bytesIn, &c.bytesOut)
		adapterConn := NewAdapterWrapper(c.iface)
		transfer, err := NewTransfer(tunnelConn, adapterConn, c.activeConfig(), c.logger)
		if err != nil {
			c.logger.Error("Failed to create transfer", zap.Error(err))
			c.activeConns.Add(-1)
			conn.Close()
			if !c.reconnect {
				return
			}
			select {
			case <-time.After(time.Second):
			case <-c.ctx.Done():
				return
			}
			continue
		}
		transfer.ShareDst() // the TUN outlives any single connection

		c.mu.Lock()
		c.currentTransfer = transfer
		c.mu.Unlock()

		c.logger.Info("Tunnel established")
		if err := transfer.Start(); err != nil {
			c.errorsTotal.Add(1)
			c.logger.Error("Transfer ended", zap.Error(err))
		}

		c.mu.Lock()
		if c.currentTransfer == transfer {
			c.currentTransfer = nil
		}
		c.mu.Unlock()

		c.conn = nil
		c.activeConns.Add(-1)
		conn.Close()
		c.logger.Info("Disconnected from server")

		if !c.reconnect {
			return
		}

		select {
		case <-time.After(time.Second):
		case <-c.ctx.Done():
			return
		}
	}
}

// activeConfig returns the current configuration snapshot for data-path
// construction. Configuration is swapped under the same mutex that guards
// the active transfer, so reloads never race connection setup.
func (c *Client) activeConfig() *config.AppConfig {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.config
}

// ApplyConfig applies the reloadable subset of a new configuration:
// throttle rates reach live and future transfers; cert-rotation timing
// reaches live certificate managers. Structural changes are logged as
// restart-required warnings.
func (c *Client) ApplyConfig(newCfg *config.AppConfig) error {
	if err := validateReloadTarget(newCfg); err != nil {
		return err
	}

	c.mu.Lock()
	oldCfg := c.config
	c.config = newCfg
	transfer := c.currentTransfer
	c.mu.Unlock()

	applyRuntimeSettings(c.logger, oldCfg, newCfg, transfer, c.tlsManager)
	return nil
}

// currentTransferSnapshotLocked returns the active transfer for sampling.
func (c *Client) currentTransferSnapshotLocked() *Transfer {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.currentTransfer
}

// Stop stops the tunnel client
func (c *Client) Stop() error {
	c.logger.Info("Stopping tunnel client")

	c.cancel()

	if c.conn != nil {
		c.conn.Close()
	}

	c.wg.Wait()

	if c.iface != nil {
		if err := c.iface.Cleanup(); err != nil {
			c.logger.Error("Failed to cleanup adapter", zap.Error(err))
		}
	}

	return nil
}
