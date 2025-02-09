package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/service"
	"github.com/o3willard-AI/SSSonector/internal/service/control"
)

var (
	network = flag.String("network", "unix", "Network type (unix, tcp)")
	address = flag.String("address", "/var/run/sssonector.sock", "Socket address")
	command = flag.String("command", "status", "Command to execute")
)

func main() {
	flag.Parse()

	// Create client
	client, err := control.NewClient(*network, *address)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	// Execute command
	var response string
	switch *command {
	case "status":
		response, err = client.GetStatus()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting status: %v\n", err)
			os.Exit(1)
		}
		printStatus(response)

	case "metrics":
		metrics, err := client.GetMetrics()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting metrics: %v\n", err)
			os.Exit(1)
		}
		printMetrics(metrics)

	case "health":
		err = client.CheckHealth()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error checking health: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Service is healthy")

	case "reload":
		err = client.Reload()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reloading service: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Service reloaded successfully")

	case "rotate-certs":
		err = client.RotateCerts()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error rotating certificates: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Certificates rotated successfully")

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", *command)
		os.Exit(1)
	}
}

func printStatus(status string) {
	var v service.ServiceStatus
	err := json.Unmarshal([]byte(status), &v)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing status: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Status: %s\n", v.State)
	fmt.Printf("PID: %d\n", v.PID)
	fmt.Printf("Uptime: %s\n", v.Uptime)
	fmt.Printf("Memory: %s\n", v.Memory)
	fmt.Printf("CPU: %s\n", v.CPU)
	fmt.Printf("Restarts: %d\n", v.Restarts)
	fmt.Printf("Start Time: %s\n", v.StartTime.Format(time.RFC3339))
	fmt.Printf("Last Reload: %s\n", v.LastReload.Format(time.RFC3339))
	fmt.Printf("Connections: %d\n", v.Connections)
	fmt.Printf("Platform: %s\n", v.Platform)
	if v.Error != "" {
		fmt.Printf("Last Error: %s\n", v.Error)
	}
}

func printMetrics(v *service.ServiceMetrics) {
	fmt.Printf("CPU Usage: %.2f%%\n", v.CPUUsage)
	fmt.Printf("Memory Usage: %d bytes\n", v.MemoryUsage)
	fmt.Printf("Connections: %d\n", v.ConnectionCount)
	fmt.Printf("Bytes Received: %d\n", v.BytesReceived)
	fmt.Printf("Bytes Sent: %d\n", v.BytesSent)
	fmt.Printf("Errors: %d\n", v.ErrorCount)
	if v.LastError != "" {
		fmt.Printf("Last Error: %s\n", v.LastError)
	}
	fmt.Printf("Uptime: %d seconds\n", v.UptimeSeconds)
	fmt.Printf("Start Time: %s\n", v.StartTime.Format(time.RFC3339))
	fmt.Printf("Last Reload: %s\n", v.LastReload.Format(time.RFC3339))
	fmt.Printf("Platform: %s\n", v.Platform)
}
