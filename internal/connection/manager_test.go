package connection

import (
	"net"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/breaker"
	"github.com/o3willard-AI/SSSonector/internal/ratelimit"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestManager(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := &Config{
		MaxConnections: 2,
		KeepAlive:      true,
		KeepAliveIdle:  time.Second,
		RetryAttempts:  3,
		RetryInterval:  time.Second,
		ConnectTimeout: 5 * time.Second,
	}

	t.Run("basic connection management", func(t *testing.T) {
		mgr := NewManager(logger, cfg)
		conn := SetupMockConn()

		err := mgr.Accept(conn)
		assert.NoError(t, err)
		assert.Equal(t, 1, mgr.GetConnectionCount())
		assert.Equal(t, ManagerStateConnected, mgr.GetState())

		mgr.Remove(conn)
		assert.Equal(t, 0, mgr.GetConnectionCount())
		assert.Equal(t, ManagerStateDisconnected, mgr.GetState())
		assert.True(t, conn.closed)
	})

	t.Run("circuit breaker", func(t *testing.T) {
		cfg := &Config{
			MaxConnections: 10,
			CircuitBreaker: &breaker.Config{
				MaxFailures:      2,
				ResetTimeout:     100 * time.Millisecond,
				HalfOpenMaxCalls: 2,
				FailureWindow:    time.Minute,
			},
		}
		mgr := NewManager(logger, cfg)

		// Set up failing dialer
		failingDialer := &MockDialer{ShouldFail: true}
		mgr.SetDialer(failingDialer)

		// First failure
		_, err := mgr.Connect("tcp", "localhost:8080")
		assert.Error(t, err)

		// Second failure should trip the circuit breaker
		_, err = mgr.Connect("tcp", "localhost:8080")
		assert.Error(t, err)

		// Next attempt should be rejected by circuit breaker
		_, err = mgr.Connect("tcp", "localhost:8080")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "circuit breaker is open")

		// Get circuit breaker stats
		stats := mgr.GetCircuitBreakerStats()
		assert.NotNil(t, stats)
		assert.Equal(t, "open", stats.State)
		assert.Equal(t, 2, stats.Failures)

		// Wait for reset timeout
		time.Sleep(200 * time.Millisecond)

		// Set up working dialer
		workingDialer := &MockDialer{
			ShouldFail: false,
			Conn:       SetupMockConn(),
		}
		mgr.SetDialer(workingDialer)

		// Should transition to half-open and then closed after successful calls
		conn, err := mgr.Connect("tcp", "localhost:8080")
		assert.NoError(t, err)
		assert.NotNil(t, conn)

		stats = mgr.GetCircuitBreakerStats()
		assert.Equal(t, "half-open", stats.State)

		// Second successful call
		conn, err = mgr.Connect("tcp", "localhost:8080")
		assert.NoError(t, err)
		assert.NotNil(t, conn)

		stats = mgr.GetCircuitBreakerStats()
		assert.Equal(t, "closed", stats.State)
	})

	t.Run("rate limiting", func(t *testing.T) {
		cfg := &Config{
			MaxConnections: 10,
			RateLimit: &ratelimit.Config{
				DefaultRate:  2,
				DefaultBurst: 1,
				CleanupTime:  time.Hour,
			},
		}
		mgr := NewManager(logger, cfg)
		defer mgr.Stop()

		// First connection should be allowed
		conn1 := SetupMockConn()
		err := mgr.Accept(conn1)
		assert.NoError(t, err)

		// Second connection should be rate limited
		conn2 := SetupMockConn()
		err = mgr.Accept(conn2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rate limit exceeded")

		// Wait for token replenishment
		time.Sleep(time.Second)

		// Third connection should be allowed after waiting
		conn3 := SetupMockConn()
		err = mgr.Accept(conn3)
		assert.NoError(t, err)

		// Update stats to trigger rate adjustment
		mgr.UpdateStats(conn1.RemoteAddr().String(), 11*1024*1024, 0) // Over 10MB
		metrics := mgr.GetRateLimitMetrics()
		assert.NotNil(t, metrics)
		assert.Equal(t, 2, metrics.ActiveBuckets)
	})

	t.Run("connection callbacks", func(t *testing.T) {
		mgr := NewManager(logger, cfg)
		conn := SetupMockConn()

		var (
			connectCalled    bool
			disconnectCalled bool
		)

		mgr.SetCallbacks(
			func(c net.Conn) { connectCalled = true },
			func(c net.Conn, err error) { disconnectCalled = true },
		)

		err := mgr.Accept(conn)
		assert.NoError(t, err)
		assert.True(t, connectCalled)

		mgr.Remove(conn)
		assert.True(t, disconnectCalled)
	})

	t.Run("max connections", func(t *testing.T) {
		mgr := NewManager(logger, cfg)

		// Add first connection
		conn1 := SetupMockConn()
		err := mgr.Accept(conn1)
		assert.NoError(t, err)

		// Add second connection
		conn2 := SetupMockConn()
		err = mgr.Accept(conn2)
		assert.NoError(t, err)

		// Try to add third connection
		conn3 := SetupMockConn()
		err = mgr.Accept(conn3)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "maximum connections reached")
	})

	t.Run("connection stats", func(t *testing.T) {
		mgr := NewManager(logger, cfg)
		conn := SetupMockConn()

		err := mgr.Accept(conn)
		assert.NoError(t, err)

		mgr.UpdateStats(conn.RemoteAddr().String(), 100, 200)

		conns := mgr.GetConnections()
		assert.Equal(t, 1, len(conns))
		assert.Equal(t, int64(100), conns[0].BytesSent)
		assert.Equal(t, int64(200), conns[0].BytesReceived)
	})

	t.Run("rate limit cleanup", func(t *testing.T) {
		cfg := &Config{
			MaxConnections: 10,
			RateLimit: &ratelimit.Config{
				DefaultRate:  10,
				DefaultBurst: 5,
				CleanupTime:  100 * time.Millisecond,
			},
		}
		mgr := NewManager(logger, cfg)
		defer mgr.Stop()

		// Add some connections
		conn1 := SetupMockConn()
		conn2 := SetupMockConn()
		mgr.Accept(conn1)
		mgr.Accept(conn2)

		// Wait for cleanup cycle
		time.Sleep(200 * time.Millisecond)

		// Remove connections
		mgr.Remove(conn1)
		mgr.Remove(conn2)

		// Wait for another cleanup cycle
		time.Sleep(200 * time.Millisecond)

		metrics := mgr.GetRateLimitMetrics()
		assert.Equal(t, 0, metrics.ActiveBuckets)
	})
}
