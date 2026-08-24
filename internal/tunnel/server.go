package tunnel

import (
	"context"
	"fmt"
	"github.com/o3willard-AI/SSSonector/internal/config"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/adapter"
	"github.com/o3willard-AI/SSSonector/internal/facade"
	"github.com/o3willard-AI/SSSonector/internal/monitor"
	"go.uber.org/zap"
)

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

	// AdapterNew creates the TUN interface; overridable in tests.
	AdapterNew func(name string, opts *adapter.Options) (adapter.Interface, error)

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
	if s.tlsManager == nil && !s.activeConfig().Config.Security.AllowPlaintext {
		return fmt.Errorf(
			"refusing to start without TLS: certificates are missing or unusable " +
				"and security.allow_plaintext is not enabled")
	}

	adapterOpts := adapter.DefaultOptions()
	adapterOpts.Logger = s.logger.Named("adapter")
	newAdapter := s.AdapterNew
	if newAdapter == nil {
		newAdapter = adapter.New
	}
	iface, err := newAdapter(s.activeConfig().Config.Network.Name, adapterOpts)
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

	if tc, ok := conn.(*net.TCPConn); ok {
		if ka := s.activeConfig().Config.Tunnel.KeepAliveSeconds; ka > 0 {
			period := time.Duration(ka) * time.Second
			_ = tc.SetKeepAliveConfig(net.KeepAliveConfig{
				Enable:   true,
				Idle:     period,
				Interval: period / 3,
				Count:    3,
			})
		}
	}

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

	idle := time.Duration(s.activeConfig().Config.Tunnel.IdleTimeoutSeconds) * time.Second
	tunnelConn = newCountingConn(tunnelConn, idle, &s.bytesIn, &s.bytesOut)

	adapterConn := NewAdapterWrapper(s.iface)
	transfer, err := NewTransfer(tunnelConn, adapterConn, s.activeConfig(), s.logger)
	if err != nil {
		s.logger.Error("Failed to create transfer", zap.Error(err), zap.String("remote", remoteAddr))
		return
	}
	transfer.ShareDst() // the TUN outlives any single client connection

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

				state := "listening"
				if s.activeConns.Load() > 0 {
					state = "connected"
				}
				s.monitor.SetHealth("server", state)
			}
		}
	}()
}

// currentTransferSnapshotLocked returns the active transfer for sampling.
func (s *Server) currentTransferSnapshotLocked() *Transfer {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentTransfer
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
