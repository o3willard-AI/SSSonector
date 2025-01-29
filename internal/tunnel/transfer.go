package tunnel

import (
	"context"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"SSSonector/internal/adapter"
	"SSSonector/internal/throttle"

	"go.uber.org/zap"
)

// Statistics tracks data transfer metrics
type Statistics struct {
	BytesReceived uint64
	BytesSent     uint64
	PacketsLost   uint64
	LastLatency   int64 // in microseconds
}

// Transfer handles data transfer between TUN interface and TLS connection
type Transfer struct {
	logger     *zap.Logger
	iface      adapter.Interface
	conn       net.Conn
	stats      Statistics
	done       chan struct{}
	ctx        context.Context
	cancel     context.CancelFunc
	reader     io.Reader
	writer     io.Writer
	bufferPool sync.Pool
}

// NewTransfer creates a new transfer instance
func NewTransfer(logger *zap.Logger, iface adapter.Interface, conn net.Conn, uploadKbps, downloadKbps int64) *Transfer {
	ctx, cancel := context.WithCancel(context.Background())
	t := &Transfer{
		logger: logger,
		iface:  iface,
		conn:   conn,
		done:   make(chan struct{}),
		ctx:    ctx,
		cancel: cancel,
		bufferPool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 4096)
			},
		},
	}

	// Initialize rate limiters if bandwidth limits are set
	if uploadKbps > 0 {
		t.writer = throttle.NewRateLimitedWriter(ctx, conn, float64(uploadKbps))
	} else {
		t.writer = conn
	}

	if downloadKbps > 0 {
		t.reader = throttle.NewRateLimitedReader(ctx, conn, float64(downloadKbps))
	} else {
		t.reader = conn
	}

	return t
}

// Start starts the data transfer
func (t *Transfer) Start() {
	var wg sync.WaitGroup
	wg.Add(2)

	// Interface -> Connection
	go func() {
		defer wg.Done()
		t.interfaceToConnection()
	}()

	// Connection -> Interface
	go func() {
		defer wg.Done()
		t.connectionToInterface()
	}()

	// Start latency measurement
	go t.measureLatency()

	wg.Wait()
}

// interfaceToConnection transfers data from interface to connection
func (t *Transfer) interfaceToConnection() {
	for {
		select {
		case <-t.done:
			return
		default:
			buffer := t.bufferPool.Get().([]byte)
			n, err := t.iface.Read(buffer)
			if err != nil {
				if err != io.EOF {
					t.logger.Error("Interface read error", zap.Error(err))
					atomic.AddUint64(&t.stats.PacketsLost, 1)
				}
				t.bufferPool.Put(buffer)
				continue
			}

			if _, err := t.writer.Write(buffer[:n]); err != nil {
				t.logger.Error("Connection write error", zap.Error(err))
				atomic.AddUint64(&t.stats.PacketsLost, 1)
				t.bufferPool.Put(buffer)
				continue
			}

			atomic.AddUint64(&t.stats.BytesSent, uint64(n))
			t.bufferPool.Put(buffer)
		}
	}
}

// connectionToInterface transfers data from connection to interface
func (t *Transfer) connectionToInterface() {
	for {
		select {
		case <-t.done:
			return
		default:
			buffer := t.bufferPool.Get().([]byte)
			n, err := t.reader.Read(buffer)
			if err != nil {
				if err != io.EOF {
					t.logger.Error("Connection read error", zap.Error(err))
					atomic.AddUint64(&t.stats.PacketsLost, 1)
				}
				t.bufferPool.Put(buffer)
				continue
			}

			if _, err := t.iface.Write(buffer[:n]); err != nil {
				t.logger.Error("Interface write error", zap.Error(err))
				atomic.AddUint64(&t.stats.PacketsLost, 1)
				t.bufferPool.Put(buffer)
				continue
			}

			atomic.AddUint64(&t.stats.BytesReceived, uint64(n))
			t.bufferPool.Put(buffer)
		}
	}
}

// measureLatency periodically measures connection latency
func (t *Transfer) measureLatency() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-t.done:
			return
		case <-ticker.C:
			start := time.Now()
			// Send ping
			if _, err := t.conn.Write([]byte{0}); err != nil {
				t.logger.Error("Latency measurement write error", zap.Error(err))
				continue
			}
			// Read pong
			buffer := make([]byte, 1)
			if _, err := t.conn.Read(buffer); err != nil {
				t.logger.Error("Latency measurement read error", zap.Error(err))
				continue
			}
			latency := time.Since(start).Microseconds()
			atomic.StoreInt64(&t.stats.LastLatency, latency)
		}
	}
}

// Stop stops the data transfer
func (t *Transfer) Stop() {
	t.cancel() // Cancel context for rate limiters
	close(t.done)
}

// GetStatistics returns current transfer statistics
func (t *Transfer) GetStatistics() Statistics {
	return Statistics{
		BytesReceived: atomic.LoadUint64(&t.stats.BytesReceived),
		BytesSent:     atomic.LoadUint64(&t.stats.BytesSent),
		PacketsLost:   atomic.LoadUint64(&t.stats.PacketsLost),
		LastLatency:   atomic.LoadInt64(&t.stats.LastLatency),
	}
}
