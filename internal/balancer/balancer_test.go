package balancer

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

type mockDialer struct {
	shouldFail bool
	conn       net.Conn
}

func (d *mockDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if d.shouldFail {
		return nil, errors.New("connection failed")
	}
	return d.conn, nil
}

type mockConn struct {
	net.Conn
	closed bool
}

func (c *mockConn) Close() error {
	c.closed = true
	return nil
}

func TestBalancer(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	t.Run("round robin strategy", func(t *testing.T) {
		cfg := &Config{
			Strategy:           RoundRobin,
			HealthCheck:        time.Second,
			RetryTimeout:       time.Second,
			MaxRetries:         3,
			HealthyThreshold:   2,
			UnhealthyThreshold: 3,
		}

		b := NewBalancer(logger, cfg)
		defer b.Stop()

		// Add endpoints
		b.AddEndpoint("localhost:8001", 1)
		b.AddEndpoint("localhost:8002", 1)
		b.AddEndpoint("localhost:8003", 1)

		// Test round robin distribution
		ep1, _ := b.Next()
		ep2, _ := b.Next()
		ep3, _ := b.Next()
		ep4, _ := b.Next()

		assert.Equal(t, "localhost:8001", ep1.Address)
		assert.Equal(t, "localhost:8002", ep2.Address)
		assert.Equal(t, "localhost:8003", ep3.Address)
		assert.Equal(t, "localhost:8001", ep4.Address)
	})

	t.Run("least connections strategy", func(t *testing.T) {
		cfg := &Config{
			Strategy:           LeastConnections,
			HealthCheck:        time.Second,
			RetryTimeout:       time.Second,
			MaxRetries:         3,
			HealthyThreshold:   2,
			UnhealthyThreshold: 3,
		}

		b := NewBalancer(logger, cfg)
		defer b.Stop()

		// Add endpoints
		b.AddEndpoint("localhost:8001", 1)
		b.AddEndpoint("localhost:8002", 1)

		// Simulate connections on first endpoint
		ep1, _ := b.Next()
		ep1.ActiveConns = 5

		// Next should choose second endpoint
		ep2, _ := b.Next()
		assert.Equal(t, "localhost:8002", ep2.Address)
	})

	t.Run("weighted round robin strategy", func(t *testing.T) {
		cfg := &Config{
			Strategy:           WeightedRoundRobin,
			HealthCheck:        time.Second,
			RetryTimeout:       time.Second,
			MaxRetries:         3,
			HealthyThreshold:   2,
			UnhealthyThreshold: 3,
		}

		b := NewBalancer(logger, cfg)
		defer b.Stop()

		// Add endpoints with different weights
		b.AddEndpoint("localhost:8001", 2) // Should get twice as many connections
		b.AddEndpoint("localhost:8002", 1)

		// Count distribution
		counts := make(map[string]int)
		for i := 0; i < 30; i++ {
			ep, _ := b.Next()
			counts[ep.Address]++
		}

		// First endpoint should get roughly twice as many connections
		assert.Greater(t, counts["localhost:8001"], counts["localhost:8002"])
	})

	t.Run("connection handling", func(t *testing.T) {
		cfg := &Config{
			Strategy:           RoundRobin,
			HealthCheck:        time.Second,
			RetryTimeout:       100 * time.Millisecond,
			MaxRetries:         2,
			HealthyThreshold:   2,
			UnhealthyThreshold: 3,
		}

		b := NewBalancer(logger, cfg)
		defer b.Stop()

		b.AddEndpoint("localhost:8001", 1)

		// Test successful connection
		mockConn := &mockConn{}
		b.dialer = &mockDialer{conn: mockConn}

		conn, err := b.Connect(context.Background())
		assert.NoError(t, err)
		assert.NotNil(t, conn)

		// Test connection failure with retry
		b.dialer = &mockDialer{shouldFail: true}

		_, err = b.Connect(context.Background())
		assert.Error(t, err)

		// Check endpoint stats
		stats := b.GetStats()
		assert.Equal(t, 1, len(stats))
		assert.Equal(t, int64(1), stats[0].TotalConns)
		assert.NotNil(t, stats[0].LastError)
	})

	t.Run("endpoint management", func(t *testing.T) {
		cfg := &Config{
			Strategy:           RoundRobin,
			HealthCheck:        time.Second,
			RetryTimeout:       time.Second,
			MaxRetries:         3,
			HealthyThreshold:   2,
			UnhealthyThreshold: 3,
		}

		b := NewBalancer(logger, cfg)
		defer b.Stop()

		// Add and remove endpoints
		b.AddEndpoint("localhost:8001", 1)
		b.AddEndpoint("localhost:8002", 1)
		assert.Equal(t, 2, len(b.endpoints))

		b.RemoveEndpoint("localhost:8001")
		assert.Equal(t, 1, len(b.endpoints))
		assert.Equal(t, "localhost:8002", b.endpoints[0].Address)
	})

	t.Run("connection release", func(t *testing.T) {
		cfg := &Config{
			Strategy:           RoundRobin,
			HealthCheck:        time.Second,
			RetryTimeout:       time.Second,
			MaxRetries:         3,
			HealthyThreshold:   2,
			UnhealthyThreshold: 3,
		}

		b := NewBalancer(logger, cfg)
		defer b.Stop()

		b.AddEndpoint("localhost:8001", 1)
		ep := b.endpoints[0]
		ep.ActiveConns = 1

		b.ReleaseConn("localhost:8001")
		assert.Equal(t, int64(0), ep.ActiveConns)
	})

	t.Run("health check", func(t *testing.T) {
		cfg := &Config{
			Strategy:           RoundRobin,
			HealthCheck:        100 * time.Millisecond,
			RetryTimeout:       time.Second,
			MaxRetries:         3,
			HealthyThreshold:   2,
			UnhealthyThreshold: 3,
		}

		b := NewBalancer(logger, cfg)
		defer b.Stop()

		b.AddEndpoint("localhost:8001", 1)

		// Start with failing health checks
		b.dialer = &mockDialer{shouldFail: true}
		time.Sleep(250 * time.Millisecond) // Wait for health checks

		stats := b.GetStats()
		assert.Greater(t, stats[0].FailureStreak, int64(0))

		// Switch to successful health checks
		b.dialer = &mockDialer{conn: &mockConn{}}
		time.Sleep(250 * time.Millisecond) // Wait for health checks

		stats = b.GetStats()
		assert.Greater(t, stats[0].SuccessStreak, int64(0))
		assert.Equal(t, int64(0), stats[0].FailureStreak)
	})

	t.Run("no endpoints", func(t *testing.T) {
		cfg := &Config{
			Strategy:           RoundRobin,
			HealthCheck:        time.Second,
			RetryTimeout:       time.Second,
			MaxRetries:         3,
			HealthyThreshold:   2,
			UnhealthyThreshold: 3,
		}

		b := NewBalancer(logger, cfg)
		defer b.Stop()

		_, err := b.Next()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no endpoints available")
	})
}
