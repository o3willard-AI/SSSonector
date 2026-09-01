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
	"github.com/o3willard-AI/SSSonector/internal/nat"
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
	natEngine    *nat.Engine
	natReverse   *nat.ReverseNAT
	natGCStop    chan struct{}
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

	// NAT engine (phase 2: forward path). Disabled config → nil engine,
	// zero data-path change.
	natCfg := s.activeConfig().Config.NAT
	if natCfg.Enabled && natCfg.Forward.Enabled {
		eng, err := nat.NewEngine(&natCfg, nat.Options{
			EgressIP: s.egressAddress(),
		}, s.logger)
		if err != nil {
			s.logger.Error("NAT engine startup failed; NAT is OFF",
				zap.Error(err))
			// Fail closed: NAT disabled, tunnel continues un-NATed.
		} else if eng != nil {
			s.natEngine = eng
			s.natGCStop = make(chan struct{})
			eng.StartGC(s.natGCStop, 0)
			s.logger.Info("NAT forward engine started",
				zap.Int("forward_rules", len(natCfg.Forward.Rules)))
		}
	}

	// Reverse PAT (phase 3): publish peer services on public ports. The
	// netstack's link writer sends frames into the tunnel; inbound
	// tunnel frames are delivered to the stack in handleConnection's
	// data path via s.natReverse.DeliverTunnelPacket.
	if natCfg.Enabled && natCfg.Reverse.Enabled && len(natCfg.Reverse.Listeners) > 0 {
		tunIP, _, _ := net.ParseCIDR(s.activeConfig().Config.Network.Address)
		if tunIP == nil {
			s.logger.Error("NAT reverse path requires a tunnel address; reverse PAT is OFF")
		} else {
			writer := &tunnelPacketWriter{server: s}
			rev, err := nat.NewReverseNAT(writer, tunIP, s.activeConfig().Config.Network.Address, s.logger)
			if err != nil {
				s.logger.Error("NAT reverse engine startup failed; reverse PAT is OFF",
					zap.Error(err))
			} else {
				s.natReverse = rev
				for _, l := range natCfg.Reverse.Listeners {
					if err := rev.StartListener(l); err != nil {
						// Non-fatal: other listeners still start.
						s.logger.Error("NAT listener bind failed",
							zap.Int("port", l.ListenPort), zap.Error(err))
					}
				}
				s.logger.Info("NAT reverse listeners started",
					zap.Int("listeners", len(natCfg.Reverse.Listeners)))
			}
		}
	}

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

// mirrorReader wraps the tunnel connection and mirrors every read frame
// into the reverse netstack while the normal data path consumes it too.
type mirrorReader struct {
	net.Conn
	mirror func([]byte)
}

// Read implements net.Conn: reads pass through and are mirrored.
func (m *mirrorReader) Read(b []byte) (int, error) {
	n, err := m.Conn.Read(b)
	if n > 0 {
		m.mirror(b[:n])
	}
	return n, err
}

// tunnelPacketWriter sends netstack-originated frames into the active
// tunnel connection. WritePacket is a no-op success when no tunnel is
// up: netstack retransmits handle the gap (frame loss until connected).
type tunnelPacketWriter struct {
	server *Server
}

// WritePacket implements nat.PacketWriter.
func (w *tunnelPacketWriter) WritePacket(p []byte) error {
	w.server.mu.Lock()
	conn := w.server.activeConn
	w.server.mu.Unlock()
	if conn == nil {
		return nil // tunnel down: netstack TCP retransmits recover
	}
	_, err := conn.Write(p)
	return err
}

// egressAddress resolves the SNAT source address: the first non-tunnel
// interface address on this host. Returns nil when discovery fails, in
// which case forward NAT fails closed (engine construction errors out).
func (s *Server) egressAddress() net.IP {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}
	_, tunnelNet, _ := net.ParseCIDR(s.activeConfig().Config.Network.Address)
	for _, a := range addrs {
		ipNet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		ip4 := ipNet.IP.To4()
		if ip4 == nil || ip4.IsLoopback() || ip4.IsLinkLocalUnicast() {
			continue
		}
		if tunnelNet != nil && tunnelNet.Contains(ip4) {
			continue
		}
		return ip4
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
	// NAT intercept on the adapter side: reads deliver packets leaving
	// toward the egress (server LAN); fromTunnel=false means egress-side
	// processing for return traffic. The tunnel side is processed when
	// packets enter via the transfer source path.
	if s.natEngine != nil {
		adapterConn = NewNATIntercept(adapterConn, s.natEngine, false)
	}

	// Reverse PAT: mirror tunnel-originated frames into the netstack so
	// published services receive peer-initiated connections. The frame
	// also continues through the normal TUN path (peer may address this
	// host's TUN IP directly).
	var mirrorToReverse func([]byte)
	if s.natReverse != nil {
		mirrorToReverse = func(p []byte) { s.natReverse.DeliverTunnelPacket(p) }
		tunnelConn = &mirrorReader{Conn: tunnelConn, mirror: mirrorToReverse}
	}

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
	natEngine := s.natEngine
	s.mu.Unlock()

	applyRuntimeSettings(s.logger, oldCfg, newCfg, transfer, s.tlsManager)

	// Hot-apply NAT rules when the engine is running and the config kept
	// it enabled (structural toggles are warned as restart-required).
	if natEngine != nil && newCfg.Config.NAT.Enabled && newCfg.Config.NAT.Forward.Enabled {
		if err := natEngine.ReloadRules(&newCfg.Config.NAT); err != nil {
			s.logger.Error("NAT rules hot-reload failed; keeping previous rules",
				zap.Error(err))
		}
	}

	// Reverse listeners hot-apply: converge running listeners to config.
	s.mu.Lock()
	rev := s.natReverse
	s.mu.Unlock()
	if rev != nil && newCfg.Config.NAT.Enabled && newCfg.Config.NAT.Reverse.Enabled {
		if err := rev.ReloadListeners(newCfg.Config.NAT.Reverse.Listeners); err != nil {
			s.logger.Error("NAT listener hot-reload failed; previous listeners kept",
				zap.Error(err))
		}
	}
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

				if s.natEngine != nil || s.natReverse != nil {
					s.sampleNATMetrics()
				}

				state := "listening"
				if s.activeConns.Load() > 0 {
					state = "connected"
				}
				s.monitor.SetHealth("server", state)
			}
		}
	}()
}

// sampleNATMetrics publishes NAT engine and reverse-path counters to
// the monitor. Called from the metrics sampler goroutine only.
func (s *Server) sampleNATMetrics() {
	var forwarded, returned, dropped, activeFlows, accepts, denies int64
	if s.natEngine != nil {
		st := s.natEngine.Stats()
		forwarded = int64(st.ForwardedPackets)
		returned = int64(st.ReturnPackets)
		dropped = int64(st.DroppedPackets)
		activeFlows = int64(st.ActiveFlows)
	}
	if s.natReverse != nil {
		acc, den, _ := s.natReverse.Stats()
		accepts = int64(acc)
		denies = int64(den)
	}
	s.monitor.UpdateNATMetrics(forwarded, returned, dropped, activeFlows, accepts, denies)
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

	// Stop the NAT GC loop first
	if s.natGCStop != nil {
		close(s.natGCStop)
		s.natGCStop = nil
	}

	// Stop reverse listeners + netstack
	if s.natReverse != nil {
		s.natReverse.Stop()
		s.natReverse = nil
	}

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
