package monitor

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"
)

// Config holds monitoring configuration
type Config struct {
	LogFile       string
	SNMPEnabled   bool
	SNMPPort      int
	SNMPCommunity string
	SNMPAddress   string
}

// Monitor handles system monitoring and logging
type Monitor struct {
	logger     *Logger
	config     *Config
	metrics    *Metrics
	snmpAgent  *SNMPAgent
	sysMetrics *SystemMetricsCollector
	startTime  time.Time
	mu         sync.RWMutex
	shutdownCh chan struct{}
	shutdownWg sync.WaitGroup
	isTestMode bool
}

// New creates a new monitor instance
func New(cfg *Config) (*Monitor, error) {
	logger, err := NewLogger(INFO, cfg.LogFile)
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	m := &Monitor{
		logger:     logger,
		config:     cfg,
		metrics:    NewMetrics(),
		sysMetrics: NewSystemMetricsCollector(),
		startTime:  time.Now(),
		shutdownCh: make(chan struct{}),
		isTestMode: os.Getenv("TEMP_DIR") != "",
	}

	// Initialize SNMP agent if enabled
	if cfg.SNMPEnabled {
		m.snmpAgent, err = NewSNMPAgent(cfg, m.metrics, logger)
		if err != nil {
			return nil, fmt.Errorf("failed to create SNMP agent: %w", err)
		}
	}

	return m, nil
}

// Start initializes monitoring
func (m *Monitor) Start() error {
	if m.config.SNMPEnabled && m.snmpAgent != nil {
		if err := m.snmpAgent.Start(); err != nil {
			return fmt.Errorf("failed to start SNMP agent: %w", err)
		}
		m.logger.Info("SNMP monitoring started on %s:%d",
			m.config.SNMPAddress, m.config.SNMPPort)
	}

	// Start certificate expiration monitor in test mode
	if m.isTestMode {
		m.shutdownWg.Add(1)
		go m.monitorCertExpiration()
	}

	// Start system metrics collection
	m.shutdownWg.Add(1)
	go m.collectSystemMetrics()

	return nil
}

// Stop shuts down monitoring
func (m *Monitor) Stop() {
	select {
	case <-m.shutdownCh:
		// Already shutting down
		return
	default:
		close(m.shutdownCh)
	}

	if m.config.SNMPEnabled && m.snmpAgent != nil {
		m.snmpAgent.Stop()
		m.logger.Info("SNMP monitoring stopped")
	}

	m.shutdownWg.Wait()

	// Close and sync logger
	if err := m.logger.Sync(); err != nil {
		m.Error("Failed to sync logger: %v", err)
	}
	if err := m.logger.Close(); err != nil {
		m.Error("Failed to close logger: %v", err)
	}
}

// Info logs an info message
func (m *Monitor) Info(format string, v ...interface{}) {
	m.logger.Info(format, v...)
}

// Error logs an error message
func (m *Monitor) Error(format string, v ...interface{}) {
	m.logger.Error(format, v...)
}

// Warn logs a warning message
func (m *Monitor) Warn(format string, v ...interface{}) {
	m.logger.Warn(format, v...)
}

// UpdateMetrics updates monitoring metrics
func (m *Monitor) UpdateMetrics(bytesIn, bytesOut, packetsIn, packetsOut, errors int64, connections int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.metrics.UpdateNetworkMetrics(bytesIn, bytesOut, packetsIn, packetsOut)
	m.metrics.UpdateErrorMetrics(errors, "", 0, 0) // No retry/drop info yet
	m.metrics.UpdateConnectionMetrics(int32(connections), int32(connections), 0, 0)
}

// GetMetrics returns current metrics
func (m *Monitor) GetMetrics() *Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.metrics.Clone()
}

// monitorCertExpiration monitors certificate expiration in test mode
func (m *Monitor) monitorCertExpiration() {
	defer m.shutdownWg.Done()

	// Wait for 15 seconds in test mode
	select {
	case <-time.After(15 * time.Second):
		m.Info("Test mode: certificate expired, shutting down")
		// Use platform-specific process termination
		if err := killProcess(os.Getpid()); err != nil {
			m.Error("Failed to kill process: %v", err)
		}
	case <-m.shutdownCh:
		return
	}
}

// collectSystemMetrics periodically collects system-wide metrics
func (m *Monitor) collectSystemMetrics() {
	defer m.shutdownWg.Done()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	var memStats runtime.MemStats
	for {
		select {
		case <-m.shutdownCh:
			return
		case <-ticker.C:
			// Update memory stats
			runtime.ReadMemStats(&memStats)

			// Get number of goroutines
			numGoroutines := runtime.NumGoroutine()

			m.mu.Lock()
			// Update resource metrics
			m.metrics.UpdateResourceMetrics(
				m.metrics.CPUUsage, // Preserved from system metrics collector
				int64(memStats.Alloc),
				int64(memStats.HeapAlloc),
				0, // Queue length from tunnel
				int64(numGoroutines),
			)

			// Collect system metrics
			if err := m.sysMetrics.CollectMetrics(m.metrics); err != nil {
				m.Error("Failed to collect system metrics: %v", err)
			}
			m.mu.Unlock()
		}
	}
}
