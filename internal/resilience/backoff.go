package resilience

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// BackoffStrategy defines different backoff algorithms
type BackoffStrategy int

const (
	StrategyExponential BackoffStrategy = iota
	StrategyLinear
	StrategyFixed
	StrategyGeometric
)

// BackoffConfig represents exponential backoff configuration
type BackoffConfig struct {
	Strategy     BackoffStrategy
	BaseDelay    time.Duration                          // Base delay for exponential backoff
	MaxDelay     time.Duration                          // Maximum delay between retries
	MaxRetries   int                                    // Maximum number of retry attempts
	JitterFactor float64                                // Jitter to add randomness (0.0-1.0)
	ResetAfter   time.Duration                          // Reset backoff after this period of success
	EnableJitter bool                                   // Whether to add jitter to prevent thundering herd
	Multiplier   float64                                // Multiplier for exponential backoff (typically 2.0)
	Name         string                                 // Name for logging and metrics
	OnBackoff    func(attempt int, delay time.Duration) // Callback before backoff
	OnReset      func()                                 // Callback when backoff resets
}

// ExponentialBackoff implements adaptive retry backoff with jitter
type ExponentialBackoff struct {
	config               *BackoffConfig
	currentDelay         time.Duration
	lastRetry            time.Time
	retryCount           int32
	consecutiveSuccesses int32
	totalRetries         int64
	lastSuccess          time.Time

	mu     sync.RWMutex
	rand   *rand.Rand
	logger *zap.Logger
}

// NewExponentialBackoff creates a new exponential backoff instance
func NewExponentialBackoff(cfg *BackoffConfig, logger *zap.Logger) *ExponentialBackoff {
	if cfg == nil {
		cfg = getDefaultBackoffConfig()
	}

	if cfg.Strategy == StrategyExponential && cfg.Multiplier == 0 {
		cfg.Multiplier = 2.0 // Default exponential multiplier
	}

	if cfg.Strategy == StrategyGeometric && cfg.Multiplier == 0 {
		cfg.Multiplier = 1.5 // Default geometric multiplier
	}

	if cfg.JitterFactor == 0 && cfg.EnableJitter {
		cfg.JitterFactor = 0.1 // Default 10% jitter
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	return &ExponentialBackoff{
		config:       cfg,
		currentDelay: cfg.BaseDelay,
		rand:         rand.New(rand.NewSource(time.Now().UnixNano())),
		logger:       logger,
	}
}

// getDefaultBackoffConfig returns default backoff configuration
func getDefaultBackoffConfig() *BackoffConfig {
	return &BackoffConfig{
		Strategy:     StrategyExponential,
		BaseDelay:    time.Second,
		MaxDelay:     30 * time.Second,
		MaxRetries:   5,
		JitterFactor: 0.1,
		ResetAfter:   5 * time.Minute,
		EnableJitter: true,
		Multiplier:   2.0,
		Name:         "default",
	}
}

// ShouldRetry determines if a retry should be attempted
func (b *ExponentialBackoff) ShouldRetry() bool {
	return atomic.LoadInt32(&b.retryCount) < int32(b.config.MaxRetries)
}

// GetNextDelay returns the next retry delay with jitter
func (b *ExponentialBackoff) GetNextDelay() time.Duration {
	b.mu.Lock()
	defer b.mu.Unlock()

	delay := b.calculateDelay()

	// Add jitter to prevent thundering herd
	if b.config.EnableJitter && b.config.JitterFactor > 0 {
		jitter := time.Duration(float64(delay) * b.config.JitterFactor * b.rand.Float64())
		if b.rand.Int()%2 == 0 {
			delay += jitter
		} else {
			delay -= jitter
		}
		// Ensure delay remains positive and within bounds
		if delay < 0 {
			delay = -delay
		}
		if delay > b.config.MaxDelay {
			delay = b.config.MaxDelay
		}
	}

	return delay
}

// calculateDelay computes the delay based on current strategy
func (b *ExponentialBackoff) calculateDelay() time.Duration {
	currentRetry := int(atomic.LoadInt32(&b.retryCount))

	switch b.config.Strategy {
	case StrategyExponential:
		delay := float64(b.config.BaseDelay) * math.Pow(b.config.Multiplier, float64(currentRetry))
		if delay > float64(b.config.MaxDelay) {
			return b.config.MaxDelay
		}
		return time.Duration(delay)

	case StrategyLinear:
		delay := b.config.BaseDelay + time.Duration(currentRetry)*b.config.BaseDelay
		if delay > b.config.MaxDelay {
			return b.config.MaxDelay
		}
		return delay

	case StrategyFixed:
		return b.config.BaseDelay

	case StrategyGeometric:
		delay := float64(b.config.BaseDelay)
		for i := 0; i < currentRetry; i++ {
			delay *= b.config.Multiplier
		}
		if time.Duration(delay) > b.config.MaxDelay {
			return b.config.MaxDelay
		}
		return time.Duration(delay)
	}

	return b.config.MaxDelay
}

// Retry executes a function with exponential backoff retry logic
func (b *ExponentialBackoff) Retry(ctx context.Context, fn func(appCtx context.Context) error) error {
	var lastErr error
	retryAttempt := 0

	for {
		if retryAttempt > 0 {
			// Check if we should retry
			if !b.ShouldRetry() {
				break
			}

			// Check if context is cancelled
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			// Call backoff callback
			delay := b.GetNextDelay()
			if b.config.OnBackoff != nil {
				b.config.OnBackoff(retryAttempt, delay)
			}

			b.logger.Info("Retrying operation with backoff",
				zap.String("name", b.config.Name),
				zap.Int("attempt", retryAttempt),
				zap.Duration("delay", delay),
				zap.Error(lastErr))

			// Wait for the backoff period
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				// Continue with retry
			}
		}

		// Execute the function
		err := fn(ctx)
		if err == nil {
			// Success - increment retry count and record success
			atomic.AddInt32(&b.retryCount, 1)
			atomic.AddInt32(&b.consecutiveSuccesses, 1)
			atomic.AddInt64(&b.totalRetries, int64(retryAttempt))
			b.lastSuccess = time.Now()
			b.lastRetry = time.Now()

			b.logger.Debug("Operation succeeded",
				zap.String("name", b.config.Name),
				zap.Int("attempt", retryAttempt))

			// Check if we should reset the backoff
			b.tryResetBackoff()

			return nil
		}

		// Failure - record and continue retrying
		lastErr = err
		atomic.AddInt32(&b.retryCount, 1)
		atomic.StoreInt32(&b.consecutiveSuccesses, 0)
		b.lastRetry = time.Now()
		totalRetries := atomic.AddInt64(&b.totalRetries, 1)

		b.logger.Warn("Operation failed, will retry",
			zap.String("name", b.config.Name),
			zap.Int("attempt", retryAttempt),
			zap.Int64("total_retries", totalRetries),
			zap.Error(err))

		retryAttempt++
	}

	// All retries exhausted
	atomic.AddInt32(&b.retryCount, 1)
	totalRetries := atomic.AddInt64(&b.totalRetries, int64(retryAttempt))

	b.logger.Error("Operation failed after all retries",
		zap.String("name", b.config.Name),
		zap.Int("max_retries", b.config.MaxRetries),
		zap.Int64("total_retries", totalRetries),
		zap.Error(lastErr))

	return lastErr
}

