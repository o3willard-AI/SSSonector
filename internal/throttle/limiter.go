package throttle

import (
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"go.uber.org/zap"
)

// Limiter implements rate limiting for read/write operations
type Limiter struct {
	enabled    bool
	inBucket   *TokenBucket
	outBucket  *TokenBucket
	reader     io.Reader
	writer     io.Writer
	logger     *zap.Logger
	mu         sync.RWMutex
	inMetrics  LimiterMetrics
	outMetrics LimiterMetrics
	bufferPool *BufferPool
}

// LimiterMetrics tracks rate limiting statistics
type LimiterMetrics struct {
	Rate      float64
	Burst     float64
	LimitHits uint64
}

// NewLimiter creates a new rate limiter
func NewLimiter(cfg *types.AppConfig, reader io.Reader, writer io.Writer, logger *zap.Logger) *Limiter {
	l := &Limiter{
		enabled: cfg.Throttle.Enabled,
		reader:  reader,
		writer:  writer,
		logger:  logger,
	}

	// Initialize token buckets with TCP overhead adjustment
	rate := float64(cfg.Throttle.Rate) * tcpOverheadFactor
	burst := float64(cfg.Throttle.Burst) * tcpOverheadFactor

	l.inBucket = NewTokenBucket(rate, burst)
	l.outBucket = NewTokenBucket(rate, burst)

	// Initialize metrics
	l.inMetrics = LimiterMetrics{
		Rate:  rate,
		Burst: burst,
	}
	l.outMetrics = LimiterMetrics{
		Rate:  rate,
		Burst: burst,
	}

	// Initialize buffer pool
	l.bufferPool = NewBufferPool(logger)

	return l
}

// Read implements io.Reader
func (l *Limiter) Read(p []byte) (n int, err error) {
	if !l.enabled {
		return l.reader.Read(p)
	}

	// Get buffer from pool
	buf := l.GetBuffer(len(p))
	defer l.PutBuffer(buf)

	// Read into buffer
	n, err = l.reader.Read(buf)
	if err != nil {
		return n, err
	}

	// Wait for tokens
	if err := l.Wait(true, n); err != nil {
		return n, err
	}

	// Copy data to output buffer
	copy(p, buf[:n])
	return n, nil
}

// Write implements io.Writer
func (l *Limiter) Write(p []byte) (n int, err error) {
	if !l.enabled {
		return l.writer.Write(p)
	}

	// Wait for tokens
	if err := l.Wait(false, len(p)); err != nil {
		return 0, err
	}

	// Write data
	return l.writer.Write(p)
}

// Wait waits for the specified number of tokens
func (l *Limiter) Wait(isRead bool, size int) error {
	if !l.enabled {
		return nil
	}

	bucket := l.inBucket
	if !isRead {
		bucket = l.outBucket
	}

	timeout := time.After(defaultTimeout)
	done := make(chan struct{})

	go func() {
		bucket.Wait(float64(size))
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-timeout:
		err := fmt.Errorf("timeout waiting for %d tokens after %v", size, defaultTimeout)
		l.logger.Warn("Rate limit wait timeout",
			zap.Bool("read", isRead),
			zap.Int("size", size),
			zap.Error(err),
		)
		return err
	}
}

// Update updates the limiter configuration
func (l *Limiter) Update(cfg *types.AppConfig) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.enabled = cfg.Throttle.Enabled
	rate := float64(cfg.Throttle.Rate) * tcpOverheadFactor
	burst := float64(cfg.Throttle.Burst) * tcpOverheadFactor

	l.inBucket.Update(rate, burst)
	l.outBucket.Update(rate, burst)

	l.inMetrics.Rate = rate
	l.inMetrics.Burst = burst
	l.outMetrics.Rate = rate
	l.outMetrics.Burst = burst

	l.logger.Info("Updated rate limiter configuration",
		zap.Bool("enabled", l.enabled),
		zap.Float64("rate", rate),
		zap.Float64("burst", burst),
	)
}

// GetMetrics returns current metrics
func (l *Limiter) GetMetrics() (LimiterMetrics, LimiterMetrics) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.inMetrics, l.outMetrics
}

// GetBuffer gets a buffer from the pool
func (l *Limiter) GetBuffer(size int) []byte {
	return l.bufferPool.Get(size)
}

// PutBuffer returns a buffer to the pool
func (l *Limiter) PutBuffer(buf []byte) {
	l.bufferPool.Put(buf)
}
