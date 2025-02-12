package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

type command struct {
	Name        string
	Description string
	Run         func(ctx context.Context, args []string) error
}

func main() {
	// Set up logging
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Set up command line flags
	var (
		showHelp   = flag.Bool("help", false, "Show help")
		configFile = flag.String("config", "", "Path to config file")
		format     = flag.String("format", "text", "Output format (text/json)")
		timeout    = flag.Duration("timeout", 30*time.Second, "Command timeout")
	)
	flag.Parse()

	if *showHelp {
		printHelp()
		return
	}

	// Load configuration
	config := defaultConfig
	if *configFile != "" {
		if err := loadConfig(*configFile, &config); err != nil {
			logger.Fatal("Failed to load config",
				zap.String("file", *configFile),
				zap.Error(err))
		}
	}

	// Set up commands
	commands := map[string]command{
		"status": {
			Name:        "status",
			Description: "Show system status",
			Run:         cmdStatus,
		},
		"connections": {
			Name:        "connections",
			Description: "List active connections",
			Run:         cmdConnections,
		},
		"endpoints": {
			Name:        "endpoints",
			Description: "Manage load balancer endpoints",
			Run:         cmdEndpoints,
		},
		"circuit-breaker": {
			Name:        "circuit-breaker",
			Description: "Show circuit breaker status",
			Run:         cmdCircuitBreaker,
		},
		"rate-limits": {
			Name:        "rate-limits",
			Description: "Show rate limit status",
			Run:         cmdRateLimits,
		},
		"pool": {
			Name:        "pool",
			Description: "Show connection pool status",
			Run:         cmdPool,
		},
		"config": {
			Name:        "config",
			Description: "Show/update configuration",
			Run: func(ctx context.Context, args []string) error {
				return cmdConfig(ctx, args, &config, *configFile)
			},
		},
		"health": {
			Name:        "health",
			Description: "Run health checks",
			Run:         cmdHealth,
		},
		"debug": {
			Name:        "debug",
			Description: "Enable/disable debug mode",
			Run:         cmdDebug,
		},
		"maintenance": {
			Name:        "maintenance",
			Description: "Enter/exit maintenance mode",
			Run:         cmdMaintenance,
		},
	}

	// Get command from args
	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Command required")
		printHelp()
		os.Exit(1)
	}

	cmdName := flag.Arg(0)
	cmd, ok := commands[cmdName]
	if !ok {
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmdName)
		printHelp()
		os.Exit(1)
	}

	// Set up context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	// Handle interrupts
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Run command
	if err := cmd.Run(ctx, flag.Args()[1:]); err != nil {
		if *format == "json" {
			json.NewEncoder(os.Stderr).Encode(map[string]string{
				"error": err.Error(),
			})
		} else {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		}
		os.Exit(1)
	}
}

func loadConfig(path string, config *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	switch ext := filepath.Ext(path); ext {
	case ".yaml", ".yml":
		if err := yaml.Unmarshal(data, config); err != nil {
			return fmt.Errorf("failed to parse YAML config: %w", err)
		}
	case ".json":
		if err := json.Unmarshal(data, config); err != nil {
			return fmt.Errorf("failed to parse JSON config: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}

	return nil
}

func saveConfig(path string, config *Config) error {
	var (
		data []byte
		err  error
	)

	switch ext := filepath.Ext(path); ext {
	case ".yaml", ".yml":
		data, err = yaml.Marshal(config)
		if err != nil {
			return fmt.Errorf("failed to marshal YAML config: %w", err)
		}
	case ".json":
		data, err = json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON config: %w", err)
		}
	default:
		return fmt.Errorf("unsupported config file format: %s", ext)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func printHelp() {
	fmt.Println("Usage: admin [options] command [args...]")
	fmt.Println("\nOptions:")
	flag.PrintDefaults()
	fmt.Println("\nCommands:")
	fmt.Println("  status           Show system status")
	fmt.Println("  connections      List active connections")
	fmt.Println("  endpoints        Manage load balancer endpoints")
	fmt.Println("  circuit-breaker  Show circuit breaker status")
	fmt.Println("  rate-limits      Show rate limit status")
	fmt.Println("  pool            Show connection pool status")
	fmt.Println("  config          Show/update configuration")
	fmt.Println("  health          Run health checks")
	fmt.Println("  debug           Enable/disable debug mode")
	fmt.Println("  maintenance     Enter/exit maintenance mode")
}
