package tunnel

import (
	"fmt"
	"github.com/o3willard-AI/SSSonector/internal/config"
	"io"
	"net"
	"time"

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
func NewTransfer(src, dst net.Conn, cfg *config.AppConfig, logger *zap.Logger) (*Transfer, error) {
	// Create rate limiters for each direction
	srcToDst, err := throttle.NewLimiter(cfg, src, dst, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create forward limiter: %w", err)
	}
	dstToSrc, err := throttle.NewLimiter(cfg, dst, src, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create reverse limiter: %w", err)
	}

	return &Transfer{
		src:      src,
		dst:      dst,
		srcToDst: srcToDst,
		dstToSrc: dstToSrc,
		logger:   logger,
	}, nil
}

// Start starts the transfer
func (t *Transfer) Start() error {
	// Start bidirectional transfer
	errChan := make(chan error, 2)

	// Forward src -> dst
	go func() {
		// Read from src and write to dst through limiter
		_, err := io.Copy(t.dst, t.srcToDst)
		// Propagate EOF: half-close the write side so the peer's read
		// unblocks and the reverse direction can drain and finish.
		if cw, ok := t.dst.(closeWriter); ok {
			cw.CloseWrite()
		}
		errChan <- err
	}()

	// Forward dst -> src
	go func() {
		// Read from dst and write to src through limiter
		_, err := io.Copy(t.src, t.dstToSrc)
		if cw, ok := t.src.(closeWriter); ok {
			cw.CloseWrite()
		}
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

// ThrottleStats returns the directional hit counters and current pacing
// values (effective rate and burst) from both directional limiters.
func (t *Transfer) ThrottleStats() (inHits, outHits uint64, rate, burst float64) {
	inMetrics, _ := t.srcToDst.GetMetrics()
	_, outMetrics := t.dstToSrc.GetMetrics()
	return inMetrics.LimitHits, outMetrics.LimitHits, inMetrics.Rate, inMetrics.Burst
}

// UpdateConfig pushes a reloaded configuration into both directional
// limiters so live transfers pick up new rates without restart.
func (t *Transfer) UpdateConfig(cfg *config.AppConfig) {
	t.srcToDst.Update(cfg)
	t.dstToSrc.Update(cfg)
}

// limiters exposes the directional limiters for white-box tests
func (t *Transfer) limiters() (*throttle.Limiter, *throttle.Limiter) {
	return t.srcToDst, t.dstToSrc
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
