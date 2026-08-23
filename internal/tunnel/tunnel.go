package tunnel

import (
	"context"
	"fmt"
	"github.com/o3willard-AI/SSSonector/internal/config"
	"net"
	"path/filepath"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/adapter"
	"github.com/o3willard-AI/SSSonector/internal/facade"
	"github.com/o3willard-AI/SSSonector/internal/monitor"
	"go.uber.org/zap"
)

// Reloadable is implemented by tunnel modes that support SIGHUP-driven
// configuration reload.
type Reloadable interface {
	ApplyConfig(newCfg *config.AppConfig) error
}

// UpdateCertificatePaths updates certificate paths to be absolute
func UpdateCertificatePaths(cfg *config.AppConfig, baseDir string) error {
	resolvePath := func(path string) string {
		if path == "" || filepath.IsAbs(path) {
			return path
		}
		return filepath.Join(baseDir, path)
	}

	cfg.Config.Auth.CertFile = resolvePath(cfg.Config.Auth.CertFile)
	cfg.Config.Auth.KeyFile = resolvePath(cfg.Config.Auth.KeyFile)
	cfg.Config.Auth.CAFile = resolvePath(cfg.Config.Auth.CAFile)

	return nil
}

// Server represents a tunnel server (point-to-point, one client per instance)
type Server struct {
	config       *config.AppConfig
	manager      config.ConfigManager
	logger       *zap.Logger
	monitor      *monitor.Monitor
	ln           net.Listener
	iface        adapter.Interface
	tlsManager   *TLSManager
	facadeServer *facade.Server
	wg           sync.WaitGroup
	ctx          context.Context
	cancel       context.CancelFunc
	mu           sync.Mutex
	activeConn   net.Conn

	bytesIn     atomic.Int64
	bytesOut    atomic.Int64
	errorsTotal atomic.Int64
	activeConns atomic.Int32

	currentTransfer *Transfer

	throttleHitsIn  atomic.Uint64
	throttleHitsOut atomic.Uint64
	lastSeenHitsIn  uint64
	lastSeenHitsOut uint64
}

// NewServer creates a new tunnel server. The monitor may be nil to run
// without metrics collection.
func NewServer(cfg *config.AppConfig, manager config.ConfigManager, logger *zap.Logger, mon *monitor.Monitor) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	var tlsManager *TLSManager
	if cfg.Config.Auth.CertFile != "" && cfg.Config.Auth.KeyFile != "" && cfg.Config.Auth.CAFile != "" {
		tlsConfig := &TLSConfig{
			CertFile:      cfg.Config.Auth.CertFile,
			KeyFile:       cfg.Config.Auth.KeyFile,
			CAFile:        cfg.Config.Auth.CAFile,
			SecurityLevel: SecurityModern,
		}
		var err error
		tlsManager, err = NewTLSManager(tlsConfig, logger)
		if err != nil {
			logger.Error("Failed to create TLS manager, running without TLS", zap.Error(err))
		}
	}

	return &Server{
		config:     cfg,
		manager:    manager,
		logger:     logger,
		monitor:    mon,
		ctx:        ctx,
		cancel:     cancel,
		tlsManager: tlsManager,
	}
}

