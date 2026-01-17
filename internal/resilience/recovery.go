package resilience

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// ErrorRecovery implements comprehensive error recovery mechanisms
type ErrorRecovery struct {
	config      *RecoveryConfig
	classifiers []ErrorClassifier
	strategies  map[string]RecoveryStrategy
	observers   []RecoveryObserver
	metrics     RecoveryMetrics

	mu     sync.RWMutex
	rand   *rand.Rand
	ctx    context.Context
	cancel context.CancelFunc
	logger *zap.Logger
}

// RecoveryStrategy defines how to recover from different error types
type RecoveryStrategy interface {
	Name() string
	CanRecover(err error) bool
	Recover(ctx context.Context, err error, attempt int) error
	Configure(config map[string]interface{}) error
}

// ErrorClassifier classifies errors for appropriate recovery
type ErrorClassifier struct {
	Name     string
	Classify func(error) ErrorCategory
	Priority int // Higher = classified first
}

// ErrorCategory represents different types of recoverable errors
type ErrorCategory int

const (
	CategoryUnknown ErrorCategory = iota
	CategoryRetryable
	CategoryNonRetryable
	CategoryRateLimit
	CategoryResourceExhaustion
	CategoryNetwork
	CategoryTimeout
	CategoryCircuitOpen
	CategoryValidation
	CategoryConfiguration
	CategoryRecoverable
)

// RecoveryObserver observes recovery operations
type RecoveryObserver interface {
	OnRecoveryStart(err error, strategy string)
	OnRecoverySuccess(err error, strategy string, duration time.Time)
	OnRecoveryFailed(err error, strategy string, newError error)
}

// RecoveryMetrics tracks recovery performance
type RecoveryMetrics struct {
	TotalRecoveries      int64
	SuccessfulRecoveries int64
	FailedRecoveries     int64
	RecoveryTime         time.Duration
	AverageRecoveryTime  time.Duration
	ErrorCategoryMetrics map[ErrorCategory]int64
	StrategyUsage        map[string]int64
}

// RecoveryConfig defines error recovery configuration
type RecoveryConfig struct {
	Classifiers []ErrorClassifier
	Strategies  map[string]RecoveryStrategy
	Observers   []RecoveryObserver
	MaxRetries  int
	Timeout     time.Duration
}

// BuiltinRecoveryStrategies provides standard recovery strategies
var BuiltinRecoveryStrategies = map[string]RecoveryStrategy{
	"backoff_retry":  &BackoffRetryStrategy{},
	"circuit_reset":  &CircuitResetStrategy{},
	"resource_retry": &ResourceRetryStrategy{},
	"network_retry":  &NetworkRetryStrategy{},
}

// Built-in recovery strategies

// BackoffRetryStrategy implements backoff-based retry
type BackoffRetryStrategy struct {
	backoff *ExponentialBackoff
	config  *BackoffConfig
}

func (s *BackoffRetryStrategy) Name() string { return "backoff_retry" }

func (s *BackoffRetryStrategy) CanRecover(err error) bool {
	cat := ClassifyCommonError(err)
	return cat != CategoryNonRetryable && cat != CategoryCircuitOpen
}

func (s *BackoffRetryStrategy) Recover(ctx context.Context, err error, attempt int) error {
	if s.backoff == nil {
		s.backoff = NewExponentialBackoff(s.config, zap.NewNop())
	}

	// Wait with backoff, but don't actually retry the operation
	duration := s.backoff.GetNextDelay()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
		return fmt.Errorf("backoff recovery simulated for attempt %d: %w", attempt, err)
	}
}

func (s *BackoffRetryStrategy) Configure(config map[string]interface{}) error {
	s.config = &BackoffConfig{
		BaseDelay:    time.Second,
		MaxDelay:     30 * time.Second,
		MaxRetries:   3,
		JitterFactor: 0.1,
		EnableJitter: true,
	}

	if baseDelay, ok := config["baseDelay"].(time.Duration); ok {
		s.config.BaseDelay = baseDelay
	}
	if maxDelay, ok := config["maxDelay"].(time.Duration); ok {
		s.config.MaxDelay = maxDelay
	}
	if maxRetries, ok := config["maxRetries"].(int); ok {
		s.config.MaxRetries = maxRetries
	}

	s.backoff = NewExponentialBackoff(s.config, zap.NewNop())
	return nil
}

