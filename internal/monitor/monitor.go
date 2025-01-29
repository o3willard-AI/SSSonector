package monitor

import (
	"fmt"
	"sync"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"go.uber.org/zap"
)

// Monitor handles system monitoring and telemetry
type Monitor struct {
	logger *zap.Logger
	config *config.MonitorConfig
	snmp   *SNMPAgent
	wg     sync.WaitGroup
}

// NewMonitor creates a new monitor instance
func NewMonitor(logger *zap.Logger, cfg *config.MonitorConfig) (*Monitor, error) {
	m := &Monitor{
		logger: logger,
		config: cfg,
	}

	if cfg.SNMPEnabled {
		snmp, err := NewSNMPAgent(logger, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create SNMP agent: %w", err)
		}
		m.snmp = snmp
	}

	return m, nil
}

// Start starts the monitoring services
func (m *Monitor) Start() error {
	if m.snmp != nil {
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			if err := m.snmp.Start(); err != nil {
				m.logger.Error("SNMP agent error",
					zap.Error(err),
				)
			}
		}()
	}

	return nil
}

// Close stops all monitoring services
func (m *Monitor) Close() error {
	if m.snmp != nil {
		if err := m.snmp.Stop(); err != nil {
			m.logger.Error("Failed to stop SNMP agent",
				zap.Error(err),
			)
		}
	}

	m.wg.Wait()
	return nil
}

// UpdateStats updates monitoring statistics
func (m *Monitor) UpdateStats(stats interface{}) {
	if m.snmp != nil {
		m.snmp.UpdateStats(stats)
	}
}
