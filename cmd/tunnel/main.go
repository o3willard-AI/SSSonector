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
	flags   = types.NewCLIFlags()
	Version string // Set by build flag
)

func init() {
	flag.StringVar(&flags.ConfigFile, "config", flags.ConfigFile, "Path to configuration file")
	flag.BoolVar(&flags.Debug, "debug", flags.Debug, "Enable debug logging")
	flag.BoolVar(&flags.Version, "version", flags.Version, "Show version information")
	flag.BoolVar(&flags.Help, "help", flags.Help, "Show help information")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\nOptions:\n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	// Show help and exit if requested
	if flags.Help {
		flag.Usage()
		os.Exit(0)
	}

	// Show version and exit if requested
	if flags.Version {
		fmt.Printf("SSSonector %s\n", Version)
		os.Exit(0)
	}

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
		Development:      flags.Debug,
		Encoding:         "console",
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
		DisableCaller:    true,
	}

	if flags.Debug {
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
	appCfg, err := config.LoadConfig(flags.ConfigFile)
	if err != nil {
		logger.Error("Failed to load configuration",
			zap.String("file", flags.ConfigFile),
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