// Start starts the tunnel server
func (s *Server) Start() error {
	adapterOpts := adapter.DefaultOptions()
	adapterOpts.Logger = s.logger.Named("adapter")
	iface, err := adapter.New(s.activeConfig().Config.Network.Name, adapterOpts)
	if err != nil {
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	if err := iface.Configure(&adapter.Config{
		Name:    s.activeConfig().Config.Network.Name,
		Address: s.activeConfig().Config.Network.Address,
		MTU:     s.activeConfig().Config.Network.MTU,
	}); err != nil {
		iface.Close()
		return fmt.Errorf("failed to configure adapter: %w", err)
	}
	s.iface = iface

	listenAddr := fmt.Sprintf("%s:%d", s.activeConfig().Config.Tunnel.ListenAddress, s.activeConfig().Config.Tunnel.ListenPort)
	s.logger.Info("Starting tunnel server",
		zap.String("address", listenAddr),
		zap.String("tun", s.config.Config.Network.Name),
		zap.String("tun_address", s.config.Config.Network.Address),
	)

	ln, err := net.Listen("tcp4", listenAddr)
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	s.ln = ln

	s.wg.Add(1)
	go s.acceptLoop()

	s.startMetricsSampler()

	// Start the HTTPS facade if enabled
	if s.config.Config.Facade.Enabled {
		facadeServer, err := facade.NewServer(&s.config.Config.Facade, &s.config.Config.Auth, s.logger)
		if err != nil {
			s.logger.Error("Failed to create HTTPS facade", zap.Error(err))
			// Non-fatal: tunnel still works on its direct port
		} else {
			if err := facadeServer.Start(); err != nil {
				s.logger.Error("Failed to start HTTPS facade", zap.Error(err))
				// Non-fatal: tunnel still works on its direct port
			} else {
				s.facadeServer = facadeServer
				s.logger.Info("HTTPS facade started",
					zap.Int("facade_port", s.config.Config.Facade.ListenPort),
					zap.Int("tunnel_port", s.config.Config.Tunnel.ListenPort),
				)
			}
		}
	}

	return nil
}

// acceptLoop accepts connections and handles them one at a time
func (s *Server) acceptLoop() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			conn, err := s.ln.Accept()
			if err != nil {
				if s.ctx.Err() == nil {
					s.logger.Error("Failed to accept connection", zap.Error(err))
				}
				continue
			}

			s.handleConnection(conn)
		}
	}
}

// handleConnection handles a client connection
func (s *Server) handleConnection(conn net.Conn) {
	s.mu.Lock()
	if s.activeConn != nil {
		s.mu.Unlock()
		s.logger.Warn("Rejecting connection - tunnel already active")
		conn.Close()
		return
	}
	s.activeConn = conn
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.activeConn = nil
		s.mu.Unlock()
		conn.Close()
	}()

	remoteAddr := conn.RemoteAddr().String()
	s.logger.Info("Client connected", zap.String("remote", remoteAddr))
	s.activeConns.Add(1)

	var tunnelConn net.Conn
	var err error

	if s.tlsManager != nil {
		tunnelConn, err = s.tlsManager.WrapConn(conn, true)
		if err != nil {
			s.logger.Error("Failed to wrap connection with TLS", zap.Error(err))
			return
		}
	} else {
		tunnelConn = conn
	}

	tunnelConn = newCountingConn(tunnelConn, &s.bytesIn, &s.bytesOut)

	adapterConn := NewAdapterWrapper(s.iface)
	transfer, err := NewTransfer(tunnelConn, adapterConn, s.activeConfig(), s.logger)
	if err != nil {
		s.logger.Error("Failed to create transfer", zap.Error(err), zap.String("remote", remoteAddr))
		return
	}

	s.mu.Lock()
	s.currentTransfer = transfer
	s.mu.Unlock()

	s.logger.Info("Tunnel established", zap.String("remote", remoteAddr))
	if err := transfer.Start(); err != nil {
		s.errorsTotal.Add(1)
		s.logger.Error("Transfer ended", zap.Error(err), zap.String("remote", remoteAddr))
	}

	s.mu.Lock()
	if s.currentTransfer == transfer {
		s.currentTransfer = nil
	}
	s.mu.Unlock()

	s.activeConns.Add(-1)
	s.logger.Info("Client disconnected", zap.String("remote", remoteAddr))
}

// activeConfig returns the current configuration snapshot for data-path
// construction. Configuration is swapped under the same mutex that guards
// the active connection, so reloads never race connection setup.
func (s *Server) activeConfig() *config.AppConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.config
}

// ApplyConfig applies the reloadable subset of a new configuration:
// throttle rates reach live and future transfers; cert-rotation timing
// reaches live certificate managers. Structural changes are logged as
// restart-required warnings.
func (s *Server) ApplyConfig(newCfg *config.AppConfig) error {
	if err := validateReloadTarget(newCfg); err != nil {
		return err
	}

	s.mu.Lock()
	oldCfg := s.config
	s.config = newCfg
	transfer := s.currentTransfer
	s.mu.Unlock()

	applyRuntimeSettings(s.logger, oldCfg, newCfg, transfer, s.tlsManager)
	return nil
}

