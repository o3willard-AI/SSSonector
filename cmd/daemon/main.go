package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"github.com/o3willard-AI/SSSonector/internal/service"
	"github.com/o3willard-AI/SSSonector/internal/service/control"
	"go.uber.org/zap"
)

var (
	// Version is set during build
	Version = "dev"

	// Command line flags
	configFile = flag.String("config", "/etc/sssonector/config.json", "Path to config file")
	socketPath = flag.String("socket", "/var/run/sssonector.sock", "Path to control socket")
)

func main() {
	// Parse command line flags
	flag.Parse()

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Load configuration
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		logger.Error("Failed to load config", zap.Error(err))
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
