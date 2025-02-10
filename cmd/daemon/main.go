package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"github.com/o3willard-AI/SSSonector/internal/service"
	"github.com/o3willard-AI/SSSonector/internal/service/control"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Version is set during build
	Version = "dev"

	// Command line flags
	configFile = flag.String("config", "/etc/sssonector/config.yaml", "Path to config file")
	socketPath = flag.String("socket", "/var/run/sssonector.sock", "Path to control socket")
	logLevel   = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
)

func getLogLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
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
	// Parse command line flags
	flag.Parse()

	// Initialize logger
	logConfig := zap.NewProductionConfig()
	logConfig.Level = zap.NewAtomicLevelAt(getLogLevel(*logLevel))
	logger, err := logConfig.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Initialize configuration manager
	configManager := config.CreateManager(*configFile)

	// Get configuration
	cfg, err := configManager.Get()
	if err != nil {
		logger.Error("Failed to get config", zap.Error(err))
		os.Exit(1)
	}

	// Create service
	svc, err := service.NewBaseService(cfg, service.ServiceOptions{
		Name:      "sssonector",
		ConfigDir: "/etc/sssonector",
		DataDir:   "/var/lib/sssonector",
		LogDir:    "/var/log/sssonector",
	})
	if err != nil {
		logger.Error("Failed to create service", zap.Error(err))
		os.Exit(1)
	}

	// Create control server
	controlServer, err := control.NewControlServer(svc)
	if err != nil {
		logger.Error("Failed to create control server", zap.Error(err))
		os.Exit(1)
	}

	// Set socket path and start control server
	controlServer.SetSocketPath(*socketPath)
	if err := controlServer.Start(); err != nil {
		logger.Error("Failed to start control server", zap.Error(err))
		os.Exit(1)
	}
	defer controlServer.Stop()

	// Start service
	if err := svc.Start(); err != nil {
		logger.Error("Failed to start service", zap.Error(err))
		os.Exit(1)
	}
	defer svc.Stop()

	// Log startup
	logger.Info("Service started",
		zap.String("version", Version),
		zap.String("config", *configFile),
		zap.String("socket", *socketPath),
		zap.String("log_level", *logLevel),
	)

	// Wait for signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Handle signals
	sig := <-sigChan
	logger.Info("Received signal", zap.String("signal", sig.String()))

	// Stop service
	if err := svc.Stop(); err != nil {
		logger.Error("Failed to stop service", zap.Error(err))
		os.Exit(1)
	}

	logger.Info("Service stopped")
}
