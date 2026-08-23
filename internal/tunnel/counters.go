package tunnel

import (
	"net"
	"sync/atomic"
)

// countingConn wraps a net.Conn and accumulates byte counters for metrics.
// Reads count bytes received from the remote peer; writes count bytes sent.
type countingConn struct {
	net.Conn
	readBytes  *atomic.Int64
	writeBytes *atomic.Int64
}

func newCountingConn(c net.Conn, readBytes, writeBytes *atomic.Int64) *countingConn {
	return &countingConn{
		Conn:       c,
		readBytes:  readBytes,
		writeBytes: writeBytes,
	}
}

func (c *countingConn) Read(p []byte) (int, error) {
	n, err := c.Conn.Read(p)
	c.readBytes.Add(int64(n))
	return n, err
}

func (c *countingConn) Write(p []byte) (int, error) {
	n, err := c.Conn.Write(p)
	c.writeBytes.Add(int64(n))
	return n, err
}

func (c *countingConn) CloseWrite() error {
	if cw, ok := c.Conn.(closeWriter); ok {
		return cw.CloseWrite()
	}
	return nil
}

// closeWriter matches net.TCPConn.CloseWrite so Transfer's half-close logic
// keeps working through the wrapper.
type closeWriter interface {
	CloseWrite() error
}

var _ closeWriter = (*countingConn)(nil)
