package tunnel

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/adapter"
	"github.com/o3willard-AI/SSSonector/internal/monitor"
	"github.com/o3willard-AI/SSSonector/internal/throttle"
)

const (
	maxRetries    = 3    // Maximum number of retries for I/O operations
	retryInterval = 50   // Base retry interval in milliseconds
	maxBackoff    = 1000 // Maximum backoff in milliseconds
)

// Tunnel represents an SSL tunnel connection
type Tunnel struct {
	conn      net.Conn
	adapter   adapter.Interface
	throttler *throttle.Limiter
	done      chan struct{}
	wg        sync.WaitGroup
	monitor   *monitor.Monitor
}

// New creates a new tunnel instance
func New(conn net.Conn, adapter adapter.Interface, throttler *throttle.Limiter, mon *monitor.Monitor) (*Tunnel, error) {
	return &Tunnel{
		conn:      conn,
		adapter:   adapter,
		throttler: throttler,
		done:      make(chan struct{}),
		monitor:   mon,
	}, nil
}

// Start begins tunnel operation
func (t *Tunnel) Start() error {
	// Wait for adapter to be ready with exponential backoff
	backoff := retryInterval
	for i := 0; i < maxRetries; i++ {
		if t.adapter.IsUp() {
			break
		}
		if i == maxRetries-1 {
			return fmt.Errorf("timeout waiting for adapter to be ready")
		}
		time.Sleep(time.Duration(backoff) * time.Millisecond)
		backoff = min(backoff*2, maxBackoff)
	}

	// Start network to tunnel transfer with worker pool
	t.wg.Add(2) // Add for both directions at once
	go t.networkToTunnel()
	go t.tunnelToNetwork()

	return nil
}

// Stop shuts down the tunnel
func (t *Tunnel) Stop() {
	select {
	case <-t.done:
		// Already closed
		return
	default:
		close(t.done)
		t.wg.Wait()
	}
}

// Done returns a channel that's closed when the tunnel is stopped
func (t *Tunnel) Done() <-chan struct{} {
	return t.done
}

func (t *Tunnel) networkToTunnel() {
	defer t.wg.Done()

	mtu := t.adapter.GetMTU()
	buf := make([]byte, mtu)

	for {
		select {
		case <-t.done:
			return
		default:
			var n int
			var err error
			if t.throttler != nil {
				n, err = t.throttler.Read(buf)
			} else {
				n, err = t.conn.Read(buf)
			}
			if err != nil {
				if err != io.EOF {
					if t.monitor != nil {
						t.monitor.GetMetrics().UpdateErrorMetrics(1, err.Error(), 0, 0)
					}
					return
				}
				// For EOF, continue reading after a short delay
				time.Sleep(10 * time.Millisecond)
				continue
			}

			if n > 0 {
				// Write data in chunks to avoid buffer overflow
				written := 0
				for written < n {
					chunkSize := min(mtu, n-written)
					w, err := t.adapter.Write(buf[written : written+chunkSize])
					if err != nil {
						if t.monitor != nil {
							t.monitor.GetMetrics().UpdateErrorMetrics(1, err.Error(), 0, 0)
						}
						return
					}
					written += w

					// Update metrics after successful write
					if t.monitor != nil {
						t.monitor.GetMetrics().UpdateNetworkMetrics(
							0,
							int64(w),
							0,
							1,
						)
					}
				}
			}
		}
	}
}

func (t *Tunnel) tunnelToNetwork() {
	defer t.wg.Done()

	mtu := t.adapter.GetMTU()
	buf := make([]byte, mtu)

	for {
		select {
		case <-t.done:
			return
		default:
			var n int
			var readErr error
			backoff := retryInterval

			for i := 0; i < maxRetries; i++ {
				n, readErr = t.adapter.Read(buf)
				if readErr == nil && n > 0 {
					break
				}
				if readErr == io.EOF {
					time.Sleep(10 * time.Millisecond)
					continue
				}
				if i == maxRetries-1 {
					if readErr != io.EOF {
						if t.monitor != nil {
							t.monitor.GetMetrics().UpdateErrorMetrics(1, readErr.Error(), int64(i+1), 0)
						}
					}
					return
				}
				time.Sleep(time.Duration(backoff) * time.Millisecond)
				backoff = min(backoff*2, maxBackoff)
			}

			if n > 0 {
				// Write data in chunks to avoid buffer overflow
				written := 0
				for written < n {
					chunkSize := min(mtu, n-written)
					var w int
					var err error
					if t.throttler != nil {
						w, err = t.throttler.Write(buf[written : written+chunkSize])
					} else {
						w, err = t.conn.Write(buf[written : written+chunkSize])
					}
					if err != nil {
						if t.monitor != nil {
							t.monitor.GetMetrics().UpdateErrorMetrics(1, err.Error(), 0, 0)
						}
						return
					}
					written += w

					// Update metrics after successful write
					if t.monitor != nil {
						t.monitor.GetMetrics().UpdateNetworkMetrics(
							int64(w),
							0,
							1,
							0,
						)
					}
				}
			}
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
