package tunnel

import (
	"net"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/adapter"
)

// adapterWrapper wraps an adapter.AdapterInterface to implement net.Conn
type adapterWrapper struct {
	adapter adapter.AdapterInterface
}

// NewAdapterWrapper creates a new adapter wrapper
func NewAdapterWrapper(adapter adapter.AdapterInterface) net.Conn {
	return &adapterWrapper{adapter: adapter}
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
	return &net.IPAddr{IP: net.ParseIP(w.adapter.GetAddress())}
}

func (w *adapterWrapper) RemoteAddr() net.Addr {
	// For TUN devices, local and remote addresses are the same
	return w.LocalAddr()
}

func (w *adapterWrapper) SetDeadline(t time.Time) error {
	// TUN devices don't support deadlines
	return nil
}

func (w *adapterWrapper) SetReadDeadline(t time.Time) error {
	// TUN devices don't support deadlines
	return nil
}

func (w *adapterWrapper) SetWriteDeadline(t time.Time) error {
	// TUN devices don't support deadlines
	return nil
}

// Ensure adapterWrapper implements net.Conn
var _ net.Conn = &adapterWrapper{}
