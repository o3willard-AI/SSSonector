package tunnel

import (
	"context"
	"fmt"
	"github.com/o3willard-AI/SSSonector/internal/config"
	"io"
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
	natProxy     *nat.ForwardProxy
	natReverse   *nat.ReverseNAT
	natGCStop    chan struct{}
	frameSink    *tunnelFrameSink
	probe        atomic.Pointer[probeConn]
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

	// Shared frame sink: netstack-emitted frames (forward proxy + reverse
	// PAT) are written into the active TLS tunnel connection through it.
	natCfg := s.activeConfig().Config.NAT
	if natCfg.Enabled && (natCfg.Forward.Enabled || natCfg.Reverse.Enabled) {
		s.frameSink = &tunnelFrameSink{}
	}

	// NAT engine (phase 2: forward path). Disabled config → nil engine,
	// zero data-path change.
	if natCfg.Enabled && natCfg.Forward.Enabled {
		tunIP, _, _ := net.ParseCIDR(s.activeConfig().Config.Network.Address)
		if tunIP == nil {
			s.logger.Error("NAT forward proxy requires a tunnel address; forward NAT is OFF")
		} else {
			// Netstack-proxy forward path (T4 decision): TCP flows from
			// the tunnel are terminated in a dedicated netstack and
			// re-originated via local sockets toward the egress — no
			// kernel forwarding, per-flow ACL enforcement in-daemon.
			fp, err := nat.NewForwardProxy(tunIP,
				s.activeConfig().Config.Network.Address,
				natCfg.Forward.Rules, s.logger)
			if err != nil {
				s.logger.Error("NAT forward proxy startup failed; forward NAT is OFF",
					zap.Error(err))
			} else {
				s.natProxy = fp
				fp.StartPump(s.logger)
				if err := fp.SetTransportHandler(natCfg.Forward.Rules, s.logger); err != nil {
					s.logger.Error("NAT forward rules failed to compile; forward NAT is OFF",
						zap.Error(err))
					s.natProxy = nil
				} else {
					s.logger.Info("NAT forward proxy started",
						zap.Int("forward_rules", len(natCfg.Forward.Rules)))
					// The forward proxy's netstack emits SYN-ACKs and
					// relay frames through the shared tunnel frame
					// sink; without it the pump silently discards
					// every frame (QA: handshake never completed).
					fp.SetFrameSink(s.frameSink)
				}
			}
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
			// Frames flow through the per-connection frame sink (set in
			// handleConnection on the TLS-wrapped conn); the nil writer
			// here is the fallback when no sink is installed.
			rev, err := nat.NewReverseNAT(nullPacketWriter{}, tunIP, s.activeConfig().Config.Network.Address, s.logger)
			if err != nil {
				s.logger.Error("NAT reverse engine startup failed; reverse PAT is OFF",
					zap.Error(err))
			} else {
				s.natReverse = rev
				rev.SetFrameSink(s.frameSink)
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

// nullPacketWriter is the construction-time fallback when no frame sink
// is installed: frames are dropped (netstack retransmits recover).
type nullPacketWriter struct{}

// WritePacket implements nat.PacketWriter.
func (nullPacketWriter) WritePacket(p []byte) error { return nil }

// tunnelRouter is the ACTIVE read path when NAT is enabled: it reads raw
// TLS chunks from the tunnel, reassembles complete IPv4 frames, routes
// netstack-owned frames to the NAT netstacks (consuming them), and serves
// everything else to the caller (the TUN data path) via an internal pipe.
//
// Why consuming is required: the server kernel RSTs any flow it doesn't
// own — netstack-relayed flows (reverse PAT replies, forward sessions)
// would otherwise be poisoned by pass-through frames (QA-verified).
type tunnelRouter struct {
	net.Conn  // underlying tunnel conn (TLS)
	assembler *nat.FrameAssembler
	route     func(frame []byte) bool // returns true if consumed
	out       *io.PipeReader
	writer    *io.PipeWriter
	closeOut  func()
}

// Read pulls from the router's output pipe; a background goroutine feeds
// it after routing (started once via start).
func (r *tunnelRouter) Read(b []byte) (int, error) {
	return r.out.Read(b)
}

// start begins draining the underlying tunnel connection.
func (r *tunnelRouter) start() {
	go func() {
		buf := make([]byte, 65535)
		for {
			n, err := r.Conn.Read(buf)
			if n > 0 {
				r.assembler.Deliver(buf[:n], func(frame []byte) {
					if !r.route(frame) {
						_, _ = r.writer.Write(frame)
					}
				})
			}
			if err != nil {
				r.closeOut()
				return
			}
		}
	}()
}

// tunnelFrameSink is the per-connection write target for netstack frames.
// It holds the TLS-wrapped tunnel conn (never the raw TCP conn — writing
// raw frames pre-TLS corrupts the client's TLS stream, QA bug T1) and is
// swapped in/out under the server mutex as connections come and go.
type tunnelFrameSink struct {
	mu   sync.Mutex
	conn net.Conn
}

// Set installs (or, with nil, removes) the active tunnel connection.
func (s *tunnelFrameSink) Set(c net.Conn) {
	s.mu.Lock()
	s.conn = c
	s.mu.Unlock()
}

// ClearIf removes the active connection only if it is still prev: the
// deferred clear of a dying connection must not clobber the fresh
// connection installed by a newer handleConnection (QA stale-sink race).
func (s *tunnelFrameSink) ClearIf(prev net.Conn) {
	s.mu.Lock()
	if s.conn == prev {
		s.conn = nil
	}
	s.mu.Unlock()
}

// WritePacket implements nat.PacketWriter. A nil conn is a no-op
// success: netstack TCP retransmit recovers when the tunnel returns.
func (s *tunnelFrameSink) WritePacket(p []byte) error {
	s.mu.Lock()
	conn := s.conn
	s.mu.Unlock()
	if conn == nil {
		return nil
	}
	_, err := conn.Write(p)
	return err
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

	// Boundary instrumentation (QA evidence gathering): observe exactly
	// what the TLS layer delivers — packet-per-record vs coalesced.
	probe := newProbeConn(tunnelConn, "server-tunnel", s.logger)
	s.probe.Store(probe)
	tunnelConn = probe

	adapterConn := NewAdapterWrapper(s.iface)
	// Forward NAT + reverse PAT: tunnel reads feed BOTH netstacks. The
	// forward proxy reassembles frames (TLS chunks aren't IP-aligned) and
	// terminates matching TCP; the reverse netstack needs the inbound
	// segments for its own listener-relayed flows. Non-matching traffic
	// continues to the normal TUN path (ICMP to this host, etc.).
	rev := s.natReverse
	if s.natProxy != nil || rev != nil {
		proxy := s.natProxy
		// Fresh stream: stale partial frames from the previous
		// connection would corrupt the new one's framing.
		if proxy != nil {
			proxy.Reset()
		}
		if rev != nil {
			rev.Reset()
		}
		tunCIDR := s.activeConfig().Config.Network.Address
		tunIP, _, _ := net.ParseCIDR(tunCIDR)
		pr, pw := io.Pipe()
		router := &tunnelRouter{
			Conn:      tunnelConn,
			assembler: &nat.FrameAssembler{},
			out:       pr,
			writer:    pw,
			closeOut:  func() { pw.CloseWithError(nil) },
		}
		router.route = func(frame []byte) bool {
			if len(frame) < 20 {
				return false
			}
			dst := net.IP(frame[16:20])
			if dst.Equal(tunIP) {
				// Reverse-PAT flows: the reverse netstack owns
				// TCP replies to its listener-dialed flows.
				if rev != nil && frame[9] == 6 {
					rev.DeliverOwned(frame)
					return true
				}
				return false // ICMP etc. pass to TUN/kernel
			}
			// Forward proxy owns peer TCP flows to egress hosts.
			if proxy != nil && frame[9] == 6 {
				proxy.DeliverOwned(frame)
				return true
			}
			return false
		}
		tunnelConn = router
		router.start()
	}

	// Reverse PAT + forward-proxy frame sink: netstack frames must flow
	// through THIS connection's TLS stream — the counting, TLS-wrapped
	// conn, never the raw TCP conn (writing pre-TLS corrupts the
	// client's TLS stream).
	if s.natProxy != nil || s.natReverse != nil {
		s.frameSink.Set(tunnelConn)
		// Clear only if the sink still holds THIS connection: a dying
		// connection's deferred clear must not clobber a newer one (QA:
		// stale-sink race blackholed server→client frames).
		defer func(prev net.Conn) { s.frameSink.ClearIf(prev) }(tunnelConn)
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

				if s.natProxy != nil || s.natReverse != nil {
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
	if s.natProxy != nil {
		sessions, pdrops, perrs, _ := s.natProxy.Stats()
		forwarded = int64(sessions)
		dropped += int64(pdrops + perrs)
		activeFlows = int64(sessions)
	}
	if s.natReverse != nil {
		acc, den, _ := s.natReverse.Stats()
		accepts = int64(acc)
		denies = int64(den)
	}
	s.monitor.UpdateNATMetrics(forwarded, returned, dropped, activeFlows, accepts, denies)

	// NAT counters (observable via monitor; the per-drop-site detail lives
	// in DropReasons() if a future QA cycle needs it again).
	if s.natProxy != nil {
		sessions, pdrops, perrs, denies := s.natProxy.Stats()
		s.logger.Info("NAT forward proxy stats",
			zap.Uint64("sessions", sessions),
			zap.Uint64("acl_denies", denies),
			zap.Uint64("relay_errors", perrs),
			zap.Uint64("dropped", pdrops),
		)
	}
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

	// Stop the forward proxy netstack first
	if s.natProxy != nil {
		s.natProxy.Stop()
		s.natProxy = nil
	}

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
