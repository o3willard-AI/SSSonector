package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/o3willard-AI/SSSonector/internal/service"
	"github.com/o3willard-AI/SSSonector/internal/service/control"
	"go.uber.org/zap"
)

var (
	// Version is set during build
	Version = "dev"

	// Command line flags
	socketPath = flag.String("socket", "/var/run/sssonector.sock", "Path to control socket")
	jsonOutput = flag.Bool("json", false, "Output in JSON format")
)

func main() {
	// Parse command line flags
	flag.Parse()

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Create control client
	client, err := control.NewClient(nil, logger)
	if err != nil {
		logger.Error("Failed to create control client", zap.Error(err))
		os.Exit(1)
	}

	// Set socket path
	client.SetSocketPath(*socketPath)

	// Get command
	args := flag.Args()
	if len(args) < 1 {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <command>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nCommands:\n")
		fmt.Fprintf(os.Stderr, "  status    Get service status\n")
		fmt.Fprintf(os.Stderr, "  metrics   Get service metrics\n")
		fmt.Fprintf(os.Stderr, "  health    Check service health\n")
		fmt.Fprintf(os.Stderr, "  start     Start service\n")
		fmt.Fprintf(os.Stderr, "  stop      Stop service\n")
		fmt.Fprintf(os.Stderr, "  reload    Reload configuration\n")
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
		os.Exit(1)
	}

	// Map command to ServiceCommand
	var cmd service.ServiceCommand
	switch args[0] {
	case "status":
		cmd = service.CmdStatus
	case "metrics":
		cmd = service.CmdMetrics
	case "health":
		cmd = service.CmdHealth
	case "start":
		cmd = service.CmdStart
	case "stop":
		cmd = service.CmdStop
	case "reload":
		cmd = service.CmdReload
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", args[0])
		os.Exit(1)
	}

	// Execute command
	resp, err := client.ExecuteCommand(cmd, nil)
	if err != nil {
		logger.Error("Command failed", zap.Error(err))
		os.Exit(1)
	}

	// Output response
	if *jsonOutput {
		// JSON output
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(resp); err != nil {
			logger.Error("Failed to encode response", zap.Error(err))
			os.Exit(1)
		}
	} else {
		// Human-readable output
		if resp.Success {
			if resp.Message != "" {
				fmt.Println(resp.Message)
			}
			if resp.Data != nil {
				data, err := json.MarshalIndent(resp.Data, "", "  ")
				if err != nil {
					logger.Error("Failed to marshal data", zap.Error(err))
					os.Exit(1)
				}
				fmt.Println(string(data))
			}
		} else {
			fmt.Fprintf(os.Stderr, "Error: %s\n", resp.Message)
			os.Exit(1)
		}
	}
}
