package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/o3willard-AI/SSSonector/internal/config/store"
	"github.com/o3willard-AI/SSSonector/internal/config/types"
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

	// Create configuration store
	configStore := store.NewFileStore(filepath.Dir(configPath))

	// Load configuration
	appCfg, err := configStore.Load()
	if err != nil {
		logger.Fatal("Failed to load configuration",
			zap.String("path", configPath),
			zap.Error(err),
		)
	}

	// Set server type if not already set
	if appCfg.Type == "" {
		appCfg.Type = types.TypeServer
		if err := configStore.Store(appCfg); err != nil {
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
		appCfg.Config = &types.Config{Mode: string(appCfg.Type)}
	}

	switch appCfg.Config.Mode {
	case string(types.TypeServer):
		t, err = NewServer(appCfg, configStore, logger)
	case string(types.TypeClient):
		t, err = NewClient(appCfg, configStore, logger)
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
