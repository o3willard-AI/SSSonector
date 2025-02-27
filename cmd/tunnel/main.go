package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/o3willard-AI/SSSonector/internal/config/validator"
	"github.com/o3willard-AI/SSSonector/internal/integrity"
	"github.com/o3willard-AI/SSSonector/internal/tunnel"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	flags      = types.NewCLIFlags()
	Version    string // Set by build flag
	BuildTime  string // Set by build flag
	CommitHash string // Set by build flag
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

// getLogLevel converts a string log level to zapcore.Level
func getLogLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
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
		fmt.Printf("SSSonector %s\nBuild Time: %s\nCommit: %s\n", Version, BuildTime, CommitHash)
		os.Exit(0)
	}

	// Verify binary integrity
	binaryPath, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get executable path: %v\n", err)
		os.Exit(1)
	}

	info, err := integrity.GetFileInfo(binaryPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get binary info: %v\n", err)
		os.Exit(1)
	}

	// Always update binary info
	infoPath := binaryPath + ".info"
	infoBytes, err := json.Marshal(info)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal binary info: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(infoPath, infoBytes, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write binary info: %v\n", err)
		os.Exit(1)
	}

	// Load and verify configuration integrity
	// First verify config file permissions and integrity
	if err := integrity.FixConfigPermissions(flags.ConfigFile); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to fix config permissions: %v\n", err)
		os.Exit(1)
	}

	configInfo, err := integrity.GetFileInfo(flags.ConfigFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get config info: %v\n", err)
		os.Exit(1)
	}

	// Verify config is of correct type
	if !configInfo.IsConfig {
		fmt.Fprintf(os.Stderr, "Not a valid config file: %s\n", flags.ConfigFile)
		os.Exit(1)
	}

	// Store config info for future verification
	configInfoPath := flags.ConfigFile + ".info"
	configInfoBytes, err := json.Marshal(configInfo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal config info: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(configInfoPath, configInfoBytes, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write config info: %v\n", err)
		os.Exit(1)
	}

	// Now load the configuration
	appCfg, err := config.LoadConfig(flags.ConfigFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Create custom encoder config for proper JSON formatting
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.EpochTimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// Create logger config based on configuration file
	logConfig := zap.Config{
		Level:            zap.NewAtomicLevelAt(getLogLevel(appCfg.Config.Logging.Level)),
		Development:      flags.Debug,
		Encoding:         "json", // Force JSON encoding
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{appCfg.Config.Logging.Output},
		ErrorOutputPaths: []string{appCfg.Config.Logging.Output},
		DisableCaller:    false,
	}

	// If file output is specified, use the file path
	if appCfg.Config.Logging.Output == "file" && appCfg.Config.Logging.File != "" {
		logConfig.OutputPaths = []string{appCfg.Config.Logging.File}
		logConfig.ErrorOutputPaths = []string{appCfg.Config.Logging.File}
	}

	if flags.Debug {
		logConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	}

	// Build logger with proper JSON formatting
	logger, err := logConfig.Build(
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

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
