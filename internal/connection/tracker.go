package connection

import (
	"net"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ConnectionStats holds statistics for a connection
type ConnectionStats struct {
	StartTime     time.Time
	BytesSent     int64
	BytesReceived int64
}

// TrackerStats holds overall connection tracking statistics
type TrackerStats struct {
	ActiveConnections         int
	PeakConnections           int
	TotalBytesSent            int64
	TotalBytesReceived        int64
	AverageConnectionDuration time.Duration
	totalDuration             time.Duration
	connectionCount           int
}

// Tracker tracks connection statistics
type Tracker struct {
	logger    *zap.Logger
	mu        sync.RWMutex
	stats     TrackerStats
	connStats map[string]*ConnectionStats
}

// NewTracker creates a new connection tracker
func NewTracker(logger *zap.Logger) *Tracker {
	return &Tracker{
		logger:    logger,
		connStats: make(map[string]*ConnectionStats),
	}
}

// Track starts tracking a connection
func (t *Tracker) Track(conn net.Conn) {
	t.mu.Lock()
	defer t.mu.Unlock()

	addr := conn.RemoteAddr().String()
	t.connStats[addr] = &ConnectionStats{
		StartTime: time.Now(),
	}

	t.stats.ActiveConnections++
	if t.stats.ActiveConnections > t.stats.PeakConnections {
		t.stats.PeakConnections = t.stats.ActiveConnections
	}

	t.logger.Debug("Started tracking connection",
		zap.String("remote_addr", addr),
		zap.Int("active_connections", t.stats.ActiveConnections))
}

// Untrack stops tracking a connection
func (t *Tracker) Untrack(conn net.Conn) {
	t.mu.Lock()
	defer t.mu.Unlock()

	addr := conn.RemoteAddr().String()
	if stats, exists := t.connStats[addr]; exists {
		duration := time.Since(stats.StartTime)
		t.stats.totalDuration += duration
		t.stats.connectionCount++
		t.stats.AverageConnectionDuration = t.stats.totalDuration / time.Duration(t.stats.connectionCount)

		delete(t.connStats, addr)
		t.stats.ActiveConnections--

		t.logger.Debug("Stopped tracking connection",
			zap.String("remote_addr", addr),
			zap.Duration("duration", duration),
			zap.Int("active_connections", t.stats.ActiveConnections))
	}
}

// UpdateStats updates statistics for a connection
func (t *Tracker) UpdateStats(remoteAddr string, bytesSent, bytesReceived int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if stats, exists := t.connStats[remoteAddr]; exists {
		// Calculate delta
		deltaBytesSent := bytesSent - stats.BytesSent
		deltaBytesReceived := bytesReceived - stats.BytesReceived

		// Update connection stats
		stats.BytesSent = bytesSent
		stats.BytesReceived = bytesReceived

		// Update total stats
		t.stats.TotalBytesSent += deltaBytesSent
		t.stats.TotalBytesReceived += deltaBytesReceived

		t.logger.Debug("Updated connection stats",
			zap.String("remote_addr", remoteAddr),
			zap.Int64("bytes_sent", bytesSent),
			zap.Int64("bytes_received", bytesReceived))
	}
}

// GetStats returns current tracking statistics
func (t *Tracker) GetStats() TrackerStats {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.stats
}

// ResetStats resets all statistics except active connections and peak
func (t *Tracker) ResetStats() {
	t.mu.Lock()
	defer t.mu.Unlock()

	activeConns := t.stats.ActiveConnections
	peakConns := t.stats.PeakConnections
	t.stats = TrackerStats{
		ActiveConnections: activeConns,
		PeakConnections:   peakConns,
	}

	// Reset connection stats but keep tracking
	for addr := range t.connStats {
		t.connStats[addr] = &ConnectionStats{
			StartTime: time.Now(),
		}
	}

	t.logger.Info("Reset connection statistics")
}