// CircuitResetStrategy attempts to reset circuit breakers
type CircuitResetStrategy struct {
	resetDelay time.Duration
	maxResets  int
	resetCount int
}

func (s *CircuitResetStrategy) Name() string { return "circuit_reset" }

func (s *CircuitResetStrategy) CanRecover(err error) bool {
	// Only recover if it's specifically a circuit breaker open error
	return errors.Is(err, ErrCircuitOpen)
}

func (s *CircuitResetStrategy) Recover(ctx context.Context, err error, attempt int) error {
	if s.resetCount >= s.maxResets {
		return fmt.Errorf("circuit reset limit exceeded: %w", err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(s.resetDelay):
		s.resetCount++
		return fmt.Errorf("circuit reset attempted at attempt %d: %w", attempt, err)
	}
}

func (s *CircuitResetStrategy) Configure(config map[string]interface{}) error {
	s.resetDelay = 5 * time.Second
	s.maxResets = 2

	if delay, ok := config["resetDelay"].(time.Duration); ok {
		s.resetDelay = delay
	}
	if max, ok := config["maxResets"].(int); ok {
		s.maxResets = max
	}
	return nil
}

// ResourceRetryStrategy handles resource exhaustion scenarios
type ResourceRetryStrategy struct {
	retryDelay time.Duration
	maxRetries int
}

func (s *ResourceRetryStrategy) Name() string { return "resource_retry" }

func (s *ResourceRetryStrategy) CanRecover(err error) bool {
	cat := ClassifyCommonError(err)
	return cat == CategoryResourceExhaustion || cat == CategoryRateLimit
}

func (s *ResourceRetryStrategy) Recover(ctx context.Context, err error, attempt int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(s.retryDelay):
		return fmt.Errorf("resource retry at attempt %d: %w", attempt, err)
	}
}

func (s *ResourceRetryStrategy) Configure(config map[string]interface{}) error {
	s.retryDelay = time.Second
	s.maxRetries = 3

	if delay, ok := config["retryDelay"].(time.Duration); ok {
		s.retryDelay = delay
	}
	if max, ok := config["maxRetries"].(int); ok {
		s.maxRetries = max
	}
	return nil
}

// NetworkRetryStrategy handles network-specific errors
type NetworkRetryStrategy struct {
	backoffStrategy *ExponentialBackoff
	retryDelay      time.Duration
}

func (s *NetworkRetryStrategy) Name() string { return "network_retry" }

func (s *NetworkRetryStrategy) CanRecover(err error) bool {
	cat := ClassifyCommonError(err)
	return cat == CategoryNetwork || cat == CategoryTimeout
}

func (s *NetworkRetryStrategy) Recover(ctx context.Context, err error, attempt int) error {
	duration := s.retryDelay
	if s.backoffStrategy != nil {
		duration = s.backoffStrategy.GetNextDelay()
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
		return fmt.Errorf("network retry at attempt %d: %w", attempt, err)
	}
}

func (s *NetworkRetryStrategy) Configure(config map[string]interface{}) error {
	s.retryDelay = 2 * time.Second

	backoffCfg := &BackoffConfig{
		BaseDelay:    time.Second,
		MaxDelay:     15 * time.Second,
		MaxRetries:   5,
		JitterFactor: 0.2,
		EnableJitter: true,
	}
	s.backoffStrategy = NewExponentialBackoff(backoffCfg, zap.NewNop())

	if delay, ok := config["retryDelay"].(time.Duration); ok {
		s.retryDelay = delay
	}
	return nil
}

// Common error classifiers

// ClassifyCommonError provides common error classification
func ClassifyCommonError(err error) ErrorCategory {
	if err == nil {
		return CategoryUnknown
	}

	switch {
	case errors.Is(err, context.Canceled):
		return CategoryNonRetryable
	case errors.Is(err, context.DeadlineExceeded):
		return CategoryTimeout
	case errors.Is(err, ErrCircuitOpen):
		return CategoryCircuitOpen
	default:
		// Check error message patterns
		errStr := err.Error()
		switch {
		case matchError(errStr, []string{"connection refused", "no such host", "network is unreachable", "dial tcp"}):
			return CategoryNetwork
		case matchError(errStr, []string{"rate limit", "too many requests", "429"}):
			return CategoryRateLimit
		case matchError(errStr, []string{"resource exhausted", "out of memory", "insufficient resources"}):
			return CategoryResourceExhaustion
		case matchError(errStr, []string{"invalid", "validation", "malformed", "bad request"}):
			return CategoryValidation
		case matchError(errStr, []string{"configuration", "config", "settings"}):
			return CategoryConfiguration
		default:
			return CategoryRecoverable
		}
	}
}

