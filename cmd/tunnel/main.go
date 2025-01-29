package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/o3willard-AI/SSSonector/internal/adapter"
	"github.com/o3willard-AI/SSSonector/internal/config"
	"github.com/o3willard-AI/SSSonector/internal/tunnel"
	"go.uber.org/zap"
)

var (
	configFile = flag.String("config", "", "Path to configuration file")
)

func main() {
	flag.Parse()

	if *configFile == "" {
		fmt.Fprintln(os.Stderr, "Error: config file path is required")
		flag.Usage()
		os.Exit(1)
	}

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
		logger.Fatal("Failed to load configuration",
			zap.Error(err),
		)
	}

	// Create network interface
	adapterManager := adapter.NewManager(logger, &cfg.Network)
	iface, err := adapterManager.CreateInterface()
	if err != nil {
		logger.Fatal("Failed to create network interface",
			zap.Error(err),
		)
	}

	// Create and start tunnel
	tun := tunnel.NewTunnel(logger, &cfg.Tunnel, iface)
	if err := tun.Start(cfg.Mode); err != nil {
		logger.Fatal("Failed to start tunnel",
			zap.Error(err),
		)
	}

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Wait for signal
	sig := <-sigChan
	logger.Info("Received signal, shutting down",
		zap.String("signal", sig.String()),
	)

	// Stop tunnel
	if err := tun.Stop(); err != nil {
		logger.Error("Failed to stop tunnel",
			zap.Error(err),
		)
	}
}
