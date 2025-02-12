package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"github.com/o3willard-AI/SSSonector/internal/service"
	"go.uber.org/zap"
)

var (
	configFile string
	debug      bool
)

func init() {
	flag.StringVar(&configFile, "config", "/etc/sssonector/config.yaml", "Path to configuration file")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")
}

func main() {
	flag.Parse()

	// Initialize logger
	var logger *zap.Logger
	var err error
	if debug {
		logger, err = zap.NewDevelopment()
	} else {
		logger, err = zap.NewProduction()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Load configuration
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		logger.Error("Failed to load configuration",
			zap.String("file", configFile),
			zap.Error(err),
		)
		os.Exit(1)
	}

	// Create and start service
	svc := service.NewBase(cfg, logger)
	if err := svc.Start(); err != nil {
		logger.Error("Service failed",
			zap.Error(err),
		)
		os.Exit(1)
	}
}
