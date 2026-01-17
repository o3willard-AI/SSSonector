package resilience

import (
	"context"
	"fmt"
	"math/rand"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// RetryStrategy defines different retry algorithms
type RetryStrategy int

const (
	StrategyFixed RetryStrategy = iota
	StrategyLinear
	StrategyExponential
	StrategyGeometric
	StrategyCustom
)

// RetryConfig defines retry behavior configuration
type RetryConfig struct {
	Strategy        RetryStrategy
	MaxAttempts     int
	BaseDelay       time.Duration
	MaxDelay        time.Duration
	Multiplier      float64
	JitterFactor    float64
	EnableJitter    bool
	ErrorClassifier func(error) RetryAction
	OnRetry         func(attempt int, delay time.Duration, err error)
	OnSuccess       func(attempt int)
	OnFailure       func(attempt int, finalErr error)
	RequestTimeout  time.Duration
	AttemptTimeout  time.Duration
	CircuitBreaker  *CircuitBreaker
	BackoffStrategy *ExponentialBackoff
	Name            string
}

// RetryAction defines what action to take for a specific error
type RetryAction int

const (
	ActionRetry RetryAction = iota // Retry the operation
	ActionFail                     // Fail immediately
	ActionSkip                     // Skip this operation
)

// RetryResult represents the result of a retry operation
type RetryResult struct {
	Success    bool
	Failed     bool
	Skipped    bool
	Attempts   int
	TotalDelay time.Duration
	LastError  error
	StartTime  time.Time
	EndTime    time.Time
}

// RetryManager implements the retry framework with multiple strategies
type RetryManager struct {
	config     *RetryConfig
	logger     *zap.Logger
	statistics RetryStatistics
	stopCh     chan struct{}
}

// RetryStatistics tracks retry performance metrics
type RetryStatistics struct {
	TotalRetries      int64
	SuccessfulRetries int64
	FailedRetries     int64
	SkippedRetries    int64
	TotalDelayTime    time.Duration
	AverageDelayTime  time.Duration
	LastRetryTime     time.Time
}

// NewRetryManager creates a new retry manager
func NewRetryManager(config *RetryConfig, logger *zap.Logger) *RetryManager {
	if config == nil {
		config = getDefaultRetryConfig()
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	if config.ErrorClassifier == nil {
		config.ErrorClassifier = defaultErrorClassifier
	}

	return &RetryManager{
		config: config,
		logger: logger,
		stopCh: make(chan struct{}),
	}
}

// Execute runs a function with retry logic
func (rm *RetryManager) Execute(ctx context.Context, fn func(ctx context.Context) error) RetryResult {
	startTime := time.Now()
	result := RetryResult{
		StartTime: startTime,
	}

	// Update statistics
	atomic.AddInt64(&rm.statistics.TotalRetries, 1)

	// Check circuit breaker if configured
	if rm.config.CircuitBreaker != nil {
		canExecute := rm.config.CircuitBreaker.CanExecute()
		if !canExecute {
			rm.logger.Warn("Circuit breaker blocked retry execution",
				zap.String("name", rm.config.Name))
			result.Skipped = true
			atomic.AddInt64(&rm.statistics.SkippedRetries, 1)
			return result
		}
	}

	var attemptErr error
	for attempt := 1; attempt <= rm.config.MaxAttempts; attempt++ {
		// Create attempt context with timeout if configured
		attemptCtx := ctx
		var attemptCancel context.CancelFunc
		if rm.config.AttemptTimeout > 0 {
			attemptCtx, attemptCancel = context.WithTimeout(ctx, rm.config.AttemptTimeout)
			defer attemptCancel()
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			result.EndTime = time.Now()
			result.LastError = ctx.Err()
			atomic.AddInt64(&rm.statistics.FailedRetries, 1)
			rm.callOnFailure(result.Attempts, ctx.Err())
			return result
		default:
		}

		// Execute the function
		err := fn(attemptCtx)
		result.Attempts = attempt
		result.LastError = err

		// Success
		if err == nil {
			result.EndTime = time.Now()
			result.Success = true
			atomic.AddInt64(&rm.statistics.SuccessfulRetries, 1)
			result.TotalDelay = result.EndTime.Sub(startTime)
			rm.callOnSuccess(attempt)

			// Update circuit breaker on success
			if rm.config.CircuitBreaker != nil {
				rm.config.CircuitBreaker.Call(attemptCtx, func(ctx context.Context) error { return nil })
			}

			rm.logger.Debug("Retry operation succeeded",
				zap.String("name", rm.config.Name),
				zap.Int("attempt", attempt),
				zap.Duration("total_delay", result.TotalDelay))

			return result
		}

		// Classify the error
		action := rm.config.ErrorClassifier(err)
		result.LastError = err

		switch action {
		case ActionFail:
			// Fail immediately for this error type
			result.EndTime = time.Now()
			result.Failed = true
			atomic.AddInt64(&rm.statistics.FailedRetries, 1)
			result.TotalDelay = result.EndTime.Sub(startTime)
			rm.callOnFailure(attempt, err)
			return result

		case ActionSkip:
			// Skip this operation
			result.EndTime = time.Now()
			result.Skipped = true
			atomic.AddInt64(&rm.statistics.SkippedRetries, 1)
			result.TotalDelay = result.EndTime.Sub(startTime)
			return result

		case ActionRetry:
			// Continue to retry logic
			result.LastError = err

		default:
			// Default to retry
		}

		// Check if we should retry (not the last attempt)
		if attempt < rm.config.MaxAttempts {
			delay := rm.calculateDelay(attempt)

			rm.logger.Info("Retrying operation",
				zap.String("name", rm.config.Name),
				zap.Int("attempt", attempt),
				zap.Duration("delay", delay),
				zap.Error(err))

			rm.callOnRetry(attempt, delay, err)

			// Wait for the calculated delay
			select {
			case <-ctx.Done():
				result.EndTime = time.Now()
				result.LastError = ctx.Err()
				atomic.AddInt64(&rm.statistics.FailedRetries, 1)
				rm.callOnFailure(attempt, ctx.Err())
				return result
			case <-time.After(delay):
				// Continue to next attempt
			}

			atomic.AddInt64(&rm.statistics.TotalDelayTime, int64(delay))
		}
	}

	// All attempts exhausted
	result.EndTime = time.Now()
	result.Failed = true
	atomic.AddInt64(&rm.statistics.FailedRetries, 1)
	result.TotalDelay = result.EndTime.Sub(startTime)

	rm.logger.Error("Retry operation failed after all attempts",
		zap.String("name", rm.config.Name),
		zap.Int("attempts", result.Attempts),
		zap.Duration("total_delay", result.TotalDelay),
		zap.Error(result.LastError))

	rm.callOnFailure(result.Attempts, result.LastError)
	atomic.StoreInt64(&rm.statistics.LastRetryTime, result.EndTime.UnixNano())

	return result
}

// calculateDelay computes the delay for the given attempt
func (rm *RetryManager) calculateDelay(attempt int) time.Duration {
	var delay time.Duration

	switch rm.config.Strategy {
	case StrategyFixed:
		delay = rm.config.BaseDelay

	case StrategyLinear:
		delay = rm.config.BaseDelay + time.Duration(attempt-1)*rm.config.BaseDelay

	case StrategyExponential:
		if rm.config.BackoffStrategy != nil {
			delay = rm.config.BackoffStrategy.GetNextDelay()
		} else {
			multiplier := rm.config.Multiplier
			if multiplier == 0 {
				multiplier = 2.0
			}
			timeValue := float64(rm.config.BaseDelay) * pow(multiplier, float64(attempt-1))
			delay = time.Duration(timeValue)
		}

	case StrategyGeometric:
		if rm.config.BackoffStrategy != nil {
			delay = rm.config.BackoffStrategy.GetNextDelay()
		} else {
			multiplier := rm.config.Multiplier
			if multiplier == 0 {
				multiplier = 1.5
			}
			timeValue := float64(rm.config.BaseDelay)
			for i := 1; i < attempt; i++ {
				timeValue *= multiplier
			}
			delay = time.Duration(timeValue)
		}

	case StrategyCustom:
		// Custom strategy implemented through backoff
		if rm.config.BackoffStrategy != nil {
			delay = rm.config.BackoffStrategy.GetNextDelay()
		} else {
			delay = rm.config.BaseDelay
		}

	default:
		delay = rm.config.BaseDelay
	}

	// Cap at maximum delay
	if delay > rm.config.MaxDelay && rm.config.MaxDelay > 0 {
		delay = rm.config.MaxDelay
	}

	// Add jitter if enabled
	if rm.config.EnableJitter && rm.config.JitterFactor > 0 {
		jitter := rm.config.JitterFactor
		if jitter > 1.0 {
			jitter = 1.0
		}
		jitterAmount := time.Duration(float64(delay) * jitter * rand.Float64())
		delay += jitterAmount
	}

	return delay
}

// GetStatistics returns current retry statistics
func (rm *RetryManager) GetStatistics() RetryStatistics {
	stats := rm.statistics

	// Calculate average delay
	totalRetries := stats.SuccessfulRetries + stats.FailedRetries + stats.SkippedRetries
	if totalRetries > 0 {
		stats.AverageDelayTime = stats.TotalDelayTime / time.Duration(totalRetries)
	}

	return stats
}

// ResetStatistics resets all retry statistics
func (rm *RetryManager) ResetStatistics() {
	atomic.StoreInt64(&rm.statistics.TotalRetries, 0)
	atomic.StoreInt64(&rm.statistics.SuccessfulRetries, 0)
	atomic.StoreInt64(&rm.statistics.FailedRetries, 0)
	atomic.StoreInt64(&rm.statistics.SkippedRetries, 0)
	atomic.StoreInt64(&rm.statistics.TotalDelayTime, 0)
	atomic.StoreInt64(&rm.statistics.LastRetryTime, 0)
	atomic.StoreInt64(&rm.statistics.AverageDelayTime, 0)
}

// GetConfig returns the current configuration
func (rm *RetryManager) GetConfig() *RetryConfig {
	return rm.config
}

// SetConfig updates the retry configuration
func (rm *RetryManager) SetConfig(config *RetryConfig) error {
	if config == nil {
		return fmt.Errorf("retry configuration cannot be nil")
	}

	rm.mu.Lock()
	defer rm.mu.Unlock()

	rm.config = config

	rm.logger.Info("Retry configuration updated",
		zap.String("name", config.Name),
		zap.String("strategy", rm.strategyName(config.Strategy)),
		zap.Int("max_attempts", config.MaxAttempts),
		zap.Duration("base_delay", config.BaseDelay),
		zap.Duration("max_delay", config.MaxDelay))

	return nil
}

// callOnRetry calls the retry callback if configured
func (rm *RetryManager) callOnRetry(attempt int, delay time.Duration, err error) {
	if rm.config.OnRetry != nil {
		rm.config.OnRetry(attempt, delay, err)
	}
}

// callOnSuccess calls the success callback if configured
func (rm *RetryManager) callOnSuccess(attempt int) {
	if rm.config.OnSuccess != nil {
		rm.config.OnSuccess(attempt)
	}
}

// callOnFailure calls the failure callback if configured
func (rm *RetryManager) callOnFailure(attempt int, err error) {
	if rm.config.OnFailure != nil {
		rm.config.OnFailure(attempt, err)
	}
}

// strategyName returns the string name of a retry strategy
func (rm *RetryManager) strategyName(strategy RetryStrategy) string {
	switch strategy {
	case StrategyFixed:
		return "fixed"
	case StrategyLinear:
		return "linear"
	case StrategyExponential:
		return "exponential"
	case StrategyGeometric:
		return "geometric"
	case StrategyCustom:
		return "custom"
	default:
		return "unknown"
	}
}

// defaultErrorClassifier provides a default error classification strategy
func defaultErrorClassifier(err error) RetryAction {
	if err == nil {
		return ActionRetry // Shouldn't happen
	}

	// Check for specific error types that should not be retried
	switch err {
	case context.Canceled:
		return ActionFail
	case context.DeadlineExceeded:
		return ActionFail
	default:
		// Default to retry for most errors
		return ActionRetry
	}
}

// getDefaultRetryConfig returns default retry configuration
func getDefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		Strategy:        StrategyExponential,
		MaxAttempts:     3,
		BaseDelay:       100 * time.Millisecond,
		MaxDelay:        5 * time.Second,
		Multiplier:      2.0,
		JitterFactor:    0.1,
		EnableJitter:    true,
		RequestTimeout:  30 * time.Second,
		AttemptTimeout:  10 * time.Second,
		Name:            "default",
		ErrorClassifier: defaultErrorClassifier,
	}
}

