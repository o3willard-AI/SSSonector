package tunnel

import (
	"net"
	"strings"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/adapter"
	"github.com/o3willard-AI/SSSonector/internal/nat"
)

// natIntercept wraps a net.Conn (the TUN adapter) and routes packets
// through the NAT engine before/after the normal data path. A nil
// engine means NAT is disabled and the wrapper is fully transparent.
type natIntercept struct {
	net.Conn
	engine *nat.Engine
	// fromTunnel: true if this conn's reads deliver tunnel-originated
	// packets (server role). False means this conn faces the egress
	// (server LAN) side and carries return traffic.
	fromTunnel bool
}

// NewNATIntercept wraps conn with NAT processing. Returns conn itself
// when engine is nil (NAT disabled: zero behavior change).
func NewNATIntercept(conn net.Conn, engine *nat.Engine, fromTunnel bool) net.Conn {
	if engine == nil {
		return conn
	}
	return &natIntercept{Conn: conn, engine: engine, fromTunnel: fromTunnel}
}

func (n *natIntercept) Read(b []byte) (int, error) {
	c, err := n.Conn.Read(b)
	if err != nil || c == 0 {
		return c, err
	}
	pkt := b[:c]
	if n.fromTunnel {
		// Packets from the tunnel: forward path (ACL + SNAT).
		out, perr := n.engine.ProcessTunnelPacket(pkt)
		if perr != nil {
			return 0, nil // drop: fail closed without killing the stream
		}
		if len(out) != len(pkt) {
			// In-place rewrite only shrinks never; sizes equal.
			return len(pkt), nil
		}
		return len(out), nil
	}
	// Egress side: return traffic of translated flows.
	if _, perr := n.engine.ProcessEgressPacket(pkt); perr != nil {
		return 0, nil // drop
	}
	return c, nil
}

func (n *natIntercept) Write(b []byte) (int, error) {
	return n.Conn.Write(b)
}

// adapterWrapper wraps an adapter.Interface to implement net.Conn
type adapterWrapper struct {
	adapter adapter.Interface
}

// NewAdapterWrapper creates a new adapter wrapper
func NewAdapterWrapper(adapter adapter.Interface) net.Conn {
	return &adapterWrapper{
		adapter: adapter,
	}
}

func (w *adapterWrapper) Read(b []byte) (n int, err error) {
	return w.adapter.Read(b)
}

func (w *adapterWrapper) Write(b []byte) (n int, err error) {
	return w.adapter.Write(b)
}

func (w *adapterWrapper) Close() error {
	return w.adapter.Close()
}

func (w *adapterWrapper) LocalAddr() net.Addr {
	// Handle CIDR format addresses
	addr := w.adapter.GetAddress()
	if strings.Contains(addr, "/") {
		addr = strings.Split(addr, "/")[0]
	}
	return &net.IPAddr{IP: net.ParseIP(addr)}
}

func (w *adapterWrapper) RemoteAddr() net.Addr {
	// Remote address is not applicable for adapter
	return nil
}

func (w *adapterWrapper) SetDeadline(t time.Time) error {
	// Deadlines not supported for adapter
	return nil
}

// deadliner is implemented by pollable adapters (TUN device files,
// in-memory pipes) so blocked reads can be aborted at teardown.
type deadliner interface {
	SetReadDeadline(t time.Time) error
}

func (w *adapterWrapper) SetReadDeadline(t time.Time) error {
	if d, ok := w.adapter.(deadliner); ok {
		return d.SetReadDeadline(t)
	}
	return nil // not pollable: reads cannot be aborted by deadline
}

func (w *adapterWrapper) SetWriteDeadline(t time.Time) error {
	// Write deadlines not supported for adapter
	return nil
}
