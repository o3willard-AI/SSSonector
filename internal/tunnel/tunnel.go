package tunnel

import (
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/o3willard-AI/SSSonector/internal/adapter"
	"github.com/o3willard-AI/SSSonector/internal/throttle"
)

// Tunnel represents an SSL tunnel connection
type Tunnel struct {
	conn      net.Conn
	adapter   adapter.Interface
	throttler *throttle.Limiter
	done      chan struct{}
	wg        sync.WaitGroup
}

// New creates a new tunnel instance
func New(conn net.Conn, adapter adapter.Interface, throttler *throttle.Limiter) (*Tunnel, error) {
	return &Tunnel{
		conn:      conn,
		adapter:   adapter,
		throttler: throttler,
		done:      make(chan struct{}),
	}, nil
}

// Start begins tunnel operation
func (t *Tunnel) Start() error {
	// Start network to tunnel transfer
	t.wg.Add(1)
	go t.networkToTunnel()

	// Start tunnel to network transfer
	t.wg.Add(1)
	go t.tunnelToNetwork()

	return nil
}

// Stop shuts down the tunnel
func (t *Tunnel) Stop() {
	close(t.done)
	t.wg.Wait()
}

// Done returns a channel that's closed when the tunnel is stopped
func (t *Tunnel) Done() <-chan struct{} {
	return t.done
}

func (t *Tunnel) networkToTunnel() {
	defer t.wg.Done()
	defer t.Stop()

	buffer := make([]byte, 1500) // Standard MTU size
	for {
		select {
		case <-t.done:
			return
		default:
			n, err := t.throttler.Read(buffer)
			if err != nil {
				if err != io.EOF {
					fmt.Printf("Error reading from network: %v\n", err)
				}
				return
			}

			_, err = t.adapter.Write(buffer[:n])
			if err != nil {
				fmt.Printf("Error writing to tunnel: %v\n", err)
				return
			}
		}
	}
}

func (t *Tunnel) tunnelToNetwork() {
	defer t.wg.Done()
	defer t.Stop()

	buffer := make([]byte, 1500) // Standard MTU size
	for {
		select {
		case <-t.done:
			return
		default:
			n, err := t.adapter.Read(buffer)
			if err != nil {
				if err != io.EOF {
					fmt.Printf("Error reading from tunnel: %v\n", err)
				}
				return
			}

			_, err = t.throttler.Write(buffer[:n])
			if err != nil {
				fmt.Printf("Error writing to network: %v\n", err)
				return
			}
		}
	}
}
