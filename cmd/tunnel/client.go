package main

import (
	"crypto/tls"
	"fmt"
	"net"

	"github.com/o3willard-AI/SSSonector/internal/adapter"
	"github.com/o3willard-AI/SSSonector/internal/cert"
	"github.com/o3willard-AI/SSSonector/internal/config"
	"github.com/o3willard-AI/SSSonector/internal/monitor"
	"github.com/o3willard-AI/SSSonector/internal/throttle"
	"github.com/o3willard-AI/SSSonector/internal/tunnel"
	"go.uber.org/zap"
)

func init() {
	defaultMode = "client"
}

type Client struct {
	config     *config.Config
	logger     *zap.Logger
	conn       net.Conn
	tlsConfig  *tls.Config
	adapter    adapter.Interface
	monitor    *monitor.Monitor
	tunnel     *tunnel.Tunnel
	shutdownCh chan struct{}
}

func NewClient(cfg *config.Config) (*Client, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Initialize TLS configuration
	tlsCfg, err := cert.NewTLSConfig(cfg.Tunnel.CertFile, cfg.Tunnel.KeyFile, cfg.Tunnel.CAFile, false)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize TLS config: %w", err)
	}

	// Initialize network adapter
	iface, err := adapter.New(cfg.Network.Interface)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize network interface: %w", err)
	}

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

	return &Client{
		config:     cfg,
		logger:     logger,
		tlsConfig:  tlsCfg,
		adapter:    iface,
		monitor:    mon,
		shutdownCh: make(chan struct{}),
	}, nil
}

func (c *Client) Start() error {
	// Configure network interface
	if err := c.adapter.Configure(&adapter.Config{
		Name:    c.config.Network.Interface,
		Address: c.config.Network.Address,
		MTU:     c.config.Network.MTU,
	}); err != nil {
		return fmt.Errorf("failed to configure network interface: %w", err)
	}

	// Start monitoring
	if err := c.monitor.Start(); err != nil {
		return fmt.Errorf("failed to start monitoring: %w", err)
	}

	// Connect to server
	if err := c.connect(); err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}

	return nil
}

func (c *Client) connect() error {
	// Connect to server
	conn, err := tls.Dial("tcp", fmt.Sprintf("%s:%d", c.config.Tunnel.ServerAddress, c.config.Tunnel.ServerPort), c.tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	c.conn = conn

	// Create throttler
	throttler := throttle.NewLimiter(conn, conn, int64(c.config.Tunnel.UploadKbps), int64(c.config.Tunnel.DownloadKbps))

	// Create tunnel
	tun, err := tunnel.New(conn, c.adapter, throttler)
	if err != nil {
		conn.Close()
		return fmt.Errorf("failed to create tunnel: %w", err)
	}
	c.tunnel = tun

	// Start tunnel
	if err := tun.Start(); err != nil {
		conn.Close()
		return fmt.Errorf("failed to start tunnel: %w", err)
	}

	// Monitor tunnel in background
	go c.monitorTunnel()

	return nil
}

func (c *Client) monitorTunnel() {
	select {
	case <-c.shutdownCh:
		return
	case <-c.tunnel.Done():
		c.monitor.Info("Tunnel closed, attempting reconnect")
		for {
			select {
			case <-c.shutdownCh:
				return
			default:
				if err := c.connect(); err != nil {
					c.monitor.Error("Failed to reconnect", err)
					continue
				}
				return
			}
		}
	}
}

func (c *Client) Stop() error {
	// Signal shutdown
	close(c.shutdownCh)

	// Stop tunnel
	if c.tunnel != nil {
		c.tunnel.Stop()
	}

	// Close connection
	if c.conn != nil {
		c.conn.Close()
	}

	// Stop monitoring
	c.monitor.Stop()

	// Cleanup network interface
	if err := c.adapter.Cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup network interface: %w", err)
	}

	return nil
}
