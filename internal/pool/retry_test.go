package pool

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestRetryManagerImmediateSuccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	failCount := 0

	factory := func(ctx context.Context) (net.Conn, error) {
		if failCount < 1 {
			failCount++
			return nil, errors.New("mock connection failed")
		}
		return &mockConn{}, nil
	}

	manager := NewRetryManager(logger, factory)
	manager.WithConfig(RetryConfig{
		MaxImmediateRetries:  3,
		ImmediateDelay:       time.Millisecond,
		MaxGradualRetries:    0,
		MaxPersistentRetries: 0,
	})

	conn, err := manager.GetConnection(context.Background())
	require.NoError(t, err)
	require.NotNil(t, conn)

	success, failures := manager.GetMetrics()
	assert.Equal(t, uint64(1), success)
	assert.Equal(t, uint64(1), failures)
}

func TestRetryManagerGradualSuccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	failCount := 0

	factory := func(ctx context.Context) (net.Conn, error) {
		if failCount < 4 {
			failCount++
			return nil, errors.New("mock connection failed")
		}
		return &mockConn{}, nil
	}

	manager := NewRetryManager(logger, factory)
	manager.WithConfig(RetryConfig{
		MaxImmediateRetries:  3,
		ImmediateDelay:       time.Millisecond,
		MaxGradualRetries:    5,
		InitialInterval:      time.Millisecond,
		MaxInterval:          10 * time.Millisecond,
		BackoffMultiplier:    2.0,
		MaxPersistentRetries: 0,
	})

	conn, err := manager.GetConnection(context.Background())
	require.NoError(t, err)
	require.NotNil(t, conn)

	success, failures := manager.GetMetrics()
	assert.Equal(t, uint64(1), success)
	assert.Equal(t, uint64(4), failures)
}

func TestRetryManagerPersistentSuccess(t *testing.T) {
	logger := zaptest.NewLogger(t)
	failCount := 0

	factory := func(ctx context.Context) (net.Conn, error) {
		if failCount < 9 {
			failCount++
			return nil, errors.New("mock connection failed")
		}
		return &mockConn{}, nil
	}

	manager := NewRetryManager(logger, factory)
	manager.WithConfig(RetryConfig{
		MaxImmediateRetries:  3,
		ImmediateDelay:       time.Millisecond,
		MaxGradualRetries:    5,
		InitialInterval:      time.Millisecond,
		MaxInterval:          10 * time.Millisecond,
		BackoffMultiplier:    2.0,
		MaxPersistentRetries: 2,
		PersistentDelay:      5 * time.Millisecond,
	})

	conn, err := manager.GetConnection(context.Background())
	require.NoError(t, err)
	require.NotNil(t, conn)

	success, failures := manager.GetMetrics()
	assert.Equal(t, uint64(1), success)
	assert.Equal(t, uint64(9), failures)
}

func TestRetryManagerContextCancellation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	factory := func(ctx context.Context) (net.Conn, error) {
		time.Sleep(20 * time.Millisecond)
		return nil, errors.New("mock connection failed")
	}

	manager := NewRetryManager(logger, factory)
	_, err := manager.GetConnection(ctx)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestRetryManagerAllPhasesFailure(t *testing.T) {
	logger := zaptest.NewLogger(t)

	factory := func(ctx context.Context) (net.Conn, error) {
		return nil, errors.New("mock connection failed")
	}

	manager := NewRetryManager(logger, factory)
	manager.WithConfig(RetryConfig{
		MaxImmediateRetries:  2,
		ImmediateDelay:       time.Millisecond,
		MaxGradualRetries:    2,
		InitialInterval:      time.Millisecond,
		MaxInterval:          10 * time.Millisecond,
		BackoffMultiplier:    2.0,
		MaxPersistentRetries: 0,
	})

	_, err := manager.GetConnection(context.Background())
	assert.ErrorIs(t, err, ErrMaxRetriesExceeded)

	success, failures := manager.GetMetrics()
	assert.Equal(t, uint64(0), success)
	assert.Equal(t, uint64(4), failures)
}

func TestRetryManagerMetricsReset(t *testing.T) {
	logger := zaptest.NewLogger(t)

	factory := func(ctx context.Context) (net.Conn, error) {
		return &mockConn{}, nil
	}

	manager := NewRetryManager(logger, factory)
	conn, err := manager.GetConnection(context.Background())
	require.NoError(t, err)
	require.NotNil(t, conn)

	success, failures := manager.GetMetrics()
	assert.Equal(t, uint64(1), success)
	assert.Equal(t, uint64(0), failures)

	manager.ResetMetrics()

	success, failures = manager.GetMetrics()
	assert.Equal(t, uint64(0), success)
	assert.Equal(t, uint64(0), failures)
}

func TestRetryManagerExponentialBackoff(t *testing.T) {
	logger := zaptest.NewLogger(t)

	factory := func(ctx context.Context) (net.Conn, error) {
		return nil, errors.New("mock connection failed")
	}

	manager := NewRetryManager(logger, factory)
	manager.WithConfig(RetryConfig{
		MaxImmediateRetries: 0,
		MaxGradualRetries:   3,
		InitialInterval:     time.Millisecond,
		MaxInterval:         10 * time.Millisecond,
		BackoffMultiplier:   2.0,
	})

	start := time.Now()
	_, err := manager.GetConnection(context.Background())
	duration := time.Since(start)

	assert.ErrorIs(t, err, ErrMaxRetriesExceeded)
	assert.GreaterOrEqual(t, duration, 7*time.Millisecond) // 1ms + 2ms + 4ms
}
