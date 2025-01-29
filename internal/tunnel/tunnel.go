package tunnel

import (
	"fmt"
	"net"
	"time"

	"SSSonector/internal/adapter"
	"SSSonector/internal/cert"
	"SSSonector/internal/config"
	"SSSonector/internal/connection"
	"SSSonector/internal/monitor"

	"go.uber.org/zap"
)

// Tunnel represents a secure tunnel connection
type Tunnel struct {
	logger      *zap.Logger
	tlsManager  *TLSManager
	config      *config.Config
	iface       adapter.Interface
	ifManager   adapter.Manager
	listener    net.Listener
	connManager *connection.Manager
	snmpAgent   *monitor.SNMPAgent
	certManager *cert.Manager
	done        chan struct{}
}

// NewTunnel creates a new tunnel instance
func NewTunnel(logger *zap.Logger, cfg *config.Config) (*Tunnel, error) {
	tlsManager := NewTLSManager(logger, &cfg.TLS)

	// Initialize certificate manager
	certManager := cert.NewManager(logger, &cert.Config{
		CertFile:     cfg.TLS.CertFile,
		KeyFile:      cfg.TLS.KeyFile,
		AutoGenerate: cfg.TLS.AutoGenerate,
		ValidityDays: cfg.TLS.ValidityDays,
		CommonName:   fmt.Sprintf("ssl-tunnel-%s", cfg.Mode),
		IPAddresses:  []string{cfg.Network.Address},
		DNSNames:     []string{"localhost"},
	})

	if err := certManager.Initialize(); err != nil {
		return nil, fmt.Errorf("certificate setup failed: %w", err)
	}

	// Create interface manager
	ifManager, err := adapter.NewManager(logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create interface manager: %w", err)
	}

	// Create network interface
	iface, err := ifManager.Create(&cfg.Network)
	if err != nil {
		return nil, fmt.Errorf("failed to create network interface: %w", err)
	}

	// Create connection manager
	connManager := connection.NewManager(logger, &connection.Config{
		RetryAttempts:  cfg.Network.RetryAttempts,
		RetryInterval:  time.Duration(cfg.Network.RetryInterval) * time.Second,
		ConnectTimeout: 30 * time.Second,
		KeepAlive:      true,
		KeepAliveIdle:  30 * time.Second,
		MaxConnections: cfg.Network.MaxClients,
	})

	// Create SNMP agent if enabled
	var snmpAgent *monitor.SNMPAgent
	if cfg.Monitor.SNMPEnabled {
		snmpAgent = monitor.NewSNMPAgent(logger, cfg.Monitor.SNMPPort, cfg.Monitor.Community)
		if err := snmpAgent.Start(); err != nil {
			return nil, fmt.Errorf("failed to start SNMP agent: %w", err)
		}
	}

	t := &Tunnel{
		logger:      logger,
		tlsManager:  tlsManager,
		config:      cfg,
		iface:       iface,
		ifManager:   ifManager,
		snmpAgent:   snmpAgent,
		certManager: certManager,
		connManager: connManager,
		done:        make(chan struct{}),
	}

	// Set connection callbacks
	connManager.SetCallbacks(
		t.handleNewConnection,
		t.handleDisconnection,
	)

	return t, nil
}

// Start starts the tunnel in either server or client mode
func (t *Tunnel) Start() error {
	switch t.config.Mode {
	case "server":
		return t.startServer()
	case "client":
		return t.startClient()
	default:
		return fmt.Errorf("invalid mode: %s", t.config.Mode)
	}
}

// startServer starts the tunnel in server mode
func (t *Tunnel) startServer() error {
	addr := fmt.Sprintf("%s:%d", t.config.Network.ListenAddress, t.config.Network.ListenPort)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	// Wrap with TLS
	tlsListener, err := t.tlsManager.WrapListener(listener)
	if err != nil {
		listener.Close()
		return fmt.Errorf("failed to create TLS listener: %w", err)
	}

	t.listener = tlsListener
	t.logger.Info("Server listening",
		zap.String("address", addr),
		zap.Int("max_clients", t.config.Network.MaxClients),
	)

	go t.acceptConnections()
	return nil
}

// startClient starts the tunnel in client mode
func (t *Tunnel) startClient() error {
	addr := fmt.Sprintf("%s:%d", t.config.Network.ServerAddress, t.config.Network.ServerPort)
	_, err := t.connManager.Connect("tcp", addr)
	return err
}

// acceptConnections accepts incoming connections
func (t *Tunnel) acceptConnections() {
	for {
		conn, err := t.listener.Accept()
		if err != nil {
			select {
			case <-t.done:
				return
			default:
				t.logger.Error("Failed to accept connection", zap.Error(err))
				continue
			}
		}

		if err := t.connManager.Accept(conn); err != nil {
			t.logger.Warn("Connection rejected",
				zap.String("remote_addr", conn.RemoteAddr().String()),
				zap.Error(err),
			)
			conn.Close()
		}
	}
}

// handleNewConnection handles a new connection
func (t *Tunnel) handleNewConnection(conn net.Conn) {
	t.logger.Info("New connection established",
		zap.String("remote_addr", conn.RemoteAddr().String()),
		zap.Int("active_connections", t.connManager.GetConnectionCount()),
	)

	// Create transfer for handling data between interface and connection
	transfer := NewTransfer(t.logger, t.iface, conn,
		t.config.Throttle.UploadKbps,
		t.config.Throttle.DownKbps)

	// Start data transfer
	transfer.Start()

	// Log and monitor transfer statistics periodically
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-t.done:
			transfer.Stop()
			return
		case <-ticker.C:
			stats := transfer.GetStatistics()
			t.logger.Info("Transfer statistics",
				zap.String("remote_addr", conn.RemoteAddr().String()),
				zap.Uint64("bytes_sent", stats.BytesSent),
				zap.Uint64("bytes_received", stats.BytesReceived),
				zap.Uint64("packets_lost", stats.PacketsLost),
				zap.Int64("latency_us", stats.LastLatency),
			)

			// Update connection statistics
			t.connManager.UpdateStats(
				conn.RemoteAddr().String(),
				stats.BytesSent,
				stats.BytesReceived,
			)

			// Update SNMP statistics if enabled
			if t.snmpAgent != nil {
				t.snmpAgent.UpdateStats(
					stats.BytesReceived,
					stats.BytesSent,
					stats.PacketsLost,
					stats.LastLatency,
				)
			}
		}
	}
}

// handleDisconnection handles a connection disconnection
func (t *Tunnel) handleDisconnection(conn net.Conn, err error) {
	t.logger.Info("Connection closed",
		zap.String("remote_addr", conn.RemoteAddr().String()),
		zap.Error(err),
		zap.Int("active_connections", t.connManager.GetConnectionCount()),
	)
}

// Stop stops the tunnel
func (t *Tunnel) Stop() error {
	close(t.done)

	// Close listener if in server mode
	if t.listener != nil {
		t.listener.Close()
	}

	// Stop connection manager
	t.connManager.Stop()

	// Close network interface
	if t.iface != nil {
		if err := t.iface.Close(); err != nil {
			t.logger.Error("Failed to close network interface", zap.Error(err))
		}
	}

	// Stop SNMP agent if enabled
	if t.snmpAgent != nil {
		if err := t.snmpAgent.Stop(); err != nil {
			t.logger.Error("Failed to stop SNMP agent", zap.Error(err))
		}
	}

	// Stop certificate manager
	if t.certManager != nil {
		t.certManager.Stop()
	}

	return nil
}
