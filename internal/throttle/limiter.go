package throttle

import (
	"io"
	"sync"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"go.uber.org/zap"
)

// Limiter implements rate limiting
type Limiter struct {
	enabled bool
	rate    float64
	burst   int
	tokens  float64
	last    time.Time
	mu      sync.Mutex
	logger  *zap.Logger
	reader  io.Reader
	writer  io.Writer
}

// NewLimiter creates a new rate limiter
func NewLimiter(cfg *config.AppConfig, reader io.Reader, writer io.Writer, logger *zap.Logger) *Limiter {
	return &Limiter{
		enabled: cfg.Throttle.Enabled,
		rate:    cfg.Throttle.Rate,
		burst:   cfg.Throttle.Burst,
		tokens:  float64(cfg.Throttle.Burst),
		last:    time.Now(),
		logger:  logger,
		reader:  reader,
		writer:  writer,
	}
}

// Allow checks if an operation should be allowed
func (l *Limiter) Allow() bool {
	if !l.enabled {
		return true
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(l.last).Seconds()
	l.tokens = min(float64(l.burst), l.tokens+elapsed*l.rate)
	l.last = now

	if l.tokens < 1 {
		l.logger.Debug("Rate limit exceeded",
			zap.Float64("tokens", l.tokens),
			zap.Float64("rate", l.rate),
			zap.Int("burst", l.burst),
		)
		return false
	}

	l.tokens--
	return true
}

// Wait waits until an operation is allowed
func (l *Limiter) Wait() {
	if !l.enabled {
		return
	}

	for !l.Allow() {
		time.Sleep(time.Second / time.Duration(l.rate))
	}
}

// Update updates the limiter configuration
func (l *Limiter) Update(cfg *config.AppConfig) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.enabled = cfg.Throttle.Enabled
	l.rate = cfg.Throttle.Rate
	l.burst = cfg.Throttle.Burst
	l.tokens = min(l.tokens, float64(l.burst))

	l.logger.Info("Updated rate limiter configuration",
		zap.Bool("enabled", l.enabled),
		zap.Float64("rate", l.rate),
		zap.Int("burst", l.burst),
	)
}

// Read implements io.Reader with rate limiting
func (l *Limiter) Read(p []byte) (n int, err error) {
	l.Wait()
	return l.reader.Read(p)
}

// Write implements io.Writer with rate limiting
func (l *Limiter) Write(p []byte) (n int, err error) {
	l.Wait()
	return l.writer.Write(p)
}
