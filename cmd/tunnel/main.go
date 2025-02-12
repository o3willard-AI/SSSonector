package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/o3willard-AI/SSSonector/internal/config/validator"
	"github.com/o3willard-AI/SSSonector/internal/tunnel"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	configFile string
	debug      bool
)

func init() {
	flag.StringVar(&configFile, "config", "/etc/sssonector/config.yaml", "Path to configuration file")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
}

func main() {
	flag.Parse()

	// Initialize logger
	var logger *zap.Logger
	var err error

	// Create custom encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "",
		LevelKey:       "",
		NameKey:        "",
		CallerKey:      "",
		MessageKey:     "msg",
		StacktraceKey:  "",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create custom config
	logConfig := zap.Config{
		Level:            zap.NewAtomicLevelAt(zap.InfoLevel),
		Development:      debug,
		Encoding:         "console",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		DisableCaller:    true,
	}

	if debug {
		logConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	// Build logger
	logger, err = logConfig.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Load configuration
	appCfg, err := config.LoadConfig(configFile)
	if err != nil {
		logger.Error("Failed to load configuration",
			zap.String("file", configFile),
			zap.Error(err),
		)
		os.Exit(1)
	}

	// Validate configuration
	v := validator.NewValidator()
	if err := v.Validate(appCfg); err != nil {
		logger.Error("Invalid configuration",
			zap.Error(err),
		)
		os.Exit(1)
	}

	// Log configuration
	logger.Info("Starting tunnel",
		zap.String("mode", appCfg.Config.Mode.String()),
		zap.String("version", appCfg.Version),
	)

	// Create tunnel based on mode
	var t tunnel.Tunnel
	switch appCfg.Config.Mode {
	case types.ModeServer:
		t = tunnel.NewServer(appCfg, nil, logger)
	case types.ModeClient:
		t = tunnel.NewClient(appCfg, nil, logger)
	default:
		logger.Error("Invalid mode",
			zap.String("mode", appCfg.Config.Mode.String()),
		)
		os.Exit(1)
	}

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start tunnel
	if err := t.Start(); err != nil {
		logger.Error("Failed to start tunnel",
			zap.Error(err),
		)
		os.Exit(1)
	}

	// Wait for signal
	<-sigChan

	// Stop tunnel
	if err := t.Stop(); err != nil {
		logger.Error("Failed to stop tunnel",
			zap.Error(err),
		)
		os.Exit(1)
	}

	logger.Info("Tunnel stopped")
}
