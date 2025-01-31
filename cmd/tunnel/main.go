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
configPath string
)

func init() {
flag.StringVar(&configPath, "config", "/etc/sssonector/config.yaml", "Path to configuration file")
}

func main() {
flag.Parse()

// Initialize logger
logger, err := zap.NewProduction()
if err != nil {
fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
os.Exit(1)
}
defer logger.Sync()

// Load configuration
cfg, err := config.LoadConfig(configPath)
if err != nil {
logger.Fatal("Failed to load configuration",
zap.String("path", configPath),
zap.Error(err),
)
}

logger.Info("Starting SSSonector",
zap.String("mode", cfg.Mode),
zap.String("interface", cfg.Network.Interface),
)

// Set up signal handling
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

// Initialize components based on mode
var tunnel interface {
Start() error
Stop() error
}

if cfg.Mode == "server" {
tunnel, err = initializeServer(logger, cfg)
} else {
tunnel, err = initializeClient(logger, cfg)
}

if err != nil {
logger.Fatal("Failed to initialize tunnel",
zap.String("mode", cfg.Mode),
zap.Error(err),
)
}

// Start the tunnel
if err := tunnel.Start(); err != nil {
logger.Fatal("Failed to start tunnel",
zap.Error(err),
)
}

// Wait for shutdown signal
sig := <-sigChan
logger.Info("Received shutdown signal",
zap.String("signal", sig.String()),
)

// Graceful shutdown
if err := tunnel.Stop(); err != nil {
logger.Error("Error during shutdown",
zap.Error(err),
)
os.Exit(1)
}

logger.Info("Shutdown complete")
}

type serverTunnel struct {
*tunnel.Tunnel
}

func (s *serverTunnel) Start() error {
return s.Tunnel.Start("server")
}

type clientTunnel struct {
*tunnel.Tunnel
}

func (c *clientTunnel) Start() error {
return c.Tunnel.Start("client")
}

func initializeServer(logger *zap.Logger, cfg *config.Config) (interface {
Start() error
Stop() error
}, error) {
// Create network interface
ifaceManager, err := adapter.NewManager(&adapter.Config{
Name:    cfg.Network.Interface,
Address: cfg.Network.Address,
MTU:     cfg.Network.MTU,
})
if err != nil {
return nil, fmt.Errorf("failed to create interface: %w", err)
}

// Create tunnel
t := tunnel.NewTunnel(logger, &cfg.Tunnel, ifaceManager.GetInterface())
return &serverTunnel{Tunnel: t}, nil
}

func initializeClient(logger *zap.Logger, cfg *config.Config) (interface {
Start() error
Stop() error
}, error) {
// Create network interface
ifaceManager, err := adapter.NewManager(&adapter.Config{
Name:    cfg.Network.Interface,
Address: cfg.Network.Address,
MTU:     cfg.Network.MTU,
})
if err != nil {
return nil, fmt.Errorf("failed to create interface: %w", err)
}

// Create tunnel
t := tunnel.NewTunnel(logger, &cfg.Tunnel, ifaceManager.GetInterface())
return &clientTunnel{Tunnel: t}, nil
}
