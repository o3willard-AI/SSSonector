package monitor

import (
	"sync/atomic"
	"time"
)

// MetricsCollector handles collection and storage of tunnel metrics
type MetricsCollector struct {
	// Network metrics
	bytesReceived uint64
	bytesSent     uint64
	packetsLost   uint64
	latency       int64 // microseconds

	// System metrics
	cpuUsage    uint64 // fixed-point: value * 100
	memoryUsage uint64 // fixed-point: value * 100
	uptime      int64  // seconds

	// Connection metrics
	activeConnections uint32
	totalConnections  uint64

	startTime time.Time
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		startTime: time.Now(),
	}
}

// UpdateNetworkMetrics updates network-related metrics
func (m *MetricsCollector) UpdateNetworkMetrics(bytesReceived, bytesSent, packetsLost uint64, latency int64) {
	atomic.StoreUint64(&m.bytesReceived, bytesReceived)
	atomic.StoreUint64(&m.bytesSent, bytesSent)
	atomic.StoreUint64(&m.packetsLost, packetsLost)
	atomic.StoreInt64(&m.latency, latency)
}

// UpdateSystemMetrics updates system-related metrics
func (m *MetricsCollector) UpdateSystemMetrics(cpuPercent, memoryPercent float64) {
	atomic.StoreUint64(&m.cpuUsage, uint64(cpuPercent*100))
	atomic.StoreUint64(&m.memoryUsage, uint64(memoryPercent*100))
	atomic.StoreInt64(&m.uptime, int64(time.Since(m.startTime).Seconds()))
}

// UpdateConnectionMetrics updates connection-related metrics
func (m *MetricsCollector) UpdateConnectionMetrics(activeConns uint32) {
	atomic.StoreUint32(&m.activeConnections, activeConns)
	atomic.AddUint64(&m.totalConnections, 1)
}

// GetNetworkMetrics returns current network metrics
func (m *MetricsCollector) GetNetworkMetrics() (uint64, uint64, uint64, int64) {
	return atomic.LoadUint64(&m.bytesReceived),
		atomic.LoadUint64(&m.bytesSent),
		atomic.LoadUint64(&m.packetsLost),
		atomic.LoadInt64(&m.latency)
}

// GetSystemMetrics returns current system metrics
func (m *MetricsCollector) GetSystemMetrics() (float64, float64, int64) {
	cpuUsage := float64(atomic.LoadUint64(&m.cpuUsage)) / 100.0
	memoryUsage := float64(atomic.LoadUint64(&m.memoryUsage)) / 100.0
	uptime := atomic.LoadInt64(&m.uptime)
	return cpuUsage, memoryUsage, uptime
}

// GetConnectionMetrics returns current connection metrics
func (m *MetricsCollector) GetConnectionMetrics() (uint32, uint64) {
	return atomic.LoadUint32(&m.activeConnections),
		atomic.LoadUint64(&m.totalConnections)
}

// Metrics represents a snapshot of all metrics
type Metrics struct {
	// Network metrics
	BytesReceived uint64
	BytesSent     uint64
	PacketsLost   uint64
	Latency       int64 // microseconds

	// System metrics
	CPUUsage    float64
	MemoryUsage float64
	Uptime      int64 // seconds

	// Connection metrics
	ActiveConnections uint32
	TotalConnections  uint64

	// Timestamp
	Timestamp time.Time
}

// GetSnapshot returns a snapshot of all current metrics
func (m *MetricsCollector) GetSnapshot() *Metrics {
	cpuUsage, memoryUsage, uptime := m.GetSystemMetrics()
	bytesReceived, bytesSent, packetsLost, latency := m.GetNetworkMetrics()
	activeConns, totalConns := m.GetConnectionMetrics()

	return &Metrics{
		BytesReceived:     bytesReceived,
		BytesSent:         bytesSent,
		PacketsLost:       packetsLost,
		Latency:           latency,
		CPUUsage:          cpuUsage,
		MemoryUsage:       memoryUsage,
		Uptime:            uptime,
		ActiveConnections: activeConns,
		TotalConnections:  totalConns,
		Timestamp:         time.Now(),
	}
}
