package pool

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"go.uber.org/zap"
)

type mockConn struct {
	net.Conn
	closed      bool
	writeErr    error
	readErr     error
	deadlineErr error
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	return len(b), nil
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	if m.readErr != nil {
		return 0, m.readErr
	}
	return len(b), nil
}

func (m *mockConn) SetDeadline(t time.Time) error {
	return m.deadlineErr
}

func TestPoolBasicOperations(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	factory := func(ctx context.Context) (net.Conn, error) {
		return &mockConn{}, nil
	}

	cfg := &Config{
		IdleTimeout:   time.Minute,
		MaxIdle:       2,
		MaxActive:     4,
		RetryInterval: time.Second,
		MaxRetries:    2,
	}

	p := NewPool(factory, cfg, logger)
	defer p.Close()

	// Test Get
	conn1, err := p.Get(context.Background())
	if err != nil {
		t.Fatalf("Failed to get connection: %v", err)
	}

	// Test Stats
	stats := p.Stats()
	if stats.ActiveCount != 1 {
		t.Errorf("Expected 1 active connection, got %d", stats.ActiveCount)
	}

	// Test Put
	if err := p.Put(conn1); err != nil {
		t.Fatalf("Failed to put connection: %v", err)
	}

	stats = p.Stats()
	if stats.IdleCount != 1 {
		t.Errorf("Expected 1 idle connection, got %d", stats.IdleCount)
	}
}

func TestPoolMaxActive(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	factory := func(ctx context.Context) (net.Conn, error) {
		return &mockConn{}, nil
	}

	cfg := &Config{
		MaxActive: 2,
	}

	p := NewPool(factory, cfg, logger)
	defer p.Close()

	// Get up to max active
	conn1, _ := p.Get(context.Background())
	conn2, _ := p.Get(context.Background())

	// Should fail to get another
	_, err := p.Get(context.Background())
	if err != ErrPoolExhausted {
		t.Errorf("Expected pool exhausted error, got %v", err)
	}

	// Return one connection
	p.Put(conn1)

	// Should be able to get another now
	conn3, err := p.Get(context.Background())
	if err != nil {
		t.Errorf("Failed to get connection after put: %v", err)
	}

	p.Put(conn2)
	p.Put(conn3)
}

func TestPoolHealthCheck(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	factory := func(ctx context.Context) (net.Conn, error) {
		return &mockConn{}, nil
	}

	cfg := &Config{
		IdleTimeout: 50 * time.Millisecond,
	}

	p := NewPool(factory, cfg, logger)
	defer p.Close()

	// Get and return a connection
	conn, _ := p.Get(context.Background())
	p.Put(conn)

	// Wait for idle timeout
	time.Sleep(100 * time.Millisecond)

	// Connection should be removed from idle pool
	stats := p.Stats()
	if stats.IdleCount != 0 {
		t.Errorf("Expected 0 idle connections after timeout, got %d", stats.IdleCount)
	}
}

func TestPoolFailedConnections(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	failCount := 0
	factory := func(ctx context.Context) (net.Conn, error) {
		failCount++
		if failCount <= 2 {
			return nil, errors.New("connection failed")
		}
		return &mockConn{}, nil
	}

	cfg := &Config{
		RetryInterval: time.Millisecond,
		MaxRetries:    3,
	}

	p := NewPool(factory, cfg, logger)
	defer p.Close()

	// Should eventually succeed after retries
	conn, err := p.Get(context.Background())
	if err != nil {
		t.Fatalf("Failed to get connection after retries: %v", err)
	}

	if failCount != 3 {
		t.Errorf("Expected 3 failures before success, got %d", failCount)
	}

	p.Put(conn)
}

func TestPoolClose(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	factory := func(ctx context.Context) (net.Conn, error) {
		return &mockConn{}, nil
	}

	p := NewPool(factory, nil, logger)

	// Get some connections
	conn1, _ := p.Get(context.Background())
	conn2, _ := p.Get(context.Background())
	p.Put(conn1) // Return one to idle pool

	// Close pool
	p.Close()

	// Verify all connections are closed
	mc1 := conn1.(*poolConn).Conn.(*mockConn)
	mc2 := conn2.(*poolConn).Conn.(*mockConn)

	if !mc1.closed {
		t.Error("Idle connection not closed")
	}
	if !mc2.closed {
		t.Error("Active connection not closed")
	}

	// Verify pool is closed
	_, err := p.Get(context.Background())
	if err != ErrPoolClosed {
		t.Errorf("Expected pool closed error, got %v", err)
	}
}

func TestPoolConnectionFailure(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	factory := func(ctx context.Context) (net.Conn, error) {
		return &mockConn{
			writeErr: errors.New("write failed"),
		}, nil
	}

	p := NewPool(factory, nil, logger)
	defer p.Close()

	// Get connection
	conn, err := p.Get(context.Background())
	if err != nil {
		t.Fatalf("Failed to get connection: %v", err)
	}

	// Put it back - should be discarded due to write error
	p.Put(conn)

	stats := p.Stats()
	if stats.IdleCount != 0 {
		t.Errorf("Expected 0 idle connections after failure, got %d", stats.IdleCount)
	}
}

func TestPoolConcurrency(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	factory := func(ctx context.Context) (net.Conn, error) {
		return &mockConn{}, nil
	}

	cfg := &Config{
		MaxActive: 10,
		MaxIdle:   5,
	}

	p := NewPool(factory, cfg, logger)
	defer p.Close()

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				conn, err := p.Get(context.Background())
				if err != nil {
					t.Errorf("Failed to get connection: %v", err)
					return
				}
				time.Sleep(time.Millisecond)
				p.Put(conn)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify pool state
	stats := p.Stats()
	if stats.ActiveCount != 0 {
		t.Errorf("Expected 0 active connections, got %d", stats.ActiveCount)
	}
	if stats.IdleCount > cfg.MaxIdle {
		t.Errorf("Expected <= %d idle connections, got %d", cfg.MaxIdle, stats.IdleCount)
	}
}
