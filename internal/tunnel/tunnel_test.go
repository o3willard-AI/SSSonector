package tunnel

import (
	"bytes"
	"log"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/adapter"
	"github.com/o3willard-AI/SSSonector/internal/config/types"
)

type mockConn struct {
	readBuf  []byte
	writeBuf []byte
	readPos  int
	mu       sync.Mutex
	readCh   chan struct{}
	writeCh  chan struct{}
}

func newMockConn() *mockConn {
	return &mockConn{
		readBuf:  make([]byte, 0),
		writeBuf: make([]byte, 0),
		readPos:  0,
		readCh:   make(chan struct{}, 10),
		writeCh:  make(chan struct{}, 10),
	}
}

func (m *mockConn) Read(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If there's no data to read, wait instead of returning EOF
	if m.readPos >= len(m.readBuf) {
		// Return 0 bytes but no error to indicate no data available yet
		// This prevents the transfer goroutine from exiting
		return 0, nil
	}

	n = copy(p, m.readBuf[m.readPos:])
	m.readPos += n
	log.Printf("mockConn Read: %d bytes, readPos: %d, total: %d", n, m.readPos, len(m.readBuf))
	select {
	case m.readCh <- struct{}{}:
	default:
	}
	return n, nil
}

func (m *mockConn) Write(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writeBuf = append(m.writeBuf, p...)
	n = len(p)
	log.Printf("mockConn Write: %d bytes, total: %d", n, len(m.writeBuf))
	select {
	case m.writeCh <- struct{}{}:
	default:
	}
	return n, nil
}

func (m *mockConn) Close() error                       { return nil }
func (m *mockConn) LocalAddr() net.Addr                { return nil }
func (m *mockConn) RemoteAddr() net.Addr               { return nil }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

type mockAdapter struct {
	readBuf  []byte
	writeBuf []byte
	readPos  int
	isUp     bool
	mtu      int
	mu       sync.Mutex
	readCh   chan struct{}
	writeCh  chan struct{}
}

func newMockAdapter() *mockAdapter {
	return &mockAdapter{
		readBuf:  make([]byte, 0),
		writeBuf: make([]byte, 0),
		readPos:  0,
		isUp:     true,
		mtu:      1500,
		readCh:   make(chan struct{}, 10),
		writeCh:  make(chan struct{}, 10),
	}
}

func (m *mockAdapter) Configure(opts *adapter.Options) error { return nil }

func (m *mockAdapter) Read(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// If there's no data to read, wait instead of returning EOF
	if m.readPos >= len(m.readBuf) {
		// Return 0 bytes but no error to indicate no data available yet
		// This prevents the transfer goroutine from exiting
		return 0, nil
	}

	n = copy(p, m.readBuf[m.readPos:])
	m.readPos += n
	log.Printf("mockAdapter Read: %d bytes, readPos: %d, total: %d", n, m.readPos, len(m.readBuf))
	select {
	case m.readCh <- struct{}{}:
	default:
	}
	return n, nil
}

func (m *mockAdapter) Write(p []byte) (n int, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.writeBuf = append(m.writeBuf, p...)
	n = len(p)
	log.Printf("mockAdapter Write: %d bytes, total: %d", n, len(m.writeBuf))
	select {
	case m.writeCh <- struct{}{}:
	default:
	}
	return n, nil
}

func (m *mockAdapter) Close() error            { return nil }
func (m *mockAdapter) GetAddress() string      { return "10.0.0.1" }
func (m *mockAdapter) GetState() adapter.State { return adapter.StateReady }
func (m *mockAdapter) GetStatus() adapter.Status {
	return adapter.Status{
		State:      adapter.StateReady,
		Statistics: adapter.Statistics{},
		LastError:  nil,
	}
}
func (m *mockAdapter) GetStatistics() adapter.Statistics {
	return adapter.Statistics{}
}
func (m *mockAdapter) Cleanup() error { return nil }

