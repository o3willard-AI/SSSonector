package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/adapter"
	"github.com/o3willard-AI/SSSonector/internal/config"
	"github.com/o3willard-AI/SSSonector/internal/monitor"
	"github.com/o3willard-AI/SSSonector/internal/throttle"
	"github.com/o3willard-AI/SSSonector/internal/tunnel"
	"go.uber.org/zap"
)

// Client represents a tunnel client
type Client struct {
	config  *config.Config
	monitor *monitor.Monitor
}

// NewClient creates a new tunnel client
func NewClient(cfg *config.Config) (*Client, error) {
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

	return &Client{
		config:  cfg,
		monitor: mon,
	}, nil
}

// Start starts the tunnel client
func (c *Client) Start() error {
	// Load CA certificate
	caCert, err := ioutil.ReadFile(c.config.Tunnel.CAFile)
	if err != nil {
		return fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return fmt.Errorf("failed to parse CA certificate")
	}

	// Load client certificate
	cert, err := tls.LoadX509KeyPair(c.config.Tunnel.CertFile, c.config.Tunnel.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to load client certificate: %w", err)
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caCertPool,
		MinVersion:   tls.VersionTLS12,
	}

	// Start monitoring
	if err := c.monitor.Start(); err != nil {
		return fmt.Errorf("failed to start monitoring: %w", err)
	}

	// Connect to server
	serverAddr := fmt.Sprintf("%s:%d", c.config.Tunnel.ServerAddress, c.config.Tunnel.ServerPort)
	conn, err := tls.Dial("tcp", serverAddr, tlsConfig)
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer conn.Close()

	c.monitor.Info("Connected to server",
		zap.String("address", serverAddr),
		zap.Bool("snmp_enabled", c.config.Monitor.SNMPEnabled))

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
		return fmt.Errorf("failed to create adapter: %w", err)
	}

	// Ensure proper cleanup on exit
	defer func() {
		c.monitor.Info("Cleaning up TUN interface", zap.String("name", tunName))
		if err := iface.Cleanup(); err != nil {
			c.monitor.Error("Failed to cleanup interface",
				zap.String("name", tunName),
				zap.Error(err))
		}
	}()

	// Configure adapter
	adapterCfg := &adapter.Config{
		Name:    tunName,
		Address: c.config.Network.Address,
		MTU:     c.config.Network.MTU,
	}
	if err := iface.Configure(adapterCfg); err != nil {
		return fmt.Errorf("failed to configure adapter: %w", err)
	}

	// Create throttler
	throttler := throttle.NewLimiter(conn, conn, c.config.Tunnel.UploadKbps*1024, c.config.Tunnel.DownloadKbps*1024)

	// Create and start tunnel
	tun, err := tunnel.New(conn, iface, throttler, c.monitor)
	if err != nil {
		return fmt.Errorf("failed to create tunnel: %w", err)
	}

	if err := tun.Start(); err != nil {
		return fmt.Errorf("failed to start tunnel: %w", err)
	}

	// Wait for tunnel to close
	<-tun.Done()
	return nil
}

// Stop stops the tunnel client
func (c *Client) Stop() error {
	c.monitor.Info("Stopping tunnel client")

	// Stop monitoring first to ensure final metrics are collected
	if c.monitor != nil {
		c.monitor.Info("Stopping monitoring")
		c.monitor.Stop()
	}

	return nil
}
