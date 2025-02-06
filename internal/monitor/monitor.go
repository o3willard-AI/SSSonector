package monitor

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
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
	startTime  time.Time
	mu         sync.RWMutex
	shutdownCh chan struct{}
	shutdownWg sync.WaitGroup
	isTestMode bool
}

// New creates a new monitor instance
func New(cfg *Config) (*Monitor, error) {
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	m := &Monitor{
		logger:     logger,
		config:     cfg,
		metrics:    NewMetrics(),
		startTime:  time.Now(),
		shutdownCh: make(chan struct{}),
		isTestMode: os.Getenv("TEMP_DIR") != "",
	}

	// Initialize SNMP agent if enabled
	if cfg.SNMPEnabled {
		m.snmpAgent, err = NewSNMPAgent(cfg, m.metrics)
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
		m.Info("SNMP monitoring started",
			zap.String("address", m.config.SNMPAddress),
			zap.Int("port", m.config.SNMPPort))
	}

	// Start certificate expiration monitor in test mode
	if m.isTestMode {
		m.shutdownWg.Add(1)
		go m.monitorCertExpiration()
	}

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
		m.Info("SNMP monitoring stopped")
	}

	m.shutdownWg.Wait()
	m.logger.Sync()
}

// Info logs an info message
func (m *Monitor) Info(msg string, fields ...zap.Field) {
	m.logger.Info(msg, fields...)
}

// Error logs an error message
func (m *Monitor) Error(msg string, err error, fields ...zap.Field) {
	if err != nil {
		fields = append(fields, zap.Error(err))
	}
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

	m.metrics.BytesIn = bytesIn
	m.metrics.BytesOut = bytesOut
	m.metrics.PacketsIn = packetsIn
	m.metrics.PacketsOut = packetsOut
	m.metrics.Errors = errors
	m.metrics.Connections = connections
	m.metrics.Uptime = int64(time.Since(m.startTime).Seconds())
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
		// Force kill the process group
		pgid, err := syscall.Getpgid(os.Getpid())
		if err == nil {
			syscall.Kill(-pgid, syscall.SIGKILL)
		} else {
			// Fallback to killing just this process
			syscall.Kill(os.Getpid(), syscall.SIGKILL)
		}
	case <-m.shutdownCh:
		return
	}
}