func TestTunnelTransfer(t *testing.T) {
	tests := []struct {
		name    string
		data    []byte
		packets int
	}{
		{
			name:    "small_packet",
			data:    bytes.Repeat([]byte("a"), 100),
			packets: 1,
		},
		{
			name:    "large_packet",
			data:    bytes.Repeat([]byte("b"), 800),
			packets: 1,
		},
		{
			name:    "multiple_packets",
			data:    bytes.Repeat([]byte("c"), 1600),
			packets: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log.Printf("Starting test %s with %d bytes", tt.name, len(tt.data))

			// Create mock connection and adapter
			conn := newMockConn()
			adapter := newMockAdapter()

			// Create a simple config for the transfer
			cfg := &types.AppConfig{
				Config: &types.ServiceConfig{
					Network: &types.NetworkConfig{
						MTU: 1500,
					},
				},
			}
			tun := NewTransfer(conn, adapter, cfg, nil)

			// Start transfer
			go func() {
				if err := tun.Start(); err != nil {
					t.Errorf("Failed to start transfer: %v", err)
				}
			}()

			// Wait for tunnel to be ready
			time.Sleep(100 * time.Millisecond)

			// Test adapter to connection transfer
			log.Printf("Starting adapter to connection transfer")
			adapter.readBuf = append(adapter.readBuf, tt.data...)
			log.Printf("Wrote %d bytes to adapter readBuf", len(tt.data))

			// Wait for data to be transferred
			deadline := time.Now().Add(1 * time.Second)
			for len(conn.writeBuf) < len(tt.data) && time.Now().Before(deadline) {
				select {
				case <-adapter.readCh:
					log.Printf("Received read notification from adapter, writeBuf size: %d", len(conn.writeBuf))
				case <-conn.writeCh:
					log.Printf("Received write notification from conn, writeBuf size: %d", len(conn.writeBuf))
				case <-time.After(10 * time.Millisecond):
				}
			}

			if len(conn.writeBuf) != len(tt.data) {
				t.Errorf("Adapter to connection transfer failed: expected to read %d bytes, got %d", len(tt.data), len(conn.writeBuf))
			} else if !bytes.Equal(conn.writeBuf, tt.data) {
				t.Errorf("Data mismatch in adapter to connection transfer")
			} else {
				log.Printf("Adapter to connection transfer complete")
			}

			// Reset buffers
			conn.writeBuf = nil
			adapter.writeBuf = nil
			conn.readPos = 0
			adapter.readPos = 0

			// Test connection to adapter transfer
			log.Printf("Starting connection to adapter transfer")
			conn.readBuf = append(conn.readBuf, tt.data...)
			log.Printf("Wrote %d bytes to connection readBuf", len(tt.data))

			// Wait for data to be transferred
			deadline = time.Now().Add(1 * time.Second)
			var lastRead, lastWrite int
			for len(adapter.writeBuf) < len(tt.data) && time.Now().Before(deadline) {
				select {
				case <-conn.readCh:
					lastRead = len(conn.readBuf)
					log.Printf("Received read notification from conn, readBuf size: %d", lastRead)
				case <-adapter.writeCh:
					lastWrite = len(adapter.writeBuf)
					log.Printf("Received write notification from adapter, writeBuf size: %d", lastWrite)
				case <-time.After(10 * time.Millisecond):
					// Check if we've made progress
					if lastRead > 0 && lastWrite > 0 && lastRead == lastWrite {
						// We've transferred all data
						break
					}
				}
			}

			if len(adapter.writeBuf) != len(tt.data) {
				t.Errorf("Connection to adapter transfer failed: expected to write %d bytes, got %d", len(tt.data), len(adapter.writeBuf))
			} else if !bytes.Equal(adapter.writeBuf, tt.data) {
				t.Errorf("Data mismatch in connection to adapter transfer")
			} else {
				log.Printf("Connection to adapter transfer complete")
			}

			// Stop tunnel (use a mutex to prevent race conditions)
			var once sync.Once
			once.Do(func() {
				tun.Stop()
			})
			log.Printf("Test %s complete", tt.name)
		})
	}
}
