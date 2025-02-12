package network

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
)

// ConnectivityChecker handles network connectivity testing
type ConnectivityChecker struct {
	cfg *types.AppConfig
}

// NewConnectivityChecker creates a new connectivity checker
func NewConnectivityChecker(cfg *types.AppConfig) *ConnectivityChecker {
	return &ConnectivityChecker{
		cfg: cfg,
	}
}

// CheckServerConnectivity verifies connectivity to the server
func (c *ConnectivityChecker) CheckServerConnectivity(ctx context.Context) error {
	if c.cfg.Config.Tunnel == nil {
		return fmt.Errorf("tunnel configuration is missing")
	}

	// Create a dialer with timeout
	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	// Try TCP connection to server
	addr := fmt.Sprintf("%s:%d", c.cfg.Config.Tunnel.ServerAddress, c.cfg.Config.Tunnel.ServerPort)
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to connect to server %s: %w", addr, err)
	}
	defer conn.Close()

	// Set read deadline for health check
	if err := conn.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Send health check packet
	if _, err := conn.Write([]byte("HEALTH")); err != nil {
		return fmt.Errorf("failed to send health check: %w", err)
	}

	// Read response
	buf := make([]byte, 5)
	n, err := conn.Read(buf)
	if err != nil {
		return fmt.Errorf("failed to read health check response: %w", err)
	}

	if n != 5 || string(buf) != "ALIVE" {
		return fmt.Errorf("invalid health check response: %s", string(buf[:n]))
	}

	return nil
}

// CheckTunnelInterface verifies the tunnel interface is up and configured
func (c *ConnectivityChecker) CheckTunnelInterface(ctx context.Context) error {
	if c.cfg.Config.Network == nil {
		return fmt.Errorf("network configuration is missing")
	}

	iface, err := net.InterfaceByName(c.cfg.Config.Network.Interface)
	if err != nil {
		return fmt.Errorf("failed to get tunnel interface %s: %w", c.cfg.Config.Network.Interface, err)
	}

	if iface.Flags&net.FlagUp == 0 {
		return fmt.Errorf("tunnel interface %s is down", c.cfg.Config.Network.Interface)
	}

	addrs, err := iface.Addrs()
	if err != nil {
		return fmt.Errorf("failed to get interface addresses: %w", err)
	}

	// Check if configured address is present
	configuredAddr := c.cfg.Config.Network.Address
	found := false
	for _, addr := range addrs {
		if addr.String() == configuredAddr {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("configured address %s not found on interface %s", configuredAddr, c.cfg.Config.Network.Interface)
	}

	return nil
}

// CheckNetworkConnectivity performs comprehensive network connectivity checks
func (c *ConnectivityChecker) CheckNetworkConnectivity(ctx context.Context) error {
	// Check tunnel interface first
	if err := c.CheckTunnelInterface(ctx); err != nil {
		return fmt.Errorf("tunnel interface check failed: %w", err)
	}

	// For client mode check server connectivity
	if c.cfg.Config.Mode == string(types.ModeClient) {
		if err := c.CheckServerConnectivity(ctx); err != nil {
			return fmt.Errorf("server connectivity check failed: %w", err)
		}
	}

	return nil
}

// MonitorConnectivity continuously monitors network connectivity
func (c *ConnectivityChecker) MonitorConnectivity(ctx context.Context, interval time.Duration) (<-chan error, error) {
	errChan := make(chan error, 1)

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		defer close(errChan)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := c.CheckNetworkConnectivity(ctx); err != nil {
					select {
					case errChan <- err:
					default:
						// Channel is full, skip this error
					}
				}
			}
		}
	}()

	return errChan, nil
}