// pow calculates base^exponent for float64
func pow(base, exponent float64) float64 {
	result := 1.0
	for i := 0; i < int(exponent); i++ {
		result *= base
	}
	return result
}

// RetryContext provides contextual information for retry operations
type RetryContext struct {
	Attempt     int
	MaxAttempts int
	Delay       time.Duration
	LastError   error
	StartTime   time.Time
	CircuitOpen bool
}

// GetContext returns current retry context for use in callbacks
func (rm *RetryManager) GetContext(attempt int, delay time.Duration, err error) RetryContext {
	isOpen := false
	if rm.config.CircuitBreaker != nil {
		isOpen = rm.config.CircuitBreaker.GetState() == StateOpen
	}

	return RetryContext{
		Attempt:     attempt,
		MaxAttempts: rm.config.MaxAttempts,
		Delay:       delay,
		LastError:   err,
		StartTime:   time.Now(),
		CircuitOpen: isOpen,
	}
}

// QuickRetry provides a simple retry function for quick use-cases
func QuickRetry(ctx context.Context, fn func() error, maxAttempts int, baseDelay time.Duration) error {
	manager := NewRetryManager(&RetryConfig{
		Strategy:    StrategyExponential,
		MaxAttempts: maxAttempts,
		BaseDelay:   baseDelay,
		MaxDelay:    30 * time.Second,
		Name:        "quick_retry",
	}, nil)

	result := manager.Execute(ctx, func(ctx context.Context) error {
		return fn()
	})

	if result.Failed {
		return result.LastError
	}

	return nil
}