// currentTransferSnapshotLocked returns the active transfer for sampling.
func (s *Server) currentTransferSnapshotLocked() *Transfer {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentTransfer
}

// startMetricsSampler periodically publishes tunnel counters to the monitor
func (s *Server) startMetricsSampler() {
	if s.monitor == nil || !s.config.Config.Metrics.Enabled {
		return
	}

	interval := s.config.Config.Metrics.Interval
	if interval < time.Second {
		interval = time.Second
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				s.monitor.UpdateMetrics(
					s.bytesIn.Load(),
					s.bytesOut.Load(),
					0, 0,
					s.errorsTotal.Load(),
					int(s.activeConns.Load()),
				)
				inH, outH, rate, burst := sampleThrottle(
					s.currentTransferSnapshotLocked(),
					&s.throttleHitsIn, &s.throttleHitsOut,
					&s.lastSeenHitsIn, &s.lastSeenHitsOut,
					&s.mu)
				s.monitor.UpdateThrottleMetrics(inH, outH, rate, burst)
			}
		}
	}()
}

// sampleThrottle accumulates per-transfer hit deltas into cumulative
// counters and returns them with the current pacing values. Safe on nil
// transfer: counters persist, pacing reports the last observed values.
func sampleThrottle(
	transfer *Transfer,
	hitsIn, hitsOut *atomic.Uint64,
	lastIn, lastOut *uint64,
	mu *sync.Mutex,
) (inHits, outHits uint64, rate, burst float64) {
	if transfer != nil {
		curIn, curOut, r, b := transfer.ThrottleStats()
		mu.Lock()
		deltaIn := curIn - *lastIn
		deltaOut := curOut - *lastOut
		*lastIn = curIn
		*lastOut = curOut
		mu.Unlock()
		hitsIn.Add(deltaIn)
		hitsOut.Add(deltaOut)
		return hitsIn.Load(), hitsOut.Load(), r, b
	}
	return hitsIn.Load(), hitsOut.Load(), 0, 0
}

// Stop stops the tunnel server
func (s *Server) Stop() error {
	s.logger.Info("Stopping tunnel server")

	// Stop the HTTPS facade first
	if s.facadeServer != nil {
		if err := s.facadeServer.Stop(); err != nil {
			s.logger.Error("Failed to stop HTTPS facade", zap.Error(err))
		}
	}

	s.cancel()

	if s.ln != nil {
		s.ln.Close()
	}

	s.mu.Lock()
	if s.activeConn != nil {
		s.activeConn.Close()
	}
	s.mu.Unlock()

	s.wg.Wait()

	if s.iface != nil {
		if err := s.iface.Cleanup(); err != nil {
			s.logger.Error("Failed to cleanup adapter", zap.Error(err))
		}
	}

	return nil
}

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
	maxRetries   int
	retryDelay   time.Duration
	maxRetryWait time.Duration

	mu              sync.Mutex
	currentTransfer *Transfer

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
		config:       cfg,
		manager:      manager,
		logger:       logger,
		monitor:      mon,
		ctx:          ctx,
		cancel:       cancel,
		tlsManager:   tlsManager,
		reconnect:    true,
		maxRetries:   10,
		retryDelay:   time.Second,
		maxRetryWait: 30 * time.Second,
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
	adapterOpts := adapter.DefaultOptions()
	adapterOpts.Logger = c.logger.Named("adapter")
	iface, err := adapter.New(c.activeConfig().Config.Network.Name, adapterOpts)
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

// currentTransferSnapshotLocked returns the active transfer for sampling.
func (c *Client) currentTransferSnapshotLocked() *Transfer {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.currentTransfer
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
			}
		}
	}()
}

// connectLoop manages connection with automatic reconnection
func (c *Client) connectLoop() {
	defer c.wg.Done()

	serverAddr := net.JoinHostPort(c.config.Config.Tunnel.ServerAddress, strconv.Itoa(c.config.Config.Tunnel.ServerPort))
	retryCount := 0
	currentDelay := c.retryDelay

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
			if retryCount > c.maxRetries {
				c.logger.Error("Max retries exceeded, stopping reconnection",
					zap.Int("attempts", retryCount),
					zap.Error(err))
				return
			}

			c.logger.Warn("Connection failed, retrying",
				zap.Int("attempt", retryCount),
				zap.Duration("delay", currentDelay),
				zap.Error(err))

			select {
			case <-time.After(currentDelay):
			case <-c.ctx.Done():
				return
			}

			currentDelay = time.Duration(float64(currentDelay) * 1.5)
			if currentDelay > c.maxRetryWait {
				currentDelay = c.maxRetryWait
			}
			continue
		}

		retryCount = 0
		currentDelay = c.retryDelay

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

