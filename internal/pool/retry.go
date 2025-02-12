package pool

import (
	"context"
	"net"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// RetryConfig represents retry configuration
type RetryConfig struct {
	// Immediate retry settings
	MaxImmediateRetries int
	ImmediateDelay      time.Duration

	// Gradual retry settings
	MaxGradualRetries int
	InitialInterval   time.Duration
	MaxInterval       time.Duration
	BackoffMultiplier float64

	// Persistent retry settings
	MaxPersistentRetries int
	PersistentDelay      time.Duration
}

// DefaultRetryConfig returns default retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		// Immediate retry settings
		MaxImmediateRetries: 3,
		ImmediateDelay:      time.Millisecond,

		// Gradual retry settings
		MaxGradualRetries: 5,
		InitialInterval:   time.Millisecond,
		MaxInterval:       10 * time.Millisecond,
		BackoffMultiplier: 2.0,

		// Persistent retry settings
		MaxPersistentRetries: 2,
		PersistentDelay:      5 * time.Second,
	}
}

// RetryManager handles connection retries with different strategies
type RetryManager struct {
	logger *zap.Logger
	config RetryConfig

	// Factory for creating new connections
	factory Factory

	// Metrics
	successCount uint64
	failureCount uint64
}

// NewRetryManager creates a new retry manager
func NewRetryManager(logger *zap.Logger, factory Factory) *RetryManager {
	return &RetryManager{
		logger:  logger,
		config:  DefaultRetryConfig(),
		factory: factory,
	}
}

// GetConnection attempts to get a connection using various retry strategies
func (r *RetryManager) GetConnection(ctx context.Context) (net.Conn, error) {
	// Try immediate retries first
	conn, err := r.tryImmediateRetries(ctx)
	if err == nil {
		return conn, nil
	}

	if err == context.DeadlineExceeded || err == context.Canceled {
		return nil, err
	}

	// If immediate retries fail, try gradual backoff
	conn, err = r.tryGradualRetries(ctx)
	if err == nil {
		return conn, nil
	}

	if err == context.DeadlineExceeded || err == context.Canceled {
		return nil, err
	}

	// If gradual retries fail, try persistent retries
	return r.tryPersistentRetries(ctx)
}

func (r *RetryManager) tryImmediateRetries(ctx context.Context) (net.Conn, error) {
	for attempt := 1; attempt <= r.config.MaxImmediateRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			conn, err := r.factory(ctx)
			if err == nil {
				atomic.AddUint64(&r.successCount, 1)
				r.logger.Info("Connection established during immediate retry phase",
					zap.Int("attempt", attempt))
				return conn, nil
			}

			atomic.AddUint64(&r.failureCount, 1)
			r.logger.Warn("Connection attempt failed during immediate retry phase",
				zap.Int("attempt", attempt),
				zap.Error(err))

			time.Sleep(r.config.ImmediateDelay)
		}
	}
	return nil, ErrMaxRetriesExceeded
}

func (r *RetryManager) tryGradualRetries(ctx context.Context) (net.Conn, error) {
	interval := r.config.InitialInterval
	for attempt := 1; attempt <= r.config.MaxGradualRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			conn, err := r.factory(ctx)
			if err == nil {
				atomic.AddUint64(&r.successCount, 1)
				r.logger.Info("Connection established during gradual retry phase",
					zap.Int("attempt", attempt))
				return conn, nil
			}

			atomic.AddUint64(&r.failureCount, 1)
			r.logger.Warn("Connection attempt failed during gradual retry phase",
				zap.Int("attempt", attempt),
				zap.Duration("interval", interval),
				zap.Error(err))

			time.Sleep(interval)
			interval = time.Duration(float64(interval) * r.config.BackoffMultiplier)
			if interval > r.config.MaxInterval {
				interval = r.config.MaxInterval
			}
		}
	}
	return nil, ErrMaxRetriesExceeded
}

func (r *RetryManager) tryPersistentRetries(ctx context.Context) (net.Conn, error) {
	for attempt := 1; attempt <= r.config.MaxPersistentRetries; attempt++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			conn, err := r.factory(ctx)
			if err == nil {
				atomic.AddUint64(&r.successCount, 1)
				r.logger.Info("Connection established during persistent retry phase",
					zap.Int("attempt", attempt))
				return conn, nil
			}

			atomic.AddUint64(&r.failureCount, 1)
			r.logger.Warn("Connection attempt failed during persistent retry phase",
				zap.Int("attempt", attempt),
				zap.Error(err))

			time.Sleep(r.config.PersistentDelay)
		}
	}
	return nil, ErrMaxRetriesExceeded
}

// GetMetrics returns the current retry metrics
func (r *RetryManager) GetMetrics() (uint64, uint64) {
	return atomic.LoadUint64(&r.successCount), atomic.LoadUint64(&r.failureCount)
}

// ResetMetrics resets the retry metrics
func (r *RetryManager) ResetMetrics() {
	atomic.StoreUint64(&r.successCount, 0)
	atomic.StoreUint64(&r.failureCount, 0)
}

// WithConfig sets the retry configuration
func (r *RetryManager) WithConfig(config RetryConfig) *RetryManager {
	r.config = config
	return r
}
