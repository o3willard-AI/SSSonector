package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func parseLevel(s string) (zapcore.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return zapcore.DebugLevel, nil
	case "info":
		return zapcore.InfoLevel, nil
	case "warn", "warning":
		return zapcore.WarnLevel, nil
	case "error":
		return zapcore.ErrorLevel, nil
	case "fatal":
		return zapcore.FatalLevel, nil
	default:
		return zapcore.InfoLevel, fmt.Errorf("invalid log level %q (expected debug, info, warn, error, or fatal)", s)
	}
}

func normalizeFormat(s string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "":
		return "json", nil
	case "json":
		return "json", nil
	case "console":
		return "console", nil
	default:
		return "", fmt.Errorf("invalid log format %q (expected json or console)", s)
	}
}

func resolveLogLevel(flagValue string, flagSet bool, configLevel string) (zapcore.Level, error) {
	if flagSet {
		return parseLevel(flagValue)
	}
	if configLevel != "" {
		return parseLevel(configLevel)
	}
	return zapcore.InfoLevel, nil
}

func buildLogger(level zapcore.Level, format, file string) (*zap.Logger, *zap.AtomicLevel, error) {
	format, err := normalizeFormat(format)
	if err != nil {
		return nil, nil, err
	}

	atomic := zap.NewAtomicLevelAt(level)

	enc := zap.NewProductionEncoderConfig()
	enc.EncodeTime = zapcore.ISO8601TimeEncoder
	if format == "console" {
		enc = zap.NewDevelopmentEncoderConfig()
		enc.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	outputPaths := []string{"stdout"}
	if file != "" {
		if dir := filepath.Dir(file); dir != "" && dir != "." {
			if mkErr := os.MkdirAll(dir, 0o750); mkErr != nil {
				return nil, nil, fmt.Errorf("failed to create log directory %s: %w", dir, mkErr)
			}
		}
		outputPaths = []string{file, "stderr"}
	}

	cfg := zap.Config{
		Level:            atomic,
		Development:      false,
		Encoding:         format,
		EncoderConfig:    enc,
		OutputPaths:      outputPaths,
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := cfg.Build()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize logger: %w", err)
	}
	return logger, &atomic, nil
}
