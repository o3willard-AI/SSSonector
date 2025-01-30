package monitor

import (
	"sync"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"go.uber.org/zap"
)

// Monitor handles monitoring and metrics collection
type Monitor struct {
	logger    *zap.Logger
	config    *config.MonitorConfig
	snmpAgent *SNMPAgent
	mu        sync.RWMutex
	metrics   *Metrics
	startTime time.Time
}

// NewMonitor creates a new monitor instance
func NewMonitor(logger *zap.Logger, cfg *config.MonitorConfig) (*Monitor, error) {
	m := &Monitor{
		logger:    logger,
		config:    cfg,
		startTime: time.Now(),
		metrics:   NewMetrics(),
	}

	if cfg.SNMPEnabled {
		agent, err := NewSNMPAgent(logger, cfg)
		if err != nil {
			return nil, err
		}
		m.snmpAgent = agent
	}

	return m, nil
}

// Start starts the monitoring
func (m *Monitor) Start() error {
	if m.snmpAgent != nil {
		if err := m.snmpAgent.Start(); err != nil {
			return err
		}
	}
	return nil
}

// Stop stops the monitoring
func (m *Monitor) Stop() error {
	if m.snmpAgent != nil {
		if err := m.snmpAgent.Stop(); err != nil {
			return err
		}
	}
	return nil
}

// UpdateMetrics updates monitoring metrics
func (m *Monitor) UpdateMetrics(bytesReceived, bytesSent uint64, connections int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics.BytesReceived = bytesReceived
	m.metrics.BytesSent = bytesSent
	m.metrics.Connections = connections
	m.metrics.Uptime = time.Since(m.startTime).Seconds()

	if m.snmpAgent != nil {
		m.snmpAgent.UpdateMetrics(bytesReceived, bytesSent, connections)
	}
}

// GetMetrics returns current metrics
func (m *Monitor) GetMetrics() *Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return &Metrics{
		BytesReceived: m.metrics.BytesReceived,
		BytesSent:     m.metrics.BytesSent,
		Connections:   m.metrics.Connections,
		Uptime:        m.metrics.Uptime,
	}
}
