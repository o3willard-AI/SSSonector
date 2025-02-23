package startup

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// StartupLogger provides enhanced logging during startup
type StartupLogger struct {
	logger *zap.Logger
	phase  types.StartupPhase
	config *types.LoggingConfig
}

// NewStartupLogger creates a new startup logger
func NewStartupLogger(logger *zap.Logger, config *types.LoggingConfig) *StartupLogger {
	return &StartupLogger{
		logger: logger.WithOptions(
			zap.AddStacktrace(zapcore.ErrorLevel),
			zap.AddCaller(),
		),
		config: config,
	}
}

// SetPhase sets the current startup phase
func (l *StartupLogger) SetPhase(phase types.StartupPhase) {
	l.phase = phase
	if l.config.StartupLogs {
		l.logger.Info(fmt.Sprintf("Entering %s phase", phase))
	}
}

// LogOperation logs a startup operation with timing and details
func (l *StartupLogger) LogOperation(component types.StartupComponent, operation string, fn func() error, details map[string]interface{}) error {
	if !l.config.StartupLogs {
		return fn()
	}

	start := time.Now()

	entry := &types.StartupLog{
		Phase:     l.phase,
		Component: component,
		Operation: operation,
		Details:   details,
		Timestamp: start,
	}

	// Execute the operation
	err := fn()

	// Calculate duration
	entry.Duration = types.NewDuration(time.Since(start))

	if err != nil {
		entry.Status = "Failed"
		entry.Error = err.Error()

		// Convert entry to JSON for structured logging
		jsonEntry, _ := json.Marshal(entry)
		l.logger.Error(fmt.Sprintf("%s failed", operation),
			zap.Duration("duration", entry.Duration.Duration),
			zap.String("phase", string(entry.Phase)),
			zap.String("component", string(entry.Component)),
			zap.String("error", entry.Error),
			zap.ByteString("startup_log", jsonEntry),
		)
		return err
	}

	entry.Status = "Success"

	// Convert entry to JSON for structured logging
	jsonEntry, _ := json.Marshal(entry)
	l.logger.Debug(fmt.Sprintf("%s completed", operation),
		zap.Duration("duration", entry.Duration.Duration),
		zap.String("phase", string(entry.Phase)),
		zap.String("component", string(entry.Component)),
		zap.ByteString("startup_log", jsonEntry),
	)
	return nil
}

// LogCheckpoint logs a major startup checkpoint
func (l *StartupLogger) LogCheckpoint(checkpoint string, details map[string]interface{}) {
	if !l.config.StartupLogs {
		return
	}

	l.logger.Info(fmt.Sprintf("Startup checkpoint: %s", checkpoint),
		zap.String("phase", string(l.phase)),
		zap.Any("details", details),
	)
}

// LogResourceState logs the state of a system resource
func (l *StartupLogger) LogResourceState(resource string, state map[string]interface{}) {
	if !l.config.StartupLogs {
		return
	}

	l.logger.Debug(fmt.Sprintf("Resource state: %s", resource),
		zap.String("phase", string(l.phase)),
		zap.Any("state", state),
	)
}
