package throttle

import (
	"io"
	"sync"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"go.uber.org/zap"
)

// Limiter implements rate limiting with TCP overhead compensation
type Limiter struct {
	enabled     bool
	inBucket    *TokenBucket
	outBucket   *TokenBucket
	mu          sync.Mutex
	logger      *zap.Logger
	reader      io.Reader
	writer      io.Writer
	metrics     *LimiterMetrics
	lastMetrics time.Time
}

// LimiterMetrics holds rate limiting metrics for SNMP monitoring
type LimiterMetrics struct {
	InRate       float64
	OutRate      float64
	InBurst      float64
	OutBurst     float64
	InLimitHits  uint64
	OutLimitHits uint64
	mu           sync.Mutex
}

// NewLimiter creates a new rate limiter with TCP overhead compensation
func NewLimiter(cfg *config.AppConfig, reader io.Reader, writer io.Writer, logger *zap.Logger) *Limiter {
	l := &Limiter{
		enabled: cfg.Throttle.Enabled,
		logger:  logger,
		reader:  reader,
		writer:  writer,
		metrics: &LimiterMetrics{},
	}

	if l.enabled {
		l.inBucket = NewTokenBucket(float64(cfg.Throttle.Rate), float64(cfg.Throttle.Burst))
		l.outBucket = NewTokenBucket(float64(cfg.Throttle.Rate), float64(cfg.Throttle.Burst))
		l.lastMetrics = time.Now()
	}

	return l
}

// Allow checks if an operation should be allowed
func (l *Limiter) Allow(isRead bool) bool {
	if !l.enabled {
		return true
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	bucket := l.inBucket
	if !isRead {
		bucket = l.outBucket
	}

	allowed := bucket.Allow()
	if !allowed {
		l.metrics.mu.Lock()
		if isRead {
			l.metrics.InLimitHits++
		} else {
			l.metrics.OutLimitHits++
		}
		l.metrics.mu.Unlock()

		baseRate, effectiveRate := bucket.GetRates()
		l.logger.Debug("Rate limit exceeded",
			zap.Bool("read", isRead),
			zap.Float64("base_rate", baseRate),
			zap.Float64("effective_rate", effectiveRate),
			zap.Float64("burst", bucket.GetBurst()),
		)
	}

	return allowed
}

// Wait waits until an operation is allowed with timeout
func (l *Limiter) Wait(isRead bool) error {
	if !l.enabled {
		return nil
	}

	bucket := l.inBucket
	if !isRead {
		bucket = l.outBucket
	}

	if err := bucket.Wait(); err != nil {
		l.logger.Warn("Rate limit wait timeout",
			zap.Bool("read", isRead),
			zap.Error(err),
		)
		return err
	}

	return nil
}

// Update updates the limiter configuration
func (l *Limiter) Update(cfg *config.AppConfig) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.enabled = cfg.Throttle.Enabled
	if l.enabled {
		if l.inBucket == nil {
			l.inBucket = NewTokenBucket(float64(cfg.Throttle.Rate), float64(cfg.Throttle.Burst))
			l.outBucket = NewTokenBucket(float64(cfg.Throttle.Rate), float64(cfg.Throttle.Burst))
		} else {
			l.inBucket.Update(float64(cfg.Throttle.Rate), float64(cfg.Throttle.Burst))
			l.outBucket.Update(float64(cfg.Throttle.Rate), float64(cfg.Throttle.Burst))
		}
	}

	l.logger.Info("Updated rate limiter configuration",
		zap.Bool("enabled", l.enabled),
		zap.Float64("rate", float64(cfg.Throttle.Rate)),
		zap.Int("burst", cfg.Throttle.Burst),
	)
}

// Read implements io.Reader with rate limiting
func (l *Limiter) Read(p []byte) (n int, err error) {
	if err := l.Wait(true); err != nil {
		return 0, err
	}
	return l.reader.Read(p)
}

// Write implements io.Writer with rate limiting
func (l *Limiter) Write(p []byte) (n int, err error) {
	if err := l.Wait(false); err != nil {
		return 0, err
	}
	return l.writer.Write(p)
}

// GetMetrics returns the current rate limiting metrics
func (l *Limiter) GetMetrics() *LimiterMetrics {
	if !l.enabled {
		return &LimiterMetrics{}
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	// Update rates and burst sizes
	_, inEffectiveRate := l.inBucket.GetRates()
	_, outEffectiveRate := l.outBucket.GetRates()

	l.metrics.mu.Lock()
	defer l.metrics.mu.Unlock()

	l.metrics.InRate = inEffectiveRate
	l.metrics.OutRate = outEffectiveRate
	l.metrics.InBurst = l.inBucket.GetBurst()
	l.metrics.OutBurst = l.outBucket.GetBurst()

	// Log metrics every minute
	if time.Since(l.lastMetrics) > time.Minute {
		l.logger.Info("Rate limiting metrics",
			zap.Float64("in_rate", inEffectiveRate),
			zap.Float64("out_rate", outEffectiveRate),
			zap.Float64("in_burst", l.metrics.InBurst),
			zap.Float64("out_burst", l.metrics.OutBurst),
			zap.Uint64("in_limit_hits", l.metrics.InLimitHits),
			zap.Uint64("out_limit_hits", l.metrics.OutLimitHits),
		)
		l.lastMetrics = time.Now()
	}

	return l.metrics
}
