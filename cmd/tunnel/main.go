package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"go.uber.org/zap"
)

var (
	configFile = flag.String("config", "/etc/sssonector/config.yaml", "Path to configuration file")
	logger     *zap.Logger
)

type tunneler interface {
	Start() error
	Stop() error
}

func main() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		fmt.Printf("Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	flag.Parse()

	// Load configuration
	cfg, err := config.Load(*configFile)
	if err != nil {
		logger.Fatal("Failed to load configuration",
			zap.String("path", *configFile),
			zap.Error(err),
		)
	}

	// Create tunnel instance based on mode
	var t tunneler
	switch cfg.Mode {
	case "server":
		t, err = NewServer(cfg)
	case "client":
		t, err = NewClient(cfg)
	default:
		logger.Fatal("Invalid mode",
			zap.String("mode", cfg.Mode),
			zap.String("valid_modes", "server, client"),
		)
	}

	if err != nil {
		logger.Fatal("Failed to create tunnel instance",
			zap.String("mode", cfg.Mode),
			zap.Error(err),
		)
	}

	// Start tunnel
	if err := t.Start(); err != nil {
		logger.Fatal("Failed to start tunnel",
			zap.String("mode", cfg.Mode),
			zap.Error(err),
		)
	}

	logger.Info("Tunnel started",
		zap.String("mode", cfg.Mode),
		zap.String("interface", cfg.Network.Interface),
	)

	// Handle shutdown signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutting down...")

	// Stop tunnel
	if err := t.Stop(); err != nil {
		logger.Error("Failed to stop tunnel cleanly",
			zap.String("mode", cfg.Mode),
			zap.Error(err),
		)
		os.Exit(1)
	}

	logger.Info("Shutdown complete")
}
