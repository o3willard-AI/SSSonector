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
	"github.com/o3willard-AI/SSSonector/internal/monitor"
	"github.com/o3willard-AI/SSSonector/internal/tunnel"
	"go.uber.org/zap"
)

var (
	Version = "dev"

	configFile = flag.String("config", "/etc/sssonector/config.yaml", "Path to config file")
	logLevel   = flag.String("log-level", "", "Log level override (debug, info, warn, error, fatal); defaults to logging.level from config, or info")
	showVer    = flag.Bool("version", false, "Print version and exit")

	activeLogLevel *zap.AtomicLevel
)

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

	logLevelSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "log-level" {
			logLevelSet = true
		}
	})

	if *showVer {
		fmt.Printf("sssonector %s\n", Version)
		return
	}

	var explicitMode string
	if args := flag.Args(); len(args) > 0 {
		if args[0] == "provision" {
			// Provisioning subcommands bypass the service lifecycle entirely.
			if err := runProvision(args[1:]); err != nil {
				fmt.Fprintf(os.Stderr, "provision: %v\n", err)
				os.Exit(1)
			}
			return
		}
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

	bootLevel, err := resolveLogLevel(*logLevel, logLevelSet, "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid -log-level: %v\n", err)
		os.Exit(1)
	}
	bootLogger, _, err := buildLogger(bootLevel, "json", "")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize bootstrap logger: %v\n", err)
		os.Exit(1)
	}
	defer bootLogger.Sync()

	cfg, err := config.LoadConfigFile(*configFile)
	if err != nil {
		bootLogger.Error("Failed to load config",
			zap.String("config", *configFile),
			zap.Error(err))
		os.Exit(1)
	}
	if err := config.NewValidator().Validate(cfg); err != nil {
		bootLogger.Error("Config validation failed",
			zap.String("config", *configFile),
			zap.Error(err))
		os.Exit(1)
	}

	cfgLevel, cfgFormat, cfgFile := "", "", ""
	if cfg.Config != nil {
		cfgLevel = cfg.Config.Logging.Level
		cfgFormat = cfg.Config.Logging.Format
		cfgFile = cfg.Config.Logging.File
	}

	logLevelResolved, err := resolveLogLevel(*logLevel, logLevelSet, cfgLevel)
	if err != nil {
		bootLogger.Error("Invalid logging.level in config", zap.Error(err))
		os.Exit(1)
	}

	logger, logLevelAtom, err := buildLogger(logLevelResolved, cfgFormat, cfgFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()
	activeLogLevel = logLevelAtom
	effectiveFormat := cfgFormat
	if effectiveFormat == "" {
		effectiveFormat = "json"
	}
	logger.Info("Logging configured",
		zap.String("level", logLevelResolved.String()),
		zap.String("format", effectiveFormat),
		zap.String("file", cfgFile),
	)

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

	var mon *monitor.Monitor
	if cfg.Config != nil && (cfg.Config.Monitor.Enabled || cfg.Config.SNMP.Enabled || cfg.Config.Metrics.Enabled) {
		monCfg := &monitor.Config{
			SNMPEnabled:       cfg.Config.SNMP.Enabled,
			SNMPAddress:       cfg.Config.SNMP.Address,
			SNMPPort:          cfg.Config.SNMP.Port,
			SNMPCommunity:     cfg.Config.SNMP.Community,
			PromEnabled:       cfg.Config.Monitor.Prometheus.Enabled && strings.EqualFold(cfg.Config.Monitor.Type, "prometheus"),
			PromPort:          cfg.Config.Monitor.Prometheus.Port,
			PromPath:          cfg.Config.Monitor.Prometheus.Path,
			PromListenAddress: cfg.Config.Monitor.Prometheus.ListenAddress,
		}
		if monCfg.SNMPAddress == "" {
			monCfg.SNMPAddress = "0.0.0.0"
		}
		if monCfg.SNMPCommunity == "" {
			monCfg.SNMPCommunity = "public"
		}

		m, err := monitor.New(logger.Named("monitor"), monCfg)
		if err != nil {
			logger.Error("Failed to create monitor", zap.Error(err))
			os.Exit(1)
		}
		if err := m.Start(); err != nil {
			logger.Warn("Monitor started in degraded state", zap.Error(err))
		}
		mon = m
		defer func() {
			if mon != nil {
				mon.Stop()
			}
		}()
	}

	var tnl interface {
		Start() error
		Stop() error
	}

	switch mode {
	case "server":
		logger.Info("Starting in server mode")
		tnl = tunnel.NewServer(cfg, configManager, logger, mon)
	case "client":
		logger.Info("Starting in client mode")
		tnl = tunnel.NewClient(cfg, configManager, logger, mon)
	default:
		logger.Error("Unknown mode", zap.String("mode", mode))
		os.Exit(1)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	prevLogLevel := logLevelResolved
	var reloadable tunnel.Reloadable
	if r, ok := tnl.(tunnel.Reloadable); ok {
		reloadable = r
	}
	reload := func() {
		prevLogLevel = applyReload(logger, *configFile, *logLevel, logLevelSet, prevLogLevel, activeLogLevel, reloadable)
	}

	hupChan := make(chan os.Signal, 1)
	signal.Notify(hupChan, syscall.SIGHUP)
	go func() {
		for range hupChan {
			reload()
		}
	}()

	errChan := make(chan error, 1)
	go func() {
		// Start returns nil once its workers are running; only a real
		// failure wakes the shutdown path. A nil would otherwise make the
		// daemon exit cleanly the instant it finished starting.
		if err := tnl.Start(); err != nil {
			errChan <- err
		}
	}()

	// A client that exhausts its reconnect schedule is fatal: surface it as
	// a non-zero exit so Restart=on-failure revives the unit.
	if c, ok := tnl.(*tunnel.Client); ok {
		go func() {
			select {
			case <-c.GiveUpChan():
				errChan <- fmt.Errorf("reconnect attempts exhausted")
			case <-sigChan:
			}
		}()
	}

	logger.Info("Service started",
		zap.String("version", Version),
		zap.String("config", *configFile),
		zap.String("mode", mode),
	)

	exitCode := 0
	select {
	case sig := <-sigChan:
		logger.Info("Received signal", zap.String("signal", sig.String()))
	case err := <-errChan:
		if err != nil {
			logger.Error("Service error", zap.Error(err))
			exitCode = 1
		}
	}

	if err := tnl.Stop(); err != nil {
		logger.Error("Failed to stop service", zap.Error(err))
	}

	logger.Info("Service stopped")
	_ = logger.Sync()
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}
