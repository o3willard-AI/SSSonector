package tunnel

import (
	"io"
	"net"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/o3willard-AI/SSSonector/internal/throttle"
	"go.uber.org/zap"
)

// Transfer handles data transfer between connections
type Transfer struct {
	src      net.Conn
	dst      net.Conn
	srcToDst *throttle.Limiter
	dstToSrc *throttle.Limiter
	logger   *zap.Logger
}

// NewTransfer creates a new transfer
func NewTransfer(src, dst net.Conn, cfg *types.AppConfig, logger *zap.Logger) *Transfer {
	// Create rate limiters for each direction
	srcToDst := throttle.NewLimiter(cfg, src, dst, logger)
	dstToSrc := throttle.NewLimiter(cfg, dst, src, logger)

	return &Transfer{
		src:      src,
		dst:      dst,
		srcToDst: srcToDst,
		dstToSrc: dstToSrc,
		logger:   logger,
	}
}

// Start starts the transfer
func (t *Transfer) Start() error {
	// Start bidirectional transfer
	errChan := make(chan error, 2)

	// Forward src -> dst
	go func() {
		// Read from src and write to dst through limiter
		_, err := io.Copy(t.dst, t.srcToDst)
		errChan <- err
	}()

	// Forward dst -> src
	go func() {
		// Read from dst and write to src through limiter
		_, err := io.Copy(t.src, t.dstToSrc)
		errChan <- err
	}()

	// Wait for first error or completion
	var err error
	for i := 0; i < 2; i++ {
		if e := <-errChan; e != nil {
			err = e
		}
	}

	// Close connections
	t.src.Close()
	t.dst.Close()

	return err
}

// Stop stops the transfer
func (t *Transfer) Stop() error {
	// Close connections
	if err := t.src.Close(); err != nil {
		t.logger.Error("Failed to close source connection", zap.Error(err))
	}
	if err := t.dst.Close(); err != nil {
		t.logger.Error("Failed to close destination connection", zap.Error(err))
	}

	return nil
}

// SetDeadline sets the read/write deadlines
func (t *Transfer) SetDeadline(deadline time.Time) {
	t.src.SetDeadline(deadline)
	t.dst.SetDeadline(deadline)
}

// SetReadDeadline sets the read deadline
func (t *Transfer) SetReadDeadline(deadline time.Time) {
	t.src.SetReadDeadline(deadline)
	t.dst.SetReadDeadline(deadline)
}

// SetWriteDeadline sets the write deadline
func (t *Transfer) SetWriteDeadline(deadline time.Time) {
	t.src.SetWriteDeadline(deadline)
	t.dst.SetWriteDeadline(deadline)
}
