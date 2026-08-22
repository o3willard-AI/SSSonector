package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"go.uber.org/zap"
)

func main() {
	configPath := flag.String("config", "", "path to configuration file")
	flag.Parse()

	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	if *configPath == "" {
		*configPath = "/etc/sssonector/config.yaml"
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	manager := config.CreateManager(filepath.Dir(*configPath))

	appCfg, err := manager.Get()
	if err != nil {
		logger.Fatal("Failed to load configuration",
			zap.String("path", *configPath),
			zap.Error(err),
		)
	}

	// The mode must come from configuration. Refuse to guess: silently
	// defaulting to server mode would start a listener on a misconfigured
	// client host.
	mode := ""
	if appCfg.Config != nil {
		mode = appCfg.Config.Mode
	}
	if mode == "" {
		mode = string(appCfg.Type)
	}

	var t interface {
		Run(context.Context) error
	}

	switch mode {
	case string(config.TypeServer):
		t, err = NewServer(appCfg, manager, logger)
	case string(config.TypeClient):
		t, err = NewClient(appCfg, manager, logger)
	default:
		logger.Fatal("configuration mode is required (set tunnel.mode or type to server|client)",
			zap.String("mode", mode),
		)
	}

	if err != nil {
		logger.Fatal("Failed to create tunnel", zap.Error(err))
	}

	if err := t.Run(ctx); err != nil && ctx.Err() == nil {
		logger.Fatal("Failed to run tunnel", zap.Error(err))
	}

	logger.Info("Service stopped")
}