func matchError(errStr string, patterns []string) bool {
	for _, pattern := range patterns {
		if containsString(errStr, pattern) {
			return true
		}
	}
	return false
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && reflect.ValueOf(s).String() == s[:len(s)-len(substr)+len(substr)]
}

// ErrorRecovery implementation

// NewErrorRecovery creates a new error recovery system
func NewErrorRecovery(cfg *RecoveryConfig, logger *zap.Logger) (*ErrorRecovery, error) {
	if cfg == nil {
		cfg = getDefaultRecoveryConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	if logger == nil {
		logger = zap.NewNop()
	}

	er := &ErrorRecovery{
		classifiers: make([]ErrorClassifier, 0),
		strategies:  make(map[string]RecoveryStrategy),
		observers:   cfg.Observers,
		metrics: RecoveryMetrics{
			ErrorCategoryMetrics: make(map[ErrorCategory]int64),
			StrategyUsage:        make(map[string]int64),
		},
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
		ctx:    ctx,
		cancel: cancel,
		logger: logger,
	}

	// Add default classifiers
	er.addDefaultClassifiers()

	// Configure provided strategies
	for name, strategy := range cfg.Strategies {
		er.strategies[name] = strategy
	}

	return er, nil
}

// AddStrategy adds a recovery strategy
func (er *ErrorRecovery) AddStrategy(name string, strategy RecoveryStrategy) error {
	er.mu.Lock()
	defer er.mu.Unlock()

	if _, exists := er.strategies[name]; exists {
		return fmt.Errorf("recovery strategy %s already exists", name)
	}

	er.strategies[name] = strategy
	er.logger.Info("Added recovery strategy", zap.String("name", name))
	return nil
}

// AddClassifier adds an error classifier
func (er *ErrorRecovery) AddClassifier(classifier ErrorClassifier) error {
	er.mu.Lock()
	defer er.mu.Unlock()

	// Insert based on priority (higher priority first)
	inserted := false
	for i, existing := range er.classifiers {
		if classifier.Priority > existing.Priority {
			// Insert before this element
			er.classifiers = append(er.classifiers[:i], append([]ErrorClassifier{classifier}, er.classifiers[i:]...)...)
			inserted = true
			break
		}
	}

	if !inserted {
		er.classifiers = append(er.classifiers, classifier)
	}

	er.logger.Info("Added error classifier",
		zap.String("name", classifier.Name),
		zap.Int("priority", classifier.Priority))
	return nil
}

// Recover attempts to recover from an error
func (er *ErrorRecovery) Recover(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	atomic.AddInt64(&er.metrics.TotalRecoveries, 1)
	startTime := time.Now()

	// Classify the error
	category := er.classifyError(err)
	atomic.AddInt64(&er.metrics.ErrorCategoryMetrics[category], 1)

	// Find appropriate strategy
	strategy := er.findStrategy(err)
	if strategy == nil {
		er.logger.Warn("No suitable recovery strategy found for error",
			zap.Error(err),
			zap.String("category", er.categoryName(category)))
		atomic.AddInt64(&er.metrics.FailedRecoveries, 1)
		return err
	}

	// Notify observers
	for _, observer := range er.observers {
		observer.OnRecoveryStart(err, strategy.Name())
	}

	// Attempt recovery
	recoveryErr := strategy.Recover(ctx, err, 1)
	atomic.AddInt64(&er.metrics.StrategyUsage[strategy.Name()], 1)

	endTime := time.Now()
	duration := endTime.Sub(startTime)
	atomic.AddInt64(&er.metrics.RecoveryTime, int64(duration))

	// Calculate running average
	total := atomic.LoadInt64(&er.metrics.TotalRecoveries)
	avg := float64(atomic.LoadInt64(&er.metrics.RecoveryTime)) / float64(total)
	atomic.StoreInt64(&er.metrics.AverageRecoveryTime, int64(avg))

	if recoveryErr == nil {
		atomic.AddInt64(&er.metrics.SuccessfulRecoveries, 1)

		for _, observer := range er.observers {
			observer.OnRecoverySuccess(err, strategy.Name(), endTime)
		}

		er.logger.Info("Error recovery successful",
			zap.String("strategy", strategy.Name()),
			zap.Duration("duration", duration),
			zap.Error(err))
		return nil
	}

	atomic.AddInt64(&er.metrics.FailedRecoveries, 1)

	for _, observer := range er.observers {
		observer.OnRecoveryFailed(err, strategy.Name(), recoveryErr)
	}

	er.logger.Warn("Error recovery failed",
		zap.String("strategy", strategy.Name()),
		zap.Duration("duration", duration),
		zap.Error(recoveryErr),
		zap.Error(err))
	return recoveryErr
}

// GetMetrics returns current recovery metrics
func (er *ErrorRecovery) GetMetrics() RecoveryMetrics {
	return er.metrics
}

// Stop stops the error recovery system
func (er *ErrorRecovery) Stop() {
	er.cancel()
}

// Helper methods

func (er *ErrorRecovery) addDefaultClassifiers() {
	classifiers := []ErrorClassifier{
		{
			Name: "network_errors",
			Classify: func(err error) ErrorCategory {
				return ClassifyCommonError(err)
			},
			Priority: 10,
		},
		{
			Name: "resource_errors",
			Classify: func(err error) ErrorCategory {
				cat := ClassifyCommonError(err)
				if cat == CategoryResourceExhaustion {
					return CategoryResourceExhaustion
				}
				return CategoryUnknown
			},
			Priority: 5,
		},
		{
			Name: "fallback",
			Classify: func(err error) ErrorCategory {
				return CategoryRecoverable
			},
			Priority: 0,
		},
	}

	for _, classifier := range classifiers {
		er.classifiers = append(er.classifiers, classifier)
	}
}

func (er *ErrorRecovery) classifyError(err error) ErrorCategory {
	for _, classifier := range er.classifiers {
		category := classifier.Classify(err)
		if category != CategoryUnknown {
			return category
		}
	}
	return CategoryUnknown
}

func (er *ErrorRecovery) findStrategy(err error) RecoveryStrategy {
	for _, strategy := range er.strategies {
		if strategy.CanRecover(err) {
			return strategy
		}
	}
	return nil
}

func (er *ErrorRecovery) categoryName(category ErrorCategory) string {
	switch category {
	case CategoryRetryable:
		return "retryable"
	case CategoryNonRetryable:
		return "non_retryable"
	case CategoryRateLimit:
		return "rate_limit"
	case CategoryResourceExhaustion:
		return "resource_exhaustion"
	case CategoryNetwork:
		return "network"
	case CategoryTimeout:
		return "timeout"
	case CategoryCircuitOpen:
		return "circuit_open"
	case CategoryValidation:
		return "validation"
	case CategoryConfiguration:
		return "configuration"
	case CategoryRecoverable:
		return "recoverable"
	default:
		return "unknown"
	}
}

func getDefaultRecoveryConfig() *RecoveryConfig {
	return &RecoveryConfig{
		Strategies: make(map[string]RecoveryStrategy),
		MaxRetries: 3,
		Timeout:    30 * time.Second,
		LogLevel:   zap.InfoLevel,
	}
}

// Recovery context and coordination

// RecoveryContext provides context for recovery operations
type RecoveryContext struct {
	Error       error
	Category    ErrorCategory
	Attempt     int
	MaxAttempts int
	Strategy    string
	StartTime   time.Time
	Timeout     time.Duration
	CircuitOpen bool
}

// GetRecoveryContext returns recovery context for the current operation
func (er *ErrorRecovery) GetRecoveryContext(err error, attempt int) RecoveryContext {
	category := er.classifyError(err)
	strategy := "none"
	if s := er.findStrategy(err); s != nil {
		strategy = s.Name()
	}

	return RecoveryContext{
		Error:       err,
		Category:    category,
		Attempt:     attempt,
		MaxAttempts: er.config.MaxRetries,
		Strategy:    strategy,
		StartTime:   time.Now(),
		Timeout:     er.config.Timeout,
		CircuitOpen: false, // This would be passed in from the breaker
	}
}
