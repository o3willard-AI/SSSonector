package monitor

import (
	"sync/atomic"
	"time"
)

// MetricType represents different types of metrics
type MetricType int

const (
	CounterMetric MetricType = iota
	GaugeMetric
	HistogramMetric
)

// Metrics holds monitoring metrics
type Metrics struct {
	// Network metrics
	BytesIn    int64
	BytesOut   int64
	PacketsIn  int64
	PacketsOut int64
	ByteRate   float64
	PacketRate float64

	// Error metrics
	Errors     int64
	LastError  string
	ErrorRate  float64
	RetryCount int64
	DropCount  int64

	// Performance metrics
	Latency        int64 // in microseconds
	Jitter         int64 // in microseconds
	RTT            int64 // in microseconds
	PacketLoss     float64
	ReorderingRate float64

	// Resource metrics
	CPUUsage     float64
	MemoryUsage  int64
	BufferSize   int64
	QueueLength  int64
	GoroutineNum int64

	// Connection metrics
	Connections    int32
	MaxConnections int32
	ConnectTime    int64 // in milliseconds
	DisconnectTime int64 // in milliseconds

	// System metrics
	Uptime     int64
	LastUpdate time.Time
	StartTime  time.Time
	SystemLoad float64
	DiskIO     int64
	NetworkIO  int64
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	now := time.Now()
	return &Metrics{
		StartTime:  now,
		LastUpdate: now,
	}
}

// Reset resets all metrics to zero
func (m *Metrics) Reset() {
	atomic.StoreInt64(&m.BytesIn, 0)
	atomic.StoreInt64(&m.BytesOut, 0)
	atomic.StoreInt64(&m.PacketsIn, 0)
	atomic.StoreInt64(&m.PacketsOut, 0)
	atomic.StoreInt64(&m.Errors, 0)
	atomic.StoreInt64(&m.RetryCount, 0)
	atomic.StoreInt64(&m.DropCount, 0)
	atomic.StoreInt64(&m.Latency, 0)
	atomic.StoreInt64(&m.Jitter, 0)
	atomic.StoreInt64(&m.RTT, 0)
	atomic.StoreInt64(&m.MemoryUsage, 0)
	atomic.StoreInt64(&m.BufferSize, 0)
	atomic.StoreInt64(&m.QueueLength, 0)
	atomic.StoreInt64(&m.GoroutineNum, 0)
	atomic.StoreInt32(&m.Connections, 0)
	atomic.StoreInt32(&m.MaxConnections, 0)
	atomic.StoreInt64(&m.ConnectTime, 0)
	atomic.StoreInt64(&m.DisconnectTime, 0)
	atomic.StoreInt64(&m.Uptime, 0)
	atomic.StoreInt64(&m.DiskIO, 0)
	atomic.StoreInt64(&m.NetworkIO, 0)
	m.LastUpdate = time.Now()
}

// Clone creates a copy of the metrics
func (m *Metrics) Clone() *Metrics {
	return &Metrics{
		BytesIn:        atomic.LoadInt64(&m.BytesIn),
		BytesOut:       atomic.LoadInt64(&m.BytesOut),
		PacketsIn:      atomic.LoadInt64(&m.PacketsIn),
		PacketsOut:     atomic.LoadInt64(&m.PacketsOut),
		ByteRate:       m.ByteRate,
		PacketRate:     m.PacketRate,
		Errors:         atomic.LoadInt64(&m.Errors),
		LastError:      m.LastError,
		ErrorRate:      m.ErrorRate,
		RetryCount:     atomic.LoadInt64(&m.RetryCount),
		DropCount:      atomic.LoadInt64(&m.DropCount),
		Latency:        atomic.LoadInt64(&m.Latency),
		Jitter:         atomic.LoadInt64(&m.Jitter),
		RTT:            atomic.LoadInt64(&m.RTT),
		PacketLoss:     m.PacketLoss,
		ReorderingRate: m.ReorderingRate,
		CPUUsage:       m.CPUUsage,
		MemoryUsage:    atomic.LoadInt64(&m.MemoryUsage),
		BufferSize:     atomic.LoadInt64(&m.BufferSize),
		QueueLength:    atomic.LoadInt64(&m.QueueLength),
		GoroutineNum:   atomic.LoadInt64(&m.GoroutineNum),
		Connections:    atomic.LoadInt32(&m.Connections),
		MaxConnections: atomic.LoadInt32(&m.MaxConnections),
		ConnectTime:    atomic.LoadInt64(&m.ConnectTime),
		DisconnectTime: atomic.LoadInt64(&m.DisconnectTime),
		Uptime:         atomic.LoadInt64(&m.Uptime),
		LastUpdate:     m.LastUpdate,
		StartTime:      m.StartTime,
		SystemLoad:     m.SystemLoad,
		DiskIO:         atomic.LoadInt64(&m.DiskIO),
		NetworkIO:      atomic.LoadInt64(&m.NetworkIO),
	}
}

