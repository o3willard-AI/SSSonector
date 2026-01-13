package resilience

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

var (
	ErrCircuitOpen = errors.New("circuit breaker is open")
)

// CircuitBreakerState represents the current state of the circuit breaker
type CircuitBreakerState int32

const (
	StateClosed CircuitBreakerState = iota
	StateHalfOpen
	StateOpen
)

// CircuitBreakerConfig represents circuit breaker configuration
type CircuitBreakerConfig struct {
	Name                string
	FailureThreshold    float64 // Percentage threshold (0.0-1.0)
	RecoveryTimeout     time.Duration
	SuccessThreshold    int // Successes needed to close half-open circuit
	RequestTimeout      time.Duration
	MinRequests         int // Minimum requests before checking failure rate
	StateChangeCallback func(string, CircuitBreakerState, CircuitBreakerState)
	ErrorClassifier     func(error) bool // Return true if error is non-retryable
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	config *CircuitBreakerConfig
	logger *zap.Logger

	state               int32  // Current state (use atomic operations)
	requests            uint64 // Total requests
	successes           uint64 // Total successes
	failures            uint64 // Total failures
	lastStateTransition time.Time

	// Half-open state tracking
	halfOpenRequests  int // Requests made in half-open state
	halfOpenSuccesses int // Successful requests in half-open state

	mu sync.RWMutex
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config *CircuitBreakerConfig, logger *zap.Logger) *CircuitBreaker {
	if config == nil {
		config = getDefaultCircuitBreakerConfig()
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	cb := &CircuitBreaker{
		config:              config,
		logger:              logger,
		lastStateTransition: time.Now(),
		state:               int32(StateClosed),
	}

	cb.logger.Info("Circuit breaker initialized",
		zap.String("name", config.Name),
		zap.Float64("failure_threshold", config.FailureThreshold),
		zap.Duration("recovery_timeout", config.RecoveryTimeout))

	return cb
}

// getDefaultCircuitBreakerConfig returns default configuration
func getDefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		Name:             "default",
		FailureThreshold: 0.5, // 50% failure rate
		RecoveryTimeout:  60 * time.Second,
		SuccessThreshold: 3, // 3 successes to close
		RequestTimeout:   5 * time.Second,
		MinRequests:      10,                               // Minimum requests before checking
		ErrorClassifier:  func(error) bool { return true }, // All errors are retryable
	}
}

// Call executes the given function with circuit breaker protection
func (cb *CircuitBreaker) Call(ctx context.Context, fn func(ctx context.Context) error) error {
	if cb.shouldFailFast() {
		atomic.AddUint64(&cb.requests, 1)
		cb.logger.Warn("Circuit breaker fast-failed request",
			zap.String("name", cb.config.Name),
			zap.String("state", cb.getStateString()))
		return ErrCircuitOpen
	}

	atomic.AddUint64(&cb.requests, 1)

	// Add timeout to context if configured
	callCtx := ctx
	var cancel context.CancelFunc
	if cb.config.RequestTimeout > 0 {
		callCtx, cancel = context.WithTimeout(ctx, cb.config.RequestTimeout)
		defer cancel()
	}

	// Execute function
	err := fn(callCtx)

	// Update state based on result
	cb.updateState(err)

	if err != nil {
		atomic.AddUint64(&cb.failures, 1)
		cb.logger.Debug("Circuit breaker request failed",
			zap.String("name", cb.config.Name),
			zap.Error(err))
	} else {
		atomic.AddUint64(&cb.successes, 1)
	}

	return err
}

// CanExecute checks if execution should be attempted without actually running
func (cb *CircuitBreaker) CanExecute() bool {
	if cb.shouldFailFast() {
		atomic.AddUint64(&cb.requests, 1)
		return false
	}
	atomic.AddUint64(&cb.requests, 1)
	return true
}

// GetState returns the current circuit breaker state
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	return CircuitBreakerState(atomic.LoadInt32(&cb.state))
}

// GetStats returns current circuit breaker statistics
func (cb *CircuitBreaker) GetStats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return CircuitBreakerStats{
		State:               cb.GetState(),
		TotalRequests:       atomic.LoadUint64(&cb.requests),
		TotalSuccesses:      atomic.LoadUint64(&cb.successes),
		TotalFailures:       atomic.LoadUint64(&cb.failures),
		Buckets:             cb.halfOpenRequests, // Simplified
		SuccessBuckets:      cb.halfOpenSuccesses,
		LastStateTransition: cb.lastStateTransition,
	}
}

// CircuitBreakerStats represents circuit breaker statistics
type CircuitBreakerStats struct {
	State               CircuitBreakerState
	TotalRequests       uint64
	TotalSuccesses      uint64
	TotalFailures       uint64
	Buckets             int
	SuccessBuckets      int
	LastStateTransition time.Time
}

