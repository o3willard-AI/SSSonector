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

	// Load configuration
	loader := config.NewLoader()
	cfg, err := loader.LoadFromFile(configPath)
	if err != nil {
		logger.Fatal("Failed to load configuration",
			zap.String("path", configPath),
			zap.Error(err),
		)
	}

	// Create configuration store and validator
	store := config.NewFileStore(filepath.Dir(configPath))
	validator := config.NewValidator()

	// Create configuration manager
	manager, err := config.NewManager(store, validator, logger)
	if err != nil {
		logger.Fatal("Failed to create configuration manager", zap.Error(err))
	}

	// Create AppConfig
	appCfg := config.NewAppConfig(cfg, config.TypeServer)

	// Update certificate paths
	if err := tunnel.UpdateCertificatePaths(appCfg, filepath.Dir(configPath)); err != nil {
		logger.Fatal("Failed to update certificate paths", zap.Error(err))
	}

	// Create and run tunnel
	var t interface {
		Run(context.Context) error
	}

	switch cfg.Mode {
	case config.ModeServer:
		t, err = NewServer(appCfg, manager, logger)
	case config.ModeClient:
		t, err = NewClient(appCfg, manager, logger)
	default:
		logger.Fatal("Invalid mode", zap.String("mode", string(cfg.Mode)))
	}

	if err != nil {
		logger.Fatal("Failed to create tunnel", zap.Error(err))
	}

	// Run tunnel
	if err := t.Run(ctx); err != nil {
		logger.Fatal("Failed to run tunnel", zap.Error(err))
	}
}
