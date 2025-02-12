package connection

import (
	"context"
	"errors"
	"net"
)

// MockConn implements a mock net.Conn for testing
type MockConn struct {
	net.Conn
	remoteAddr net.Addr
	closed     bool
}

func (c *MockConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *MockConn) Close() error {
	c.closed = true
	return nil
}

// SetupMockConn creates a new mock connection for testing
func SetupMockConn() *MockConn {
	return &MockConn{
		remoteAddr: &net.TCPAddr{
			IP:   net.ParseIP("192.168.1.1"),
			Port: 12345,
		},
	}
}

// MockDialer implements a mock Dialer for testing
type MockDialer struct {
	ShouldFail bool
	Conn       net.Conn
}

func (d *MockDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if d.ShouldFail {
		return nil, errors.New("connection failed")
	}
	return d.Conn, nil
}
