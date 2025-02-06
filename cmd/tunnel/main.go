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

	// Create signal channel for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start server or client based on mode
	var instance interface {
		Start() error
		Stop() error
	}

	var initErr error
	switch cfg.Mode {
	case "server":
		instance, initErr = NewServer(cfg)
	case "client":
		instance, initErr = NewClient(cfg)
	default:
		log.Fatalf("Invalid mode: %s", cfg.Mode)
	}

	if initErr != nil {
		log.Fatalf("Failed to create %s: %v", cfg.Mode, initErr)
	}

	// Start instance in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- instance.Start()
	}()

	// Wait for signal or error
	select {
	case <-sigCh:
		fmt.Println("\nShutting down...")
		instance.Stop()
	case runErr := <-errCh:
		if runErr != nil {
			log.Fatalf("Error running %s: %v", cfg.Mode, runErr)
		}
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
