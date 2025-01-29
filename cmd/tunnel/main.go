package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"SSSonector/internal/config"
	"SSSonector/internal/monitor"
	"SSSonector/internal/tunnel"

	"go.uber.org/zap"
)

var (
	configFile = flag.String("config", "", "Path to configuration file")
	version    = "0.1.0"
)

func main() {
	flag.Parse()

	if *configFile == "" {
		log.Fatal("Configuration file is required")
	}

	fmt.Println("Starting SSL Tunnel application...")

	// Initialize default logging
	if err := monitor.InitLogging(); err != nil {
		log.Fatalf("Failed to initialize logging: %v", err)
	}
	logger := monitor.GetLogger()
	defer monitor.Sync()

	fmt.Println("Loading configuration...")
	// Load configuration
	cfg, err := config.Load(*configFile)
	if err != nil {
		logger.Fatal("Failed to load configuration", zap.Error(err))
	}

	fmt.Println("Configuring logging...")
	// Configure logging with loaded config
	if err := monitor.ConfigureLogging(&cfg.Logging); err != nil {
		logger.Fatal("Failed to configure logging", zap.Error(err))
	}
	logger = monitor.GetLogger() // Get new logger after configuration

	logger.Info("Starting SSL Tunnel", zap.String("version", version))
	fmt.Printf("Mode: %s\n", cfg.Mode)

	// Create and start tunnel
	t, err := tunnel.NewTunnel(logger, cfg)
	if err != nil {
		logger.Fatal("Failed to create tunnel", zap.Error(err))
	}

	// Start tunnel in background
	if err := t.Start(); err != nil {
		logger.Fatal("Failed to start tunnel", zap.Error(err))
	}

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Printf("SSL Tunnel %s running in %s mode\n", version, cfg.Mode)
	fmt.Printf("Press Ctrl+C to shutdown\n")

	// Wait for shutdown signal
	sig := <-sigChan
	fmt.Printf("Received signal %v, shutting down...\n", sig)

	// Stop tunnel gracefully
	if err := t.Stop(); err != nil {
		logger.Error("Error stopping tunnel", zap.Error(err))
	}
}
