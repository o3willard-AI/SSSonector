package connection

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestConnectionManager(t *testing.T) {
	logger := zap.NewNop()

	t.Run("respects connection limit", func(t *testing.T) {
		config := &Config{
			MaxConnections: 2,
			KeepAlive:      true,
			KeepAliveIdle:  30 * time.Second,
		}

		manager := NewManager(logger, config)
		defer manager.Stop()

		// Create test connections
		conn1 := newTestConn("client1")
		conn2 := newTestConn("client2")
		conn3 := newTestConn("client3")

		// First two connections should succeed
		if err := manager.Accept(conn1); err != nil {
			t.Errorf("first connection failed: %v", err)
		}
		if err := manager.Accept(conn2); err != nil {
			t.Errorf("second connection failed: %v", err)
		}

		// Third connection should fail
		if err := manager.Accept(conn3); err == nil {
			t.Error("expected third connection to fail")
		}

		// Verify connection count
		if count := manager.GetConnectionCount(); count != 2 {
			t.Errorf("expected 2 connections, got %d", count)
		}
	})

	t.Run("handles connection callbacks", func(t *testing.T) {
		config := &Config{
			MaxConnections: 1,
		}

		var (
			connectCalled    bool
			disconnectCalled bool
		)

		manager := NewManager(logger, config)
		defer manager.Stop()

		manager.SetCallbacks(
			func(conn net.Conn) { connectCalled = true },
			func(conn net.Conn, err error) { disconnectCalled = true },
		)

		// Test connect callback
		conn := newTestConn("client")
		if err := manager.Accept(conn); err != nil {
			t.Errorf("connection failed: %v", err)
		}

		if !connectCalled {
			t.Error("connect callback not called")
		}

		// Test disconnect callback
		manager.Stop()

		if !disconnectCalled {
			t.Error("disconnect callback not called")
		}
	})

	t.Run("tracks connection info", func(t *testing.T) {
		config := &Config{
			MaxConnections: 1,
		}

		manager := NewManager(logger, config)
		defer manager.Stop()

		// Add connection
		conn := newTestConn("client")
		if err := manager.Accept(conn); err != nil {
			t.Errorf("connection failed: %v", err)
		}

		// Update stats
		manager.UpdateStats("client", 100, 200)

		// Get connection info
		infos := manager.GetConnections()
		if len(infos) != 1 {
			t.Fatalf("expected 1 connection info, got %d", len(infos))
		}

		info := infos[0]
		if info.RemoteAddr != "client" {
			t.Errorf("expected remote addr 'client', got %s", info.RemoteAddr)
		}
		if info.State != StateConnected {
			t.Errorf("expected state Connected, got %s", info.State)
		}
		if info.BytesSent != 100 {
			t.Errorf("expected 100 bytes sent, got %d", info.BytesSent)
		}
		if info.BytesReceived != 200 {
			t.Errorf("expected 200 bytes received, got %d", info.BytesReceived)
		}
	})

	t.Run("handles connection retry", func(t *testing.T) {
		config := &Config{
			RetryAttempts:  2,
			RetryInterval:  10 * time.Millisecond,
			ConnectTimeout: 50 * time.Millisecond,
		}

		manager := NewManager(logger, config)
		defer manager.Stop()

		// Try to connect to non-existent server
		_, err := manager.Connect("tcp", "localhost:1")
		if err == nil {
			t.Error("expected connection to fail")
		}

		// Verify retry attempts
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("expected deadline exceeded error, got %v", err)
		}

		state := manager.GetState()
		if state != StateDisconnected {
			t.Errorf("expected state Disconnected, got %s", state)
		}
	})
}

// testConn implements a mock net.Conn for testing
type testConn struct {
	addr string
}

func newTestConn(addr string) net.Conn {
	return &testConn{addr: addr}
}

func (c *testConn) Read(b []byte) (n int, err error)   { return 0, nil }
func (c *testConn) Write(b []byte) (n int, err error)  { return len(b), nil }
func (c *testConn) Close() error                       { return nil }
func (c *testConn) LocalAddr() net.Addr                { return &testAddr{c.addr} }
func (c *testConn) RemoteAddr() net.Addr               { return &testAddr{c.addr} }
func (c *testConn) SetDeadline(t time.Time) error      { return nil }
func (c *testConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *testConn) SetWriteDeadline(t time.Time) error { return nil }

// testAddr implements net.Addr for testing
type testAddr struct {
	addr string
}

func (a *testAddr) Network() string { return "test" }
func (a *testAddr) String() string  { return a.addr }
