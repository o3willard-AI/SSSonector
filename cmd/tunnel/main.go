package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/o3willard-AI/SSSonector/internal/config"
)

var (
	configFile      string
	testWithoutCert bool
	genCertsOnly    bool
	keyFile         string
	keygen          bool
	validateCerts   bool
)

func init() {
	flag.StringVar(&configFile, "config", "", "Path to configuration file")
	flag.BoolVar(&testWithoutCert, "test-without-certs", false, "Run with temporary certificates")
	flag.BoolVar(&genCertsOnly, "generate-certs-only", false, "Generate certificates without starting service")
	flag.StringVar(&keyFile, "keyfile", "", "Specify certificate directory")
	flag.BoolVar(&keygen, "keygen", false, "Generate production certificates")
	flag.BoolVar(&validateCerts, "validate-certs", false, "Validate existing certificates")
}

func main() {
	flag.Parse()

	// Load configuration
	if configFile == "" {
		configFile = filepath.Join(os.Getenv("HOME"), ".config", "sssonector", "config.yaml")
	}

	cfg, loadErr := config.LoadConfig(configFile)
	if loadErr != nil {
		log.Fatalf("Failed to load configuration: %v", loadErr)
	}

	// Handle certificate operations
	if genCertsOnly || keygen {
		if certErr := generateCertificates(cfg); certErr != nil {
			log.Fatalf("Failed to generate certificates: %v", certErr)
		}
		return
	}

	if validateCerts {
		if validateErr := validateCertificates(cfg); validateErr != nil {
			log.Fatalf("Failed to validate certificates: %v", validateErr)
		}
		return
	}

	// Update certificate paths if keyfile specified
	if keyFile != "" {
		cfg.UpdateCertificatePaths(keyFile)
	}

	// Create signal channel for graceful shutdown and reload
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Start server or client based on mode
	// Create instance based on mode
	switch cfg.Mode {
	case "server":
		if err := runServer(configFile, sigCh); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	case "client":
		instance, initErr := NewClient(cfg)
		if initErr != nil {
			log.Fatalf("Failed to create client: %v", initErr)
		}
		if err := runInstance(instance, sigCh); err != nil {
			log.Fatalf("Client error: %v", err)
		}
	default:
		log.Fatalf("Invalid mode: %s", cfg.Mode)
	}
}

func runServer(configPath string, sigCh chan os.Signal) error {
	// Create server with config path for hot reloading
	server, err := NewServer(configPath)
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Start server in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Start()
	}()

	// Wait for signal or error
	for {
		select {
		case sig := <-sigCh:
			switch sig {
			case syscall.SIGHUP:
				fmt.Println("\nReloading configuration...")
				// ConfigManager will handle the reload
			case syscall.SIGINT, syscall.SIGTERM:
				fmt.Println("\nShutting down...")
				return server.Stop()
			}
		case err := <-errCh:
			return err
		}
	}
}

func runInstance(instance interface {
	Start() error
	Stop() error
}, sigCh chan os.Signal) error {
	// Start instance in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- instance.Start()
	}()

	// Wait for signal or error
	select {
	case <-sigCh:
		fmt.Println("\nShutting down...")
		return instance.Stop()
	case err := <-errCh:
		return err
	}
}

func generateCertificates(cfg *config.Config) error {
	// Certificate generation logic
	return nil
}

func validateCertificates(cfg *config.Config) error {
	// Certificate validation logic
	return nil
}
