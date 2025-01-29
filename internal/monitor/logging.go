package monitor

import (
	"fmt"
	"os"
	"path/filepath"

	"SSSonector/internal/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var (
	logger *zap.Logger
)

// InitLogging initializes the logging system based on configuration
func InitLogging() error {
	// Create default logger for startup
	var err error
	logger, err = zap.NewDevelopment()
	if err != nil {
		return fmt.Errorf("failed to create default logger: %w", err)
	}
	zap.ReplaceGlobals(logger)
	return nil
}

// ConfigureLogging configures logging based on the provided configuration
func ConfigureLogging(cfg *config.LoggingConfig) error {
	// Ensure log directory exists
	logDir := filepath.Dir(cfg.FilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Configure log rotation
	writer := &lumberjack.Logger{
		Filename:   cfg.FilePath,
		MaxSize:    cfg.MaxSize,
		MaxBackups: 3,
		MaxAge:     28,   // days
		Compress:   true, // compress rotated files
	}

	// Parse log level
	var level zapcore.Level
	switch cfg.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	default:
		return fmt.Errorf("invalid log level: %s (must be 'debug' or 'info')", cfg.Level)
	}

	// Create encoder config
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Create core for file logging
	fileCore := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(writer),
		level,
	)

	// Create new logger
	newLogger := zap.New(fileCore, zap.AddCaller())

	// Replace global logger
	zap.ReplaceGlobals(newLogger)
	logger = newLogger

	return nil
}

// GetLogger returns the configured logger instance
func GetLogger() *zap.Logger {
	return logger
}

// Sync flushes any buffered log entries
func Sync() error {
	return logger.Sync()
}
