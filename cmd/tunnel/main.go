package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/o3willard-AI/SSSonector/internal/cert"
	"github.com/o3willard-AI/SSSonector/internal/config"
)

var (
	// Command-line flags
	configFile      string
	keygenFlag      bool
	keyFileDir      string
	testWithoutCert bool
	mode            string
)

func init() {
	flag.StringVar(&configFile, "config", "", "Path to configuration file")
	flag.BoolVar(&keygenFlag, "keygen", false, "Generate SSL certificates")
	flag.StringVar(&keyFileDir, "keyfile", "", "Directory containing SSL certificates")
	flag.BoolVar(&testWithoutCert, "test-without-certs", false, "Run a 15-second test connection without certificates")
	flag.StringVar(&mode, "mode", "", "Operation mode (server/client)")
}

func main() {
	flag.Parse()

	// Handle certificate generation
	if keygenFlag {
		if err := generateCertificates(); err != nil {
			log.Fatalf("Failed to generate certificates: %v", err)
		}
		return
	}

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Handle test mode
	if testWithoutCert {
		if err := runTestMode(cfg); err != nil {
			log.Fatalf("Test connection failed: %v", err)
		}
		return
	}

	// Normal operation
	if err := run(cfg); err != nil {
		log.Fatalf("Application error: %v", err)
	}
}

func generateCertificates() error {
	// Use current directory if not specified
	outputDir := "."
	if keyFileDir != "" {
		outputDir = keyFileDir
	}

	// Create certificate generator
	generator := cert.NewCertificateGenerator(outputDir)

	// Generate CA certificate
	if err := generator.GenerateCA(); err != nil {
		return fmt.Errorf("failed to generate CA certificate: %v", err)
	}

	// Generate server certificate
	if err := generator.GenerateServerCert(); err != nil {
		return fmt.Errorf("failed to generate server certificate: %v", err)
	}

	// Generate client certificate
	if err := generator.GenerateClientCert(); err != nil {
		return fmt.Errorf("failed to generate client certificate: %v", err)
	}

	fmt.Println("Certificates generated successfully:")
	fmt.Printf("  CA Certificate:     %s\n", filepath.Join(outputDir, "ca.crt"))
	fmt.Printf("  Server Certificate: %s\n", filepath.Join(outputDir, "server.crt"))
	fmt.Printf("  Server Key:        %s\n", filepath.Join(outputDir, "server.key"))
	fmt.Printf("  Client Certificate: %s\n", filepath.Join(outputDir, "client.crt"))
	fmt.Printf("  Client Key:        %s\n", filepath.Join(outputDir, "client.key"))

	return nil
}

func loadConfig() (*config.Config, error) {
	// If config file is not specified, use default location
	if configFile == "" {
		configFile = "/etc/sssonector/config.yaml"
	}

	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %v", err)
	}

	// Override mode if specified in command line
	if mode != "" {
		cfg.Mode = mode
	}

	// Validate mode
	if cfg.Mode != "server" && cfg.Mode != "client" {
		return nil, fmt.Errorf("invalid mode: %s (must be 'server' or 'client')", cfg.Mode)
	}

	// Handle certificate locations
	if keyFileDir != "" {
		// Update certificate paths if keyfile directory is specified
		cfg.Tunnel.CertFile = filepath.Join(keyFileDir, filepath.Base(cfg.Tunnel.CertFile))
		cfg.Tunnel.KeyFile = filepath.Join(keyFileDir, filepath.Base(cfg.Tunnel.KeyFile))
		cfg.Tunnel.CAFile = filepath.Join(keyFileDir, filepath.Base(cfg.Tunnel.CAFile))
	}

	// Validate certificate files exist
	locator := cert.NewCertificateLocator(filepath.Dir(cfg.Tunnel.CertFile))
	certPath, keyPath, err := locator.FindCertificates(
		filepath.Base(cfg.Tunnel.CertFile),
		filepath.Base(cfg.Tunnel.KeyFile),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to locate certificates: %v", err)
	}

	// Update paths to found certificates
	cfg.Tunnel.CertFile = certPath
	cfg.Tunnel.KeyFile = keyPath

	// Validate certificates
	validator := cert.NewCertificateValidator(false)
	if err := validator.ValidateCertificateFiles(certPath, keyPath, cfg.Tunnel.CAFile); err != nil {
		return nil, fmt.Errorf("certificate validation failed: %v", err)
	}

	return cfg, nil
}

func runTestMode(cfg *config.Config) error {
	fmt.Println("Starting 15-second test connection...")

	// Create certificate generator for test certificates
	generator := cert.NewCertificateGenerator(".")

	// Generate test certificates
	if err := generator.GenerateTestCerts(); err != nil {
		return fmt.Errorf("failed to generate test certificates: %v", err)
	}

	// Update config to use test certificates
	cfg.Tunnel.CertFile = "test_" + filepath.Base(cfg.Tunnel.CertFile)
	cfg.Tunnel.KeyFile = "test_" + filepath.Base(cfg.Tunnel.KeyFile)
	cfg.Tunnel.CAFile = "test_ca.crt"

	// Run the tunnel with test certificates
	if err := run(cfg); err != nil {
		return fmt.Errorf("test connection failed: %v", err)
	}

	// Clean up test certificates
	os.Remove(cfg.Tunnel.CertFile)
	os.Remove(cfg.Tunnel.KeyFile)
	os.Remove(cfg.Tunnel.CAFile)

	fmt.Println("Test connection completed successfully")
	return nil
}

func run(cfg *config.Config) error {
	// Implementation of the main tunnel functionality
	// This will be handled by the existing tunnel package
	return fmt.Errorf("tunnel implementation not yet complete")
}
