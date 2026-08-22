// Command sssonector is the unified service binary for SSSonector.
//
// Usage:
//
//	sssonector [flags]              run using the mode from the config file
//	sssonector server [flags]       force server mode
//	sssonector client [flags]       force client mode
//	sssonector -version             print version and exit
package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"github.com/o3willard-AI/SSSonector/internal/tunnel"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Version = "dev"

	configFile = flag.String("config", "/etc/sssonector/config.yaml", "Path to config file")
	logLevel   = flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	showVer    = flag.Bool("version", false, "Print version and exit")
)

func getLogLevel(level string) zapcore.Level {
	switch strings.ToLower(level) {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

// resolveMode determines the run mode. An explicit subcommand wins; otherwise
// the mode comes from configuration. Ambiguity is fatal: this binary must
// never guess itself into a listening state.
func resolveMode(explicit string, cfg *config.AppConfig) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	if cfg.Config != nil && cfg.Config.Mode != "" {
		return cfg.Config.Mode, nil
	}
	if cfg.Type != "" {
		return string(cfg.Type), nil
	}
	return "", fmt.Errorf("run mode is not set: provide a 'server' or 'client' subcommand or set tunnel.mode in %s", *configFile)
}

func main() {
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, "Usage: sssonector [-config PATH] [-log-level LEVEL] [server|client]\n"+
			"       sssonector -version")
		flag.PrintDefaults()
	}
	flag.Parse()

	if *showVer {
		fmt.Printf("sssonector %s\n", Version)
		return
	}

	var explicitMode string
	if args := flag.Args(); len(args) > 0 {
		explicitMode = strings.ToLower(strings.TrimSpace(args[0]))
		if explicitMode == "version" {
			fmt.Printf("sssonector %s\n", Version)
			return
		}
		if explicitMode != "server" && explicitMode != "client" {
			fmt.Fprintf(os.Stderr, "unknown command %q: expected 'server' or 'client'\n", args[0])
			os.Exit(1)
		}
	}

	logConfig := zap.NewProductionConfig()
	logConfig.Level = zap.NewAtomicLevelAt(getLogLevel(*logLevel))
	logger, err := logConfig.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	cfg, err := config.LoadConfigFile(*configFile)
	if err != nil {
		logger.Error("Failed to load config", zap.Error(err))
		os.Exit(1)
	}

	mode, err := resolveMode(explicitMode, cfg)
	if err != nil {
		logger.Error("Cannot determine run mode", zap.Error(err))
		os.Exit(1)
	}

	configDir := filepath.Dir(*configFile)
	instanceDir := filepath.Join(configDir, "instances")
	instanceName := ""
	for i, part := range strings.Split(*configFile, string(filepath.Separator)) {
		if part == "instances" && i+1 < len(strings.Split(*configFile, string(filepath.Separator))) {
			instanceName = strings.Split(*configFile, string(filepath.Separator))[i+1]
			instanceDir = filepath.Join(configDir, "instances", instanceName)
			break
		}
	}
	if instanceName != "" {
		if err := tunnel.UpdateCertificatePaths(cfg, instanceDir); err != nil {
			logger.Error("Failed to update certificate paths", zap.Error(err))
			os.Exit(1)
		}
	}

	configManager := config.CreateManager(filepath.Dir(*configFile))

	var tnl interface {
		Start() error
		Stop() error
	}

	switch mode {
	case "server":
		logger.Info("Starting in server mode")
		tnl = tunnel.NewServer(cfg, configManager, logger)
	case "client":
		logger.Info("Starting in client mode")
		tnl = tunnel.NewClient(cfg, configManager, logger)
	default:
		logger.Error("Unknown mode", zap.String("mode", mode))
		os.Exit(1)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	errChan := make(chan error, 1)
	go func() {
		errChan <- tnl.Start()
	}()

	logger.Info("Service started",
		zap.String("version", Version),
		zap.String("config", *configFile),
		zap.String("mode", mode),
	)

	select {
	case sig := <-sigChan:
		logger.Info("Received signal", zap.String("signal", sig.String()))
	case err := <-errChan:
		if err != nil {
			logger.Error("Service error", zap.Error(err))
		}
	}

	if err := tnl.Stop(); err != nil {
		logger.Error("Failed to stop service", zap.Error(err))
	}

	logger.Info("Service stopped")
}
