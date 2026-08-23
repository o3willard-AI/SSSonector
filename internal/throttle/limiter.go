package throttle

import (
	"fmt"
	"github.com/o3willard-AI/SSSonector/internal/config"
	"io"
	"sync"

	"go.uber.org/zap"
)

// Limiter implements rate limiting for read/write operations.
//
// The limiter wraps an io.Reader and io.Writer pair and paces transfers at
// the configured rate (bytes/second, adjusted by TCPOverheadFactor). The
// burst cap is 100ms worth of effective rate; buckets start empty so
// throughput is paced from the first byte.
type Limiter struct {
	enabled   bool
	inBucket  *tokenBucket // read direction (into this process)
	outBucket *tokenBucket // write direction (out of this process)
	reader    io.Reader
	writer    io.Writer
	logger    *zap.Logger

	mu         sync.RWMutex
	inMetrics  LimiterMetrics
	outMetrics LimiterMetrics
}

// LimiterMetrics tracks rate limiting statistics.
type LimiterMetrics struct {
	Rate      float64 // effective rate (bytes/second incl. TCP overhead)
	Burst     float64 // burst cap in bytes (100ms of effective rate)
	LimitHits uint64  // number of requests that had to wait
}

// NewLimiter creates a new rate limiter.
func NewLimiter(cfg *config.AppConfig, reader io.Reader, writer io.Writer, logger *zap.Logger) (*Limiter, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	rate := float64(cfg.Throttle.Rate) * TCPOverheadFactor
	burst := rate * 0.1 // 100ms worth of data

	l := &Limiter{
		enabled:    cfg.Throttle.Enabled,
		reader:     reader,
		writer:     writer,
		logger:     logger,
		inBucket:   newTokenBucket(rate, burst),
		outBucket:  newTokenBucket(rate, burst),
		inMetrics:  LimiterMetrics{Rate: rate, Burst: burst},
		outMetrics: LimiterMetrics{Rate: rate, Burst: burst},
	}

	return l, nil
}

// Read implements io.Reader. Data is read from the underlying reader and
// then paced at the configured inbound rate before being returned.
func (l *Limiter) Read(p []byte) (n int, err error) {
	if !l.enabled {
		return l.reader.Read(p)
	}

	n, err = l.reader.Read(p)
	if err != nil {
		return n, err
	}
	if n > 0 {
		if werr := l.Wait(true, n); werr != nil {
			return n, werr
		}
	}
	return n, nil
}

// Write implements io.Writer. The write is paced at the configured outbound
// rate before the data is handed to the underlying writer.
func (l *Limiter) Write(p []byte) (n int, err error) {
	if !l.enabled {
		return l.writer.Write(p)
	}

	if err := l.Wait(false, len(p)); err != nil {
		return 0, err
	}
	return l.writer.Write(p)
}

// Wait blocks until size bytes are admitted by the direction's bucket,
// recording a limit hit if any waiting was required. When the limiter is
// disabled it returns immediately.
func (l *Limiter) Wait(isRead bool, size int) error {
	if !l.enabled {
		return nil
	}

	bucket := l.inBucket
	if !isRead {
		bucket = l.outBucket
	}

	waited, err := bucket.acquire(float64(size))
	if waited {
		l.recordLimitHit(isRead)
	}
	if err != nil {
		l.logger.Warn("Rate limit wait timeout",
			zap.Bool("read", isRead),
			zap.Int("size", size),
			zap.Error(err),
		)
		return fmt.Errorf("wait for %d tokens: %w", size, err)
	}
	return nil
}

// Update updates the limiter configuration (hot reload).
func (l *Limiter) Update(cfg *config.AppConfig) {
	rate := float64(cfg.Throttle.Rate) * TCPOverheadFactor
	burst := rate * 0.1

	l.inBucket.Update(rate, burst)
	l.outBucket.Update(rate, burst)

	l.mu.Lock()
	l.enabled = cfg.Throttle.Enabled
	l.inMetrics.Rate = rate
	l.inMetrics.Burst = burst
	l.outMetrics.Rate = rate
	l.outMetrics.Burst = burst
	l.mu.Unlock()

	l.logger.Info("Updated rate limiter configuration",
		zap.Bool("enabled", cfg.Throttle.Enabled),
		zap.Float64("rate", rate),
		zap.Float64("burst", burst),
	)
}

// IsEnabled reports whether pacing is currently active
func (l *Limiter) IsEnabled() bool {
	return l.enabled
}

// GetMetrics returns current inbound and outbound metrics.
func (l *Limiter) GetMetrics() (LimiterMetrics, LimiterMetrics) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.inMetrics, l.outMetrics
}

func (l *Limiter) recordLimitHit(isRead bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if isRead {
		l.inMetrics.LimitHits++
	} else {
		l.outMetrics.LimitHits++
	}
}
