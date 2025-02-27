package startup

import (
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
	// Create a new logger with proper JSON formatting
	newLogger := logger.WithOptions(
		zap.AddStacktrace(zapcore.ErrorLevel),
		zap.AddCaller(),
	)

	return &StartupLogger{
		logger: newLogger,
		config: config,
	}
}

// SetPhase sets the current startup phase
func (l *StartupLogger) SetPhase(phase types.StartupPhase) error {
	if err := types.ValidatePhaseTransition(l.phase, phase); err != nil {
		entry := &types.StartupLog{
			Phase:     l.phase,
			Operation: "Phase transition",
			Error:     err.Error(),
			Status:    "Failed",
			Timestamp: time.Now(),
			Details: map[string]interface{}{
				"current_phase": string(l.phase),
				"next_phase":    string(phase),
			},
		}

		l.logger.Error("Invalid phase transition",
			zap.String("current", string(l.phase)),
			zap.String("next", string(phase)),
			zap.Error(err),
			zap.Object("startup_log", entry),
		)
		return err
	}

	oldPhase := l.phase
	l.phase = phase

	if l.config.StartupLogs {
		entry := &types.StartupLog{
			Phase:     phase,
			Operation: "Phase transition",
			Status:    "Success",
			Timestamp: time.Now(),
			Details: map[string]interface{}{
				"from_phase": string(oldPhase),
				"to_phase":   string(phase),
			},
		}
		l.logger.Info(fmt.Sprintf("Entering %s phase", phase),
			zap.Object("startup_log", entry),
		)
	}
	return nil
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

		l.logger.Error(fmt.Sprintf("%s failed", operation),
			zap.Duration("duration", entry.Duration.Duration),
			zap.String("phase", string(entry.Phase)),
			zap.String("component", string(entry.Component)),
			zap.String("error", entry.Error),
			zap.Object("startup_log", entry),
		)

		// Log cleanup for operations that require it
		if component == types.StartupComponentAdapter || component == types.StartupComponentConnection {
			cleanupEntry := &types.StartupLog{
				Phase:     l.phase,
				Component: component,
				Operation: "Cleanup",
				Status:    "Success",
				Timestamp: time.Now(),
			}
			l.logger.Info("Performing cleanup",
				zap.String("component", string(component)),
				zap.Object("startup_log", cleanupEntry),
			)
		}

		return err
	}

	entry.Status = "Success"

	l.logger.Debug(fmt.Sprintf("%s completed", operation),
		zap.Duration("duration", entry.Duration.Duration),
		zap.String("phase", string(entry.Phase)),
		zap.String("component", string(entry.Component)),
		zap.Object("startup_log", entry),
	)
	return nil
}

// LogCheckpoint logs a major startup checkpoint
func (l *StartupLogger) LogCheckpoint(checkpoint string, details map[string]interface{}) {
	if !l.config.StartupLogs {
		return
	}

	entry := &types.StartupLog{
		Phase:     l.phase,
		Operation: "Checkpoint",
		Status:    "Success",
		Timestamp: time.Now(),
		Details:   details,
	}

	l.logger.Info(fmt.Sprintf("Startup checkpoint: %s", checkpoint),
		zap.String("phase", string(l.phase)),
		zap.Any("details", details),
		zap.Object("startup_log", entry),
	)
}

// LogResourceState logs the state of a system resource
func (l *StartupLogger) LogResourceState(resource string, state map[string]interface{}) {
	if !l.config.StartupLogs {
		return
	}

	entry := &types.StartupLog{
		Phase:     l.phase,
		Operation: fmt.Sprintf("Resource state: %s", resource),
		Details:   state,
		Status:    "Success",
		Timestamp: time.Now(),
	}

	l.logger.Debug(fmt.Sprintf("Resource state: %s", resource),
		zap.String("phase", string(l.phase)),
		zap.Any("state", state),
		zap.Object("startup_log", entry),
	)
}
