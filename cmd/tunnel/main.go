package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/o3willard-AI/SSSonector/internal/adapter"
	"github.com/o3willard-AI/SSSonector/internal/config"
	"github.com/o3willard-AI/SSSonector/internal/monitor"
	"github.com/o3willard-AI/SSSonector/internal/tunnel"
	"go.uber.org/zap"
)

var (
	configFile string
	version    = "dev"
)

func init() {
	flag.StringVar(&configFile, "config", "config.yaml", "Path to configuration file")
}

func main() {
	flag.Parse()

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	defer logger.Sync()

	// Load configuration
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		logger.Fatal("Failed to load configuration",
			zap.String("file", configFile),
			zap.Error(err),
		)
	}

	// Initialize monitoring
	mon, err := monitor.NewMonitor(logger, cfg.Monitor)
	if err != nil {
		logger.Fatal("Failed to initialize monitoring",
			zap.Error(err),
		)
	}
	defer mon.Close()

	// Initialize network adapter
	adapterMgr, err := adapter.NewManager(logger)
	if err != nil {
		logger.Fatal("Failed to create adapter manager",
			zap.Error(err),
		)
	}

	iface, err := adapterMgr.Create(cfg.Network)
	if err != nil {
		logger.Fatal("Failed to create network interface",
			zap.Error(err),
		)
	}
	defer iface.Close()

	// Initialize tunnel
	tun, err := tunnel.NewTunnel(logger, cfg.Tunnel, iface)
	if err != nil {
		logger.Fatal("Failed to create tunnel",
			zap.Error(err),
		)
	}
	defer tun.Close()

	// Start tunnel
	if err := tun.Start(); err != nil {
		logger.Fatal("Failed to start tunnel",
			zap.Error(err),
		)
	}

	logger.Info("SSSonector started",
		zap.String("version", version),
		zap.String("mode", cfg.Mode),
		zap.String("interface", iface.GetName()),
	)

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Info("Shutting down...")
}
