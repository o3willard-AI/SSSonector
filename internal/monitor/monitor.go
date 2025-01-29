package monitor

import (
	"fmt"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Monitor handles logging and monitoring
type Monitor struct {
	logger *zap.Logger
	snmp   *SNMPAgent
}

// NewMonitor creates a new monitor instance
func NewMonitor(logger *zap.Logger, cfg *config.MonitorConfig) (*Monitor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("monitor configuration is required")
	}

	// Configure file logging if specified
	if cfg.LogFile != "" {
		logRotator := &lumberjack.Logger{
			Filename:   cfg.LogFile,
			MaxSize:    100, // megabytes
			MaxBackups: 3,
			MaxAge:     28,   // days
			Compress:   true, // compress rotated files
		}

		// Create a new logger that writes to both console and file
		fileCore := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(logRotator),
			zap.InfoLevel,
		)
		logger = logger.WithOptions(zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewTee(core, fileCore)
		}))
	}

	m := &Monitor{
		logger: logger,
	}

	// Initialize SNMP if enabled
	if cfg.SNMPEnabled {
		snmpAgent, err := NewSNMPAgent(logger, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize SNMP agent: %w", err)
		}
		m.snmp = snmpAgent
	}

	return m, nil
}

// Close closes the monitor and its resources
func (m *Monitor) Close() error {
	if m.snmp != nil {
		if err := m.snmp.Close(); err != nil {
			m.logger.Error("Failed to close SNMP agent", zap.Error(err))
		}
	}
	return nil
}

// LogTunnelStats logs tunnel statistics
func (m *Monitor) LogTunnelStats(bytesIn, bytesOut int64, activeConnections int) {
	m.logger.Info("Tunnel statistics",
		zap.Int64("bytes_in", bytesIn),
		zap.Int64("bytes_out", bytesOut),
		zap.Int("active_connections", activeConnections),
	)

	if m.snmp != nil {
		m.snmp.UpdateStats(bytesIn, bytesOut, activeConnections)
	}
}

// LogConnectionEvent logs connection events
func (m *Monitor) LogConnectionEvent(event string, remoteAddr string, err error) {
	if err != nil {
		m.logger.Error("Connection event",
			zap.String("event", event),
			zap.String("remote_addr", remoteAddr),
			zap.Error(err),
		)
	} else {
		m.logger.Info("Connection event",
			zap.String("event", event),
			zap.String("remote_addr", remoteAddr),
		)
	}
}

// LogBandwidthThrottle logs bandwidth throttling events
func (m *Monitor) LogBandwidthThrottle(bytesThrottled int64, duration float64) {
	m.logger.Info("Bandwidth throttled",
		zap.Int64("bytes_throttled", bytesThrottled),
		zap.Float64("duration_ms", duration),
	)
}
