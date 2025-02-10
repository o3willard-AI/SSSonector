package tunnel

import (
	"net"
	"strings"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/adapter"
)

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

func (w *adapterWrapper) SetReadDeadline(t time.Time) error {
	// Read deadlines not supported for adapter
	return nil
}

func (w *adapterWrapper) SetWriteDeadline(t time.Time) error {
	// Write deadlines not supported for adapter
	return nil
}
