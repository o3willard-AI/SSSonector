package connection

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestTracker(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	t.Run("track connection", func(t *testing.T) {
		tracker := NewTracker(logger)
		conn := SetupMockConn()

		// Track connection
		tracker.Track(conn)

		// Verify connection is tracked
		stats := tracker.GetStats()
		assert.Equal(t, 1, stats.ActiveConnections)
		assert.Equal(t, int64(0), stats.TotalBytesReceived)
		assert.Equal(t, int64(0), stats.TotalBytesSent)
	})

	t.Run("untrack connection", func(t *testing.T) {
		tracker := NewTracker(logger)
		conn := SetupMockConn()

		// Track and untrack connection
		tracker.Track(conn)
		tracker.Untrack(conn)

		// Verify connection is untracked
		stats := tracker.GetStats()
		assert.Equal(t, 0, stats.ActiveConnections)
	})

	t.Run("update stats", func(t *testing.T) {
		tracker := NewTracker(logger)
		conn := SetupMockConn()

		// Track connection and update stats
		tracker.Track(conn)
		tracker.UpdateStats(conn.RemoteAddr().String(), 100, 200)

		// Verify stats are updated
		stats := tracker.GetStats()
		assert.Equal(t, int64(100), stats.TotalBytesSent)
		assert.Equal(t, int64(200), stats.TotalBytesReceived)
	})

	t.Run("connection duration", func(t *testing.T) {
		tracker := NewTracker(logger)
		conn := SetupMockConn()

		// Track connection
		tracker.Track(conn)
		time.Sleep(100 * time.Millisecond)
		tracker.Untrack(conn)

		// Verify duration is tracked
		stats := tracker.GetStats()
		assert.True(t, stats.AverageConnectionDuration >= 100*time.Millisecond)
	})

	t.Run("multiple connections", func(t *testing.T) {
		tracker := NewTracker(logger)
		conn1 := SetupMockConn()
		conn2 := SetupMockConn()

		// Track multiple connections
		tracker.Track(conn1)
		tracker.Track(conn2)

		// Update stats for both connections
		tracker.UpdateStats(conn1.RemoteAddr().String(), 100, 200)
		tracker.UpdateStats(conn2.RemoteAddr().String(), 300, 400)

		// Verify combined stats
		stats := tracker.GetStats()
		assert.Equal(t, 2, stats.ActiveConnections)
		assert.Equal(t, int64(400), stats.TotalBytesSent)
		assert.Equal(t, int64(600), stats.TotalBytesReceived)

		// Untrack one connection
		tracker.Untrack(conn1)
		stats = tracker.GetStats()
		assert.Equal(t, 1, stats.ActiveConnections)
	})

	t.Run("peak connections", func(t *testing.T) {
		tracker := NewTracker(logger)
		conn1 := SetupMockConn()
		conn2 := SetupMockConn()

		// Track and verify peak
		tracker.Track(conn1)
		tracker.Track(conn2)
		stats := tracker.GetStats()
		assert.Equal(t, 2, stats.PeakConnections)

		// Untrack and verify peak remains
		tracker.Untrack(conn1)
		stats = tracker.GetStats()
		assert.Equal(t, 2, stats.PeakConnections)
	})

	t.Run("reset stats", func(t *testing.T) {
		tracker := NewTracker(logger)
		conn := SetupMockConn()

		// Track and update stats
		tracker.Track(conn)
		tracker.UpdateStats(conn.RemoteAddr().String(), 100, 200)

		// Reset stats
		tracker.ResetStats()

		// Verify stats are reset
		stats := tracker.GetStats()
		assert.Equal(t, 1, stats.ActiveConnections) // Active connections remain
		assert.Equal(t, int64(0), stats.TotalBytesSent)
		assert.Equal(t, int64(0), stats.TotalBytesReceived)
		assert.Equal(t, 1, stats.PeakConnections) // Peak remains
	})
}
