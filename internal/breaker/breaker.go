package breaker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// State represents the state of the circuit breaker
type State int

const (
	// StateClosed represents a closed (normal) state
	StateClosed State = iota
	// StateHalfOpen represents a half-open state (testing the waters)
	StateHalfOpen
	// StateOpen represents an open (failing) state
	StateOpen
)

func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateHalfOpen:
		return "half-open"
	case StateOpen:
		return "open"
	default:
		return "unknown"
	}
}

// Config holds circuit breaker configuration
type Config struct {
	MaxFailures      int           // Maximum number of failures before opening
	ResetTimeout     time.Duration // Time to wait before attempting reset
	HalfOpenMaxCalls int           // Maximum number of calls in half-open state
	FailureWindow    time.Duration // Time window for counting failures
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	mu             sync.RWMutex
	config         *Config
	logger         *zap.Logger
	state          State
	failures       int
	lastFailure    time.Time
	openTime       time.Time
	halfOpenCalls  int
	halfOpenFailed bool
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(cfg *Config, logger *zap.Logger) *CircuitBreaker {
	return &CircuitBreaker{
		config: cfg,
		logger: logger,
		state:  StateClosed,
	}
}

// Execute executes a function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() error) error {
	if err := cb.beforeExecute(); err != nil {
		return err
	}

	err := fn()
	cb.afterExecute(err)
	return err
}

// beforeExecute checks if execution is allowed
func (cb *CircuitBreaker) beforeExecute() error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return nil

	case StateHalfOpen:
		if cb.halfOpenCalls >= cb.config.HalfOpenMaxCalls {
			return fmt.Errorf("circuit breaker is half-open and at call limit")
		}
		cb.halfOpenCalls++
		return nil

	case StateOpen:
		if time.Since(cb.openTime) > cb.config.ResetTimeout {
			cb.logger.Info("circuit breaker transitioning from open to half-open")
			cb.toHalfOpen()
			return nil
		}
		return fmt.Errorf("circuit breaker is open")

	default:
		return fmt.Errorf("circuit breaker in unknown state")
	}
}

// afterExecute processes the execution result
func (cb *CircuitBreaker) afterExecute(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		switch cb.state {
		case StateClosed:
			cb.failures++
			cb.lastFailure = time.Now()

			// Clear old failures outside the window
			if time.Since(cb.lastFailure) > cb.config.FailureWindow {
				cb.failures = 1
			}

			if cb.failures >= cb.config.MaxFailures {
				cb.logger.Info("circuit breaker transitioning from closed to open",
					zap.Int("failures", cb.failures))
				cb.toOpen()
			}

		case StateHalfOpen:
			cb.logger.Info("circuit breaker transitioning from half-open to open due to failure")
			cb.halfOpenFailed = true
			cb.toOpen()
		}
	} else {
		switch cb.state {
		case StateHalfOpen:
			if cb.halfOpenCalls >= cb.config.HalfOpenMaxCalls && !cb.halfOpenFailed {
				cb.logger.Info("circuit breaker transitioning from half-open to closed")
				cb.toClosed()
			}
		case StateClosed:
			// Reset failures on success
			if time.Since(cb.lastFailure) > cb.config.FailureWindow {
				cb.failures = 0
			}
		}
	}
}

// toOpen transitions to open state
func (cb *CircuitBreaker) toOpen() {
	cb.state = StateOpen
	cb.openTime = time.Now()
}

// toHalfOpen transitions to half-open state
func (cb *CircuitBreaker) toHalfOpen() {
	cb.state = StateHalfOpen
	cb.halfOpenCalls = 0
	cb.halfOpenFailed = false
}

// toClosed transitions to closed state
func (cb *CircuitBreaker) toClosed() {
	cb.state = StateClosed
	cb.failures = 0
	cb.halfOpenCalls = 0
	cb.halfOpenFailed = false
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats holds circuit breaker statistics
type Stats struct {
	State          string
	Failures       int
	LastFailure    time.Time
	OpenTime       time.Time
	HalfOpenCalls  int
	HalfOpenFailed bool
}

// GetStats returns current statistics
func (cb *CircuitBreaker) GetStats() Stats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	return Stats{
		State:          cb.state.String(),
		Failures:       cb.failures,
		LastFailure:    cb.lastFailure,
		OpenTime:       cb.openTime,
		HalfOpenCalls:  cb.halfOpenCalls,
		HalfOpenFailed: cb.halfOpenFailed,
	}
}

// Reset forces the circuit breaker back to closed state
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.toClosed()
	cb.logger.Info("circuit breaker manually reset to closed state")
}
