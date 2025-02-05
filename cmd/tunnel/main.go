package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/o3willard-AI/SSSonector/internal/cert/generator"
	"github.com/o3willard-AI/SSSonector/internal/cert/validator"
	"github.com/o3willard-AI/SSSonector/internal/config"
)

var (
	// Command line flags
	configPath    string
	generateCerts bool
	certDir       string
	validateCerts bool
	testMode      bool
	mode          string
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
}

func main() {
	flag.Parse()

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

	// Handle test mode
	if testMode {
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
		runServer(cfg)
	case "client":
		runClient(cfg)
	default:
		log.Fatal("Must specify -mode (server/client)")
	}
}

func runServer(cfg *config.Config) {
	// Server implementation
	fmt.Println("Running in server mode...")
}

func runClient(cfg *config.Config) {
	// Client implementation
	fmt.Println("Running in client mode...")
}