// GetFailureRate returns the current failure rate
func (cb *CircuitBreaker) GetFailureRate() float64 {
	totalRequests := atomic.LoadUint64(&cb.requests)
	totalFailures := atomic.LoadUint64(&cb.failures)

	if totalRequests == 0 {
		return 0.0
	}

	return float64(totalFailures) / float64(totalRequests)
}

// shouldFailFast returns true if requests should fail fast
func (cb *CircuitBreaker) shouldFailFast() bool {
	state := cb.GetState()

	switch state {
	case StateClosed:
		// Check if we should open based on failure rate
		if atomic.LoadUint64(&cb.requests) >= uint64(cb.config.MinRequests) {
			if cb.GetFailureRate() > cb.config.FailureThreshold {
				cb.transitionToState(StateOpen)
				return true
			}
		}
		return false

	case StateOpen:
		// Check if recovery timeout has expired
		if time.Since(cb.lastStateTransition) >= cb.config.RecoveryTimeout {
			cb.transitionToState(StateHalfOpen)
			return false // Allow one request to test
		}
		return true // Still in open state

	case StateHalfOpen:
		return false // Allow requests in half-open state

	default:
		return true
	}
}

// updateState updates the circuit breaker state based on operation result
func (cb *CircuitBreaker) updateState(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state := cb.GetState()
	isSuccess := err == nil || (cb.config.ErrorClassifier != nil && !cb.config.ErrorClassifier(err))

	switch state {
	case StateHalfOpen:
		cb.halfOpenRequests++
		if isSuccess {
			cb.halfOpenSuccesses++
		}

		// Check if we've had enough successes to close the circuit
		if cb.halfOpenSuccesses >= cb.config.SuccessThreshold {
			cb.transitionToState(StateClosed)
		}
	case StateClosed, StateOpen:
		// Standard operation - monitoring for failure rate handled in shouldFailFast
	}
}

// transitionToState changes the circuit breaker state
func (cb *CircuitBreaker) transitionToState(newState CircuitBreakerState) {
	oldState := CircuitBreakerState(atomic.LoadInt32(&cb.state))
	atomic.StoreInt32(&cb.state, int32(newState))

	cb.lastStateTransition = time.Now()

	// Reset half-open counters if transitioning to half-open
	if newState == StateHalfOpen {
		cb.halfOpenRequests = 0
		cb.halfOpenSuccesses = 0
	}

	cb.logger.Info("Circuit breaker state transition",
		zap.String("name", cb.config.Name),
		zap.String("from", cb.stateString(oldState)),
		zap.String("to", cb.stateString(newState)),
		zap.Float64("failure_rate", cb.GetFailureRate()))

	if cb.config.StateChangeCallback != nil {
		cb.config.StateChangeCallback(cb.config.Name, oldState, newState)
	}
}

// getStateString returns a string representation of the state
func (cb *CircuitBreaker) getStateString() string {
	return cb.stateString(cb.GetState())
}

// stateString converts state to string
func (cb *CircuitBreaker) stateString(state CircuitBreakerState) string {
	switch state {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	atomic.StoreInt32(&cb.state, int32(StateClosed))
	atomic.StoreUint64(&cb.requests, 0)
	atomic.StoreUint64(&cb.successes, 0)
	atomic.StoreUint64(&cb.failures, 0)
	cb.halfOpenRequests = 0
	cb.halfOpenSuccesses = 0
	cb.lastStateTransition = time.Now()

	cb.logger.Info("Circuit breaker reset",
		zap.String("name", cb.config.Name))
}

// SetStateChangeCallback sets the callback for state changes
func (cb *CircuitBreaker) SetStateChangeCallback(callback func(string, CircuitBreakerState, CircuitBreakerState)) {
	cb.config.StateChangeCallback = callback
}

// SetThreshold updates the failure threshold
func (cb *CircuitBreaker) SetThreshold(threshold float64) {
	cb.config.FailureThreshold = threshold
	cb.logger.Info("Circuit breaker threshold updated",
		zap.String("name", cb.config.Name),
		zap.Float64("threshold", threshold))
}

// IsOpen returns true if the circuit breaker is currently open
func (cb *CircuitBreaker) IsOpen() bool {
	return cb.GetState() == StateOpen
}

// IsHalfOpen returns true if the circuit breaker is currently half-open
func (cb *CircuitBreaker) IsHalfOpen() bool {
	return cb.GetState() == StateHalfOpen
}

// IsClosed returns true if the circuit breaker is currently closed
func (cb *CircuitBreaker) IsClosed() bool {
	return cb.GetState() == StateClosed
}

// ForceOpen forces the circuit breaker to open state
func (cb *CircuitBreaker) ForceOpen() {
	cb.transitionToState(StateOpen)
}

// ForceClose forces the circuit breaker to closed state
func (cb *CircuitBreaker) ForceClose() {
	cb.transitionToState(StateClosed)
}
