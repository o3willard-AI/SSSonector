package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/adapter"
	"github.com/o3willard-AI/SSSonector/internal/cert"
	"github.com/o3willard-AI/SSSonector/internal/config"
	"github.com/o3willard-AI/SSSonector/internal/connection"
	"github.com/o3willard-AI/SSSonector/internal/monitor"
	"github.com/o3willard-AI/SSSonector/internal/throttle"
	"github.com/o3willard-AI/SSSonector/internal/tunnel"
	"go.uber.org/zap"
)

func init() {
	defaultMode = "server"
}

type Server struct {
	config     *config.Config
	logger     *zap.Logger
	listener   net.Listener
	tlsConfig  *tls.Config
	adapter    adapter.Interface
	connMgr    *connection.Manager
	monitor    *monitor.Monitor
	shutdownCh chan struct{}
	testMode   bool
	wg         sync.WaitGroup
}

func NewServer(cfg *config.Config, testMode bool) (*Server, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Initialize TLS configuration
	tlsCfg, err := cert.NewTLSConfig(cfg.Tunnel.CertFile, cfg.Tunnel.KeyFile, cfg.Tunnel.CAFile, true, testMode)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize TLS config: %w", err)
	}

	// Initialize network adapter
	iface, err := adapter.New(cfg.Network.Interface)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize network interface: %w", err)
	}

	// Initialize connection manager
	connMgr := connection.NewManager(logger, &connection.Config{
		MaxConnections: cfg.Tunnel.MaxClients,
	})

	// Initialize monitoring
	mon, err := monitor.New(&monitor.Config{
		LogFile:       cfg.Monitor.LogFile,
		SNMPEnabled:   cfg.Monitor.SNMPEnabled,
		SNMPPort:      cfg.Monitor.SNMPPort,
		SNMPCommunity: cfg.Monitor.SNMPCommunity,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize monitoring: %w", err)
	}

	return &Server{
		config:     cfg,
		logger:     logger,
		tlsConfig:  tlsCfg,
		adapter:    iface,
		connMgr:    connMgr,
		monitor:    mon,
		shutdownCh: make(chan struct{}),
		testMode:   testMode,
	}, nil
}

func (s *Server) Start() error {
	// Configure network interface
	if err := s.adapter.Configure(&adapter.Config{
		Name:    s.config.Network.Interface,
		Address: s.config.Network.Address,
		MTU:     s.config.Network.MTU,
	}); err != nil {
		return fmt.Errorf("failed to configure network interface: %w", err)
	}

	// Create TLS listener
	listener, err := tls.Listen("tcp", fmt.Sprintf("%s:%d", s.config.Tunnel.ListenAddress, s.config.Tunnel.ListenPort), s.tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to create TLS listener: %w", err)
	}
	s.listener = listener

	// Start monitoring
	if err := s.monitor.Start(); err != nil {
		return fmt.Errorf("failed to start monitoring: %w", err)
	}

	// Start accepting connections in background
	s.wg.Add(1)
	go s.acceptConnections()

	// Wait a bit for the server to start
	time.Sleep(2 * time.Second)

	// Verify server is listening
	if _, err := net.Listen("tcp", fmt.Sprintf("%s:%d", s.config.Tunnel.ListenAddress, s.config.Tunnel.ListenPort)); err == nil {
		return fmt.Errorf("port %d is not in use, server failed to start", s.config.Tunnel.ListenPort)
	}

	return nil
}

func (s *Server) acceptConnections() {
	defer s.wg.Done()
	for {
		select {
		case <-s.shutdownCh:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				s.monitor.Error("Failed to accept connection", err)
				if s.testMode {
					s.monitor.Info("Test mode: certificate expired, shutting down")
					s.initiateShutdown()
				}
				return
			}

			if !s.connMgr.CanAcceptMore() {
				s.monitor.Warn("Maximum client connections reached, rejecting connection")
				conn.Close()
				continue
			}

			s.wg.Add(1)
			go s.handleConnection(conn)
		}
	}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()

	// Register connection
	if err := s.connMgr.Add(conn); err != nil {
		s.monitor.Error("Failed to register connection", err)
		conn.Close()
		return
	}
	defer s.connMgr.Remove(conn)

	// Create throttler
	throttler := throttle.NewLimiter(conn, conn, int64(s.config.Tunnel.UploadKbps), int64(s.config.Tunnel.DownloadKbps))

	// Create tunnel
	tun, err := tunnel.New(conn, s.adapter, throttler)
	if err != nil {
		s.monitor.Error("Failed to create tunnel", err)
		return
	}

	// Start tunnel
	if err := tun.Start(); err != nil {
		s.monitor.Error("Failed to start tunnel", err)
		return
	}

	// Wait for tunnel completion
	<-tun.Done()

	// In test mode, if the tunnel closes, shut down the server
	if s.testMode {
		s.monitor.Info("Test mode: certificate expired, shutting down")
		s.initiateShutdown()
	}
}

func (s *Server) initiateShutdown() {
	select {
	case <-s.shutdownCh:
		// Already shutting down
		return
	default:
		close(s.shutdownCh)
	}
}

func (s *Server) Stop() error {
	s.initiateShutdown()

	// Close listener
	if s.listener != nil {
		s.listener.Close()
	}

	// Wait for all goroutines to finish
	s.wg.Wait()

	// Stop monitoring
	s.monitor.Stop()

	// Close all connections
	s.connMgr.CloseAll()

	// Cleanup network interface
	if err := s.adapter.Cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup network interface: %w", err)
	}

	return nil
}
