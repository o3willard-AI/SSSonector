package helpers

import (
	"bytes"
	"sync"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// TestLogger wraps a zap logger with test-specific functionality
type TestLogger struct {
	mu      sync.Mutex
	entries []LogEntry
	core    *testCore
}

// testCore implements zapcore.Core
type testCore struct {
	zapcore.LevelEnabler
	enc    zapcore.Encoder
	logger *TestLogger
	fields []zapcore.Field
	syncer zapcore.WriteSyncer
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Phase     types.StartupPhase     `json:"phase"`
	Component types.StartupComponent `json:"component"`
	Operation string                 `json:"operation"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Duration  types.Duration         `json:"duration,omitempty"`
	Status    string                 `json:"status"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// MarshalLogObject implements zapcore.ObjectMarshaler
func (e *LogEntry) MarshalLogObject(enc zapcore.ObjectEncoder) error {
	enc.AddString("phase", string(e.Phase))
	enc.AddString("component", string(e.Component))
	enc.AddString("operation", e.Operation)
	enc.AddString("status", e.Status)
	enc.AddTime("timestamp", e.Timestamp)

	if e.Error != "" {
		enc.AddString("error", e.Error)
	}

	if e.Duration.Duration > 0 {
		enc.AddDuration("duration", e.Duration.Duration)
	}

	if len(e.Details) > 0 {
		enc.AddObject("details", zapcore.ObjectMarshalerFunc(func(enc zapcore.ObjectEncoder) error {
			for k, v := range e.Details {
				switch val := v.(type) {
				case string:
					enc.AddString(k, val)
				case int:
					enc.AddInt(k, val)
				case bool:
					enc.AddBool(k, val)
				case float64:
					enc.AddFloat64(k, val)
				case time.Time:
					enc.AddTime(k, val)
				case time.Duration:
					enc.AddDuration(k, val)
				default:
					enc.AddReflected(k, val)
				}
			}
			return nil
		}))
	}

	return nil
}

// With implements zapcore.Core
func (c *testCore) With(fields []zapcore.Field) zapcore.Core {
	clone := *c
	clone.fields = make([]zapcore.Field, len(c.fields)+len(fields))
	copy(clone.fields, c.fields)
	copy(clone.fields[len(c.fields):], fields)
	return &clone
}

// Check implements zapcore.Core
func (c *testCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

// Write implements zapcore.Core
func (c *testCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	allFields := make([]zapcore.Field, 0, len(fields)+len(c.fields))
	allFields = append(allFields, c.fields...)
	allFields = append(allFields, fields...)

	// Extract startup log entry
	for _, field := range allFields {
		if field.Key == "startup_log" {
			if logEntry, ok := field.Interface.(*types.StartupLog); ok {
				entry := LogEntry{
					Phase:     logEntry.Phase,
					Component: logEntry.Component,
					Operation: logEntry.Operation,
					Details:   logEntry.Details,
					Duration:  logEntry.Duration,
					Status:    logEntry.Status,
					Error:     logEntry.Error,
					Timestamp: logEntry.Timestamp,
				}
				c.logger.Log(entry)
				break
			}
		}
	}

	buf, err := c.enc.EncodeEntry(ent, allFields)
	if err != nil {
		return err
	}
	_, err = c.syncer.Write(buf.Bytes())
	buf.Free()
	return err
}

// Sync implements zapcore.Core
func (c *testCore) Sync() error {
	return c.syncer.Sync()
}

// NewTestLoggerWithCapture creates a new test logger with log entry capture
func NewTestLoggerWithCapture() (*TestLogger, *zap.Logger) {
	testLogger := &TestLogger{
		entries: make([]LogEntry, 0),
	}

	encoderConfig := zap.NewDevelopmentEncoderConfig()
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeDuration = zapcore.StringDurationEncoder

	// Create a custom encoder that preserves the original JSON string
	enc := zapcore.NewJSONEncoder(encoderConfig)

	core := &testCore{
		LevelEnabler: zapcore.DebugLevel,
		enc:          enc,
		logger:       testLogger,
		fields:       make([]zapcore.Field, 0),
		syncer:       zapcore.AddSync(&bytes.Buffer{}),
	}
	testLogger.core = core

	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
	return testLogger, logger
}

// Log captures a log entry
func (t *TestLogger) Log(entry LogEntry) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.entries = append(t.entries, entry)
}

// GetEntries returns all captured log entries
func (t *TestLogger) GetEntries() []LogEntry {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]LogEntry{}, t.entries...)
}

// AssertPhaseOrder verifies that phases occurred in the expected order
func AssertPhaseOrder(t *testing.T, entries []LogEntry, expectedPhases []types.StartupPhase) {
	t.Helper()

	var phases []types.StartupPhase
	for _, entry := range entries {
		if entry.Phase != "" {
			phases = append(phases, entry.Phase)
		}
	}

	if len(phases) != len(expectedPhases) {
		t.Errorf("Expected %d phases, got %d", len(expectedPhases), len(phases))
		return
	}

	for i, phase := range phases {
		if phase != expectedPhases[i] {
			t.Errorf("Phase at position %d: expected %s, got %s", i, string(expectedPhases[i]), string(phase))
		}
	}
}

// AssertDuration verifies that an operation completed within expected time
func AssertDuration(t *testing.T, entries []LogEntry, operation string, maxDuration time.Duration) {
	t.Helper()

	for _, entry := range entries {
		if entry.Operation == operation {
			if entry.Duration.Duration > maxDuration {
				t.Errorf("Operation %s took %v, expected <= %v", operation, entry.Duration.Duration, maxDuration)
			}
			return
		}
	}
	t.Errorf("Operation %s not found in log entries", operation)
}

// AssertNoErrors verifies that no error entries exist in the logs
func AssertNoErrors(t *testing.T, entries []LogEntry) {
	t.Helper()

	for _, entry := range entries {
		if entry.Error != "" {
			t.Errorf("Unexpected error in log entry: %s", entry.Error)
		}
	}
}

// AssertResourceState verifies resource state transitions
func AssertResourceState(t *testing.T, entries []LogEntry, resource string, expectedState string) {
	t.Helper()

	for _, entry := range entries {
		details, ok := entry.Details["resource"].(string)
		if !ok {
			continue
		}
		if details == resource {
			state, ok := entry.Details["state"].(string)
			if !ok {
				t.Errorf("Resource %s has no state information", resource)
				return
			}
			if state != expectedState {
				t.Errorf("Resource %s state: expected %s, got %s", resource, expectedState, state)
			}
			return
		}
	}
	t.Errorf("Resource %s not found in log entries", resource)
}

// CreateTestConfig creates a test configuration
func CreateTestConfig() *types.LoggingConfig {
	return &types.LoggingConfig{
		Level:       "debug",
		Format:      "json",
		Output:      "stdout",
		StartupLogs: true,
	}
}

// WaitForOperation waits for a specific operation to complete
func WaitForOperation(entries []LogEntry, operation string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, entry := range entries {
			if entry.Operation == operation && entry.Status == "completed" {
				return true
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}
