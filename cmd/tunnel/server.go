package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/adapter"
	"github.com/o3willard-AI/SSSonector/internal/config"
	"github.com/o3willard-AI/SSSonector/internal/monitor"
	"github.com/o3willard-AI/SSSonector/internal/throttle"
	"github.com/o3willard-AI/SSSonector/internal/tunnel"
	"go.uber.org/zap"
)

// Server represents a tunnel server
type Server struct {
	config   *config.Config
	listener net.Listener
	monitor  *monitor.Monitor
}

// NewServer creates a new tunnel server
func NewServer(cfg *config.Config) (*Server, error) {
	// Initialize monitoring
	monCfg := &monitor.Config{
		LogFile:       cfg.Monitor.LogFile,
		SNMPEnabled:   cfg.Monitor.SNMPEnabled,
		SNMPPort:      cfg.Monitor.SNMPPort,
		SNMPCommunity: cfg.Monitor.SNMPCommunity,
		SNMPAddress:   cfg.Monitor.SNMPAddress,
	}
	mon, err := monitor.New(monCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize monitoring: %w", err)
	}

	return &Server{
		config:  cfg,
		monitor: mon,
	}, nil
}

// Start starts the tunnel server
func (s *Server) Start() error {
	// Create TLS config
	cert, err := tls.LoadX509KeyPair(s.config.Tunnel.CertFile, s.config.Tunnel.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to load certificates: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	// Create TLS listener
	listenAddr := fmt.Sprintf("%s:%d", s.config.Tunnel.ListenAddress, s.config.Tunnel.ListenPort)
	listener, err := tls.Listen("tcp", listenAddr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}
	s.listener = listener

	// Start monitoring
	if err := s.monitor.Start(); err != nil {
		return fmt.Errorf("failed to start monitoring: %w", err)
	}

	s.monitor.Info("Server started",
		zap.String("address", listenAddr),
		zap.Bool("snmp_enabled", s.config.Monitor.SNMPEnabled))

	// Accept connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			s.monitor.Error("Failed to accept connection", zap.Error(err))
			continue
		}

		go s.handleConnection(conn)
	}
}

// Stop stops the tunnel server
func (s *Server) Stop() error {
	s.monitor.Info("Stopping tunnel server")

	// Close listener first to stop accepting new connections
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			s.monitor.Error("Failed to close listener", zap.Error(err))
		}
	}

	// Stop monitoring last to ensure all cleanup metrics are collected
	if s.monitor != nil {
		s.monitor.Info("Stopping monitoring")
		s.monitor.Stop()
	}

	return nil
}

// handleConnection handles a client connection
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	remoteAddr := conn.RemoteAddr().String()
	s.monitor.Info("New client connection", zap.String("remote_addr", remoteAddr))

	// Create TUN adapter with robust options
	tunName := fmt.Sprintf("tun%d", time.Now().UnixNano())
	adapterOpts := &adapter.Options{
		RetryAttempts:  5,     // More retries for initial setup
		RetryDelay:     200,   // 200ms between retries
		CleanupTimeout: 10000, // 10 seconds for cleanup
		ValidateState:  true,  // Always validate interface state
	}
	iface, err := adapter.New(tunName, adapterOpts)
	if err != nil {
		s.monitor.Error("Failed to create adapter",
			zap.String("name", tunName),
			zap.Error(err))
		return
	}

	// Ensure proper cleanup on connection close
	defer func() {
		s.monitor.Info("Cleaning up TUN interface", zap.String("name", tunName))
		if err := iface.Cleanup(); err != nil {
			s.monitor.Error("Failed to cleanup interface",
				zap.String("name", tunName),
				zap.Error(err))
		}
	}()

	// Configure adapter
	adapterCfg := &adapter.Config{
		Name:    tunName,
		Address: s.config.Network.Address,
		MTU:     s.config.Network.MTU,
	}
	if err := iface.Configure(adapterCfg); err != nil {
		s.monitor.Error("Failed to configure adapter",
			zap.String("name", tunName),
			zap.Error(err))
		return
	}

	s.monitor.Info("TUN interface configured",
		zap.String("name", tunName),
		zap.String("address", s.config.Network.Address),
		zap.Int("mtu", s.config.Network.MTU))

	// Create throttler
	throttler := throttle.NewLimiter(conn, conn,
		s.config.Tunnel.UploadKbps*1024,
		s.config.Tunnel.DownloadKbps*1024)

	// Create and start tunnel
	tun, err := tunnel.New(conn, iface, throttler, s.monitor)
	if err != nil {
		s.monitor.Error("Failed to create tunnel",
			zap.String("remote_addr", remoteAddr),
			zap.Error(err))
		return
	}

	if err := tun.Start(); err != nil {
		s.monitor.Error("Failed to start tunnel",
			zap.String("remote_addr", remoteAddr),
			zap.Error(err))
		return
	}

	s.monitor.Info("Tunnel started",
		zap.String("remote_addr", remoteAddr),
		zap.String("tun_name", tunName))

	// Wait for tunnel to close
	<-tun.Done()

	s.monitor.Info("Tunnel closed",
		zap.String("remote_addr", remoteAddr),
		zap.String("tun_name", tunName))
}
