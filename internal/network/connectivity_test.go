package network

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/stretchr/testify/assert"
)

func setupTestServer(t *testing.T) (string, func()) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	assert.NoError(t, err)

	// Start test server
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return // listener closed
			}

			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 6)
				n, err := c.Read(buf)
				if err != nil {
					return
				}

				if n == 6 && string(buf[:n]) == "HEALTH" {
					c.Write([]byte("ALIVE"))
				}
			}(conn)
		}
	}()

	return listener.Addr().String(), func() {
		listener.Close()
	}
}

func TestConnectivityChecker(t *testing.T) {
	addr, cleanup := setupTestServer(t)
	defer cleanup()

	host, port, err := net.SplitHostPort(addr)
	assert.NoError(t, err)

	cfg := &types.AppConfig{
		Type:    types.TypeServer,
		Version: "1.0.0",
		Config: &types.ServiceConfig{
			Mode: types.ModeClient,
			Network: &types.NetworkConfig{
				Interface: "lo", // Use loopback for testing
				Address:   "127.0.0.1/8",
			},
			Tunnel: &types.TunnelConfig{
				ServerAddress: host,
				ServerPort:    atoi(port),
			},
		},
	}

	checker := NewConnectivityChecker(cfg)

	t.Run("check server connectivity", func(t *testing.T) {
		err := checker.CheckServerConnectivity(context.Background())
		assert.NoError(t, err)
	})

	t.Run("check tunnel interface", func(t *testing.T) {
		err := checker.CheckTunnelInterface(context.Background())
		assert.NoError(t, err)
	})

	t.Run("check network connectivity", func(t *testing.T) {
		err := checker.CheckNetworkConnectivity(context.Background())
		assert.NoError(t, err)
	})

	t.Run("monitor connectivity", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		errChan, err := checker.MonitorConnectivity(ctx, 500*time.Millisecond)
		assert.NoError(t, err)

		// Should not receive any errors
		select {
		case err := <-errChan:
			t.Errorf("unexpected error: %v", err)
		case <-ctx.Done():
			// Success - no errors received
		}
	})
}

func TestConnectivityCheckerErrors(t *testing.T) {
	t.Run("missing tunnel config", func(t *testing.T) {
		cfg := &types.AppConfig{
			Type:    types.TypeServer,
			Version: "1.0.0",
			Config: &types.ServiceConfig{
				Mode: types.ModeClient,
			},
		}
		checker := NewConnectivityChecker(cfg)
		err := checker.CheckServerConnectivity(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tunnel configuration is missing")
	})

	t.Run("missing network config", func(t *testing.T) {
		cfg := &types.AppConfig{
			Type:    types.TypeServer,
			Version: "1.0.0",
			Config: &types.ServiceConfig{
				Mode: types.ModeClient,
			},
		}
		checker := NewConnectivityChecker(cfg)
		err := checker.CheckTunnelInterface(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "network configuration is missing")
	})

	t.Run("invalid interface", func(t *testing.T) {
		cfg := &types.AppConfig{
			Type:    types.TypeServer,
			Version: "1.0.0",
			Config: &types.ServiceConfig{
				Mode: types.ModeClient,
				Network: &types.NetworkConfig{
					Interface: "invalid0",
					Address:   "192.168.1.1/24",
				},
			},
		}
		checker := NewConnectivityChecker(cfg)
		err := checker.CheckTunnelInterface(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get tunnel interface")
	})

	t.Run("server connection failure", func(t *testing.T) {
		cfg := &types.AppConfig{
			Type:    types.TypeServer,
			Version: "1.0.0",
			Config: &types.ServiceConfig{
				Mode: types.ModeClient,
				Tunnel: &types.TunnelConfig{
					ServerAddress: "invalid.host",
					ServerPort:    12345,
				},
			},
		}
		checker := NewConnectivityChecker(cfg)
		err := checker.CheckServerConnectivity(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to connect to server")
	})
}

// Helper function to convert string port to int
func atoi(s string) int {
	var port int
	fmt.Sscanf(s, "%d", &port)
	return port
}
