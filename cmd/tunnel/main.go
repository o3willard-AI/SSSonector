package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/o3willard-AI/SSSonector/internal/cert/generator"
	"github.com/o3willard-AI/SSSonector/internal/cert/validator"
	"github.com/o3willard-AI/SSSonector/internal/config"
)

var (
	// Command line flags
	configPath        string
	generateCerts     bool
	certDir           string
	validateCerts     bool
	testMode          bool
	mode              string
	generateCertsOnly bool
)

func init() {
	// Basic flags
	flag.StringVar(&configPath, "config", "/etc/sssonector/config.yaml", "Path to configuration file")
	flag.StringVar(&mode, "mode", "", "Operation mode (server/client)")

	// Certificate management flags
	flag.BoolVar(&generateCerts, "keygen", false, "Generate SSL certificates")
	flag.StringVar(&certDir, "keyfile", "/etc/sssonector/certs", "Certificate directory")
	flag.BoolVar(&validateCerts, "validate-certs", false, "Validate existing certificates")
	flag.BoolVar(&testMode, "test-without-certs", false, "Run in test mode with temporary certificates")
	flag.BoolVar(&generateCertsOnly, "generate-certs-only", false, "Generate certificates without starting the service")

	// Parse flags early to handle certificate operations
	flag.Parse()
}

func main() {
	// Handle certificate generation
	if generateCerts {
		if err := os.MkdirAll(certDir, 0755); err != nil {
			log.Fatalf("Failed to create certificate directory: %v", err)
		}

		if err := generator.GenerateCertificates(certDir); err != nil {
			log.Fatalf("Failed to generate certificates: %v", err)
		}
		fmt.Printf("Successfully generated certificates in %s\n", certDir)
		return
	}

	// Handle certificate validation
	if validateCerts {
		if err := validator.ValidateCertificates(certDir); err != nil {
			log.Fatalf("Certificate validation failed: %v", err)
		}
		fmt.Println("Certificate validation successful")
		return
	}

	// Validate mode flag
	if mode == "" && !generateCerts && !validateCerts && !generateCertsOnly {
		log.Fatal("Must specify -mode (server/client) or use -keygen/-validate-certs/-generate-certs-only")
	}

	if mode != "server" && mode != "client" && !generateCerts && !validateCerts && !generateCertsOnly {
		log.Fatalf("Invalid mode: %s (must be 'server' or 'client')", mode)
	}

	// Handle test mode
	if testMode {
		if generateCertsOnly {
			// Use provided directory for certificate generation
			if err := generator.GenerateTemporaryCertificates(certDir); err != nil {
				log.Fatalf("Failed to generate temporary certificates: %v", err)
			}
			fmt.Printf("Successfully generated temporary certificates in %s\n", certDir)
			return
		} else {
			// Create temporary directory for running the service
			tempDir, err := os.MkdirTemp("", "sssonector-test-certs")
			if err != nil {
				log.Fatalf("Failed to create temporary directory: %v", err)
			}
			defer os.RemoveAll(tempDir)

			if err := generator.GenerateTemporaryCertificates(tempDir); err != nil {
				log.Fatalf("Failed to generate temporary certificates: %v", err)
			}
			certDir = tempDir
		}
	}

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Update certificate paths in configuration
	if certDir != "/etc/sssonector/certs" {
		cfg.UpdateCertificatePaths(certDir)
	}

	// Run in specified mode
	switch mode {
	case "server":
		runServer(cfg, testMode)
	case "client":
		runClient(cfg, testMode)
	}
}

func runServer(cfg *config.Config, testMode bool) {
	server, err := NewServer(cfg, testMode)
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	<-sigCh

	if err := server.Stop(); err != nil {
		log.Printf("Error stopping server: %v", err)
	}
}

func runClient(cfg *config.Config, testMode bool) {
	client, err := NewClient(cfg, testMode)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	if err := client.Start(); err != nil {
		log.Fatalf("Failed to start client: %v", err)
	}

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	<-sigCh

	if err := client.Stop(); err != nil {
		log.Printf("Error stopping client: %v", err)
	}
}
