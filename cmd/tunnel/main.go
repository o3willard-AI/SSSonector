package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"github.com/o3willard-AI/SSSonector/internal/tunnel"
	"go.uber.org/zap"
)

var (
	configPath string
	logger     *zap.Logger
)

func init() {
	// Parse command line flags
	flag.StringVar(&configPath, "config", "", "path to configuration file")
	flag.Parse()

	// Initialize logger
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
}

func main() {
	// Create context
	ctx := context.Background()

	// Validate config path
	if configPath == "" {
		configPath = "/etc/sssonector/config.yaml"
	}

	// Create configuration manager
	manager := config.CreateManager(filepath.Dir(configPath))

	// Copy config file to manager's directory if it's not already there
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		logger.Fatal("Failed to create config directory", zap.Error(err))
	}

	// Load configuration
	appCfg, err := manager.Get()
	if err != nil {
		logger.Fatal("Failed to load configuration",
			zap.String("path", configPath),
			zap.Error(err),
		)
	}

	// Set server type if not already set
	if appCfg.Type == "" {
		appCfg.Type = config.TypeServer
		if err := manager.Set(appCfg); err != nil {
			logger.Fatal("Failed to set configuration type", zap.Error(err))
		}
	}

	// Update certificate paths
	if err := tunnel.UpdateCertificatePaths(appCfg, filepath.Dir(configPath)); err != nil {
		logger.Fatal("Failed to update certificate paths", zap.Error(err))
	}

	// Create and run tunnel
	var t interface {
		Run(context.Context) error
	}

	if appCfg.Config == nil {
		appCfg.Config = &config.Config{Mode: string(appCfg.Type)}
	}

	switch appCfg.Config.Mode {
	case string(config.TypeServer):
		t, err = NewServer(appCfg, manager, logger)
	case string(config.TypeClient):
		t, err = NewClient(appCfg, manager, logger)
	default:
		logger.Fatal("Invalid mode", zap.String("mode", appCfg.Config.Mode))
	}

	if err != nil {
		logger.Fatal("Failed to create tunnel", zap.Error(err))
	}

	// Run tunnel
	if err := t.Run(ctx); err != nil {
		logger.Fatal("Failed to run tunnel", zap.Error(err))
	}
}
