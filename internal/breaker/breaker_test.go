package breaker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestCircuitBreaker(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	t.Run("basic operation", func(t *testing.T) {
		cb := NewCircuitBreaker(&Config{
			MaxFailures:      3,
			ResetTimeout:     time.Second,
			HalfOpenMaxCalls: 2,
			FailureWindow:    time.Minute,
		}, logger)

		// Should start closed
		assert.Equal(t, StateClosed, cb.GetState())

		// Successful execution
		err := cb.Execute(context.Background(), func() error {
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, StateClosed, cb.GetState())

		stats := cb.GetStats()
		assert.Equal(t, "closed", stats.State)
		assert.Equal(t, 0, stats.Failures)
	})

	t.Run("transition to open", func(t *testing.T) {
		cb := NewCircuitBreaker(&Config{
			MaxFailures:      2,
			ResetTimeout:     time.Second,
			HalfOpenMaxCalls: 2,
			FailureWindow:    time.Minute,
		}, logger)

		// First failure
		err := cb.Execute(context.Background(), func() error {
			return errors.New("test error")
		})
		assert.Error(t, err)
		assert.Equal(t, StateClosed, cb.GetState())

		// Second failure - should trip
		err = cb.Execute(context.Background(), func() error {
			return errors.New("test error")
		})
		assert.Error(t, err)
		assert.Equal(t, StateOpen, cb.GetState())

		// Should reject when open
		err = cb.Execute(context.Background(), func() error {
			return nil
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "circuit breaker is open")
	})

	t.Run("half open transition", func(t *testing.T) {
		cb := NewCircuitBreaker(&Config{
			MaxFailures:      2,
			ResetTimeout:     100 * time.Millisecond,
			HalfOpenMaxCalls: 2,
			FailureWindow:    time.Minute,
		}, logger)

		// Trip the breaker
		for i := 0; i < 2; i++ {
			cb.Execute(context.Background(), func() error {
				return errors.New("test error")
			})
		}
		assert.Equal(t, StateOpen, cb.GetState())

		// Wait for reset timeout
		time.Sleep(200 * time.Millisecond)

		// Should transition to half-open
		err := cb.Execute(context.Background(), func() error {
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, StateHalfOpen, cb.GetState())
	})

	t.Run("half open success", func(t *testing.T) {
		cb := NewCircuitBreaker(&Config{
			MaxFailures:      2,
			ResetTimeout:     100 * time.Millisecond,
			HalfOpenMaxCalls: 2,
			FailureWindow:    time.Minute,
		}, logger)

		// Trip the breaker
		for i := 0; i < 2; i++ {
			cb.Execute(context.Background(), func() error {
				return errors.New("test error")
			})
		}

		// Wait for reset timeout
		time.Sleep(200 * time.Millisecond)

		// Two successful calls in half-open state
		for i := 0; i < 2; i++ {
			err := cb.Execute(context.Background(), func() error {
				return nil
			})
			assert.NoError(t, err)
		}

		// Should transition back to closed
		assert.Equal(t, StateClosed, cb.GetState())
	})

	t.Run("half open failure", func(t *testing.T) {
		cb := NewCircuitBreaker(&Config{
			MaxFailures:      2,
			ResetTimeout:     100 * time.Millisecond,
			HalfOpenMaxCalls: 2,
			FailureWindow:    time.Minute,
		}, logger)

		// Trip the breaker
		for i := 0; i < 2; i++ {
			cb.Execute(context.Background(), func() error {
				return errors.New("test error")
			})
		}

		// Wait for reset timeout
		time.Sleep(200 * time.Millisecond)

		// Failure in half-open state
		err := cb.Execute(context.Background(), func() error {
			return errors.New("test error")
		})
		assert.Error(t, err)

		// Should transition back to open
		assert.Equal(t, StateOpen, cb.GetState())
	})

	t.Run("failure window", func(t *testing.T) {
		cb := NewCircuitBreaker(&Config{
			MaxFailures:      2,
			ResetTimeout:     time.Second,
			HalfOpenMaxCalls: 2,
			FailureWindow:    100 * time.Millisecond,
		}, logger)

		// First failure
		cb.Execute(context.Background(), func() error {
			return errors.New("test error")
		})

		// Wait for failure window
		time.Sleep(200 * time.Millisecond)

		// This failure should be counted as first in new window
		cb.Execute(context.Background(), func() error {
			return errors.New("test error")
		})

		// Should still be closed as failures were in different windows
		assert.Equal(t, StateClosed, cb.GetState())
	})

	t.Run("manual reset", func(t *testing.T) {
		cb := NewCircuitBreaker(&Config{
			MaxFailures:      2,
			ResetTimeout:     time.Second,
			HalfOpenMaxCalls: 2,
			FailureWindow:    time.Minute,
		}, logger)

		// Trip the breaker
		for i := 0; i < 2; i++ {
			cb.Execute(context.Background(), func() error {
				return errors.New("test error")
			})
		}
		assert.Equal(t, StateOpen, cb.GetState())

		// Manual reset
		cb.Reset()
		assert.Equal(t, StateClosed, cb.GetState())

		// Should allow new calls
		err := cb.Execute(context.Background(), func() error {
			return nil
		})
		assert.NoError(t, err)
	})

	t.Run("concurrent operations", func(t *testing.T) {
		cb := NewCircuitBreaker(&Config{
			MaxFailures:      100,
			ResetTimeout:     time.Second,
			HalfOpenMaxCalls: 2,
			FailureWindow:    time.Minute,
		}, logger)

		// Run concurrent operations
		done := make(chan bool)
		for i := 0; i < 10; i++ {
			go func() {
				for j := 0; j < 100; j++ {
					cb.Execute(context.Background(), func() error {
						return nil
					})
				}
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}

		// Should still be in a valid state
		assert.Equal(t, StateClosed, cb.GetState())
	})
}
