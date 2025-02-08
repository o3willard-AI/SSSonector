package monitor

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// LogLevel represents different logging levels
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

// Logger handles structured logging for the SNMP agent
type Logger struct {
	level    LogLevel
	debugLog *log.Logger
	infoLog  *log.Logger
	warnLog  *log.Logger
	errorLog *log.Logger
	mu       sync.Mutex
	logFile  *os.File
}

// NewLogger creates a new logger with the specified level
func NewLogger(level LogLevel, logPath string) (*Logger, error) {
	var logFile *os.File
	var err error

	if logPath != "" {
		logFile, err = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
	}

	flags := log.Ldate | log.Ltime | log.LUTC | log.Lmicroseconds

	l := &Logger{
		level:    level,
		logFile:  logFile,
		debugLog: log.New(logFile, "DEBUG: ", flags),
		infoLog:  log.New(logFile, "INFO:  ", flags),
		warnLog:  log.New(logFile, "WARN:  ", flags),
		errorLog: log.New(logFile, "ERROR: ", flags),
	}

	// If no log file specified, use stdout/stderr
	if logFile == nil {
		l.debugLog.SetOutput(os.Stdout)
		l.infoLog.SetOutput(os.Stdout)
		l.warnLog.SetOutput(os.Stderr)
		l.errorLog.SetOutput(os.Stderr)
	}

	return l, nil
}

// Close closes the log file if one was opened
func (l *Logger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Only sync the log file if we have one
	if l.logFile != nil {
		return l.logFile.Sync()
	}

	// When using stdout/stderr, no need to sync
	return nil
}

// Debug logs a debug message
func (l *Logger) Debug(format string, v ...interface{}) {
	if l.level <= DEBUG {
		l.mu.Lock()
		l.debugLog.Printf(format, v...)
		l.mu.Unlock()
	}
}

// Info logs an info message
func (l *Logger) Info(format string, v ...interface{}) {
	if l.level <= INFO {
		l.mu.Lock()
		l.infoLog.Printf(format, v...)
		l.mu.Unlock()
	}
}

// Warn logs a warning message
func (l *Logger) Warn(format string, v ...interface{}) {
	if l.level <= WARN {
		l.mu.Lock()
		l.warnLog.Printf(format, v...)
		l.mu.Unlock()
	}
}

// Error logs an error message
func (l *Logger) Error(format string, v ...interface{}) {
	if l.level <= ERROR {
		l.mu.Lock()
		l.errorLog.Printf(format, v...)
		l.mu.Unlock()
	}
}

// SetLevel changes the logging level
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	l.level = level
	l.mu.Unlock()
}

// GetLevel returns the current logging level
func (l *Logger) GetLevel() LogLevel {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.level
}

// FormatTime formats a time value consistently
func (l *Logger) FormatTime(t time.Time) string {
	return t.UTC().Format(time.RFC3339Nano)
}
