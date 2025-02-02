package monitor

import (
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogConfig holds logging configuration
type LogConfig struct {
	LogFile  string
	LogLevel string
	MaxSize  int // megabytes
	MaxAge   int // days
}

// NewLogger creates a new logger instance
func NewLogger(cfg *LogConfig) (*zap.Logger, error) {
	// Create log directory if it doesn't exist
	if cfg.LogFile != "" {
		logDir := filepath.Dir(cfg.LogFile)
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}
	}

	// Configure encoder
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "ts"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// Create core
	var core zapcore.Core
	if cfg.LogFile == "" {
		// Log to stderr if no file specified
		core = zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.Lock(os.Stderr),
			zap.NewAtomicLevelAt(zap.InfoLevel),
		)
	} else {
		// Log to file
		file, err := os.OpenFile(cfg.LogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}

		core = zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.Lock(file),
			zap.NewAtomicLevelAt(zap.InfoLevel),
		)
	}

	// Create logger
	logger := zap.New(core)

	return logger, nil
}