// tryResetBackoff checks if the backoff should be reset after a period of success
func (b *ExponentialBackoff) tryResetBackoff() {
	if b.config.ResetAfter <= 0 {
		return
	}

	// Check if we have had continuous success for the reset period
	if b.consecutiveSuccesses > int32(b.config.MaxRetries) {
		if time.Since(b.lastRetry) > b.config.ResetAfter {
			b.Reset()
		}
	}
}

// Reset resets the backoff state
func (b *ExponentialBackoff) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	atomic.StoreInt32(&b.retryCount, 0)
	atomic.StoreInt32(&b.consecutiveSuccesses, 0)
	b.currentDelay = b.config.BaseDelay
	b.lastRetry = time.Time{}
	b.totalRetries = 0

	if b.config.OnReset != nil {
		b.config.OnReset()
	}

	b.logger.Info("Exponential backoff reset",
		zap.String("name", b.config.Name))
}

// GetRetryCount returns the current retry count
func (b *ExponentialBackoff) GetRetryCount() int {
	return int(atomic.LoadInt32(&b.retryCount))
}

// GetConsecutiveSuccesses returns consecutive successes count
func (b *ExponentialBackoff) GetConsecutiveSuccesses() int {
	return int(atomic.LoadInt32(&b.consecutiveSuccesses))
}

// GetTotalRetries returns the total number of retries across all operations
func (b *ExponentialBackoff) GetTotalRetries() int64 {
	return atomic.LoadInt64(&b.totalRetries)
}