// UpdateNetworkMetrics updates network-related metrics
func (m *Metrics) UpdateNetworkMetrics(bytesIn, bytesOut, packetsIn, packetsOut int64) {
	atomic.AddInt64(&m.BytesIn, bytesIn)
	atomic.AddInt64(&m.BytesOut, bytesOut)
	atomic.AddInt64(&m.PacketsIn, packetsIn)
	atomic.AddInt64(&m.PacketsOut, packetsOut)

	// Calculate rates
	elapsed := time.Since(m.LastUpdate).Seconds()
	if elapsed > 0 {
		m.ByteRate = float64(bytesIn+bytesOut) / elapsed
		m.PacketRate = float64(packetsIn+packetsOut) / elapsed
	}
	m.LastUpdate = time.Now()
}

// UpdateErrorMetrics updates error-related metrics
func (m *Metrics) UpdateErrorMetrics(errors int64, lastError string, retries, drops int64) {
	atomic.AddInt64(&m.Errors, errors)
	atomic.AddInt64(&m.RetryCount, retries)
	atomic.AddInt64(&m.DropCount, drops)
	m.LastError = lastError

	elapsed := time.Since(m.LastUpdate).Seconds()
	if elapsed > 0 {
		m.ErrorRate = float64(errors) / elapsed
	}
}

// UpdatePerformanceMetrics updates performance-related metrics
func (m *Metrics) UpdatePerformanceMetrics(latency, jitter, rtt int64, packetLoss, reorderingRate float64) {
	atomic.StoreInt64(&m.Latency, latency)
	atomic.StoreInt64(&m.Jitter, jitter)
	atomic.StoreInt64(&m.RTT, rtt)
	m.PacketLoss = packetLoss
	m.ReorderingRate = reorderingRate
}

// UpdateResourceMetrics updates resource utilization metrics
func (m *Metrics) UpdateResourceMetrics(cpu float64, memory, bufferSize, queueLen, goroutines int64) {
	m.CPUUsage = cpu
	atomic.StoreInt64(&m.MemoryUsage, memory)
	atomic.StoreInt64(&m.BufferSize, bufferSize)
	atomic.StoreInt64(&m.QueueLength, queueLen)
	atomic.StoreInt64(&m.GoroutineNum, goroutines)
}

// UpdateConnectionMetrics updates connection-related metrics
func (m *Metrics) UpdateConnectionMetrics(conns, maxConns int32, connectTime, disconnectTime int64) {
	atomic.StoreInt32(&m.Connections, conns)
	atomic.StoreInt32(&m.MaxConnections, maxConns)
	atomic.StoreInt64(&m.ConnectTime, connectTime)
	atomic.StoreInt64(&m.DisconnectTime, disconnectTime)
}

// UpdateSystemMetrics updates system-wide metrics
func (m *Metrics) UpdateSystemMetrics(load float64, diskIO, networkIO int64) {
	m.SystemLoad = load
	atomic.StoreInt64(&m.DiskIO, diskIO)
	atomic.StoreInt64(&m.NetworkIO, networkIO)
	atomic.StoreInt64(&m.Uptime, int64(time.Since(m.StartTime).Seconds()))
}
