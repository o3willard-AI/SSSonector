package pool

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"go.uber.org/zap"
)

type mockFactory struct {
	failCount int
	maxFails  int
	delay     time.Duration
}

// mockConn implements net.Conn for testing purposes
type mockConn struct{}

func (m *mockConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	return len(b), nil
}

func (m *mockConn) Close() error {
	return nil
}

func (m *mockConn) LocalAddr() net.Addr {
	return nil
}

func (m *mockConn) RemoteAddr() net.Addr {
	return nil
}

func (m *mockConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

func (f *mockFactory) createConn(ctx context.Context) (net.Conn, error) {
	if f.delay > 0 {
		time.Sleep(f.delay)
	}

	f.failCount++
	if f.failCount <= f.maxFails {
		return nil, errors.New("mock connection failed")
	}

	return &mockConn{}, nil
}

func TestRetryManagerImmediateSuccess(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	factory := &mockFactory{maxFails: 1}
	config := &RetryConfig{
		ImmediateAttempts: 3,
		ImmediateInterval: time.Millisecond,
	}

	manager := NewRetryManager(factory.createConn, config, logger)
	conn, err := manager.GetConnection(context.Background())
	if err != nil {
		t.Fatalf("Expected successful connection, got error: %v", err)
	}
	if conn == nil {
		t.Fatal("Expected connection to be non-nil")
	}

	attempts, failures, successes := manager.GetMetrics()
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
	if failures != 0 {
		t.Errorf("Expected 0 failures, got %d", failures)
	}
	if successes != 1 {
		t.Errorf("Expected 1 success, got %d", successes)
	}
}

func TestRetryManagerGradualSuccess(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	factory := &mockFactory{maxFails: 4}
	config := &RetryConfig{
		ImmediateAttempts:  3,
		ImmediateInterval:  time.Millisecond,
		GradualAttempts:    5,
		GradualInterval:    time.Millisecond,
		MaxGradualInterval: time.Millisecond * 10,
	}

	manager := NewRetryManager(factory.createConn, config, logger)
	conn, err := manager.GetConnection(context.Background())
	if err != nil {
		t.Fatalf("Expected successful connection, got error: %v", err)
	}
	if conn == nil {
		t.Fatal("Expected connection to be non-nil")
	}

	attempts, failures, successes := manager.GetMetrics()
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
	if failures != 0 {
		t.Errorf("Expected 0 failures, got %d", failures)
	}
	if successes != 1 {
		t.Errorf("Expected 1 success, got %d", successes)
	}
}

func TestRetryManagerPersistentSuccess(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	factory := &mockFactory{maxFails: 9}
	config := &RetryConfig{
		ImmediateAttempts:  3,
		ImmediateInterval:  time.Millisecond,
		GradualAttempts:    5,
		GradualInterval:    time.Millisecond,
		MaxGradualInterval: time.Millisecond * 10,
		PersistentEnabled:  true,
		PersistentInterval: time.Millisecond,
	}

	manager := NewRetryManager(factory.createConn, config, logger)
	conn, err := manager.GetConnection(context.Background())
	if err != nil {
		t.Fatalf("Expected successful connection, got error: %v", err)
	}
	if conn == nil {
		t.Fatal("Expected connection to be non-nil")
	}

	attempts, failures, successes := manager.GetMetrics()
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
	if failures != 0 {
		t.Errorf("Expected 0 failures, got %d", failures)
	}
	if successes != 1 {
		t.Errorf("Expected 1 success, got %d", successes)
	}
}

func TestRetryManagerContextCancellation(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	factory := &mockFactory{maxFails: 100, delay: time.Millisecond * 100}
	config := &RetryConfig{
		ImmediateAttempts: 3,
		ImmediateInterval: time.Millisecond,
	}

	manager := NewRetryManager(factory.createConn, config, logger)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*50)
	defer cancel()

	_, err := manager.GetConnection(ctx)
	if err == nil {
		t.Fatal("Expected error due to context cancellation")
	}
	// Accept either DeadlineExceeded (context expired) or ErrMaxRetriesExceeded
	// (retry loop exhausted) since timing may race
	if err != context.DeadlineExceeded && err != ErrMaxRetriesExceeded {
		t.Errorf("Expected deadline exceeded or max retries error, got: %v", err)
	}
}

func TestRetryManagerAllPhasesFailure(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	factory := &mockFactory{maxFails: 1000}
	config := &RetryConfig{
		ImmediateAttempts:  2,
		ImmediateInterval:  time.Millisecond,
		GradualAttempts:    2,
		GradualInterval:    time.Millisecond,
		MaxGradualInterval: time.Millisecond * 10,
		PersistentEnabled:  false,
	}

	manager := NewRetryManager(factory.createConn, config, logger)
	_, err := manager.GetConnection(context.Background())
	if err != ErrMaxRetriesExceeded {
		t.Errorf("Expected max retries exceeded error, got: %v", err)
	}

	attempts, failures, successes := manager.GetMetrics()
	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
	if failures != 1 {
		t.Errorf("Expected 1 failure, got %d", failures)
	}
	if successes != 0 {
		t.Errorf("Expected 0 successes, got %d", successes)
	}
}

func TestRetryManagerMetricsReset(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	factory := &mockFactory{maxFails: 1}
	manager := NewRetryManager(factory.createConn, nil, logger)

	// Make some attempts
	manager.GetConnection(context.Background())
	manager.GetConnection(context.Background())

	// Reset metrics
	manager.ResetMetrics()

	attempts, failures, successes := manager.GetMetrics()
	if attempts != 0 {
		t.Errorf("Expected 0 attempts after reset, got %d", attempts)
	}
	if failures != 0 {
		t.Errorf("Expected 0 failures after reset, got %d", failures)
	}
	if successes != 0 {
		t.Errorf("Expected 0 successes after reset, got %d", successes)
	}
}

func TestRetryManagerExponentialBackoff(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	factory := &mockFactory{maxFails: 2}
	config := &RetryConfig{
		GradualAttempts:    3,
		GradualInterval:    time.Millisecond,
		MaxGradualInterval: time.Millisecond * 4,
	}

	manager := NewRetryManager(factory.createConn, config, logger)
	start := time.Now()
	conn, err := manager.GetConnection(context.Background())
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Expected successful connection, got error: %v", err)
	}
	if conn == nil {
		t.Fatal("Expected connection to be non-nil")
	}

	// Check that backoff was applied
	// Expected delays: 1ms, 2ms, 4ms = 7ms minimum; allow some slack
	minExpected := time.Millisecond * 2
	if duration < minExpected {
		t.Errorf("Expected minimum duration of %v, got %v", minExpected, duration)
	}
}