// GetAverageDelay calculates the average delay based on current retry count
func (b *ExponentialBackoff) GetAverageDelay() time.Duration {
	b.mu.RLock()
	defer b.mu.RUnlock()

	retryCount := atomic.LoadInt32(&b.retryCount)
	totalDelay := time.Duration(0)

	for i := 0; i < int(retryCount); i++ {
		switch b.config.Strategy {
		case StrategyExponential:
			delay := float64(b.config.BaseDelay) * math.Pow(b.config.Multiplier, float64(i))
			totalDelay += time.Duration(delay)
		case StrategyLinear:
			delay := b.config.BaseDelay + time.Duration(i)*b.config.BaseDelay
			totalDelay += delay
		case StrategyFixed:
			totalDelay += b.config.BaseDelay
		case StrategyGeometric:
			delay := float64(b.config.BaseDelay)
			for j := 0; j < i; j++ {
				delay *= b.config.Multiplier
			}
			totalDelay += time.Duration(delay)
		}
	}

	if retryCount == 0 {
		return 0
	}

	return totalDelay / time.Duration(retryCount)
}

// GetStats returns current backoff statistics
func (b *ExponentialBackoff) GetStats() BackoffStats {
	b.mu.RLock()
	defer b.mu.RUnlock()

	return BackoffStats{
		CurrentRetry:         int(atomic.LoadInt32(&b.retryCount)),
		ConsecutiveSuccesses: int(atomic.LoadInt32(&b.consecutiveSuccesses)),
		TotalRetries:         atomic.LoadInt64(&b.totalRetries),
		LastRetry:            b.lastRetry,
		LastSuccess:          b.lastSuccess,
		NextDelay:            b.GetNextDelay(),
		AverageDelay:         b.GetAverageDelay(),
		Strategy:             b.config.Strategy,
		ShouldRetry:          b.ShouldRetry(),
	}
}

// BackoffStats represents backoff statistics
type BackoffStats struct {
	CurrentRetry         int
	ConsecutiveSuccesses int
	TotalRetries         int64
	LastRetry            time.Time
	LastSuccess          time.Time
	NextDelay            time.Duration
	AverageDelay         time.Duration
	Strategy             BackoffStrategy
	ShouldRetry          bool
}

// SetConfig updates the backoff configuration
func (b *ExponentialBackoff) SetConfig(cfg *BackoffConfig) error {
	if cfg == nil {
		return fmt.Errorf("invalid backoff configuration")
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	b.config = cfg

	// Reset state when configuration changes
	atomic.StoreInt32(&b.retryCount, 0)
	atomic.StoreInt32(&b.consecutiveSuccesses, 0)
	b.currentDelay = cfg.BaseDelay
	b.lastRetry = time.Time{}

	b.logger.Info("Backoff configuration updated",
		zap.String("name", cfg.Name),
		zap.String("strategy", b.strategyName(b.config.Strategy)),
		zap.Duration("base_delay", cfg.BaseDelay),
		zap.Duration("max_delay", cfg.MaxDelay),
		zap.Int("max_retries", cfg.MaxRetries))

	return nil
}

// strategyName returns the string name of a backoff strategy
func (b *ExponentialBackoff) strategyName(strategy BackoffStrategy) string {
	switch strategy {
	case StrategyExponential:
		return "exponential"
	case StrategyLinear:
		return "linear"
	case StrategyFixed:
		return "fixed"
	case StrategyGeometric:
		return "geometric"
	default:
		return "unknown"
	}
}

// Wait performs a simple backoff wait without retry logic
func (b *ExponentialBackoff) Wait(ctx context.Context) error {
	delay := b.GetNextDelay()
	atomic.AddInt32(&b.retryCount, 1)

	b.logger.Debug("Waiting with backoff",
		zap.String("name", b.config.Name),
		zap.Duration("delay", delay))

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
		return nil
	}
}

// ForceRetryCount sets the current retry count (for testing)
func (b *ExponentialBackoff) ForceRetryCount(count int) {
	atomic.StoreInt32(&b.retryCount, int32(count))
}

// Predefined backoff configurations for common use cases
var (
	// FastRetryConfig for quick-retrying operations
	FastRetryConfig = &BackoffConfig{
		Strategy:     StrategyExponential,
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		MaxRetries:   10,
		JitterFactor: 0.2,
		EnableJitter: true,
		ResetAfter:   time.Minute,
		Multiplier:   1.5,
		Name:         "fast_retry",
	}

	// NetworkRetryConfig for network operations
	NetworkRetryConfig = &BackoffConfig{
		Strategy:     StrategyExponential,
		BaseDelay:    time.Second,
		MaxDelay:     time.Minute,
		MaxRetries:   5,
		JitterFactor: 0.1,
		EnableJitter: true,
		ResetAfter:   5 * time.Minute,
		Multiplier:   2.0,
		Name:         "network_retry",
	}

	// DatabaseRetryConfig for database operations
	DatabaseRetryConfig = &BackoffConfig{
		Strategy:     StrategyGeometric,
		BaseDelay:    500 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		MaxRetries:   8,
		JitterFactor: 0.05,
		EnableJitter: true,
		ResetAfter:   10 * time.Minute,
		Multiplier:   1.6,
		Name:         "database_retry",
	}
)
