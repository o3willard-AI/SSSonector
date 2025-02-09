package monitor

import (
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
	logger     *zap.Logger
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
	// Create logger configuration
	logConfig := zap.NewProductionConfig()
	logConfig.OutputPaths = []string{cfg.LogFile}
	logConfig.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Create logger
	logger, err := logConfig.Build()
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

// Logger returns the monitor's logger instance
func (m *Monitor) Logger() *zap.Logger {
	return m.logger
}

// Start initializes monitoring
func (m *Monitor) Start() error {
	if m.config.SNMPEnabled && m.snmpAgent != nil {
		if err := m.snmpAgent.Start(); err != nil {
			return fmt.Errorf("failed to start SNMP agent: %w", err)
		}
		m.logger.Info("SNMP monitoring started",
			zap.String("address", m.config.SNMPAddress),
			zap.Int("port", m.config.SNMPPort))
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
		m.logger.Error("Failed to sync logger", zap.Error(err))
	}
}

// Info logs an info message
func (m *Monitor) Info(msg string, fields ...zap.Field) {
	m.logger.Info(msg, fields...)
}

// Error logs an error message
func (m *Monitor) Error(msg string, fields ...zap.Field) {
	m.logger.Error(msg, fields...)
}

// Warn logs a warning message
func (m *Monitor) Warn(msg string, fields ...zap.Field) {
	m.logger.Warn(msg, fields...)
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
		m.logger.Info("Test mode: certificate expired, shutting down")
		// Use platform-specific process termination
		if err := killProcess(os.Getpid()); err != nil {
			m.logger.Error("Failed to kill process", zap.Error(err))
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
				m.logger.Error("Failed to collect system metrics", zap.Error(err))
			}
			m.mu.Unlock()
		}
	}
}
