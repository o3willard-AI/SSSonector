package tunnel

import (
	"net"
	"sync/atomic"
	"time"
)

// countingConn wraps a net.Conn and accumulates byte counters for metrics.
// Reads count bytes received from the remote peer; writes count bytes sent.
type countingConn struct {
	net.Conn
	readBytes  *atomic.Int64
	writeBytes *atomic.Int64

	// idleTimeout arms read/write deadlines that are pushed forward on
	// every successful transfer of bytes; a silent connection therefore
	// fails its next deadline instead of lingering half-open.
	idleTimeout time.Duration
	armedRead   bool
	armedWrite  bool
}

func newCountingConn(c net.Conn, idle time.Duration, readBytes, writeBytes *atomic.Int64) *countingConn {
	return &countingConn{
		Conn:        c,
		readBytes:   readBytes,
		writeBytes:  writeBytes,
		idleTimeout: idle,
	}
}

func (c *countingConn) armIdle(arm *bool, set func(time.Time) error) {
	if c.idleTimeout <= 0 || *arm {
		return
	}
	if err := set(time.Now().Add(c.idleTimeout)); err == nil {
		*arm = true
	}
}

func (c *countingConn) Read(p []byte) (int, error) {
	c.armIdle(&c.armedRead, func(t time.Time) error { return c.SetReadDeadline(t) })
	n, err := c.Conn.Read(p)
	c.readBytes.Add(int64(n))
	if n > 0 && err == nil && c.idleTimeout > 0 {
		_ = c.SetReadDeadline(time.Now().Add(c.idleTimeout))
	}
	return n, err
}

func (c *countingConn) Write(p []byte) (int, error) {
	c.armIdle(&c.armedWrite, func(t time.Time) error { return c.SetWriteDeadline(t) })
	n, err := c.Conn.Write(p)
	c.writeBytes.Add(int64(n))
	if n > 0 && err == nil && c.idleTimeout > 0 {
		_ = c.SetWriteDeadline(time.Now().Add(c.idleTimeout))
	}
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
