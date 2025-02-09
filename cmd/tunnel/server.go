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
	configManager *config.ConfigManager
	listener      net.Listener
	monitor       *monitor.Monitor
	done          chan struct{}
}

// NewServer creates a new tunnel server
func NewServer(configPath string) (*Server, error) {
	// Initialize monitoring with default config first
	defaultCfg := config.DefaultConfig()
	monCfg := &monitor.Config{
		LogFile:       defaultCfg.Monitor.LogFile,
		SNMPEnabled:   defaultCfg.Monitor.SNMPEnabled,
		SNMPPort:      defaultCfg.Monitor.SNMPPort,
		SNMPCommunity: defaultCfg.Monitor.SNMPCommunity,
		SNMPAddress:   defaultCfg.Monitor.SNMPAddress,
	}
	mon, err := monitor.New(monCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize monitoring: %w", err)
	}

	// Create config manager
	configManager, err := config.NewConfigManager(configPath, mon.Logger())
	if err != nil {
		mon.Stop()
		return nil, fmt.Errorf("failed to create config manager: %w", err)
	}

	return &Server{
		configManager: configManager,
		monitor:       mon,
		done:          make(chan struct{}),
	}, nil
}

// Start starts the tunnel server
func (s *Server) Start() error {
	cfg := s.configManager.GetConfig()

	// Create TLS config
	cert, err := tls.LoadX509KeyPair(cfg.Tunnel.CertFile, cfg.Tunnel.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to load certificates: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	// Create TLS listener
	listenAddr := fmt.Sprintf("%s:%d", cfg.Tunnel.ListenAddress, cfg.Tunnel.ListenPort)
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
		zap.Bool("snmp_enabled", cfg.Monitor.SNMPEnabled))

	// Accept connections
	go s.acceptConnections(listener)

	// Wait for shutdown
	<-s.done
	return nil
}

// Stop stops the tunnel server
func (s *Server) Stop() error {
	s.monitor.Info("Stopping tunnel server")

	// Signal shutdown
	close(s.done)

	// Close listener first to stop accepting new connections
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			s.monitor.Error("Failed to close listener", zap.Error(err))
		}
	}

	// Stop config manager
	if err := s.configManager.Close(); err != nil {
		s.monitor.Error("Failed to close config manager", zap.Error(err))
	}

	// Stop monitoring last to ensure all cleanup metrics are collected
	if s.monitor != nil {
		s.monitor.Info("Stopping monitoring")
		s.monitor.Stop()
	}

	return nil
}

// acceptConnections handles incoming connections
func (s *Server) acceptConnections(listener net.Listener) {
	for {
		select {
		case <-s.done:
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				if !isClosedError(err) {
					s.monitor.Error("Failed to accept connection", zap.Error(err))
				}
				continue
			}
			go s.handleConnection(conn)
		}
	}
}

// isClosedError checks if the error is due to closed listener
func isClosedError(err error) bool {
	return err.Error() == "use of closed network connection"
}

// handleConnection handles a client connection
func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	cfg := s.configManager.GetConfig()
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
		Address: cfg.Network.Address,
		MTU:     cfg.Network.MTU,
	}
	if err := iface.Configure(adapterCfg); err != nil {
		s.monitor.Error("Failed to configure adapter",
			zap.String("name", tunName),
			zap.Error(err))
		return
	}

	s.monitor.Info("TUN interface configured",
		zap.String("name", tunName),
		zap.String("address", cfg.Network.Address),
		zap.Int("mtu", cfg.Network.MTU))

	// Create throttler with hot reload support
	throttler := throttle.NewLimiter(conn, conn,
		cfg.Tunnel.UploadKbps*1024,
		cfg.Tunnel.DownloadKbps*1024)

	// Register throttler for config updates
	s.configManager.RegisterWatcher(throttler)

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
