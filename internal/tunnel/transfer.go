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

	// shareDst keeps Start() from closing dst when the transfer ends,
	// for destinations shared across connections (the process-wide TUN).
	shareDst bool
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

// ShareDst marks dst as shared across connections: Start will not close it
// on return (the owner shuts it down during service Stop).
func (t *Transfer) ShareDst() { t.shareDst = true }

// Start starts the transfer. It returns as soon as either direction
// finishes, closing src (and dst unless ShareDst was set) so the remaining
// direction's blocked Read unblocks instead of leaking.
//
// For shared destinations that cannot be closed, a past read deadline
// aborts the pending adapter Read; it is cleared when the next transfer
// on the same destination starts.
func (t *Transfer) Start() error {
	if t.shareDst {
		if d, ok := t.dst.(deadliner); ok {
			_ = d.SetReadDeadline(time.Time{}) // clear any prior abort
		}
	}

	errChan := make(chan error, 2)

	// Forward src -> dst
	go func() {
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
		_, err := io.Copy(t.src, t.dstToSrc)
		if cw, ok := t.src.(closeWriter); ok {
			cw.CloseWrite()
		}
		errChan <- err
	}()

	var err error
	if e := <-errChan; e != nil {
		err = e
	}

	t.src.Close()
	if !t.shareDst {
		t.dst.Close()
	} else if d, ok := t.dst.(deadliner); ok {
		// Abort the surviving adapter-side Read so its goroutine exits;
		// the next Start clears this deadline.
		_ = d.SetReadDeadline(time.Unix(1, 0))
	}

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
