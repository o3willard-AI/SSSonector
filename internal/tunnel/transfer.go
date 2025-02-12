package tunnel

import (
	"io"
	"sync"
	"sync/atomic"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"go.uber.org/zap"
)

// Transfer handles data transfer between two connections
type Transfer struct {
	conn1  io.ReadWriteCloser
	conn2  io.ReadWriteCloser
	config *types.AppConfig
	logger *zap.Logger
	wg     sync.WaitGroup
	done   chan struct{}
	bytes  atomic.Uint64
}

// NewTransfer creates a new transfer
func NewTransfer(conn1, conn2 io.ReadWriteCloser, cfg *types.AppConfig, logger *zap.Logger) *Transfer {
	return &Transfer{
		conn1:  conn1,
		conn2:  conn2,
		config: cfg,
		logger: logger,
		done:   make(chan struct{}),
	}
}

// Start starts the transfer
func (t *Transfer) Start() error {
	if t.logger != nil {
		t.logger.Debug("Starting transfer")
	}

	// Start bidirectional copy
	t.wg.Add(2)
	go t.copy(t.conn1, t.conn2)
	go t.copy(t.conn2, t.conn1)

	// Wait for both copies to complete
	t.wg.Wait()
	close(t.done)

	if t.logger != nil {
		t.logger.Debug("Transfer complete",
			zap.Uint64("bytes_transferred", t.bytes.Load()),
		)
	}

	return nil
}

// copy copies data from src to dst
func (t *Transfer) copy(dst io.Writer, src io.Reader) {
	defer t.wg.Done()

	// Create buffer for copying
	buf := make([]byte, t.config.Config.Network.MTU)

	// Copy data
	for {
		select {
		case <-t.done:
			return
		default:
			n, err := src.Read(buf)
			if err != nil {
				if err != io.EOF && t.logger != nil {
					t.logger.Error("Read failed", zap.Error(err))
				}
				return
			}

			if n > 0 {
				_, err := dst.Write(buf[:n])
				if err != nil {
					if t.logger != nil {
						t.logger.Error("Write failed", zap.Error(err))
					}
					return
				}
				t.bytes.Add(uint64(n))
			}
		}
	}
}

// Stop stops the transfer
func (t *Transfer) Stop() error {
	close(t.done)
	t.wg.Wait()
	return nil
}
