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
	timeout    types.Duration
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
		timeout: types.DefaultTimeout,
	}

	// Initialize buffer pool
	l.bufferPool = NewBufferPool(logger)

	if !cfg.Throttle.Enabled {
		return l
	}

	// Get base rates from config
	rate := float64(cfg.Throttle.Rate)
	burst := float64(cfg.Throttle.Burst)

	// Apply TCP overhead factor for token buckets
	adjustedRate := rate * tcpOverheadFactor
	adjustedBurst := burst * tcpOverheadFactor

	l.inBucket = NewTokenBucket(adjustedRate, adjustedBurst)
	l.outBucket = NewTokenBucket(adjustedRate, adjustedBurst)

	// Initialize metrics with base values
	l.inMetrics = LimiterMetrics{
		Rate:  rate,
		Burst: burst,
	}
	l.outMetrics = LimiterMetrics{
		Rate:  rate,
		Burst: burst,
	}

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

	timeout := time.After(l.timeout.Duration)
	done := make(chan struct{})

	go func() {
		bucket.Wait(float64(size))
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-timeout:
		err := fmt.Errorf("timeout waiting for %d tokens after %v", size, l.timeout)
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

	wasEnabled := l.enabled
	l.enabled = cfg.Throttle.Enabled

	// Get base rates from config
	rate := float64(cfg.Throttle.Rate)
	burst := float64(cfg.Throttle.Burst)

	// Apply TCP overhead factor for token buckets
	adjustedRate := rate * tcpOverheadFactor
	adjustedBurst := burst * tcpOverheadFactor

	if !l.enabled {
		// If disabling, keep the current metrics but disable operation
		return
	}

	// Initialize or update token buckets
	if !wasEnabled {
		l.inBucket = NewTokenBucket(adjustedRate, adjustedBurst)
		l.outBucket = NewTokenBucket(adjustedRate, adjustedBurst)
	} else {
		l.inBucket.Update(adjustedRate, adjustedBurst)
		l.outBucket.Update(adjustedRate, adjustedBurst)
	}

	// Store base values in metrics
	l.inMetrics.Rate = rate
	l.inMetrics.Burst = burst
	l.outMetrics.Rate = rate
	l.outMetrics.Burst = burst

	l.logger.Info("Updated rate limiter configuration",
		zap.Bool("enabled", l.enabled),
		zap.Float64("base_rate", rate),
		zap.Float64("base_burst", burst),
		zap.Float64("adjusted_rate", adjustedRate),
		zap.Float64("adjusted_burst", adjustedBurst),
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

// SetTimeout sets the timeout duration for rate limiting operations
func (l *Limiter) SetTimeout(timeout types.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.timeout = timeout
}

// GetTimeout returns the current timeout duration
func (l *Limiter) GetTimeout() types.Duration {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.timeout
}