// validateReloadTarget rejects unusable reload payloads
func validateReloadTarget(newCfg *config.AppConfig) error {
	if newCfg == nil {
		return fmt.Errorf("config is required")
	}
	if newCfg.Config == nil {
		return fmt.Errorf("config.Config is required")
	}
	return nil
}

// applyRuntimeSettings pushes the hot-reloadable subset into live objects
// and warns about anything that needs a restart instead.
func applyRuntimeSettings(logger *zap.Logger, oldCfg, newCfg *config.AppConfig, transfer *Transfer, tlsManager *TLSManager) {
	if oldCfg != nil && oldCfg.Throttle != newCfg.Throttle {
		logger.Info("Applying reloaded throttle settings",
			zap.Bool("enabled", newCfg.Throttle.Enabled),
			zap.Float64("rate", newCfg.Throttle.Rate),
			zap.Int("burst", newCfg.Throttle.Burst),
		)
		if transfer != nil {
			transfer.UpdateConfig(newCfg)
		}
	}

	if oldCfg != nil && tlsManager != nil &&
		oldCfg.Config.Auth.CertRotation.Interval != newCfg.Config.Auth.CertRotation.Interval &&
		newCfg.Config.Auth.CertRotation.Interval > 0 {
		tlsManager.SetCertTunables(newCfg.Config.Auth.CertRotation.Interval, 0)
		logger.Info("Applied reloaded certificate check interval",
			zap.Duration("interval", newCfg.Config.Auth.CertRotation.Interval),
		)
	}

	logRestartRequiredChanges(logger, oldCfg, newCfg)
}

// logRestartRequiredChanges warns about configuration differences that only
// take effect after a service restart.
func logRestartRequiredChanges(logger *zap.Logger, oldCfg, newCfg *config.AppConfig) {
	if oldCfg == nil || oldCfg.Config == nil {
		return
	}
	oldC, newC := oldCfg.Config, newCfg.Config

	warnIfChanged := func(field string, changed bool) {
		if changed {
			logger.Warn("Configuration change requires restart",
				zap.String("field", field),
			)
		}
	}

	warnIfChanged("mode", oldC.Mode != newC.Mode)
	warnIfChanged("logging.file", oldC.Logging.File != newC.Logging.File)
	warnIfChanged("logging.format", oldC.Logging.Format != newC.Logging.Format)
	warnIfChanged("network.name", oldC.Network.Name != newC.Network.Name)
	warnIfChanged("network.address", oldC.Network.Address != newC.Network.Address)
	warnIfChanged("network.mtu", oldC.Network.MTU != newC.Network.MTU)
	warnIfChanged("tunnel.listen_address", oldC.Tunnel.ListenAddress != newC.Tunnel.ListenAddress)
	warnIfChanged("tunnel.listen_port", oldC.Tunnel.ListenPort != newC.Tunnel.ListenPort)
	warnIfChanged("tunnel.server_address", oldC.Tunnel.ServerAddress != newC.Tunnel.ServerAddress)
	warnIfChanged("tunnel.server_port", oldC.Tunnel.ServerPort != newC.Tunnel.ServerPort)
	warnIfChanged("facade.enabled", oldC.Facade.Enabled != newC.Facade.Enabled)
	warnIfChanged("monitor.type", oldC.Monitor.Type != newC.Monitor.Type)
	warnIfChanged("monitor.prometheus.port", oldC.Monitor.Prometheus.Port != newC.Monitor.Prometheus.Port)
	warnIfChanged("snmp.enabled", oldC.SNMP.Enabled != newC.SNMP.Enabled)
	warnIfChanged("snmp.port", oldC.SNMP.Port != newC.SNMP.Port)
	warnIfChanged("metrics.interval", oldC.Metrics.Interval != newC.Metrics.Interval)
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
