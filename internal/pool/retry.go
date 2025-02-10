package pool

import (
	"context"
	"net"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// RetryConfig holds retry configuration for different phases
type RetryConfig struct {
	// Immediate retry phase
	ImmediateAttempts int
	ImmediateInterval time.Duration

	// Gradual retry phase
	GradualAttempts    int
	GradualInterval    time.Duration
	MaxGradualInterval time.Duration

	// Persistent retry phase
	PersistentEnabled  bool
	PersistentInterval time.Duration
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		ImmediateAttempts:  3,
		ImmediateInterval:  5 * time.Second,
		GradualAttempts:    5,
		GradualInterval:    30 * time.Second,
		MaxGradualInterval: 480 * time.Second,
		PersistentEnabled:  true,
		PersistentInterval: 10 * time.Minute,
	}
}

// RetryManager handles connection retry logic
type RetryManager struct {
	config *RetryConfig
	logger *zap.Logger

	// Factory function to create new connections
	factory Factory

	// Metrics for monitoring
	metrics struct {
		attempts  int64
		failures  int64
		successes int64
	}
}

// NewRetryManager creates a new retry manager
func NewRetryManager(factory Factory, config *RetryConfig, logger *zap.Logger) *RetryManager {
	if config == nil {
		config = DefaultRetryConfig()
	}

	return &RetryManager{
		config:  config,
		logger:  logger,
		factory: factory,
	}
}

// GetConnection attempts to get a connection using the three-phase retry strategy
func (r *RetryManager) GetConnection(ctx context.Context) (net.Conn, error) {
	atomic.AddInt64(&r.metrics.attempts, 1)

	// Phase 1: Immediate retries
	conn, err := r.immediateRetries(ctx)
	if err == nil {
		atomic.AddInt64(&r.metrics.successes, 1)
		return conn, nil
	}

	// Phase 2: Gradual retries with exponential backoff
	conn, err = r.gradualRetries(ctx)
	if err == nil {
		atomic.AddInt64(&r.metrics.successes, 1)
		return conn, nil
	}

	// Phase 3: Persistent retries
	if r.config.PersistentEnabled {
		conn, err = r.persistentRetries(ctx)
		if err == nil {
			atomic.AddInt64(&r.metrics.successes, 1)
			return conn, nil
		}
	}

	atomic.AddInt64(&r.metrics.failures, 1)
	return nil, ErrMaxRetriesExceeded
}

// immediateRetries handles the immediate retry phase
func (r *RetryManager) immediateRetries(ctx context.Context) (net.Conn, error) {
	for i := 0; i < r.config.ImmediateAttempts; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			conn, err := r.factory(ctx)
			if err == nil {
				r.logger.Info("Connection established during immediate retry phase",
					zap.Int("attempt", i+1),
				)
				return conn, nil
			}

			r.logger.Warn("Connection attempt failed during immediate retry phase",
				zap.Int("attempt", i+1),
				zap.Error(err),
			)

			time.Sleep(r.config.ImmediateInterval)
		}
	}

	return nil, ErrMaxRetriesExceeded
}

// gradualRetries handles the gradual retry phase with exponential backoff
func (r *RetryManager) gradualRetries(ctx context.Context) (net.Conn, error) {
	interval := r.config.GradualInterval

	for i := 0; i < r.config.GradualAttempts; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			conn, err := r.factory(ctx)
			if err == nil {
				r.logger.Info("Connection established during gradual retry phase",
					zap.Int("attempt", i+1),
				)
				return conn, nil
			}

			r.logger.Warn("Connection attempt failed during gradual retry phase",
				zap.Int("attempt", i+1),
				zap.Duration("interval", interval),
				zap.Error(err),
			)

			time.Sleep(interval)
			interval = time.Duration(float64(interval) * 2)
			if interval > r.config.MaxGradualInterval {
				interval = r.config.MaxGradualInterval
			}
		}
	}

	return nil, ErrMaxRetriesExceeded
}

// persistentRetries handles the persistent retry phase
func (r *RetryManager) persistentRetries(ctx context.Context) (net.Conn, error) {
	attempt := 0
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			attempt++
			conn, err := r.factory(ctx)
			if err == nil {
				r.logger.Info("Connection established during persistent retry phase",
					zap.Int("attempt", attempt),
				)
				return conn, nil
			}

			r.logger.Warn("Connection attempt failed during persistent retry phase",
				zap.Int("attempt", attempt),
				zap.Error(err),
			)

			time.Sleep(r.config.PersistentInterval)
		}
	}
}

// GetMetrics returns retry metrics
func (r *RetryManager) GetMetrics() (attempts, failures, successes int64) {
	return atomic.LoadInt64(&r.metrics.attempts),
		atomic.LoadInt64(&r.metrics.failures),
		atomic.LoadInt64(&r.metrics.successes)
}

// ResetMetrics resets retry metrics
func (r *RetryManager) ResetMetrics() {
	atomic.StoreInt64(&r.metrics.attempts, 0)
	atomic.StoreInt64(&r.metrics.failures, 0)
	atomic.StoreInt64(&r.metrics.successes, 0)
}
