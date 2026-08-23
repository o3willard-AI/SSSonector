package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestConfigManager(t *testing.T) {
	// Create temporary directory for test configs
	tempDir, err := os.MkdirTemp("", "sssonector-config-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create manager
	manager := CreateManager(tempDir)

	// Test default config
	config, err := manager.Get()
	if err != nil {
		t.Fatalf("Failed to get default config: %v", err)
	}
	if config == nil {
		t.Fatal("Expected non-nil default config")
	}
	if config.Type != TypeServer {
		t.Errorf("Expected default type %s, got %s", TypeServer, config.Type)
	}

	// Test setting new config
	newConfig := &AppConfig{
		Type:    TypeClient,
		Version: "2.0.0",
		Config: &Config{
			Mode: ModeClient,
			Logging: LoggingConfig{
				Level:  "debug",
				File:   "/var/log/sssonector.log",
				Format: "json",
			},
			Network: NetworkConfig{
				Interface:  "tun0",
				MTU:        1500,
				Address:    "10.0.0.1/24",
				DNSServers: []string{"8.8.8.8"},
			},
			Tunnel: TunnelConfig{
				ServerAddress: "192.168.1.1",
				ServerPort:    8080,
				Compression:   true,
			},
			Security: SecurityConfig{
				TLS: TLSConfigOptions{
					MinVersion: "1.2",
					MaxVersion: "1.3",
				},
			},
		},
		Metadata: ConfigMetadata{
			Version:       "2.0.0",
			SchemaVersion: CurrentSchemaVersion,
			Created:       time.Now(),
			Modified:      time.Now(),
			CreatedBy:     "test",
			Environment:   "test",
		},
	}

	if err := manager.Set(newConfig); err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Test getting updated config
	updatedConfig, err := manager.Get()
	if err != nil {
		t.Fatalf("Failed to get updated config: %v", err)
	}
	if updatedConfig.Type != TypeClient {
		t.Errorf("Expected type %s, got %s", TypeClient, updatedConfig.Type)
	}
	if updatedConfig.Config.Mode != ModeClient {
		t.Errorf("Expected mode %s, got %s", ModeClient, updatedConfig.Config.Mode)
	}

	// Test config file was created
	files, err := os.ReadDir(tempDir)
	if err != nil {
		t.Fatalf("Failed to read temp dir: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("Expected 1 config file, got %d", len(files))
	}
	expectedFilename := filepath.Join(tempDir, "config.yaml")
	if _, err := os.Stat(expectedFilename); os.IsNotExist(err) {
		t.Errorf("Expected config file %s does not exist", expectedFilename)
	}

	// Test watching config changes
	watcher, err := manager.Watch()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	// Update config and verify watcher receives it
	newConfig.Config.Network.MTU = 1400
	if err := manager.Update(newConfig); err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	select {
	case updatedConfig := <-watcher:
		if updatedConfig.Config.Network.MTU != 1400 {
			t.Errorf("Expected MTU 1400, got %d", updatedConfig.Config.Network.MTU)
		}
	case <-time.After(time.Second):
		t.Error("Timed out waiting for config update")
	}

	// Test closing manager
	if err := manager.Close(); err != nil {
		t.Fatalf("Failed to close manager: %v", err)
	}
}

func TestValidateLoggingFormat(t *testing.T) {
	v := NewValidator()

	valid := []string{"", "json", "console", "JSON"}
	for _, format := range valid {
		cfg := DefaultConfig()
		cfg.Config.Logging = LoggingConfig{Level: "info", Format: format}
		if err := v.validateLogging(cfg.Config.Logging); err != nil {
			t.Errorf("validateLogging(format=%q) unexpected error: %v", format, err)
		}
	}

	invalid := []string{"xml", "logfmt", "text"}
	for _, format := range invalid {
		cfg := DefaultConfig()
		cfg.Config.Logging = LoggingConfig{Level: "info", Format: format}
		if err := v.validateLogging(cfg.Config.Logging); err == nil {
			t.Errorf("validateLogging(format=%q) expected error, got nil", format)
		}
	}
}

func TestValidateNetworkRequiresCIDR(t *testing.T) {
	v := NewValidator()

	valid := []string{"10.0.0.1/24", "192.168.50.10/32"}
	for _, addr := range valid {
		cfg := NetworkConfig{Interface: "tun0", MTU: 1500, Address: addr}
		if err := v.validateNetwork(cfg); err != nil {
			t.Errorf("validateNetwork(address=%q) unexpected error: %v", addr, err)
		}
	}

	invalid := []string{"10.0.0.1", "not-an-ip", "10.0.0.1/99"}
	for _, addr := range invalid {
		cfg := NetworkConfig{Interface: "tun0", MTU: 1500, Address: addr}
		if err := v.validateNetwork(cfg); err == nil {
			t.Errorf("validateNetwork(address=%q) expected error, got nil", addr)
		}
	}
}

func TestValidateTunnelModeAware(t *testing.T) {
	v := NewValidator()

	serverCfg := TunnelConfig{ListenPort: 8443}
	if err := v.validateTunnel("server", serverCfg); err != nil {
		t.Errorf("valid server tunnel rejected: %v", err)
	}
	if err := v.validateTunnel("server", TunnelConfig{}); err == nil {
		t.Error("server without listen_port should fail")
	}

	clientCfg := TunnelConfig{ServerAddress: "203.0.113.5", ServerPort: 8443}
	if err := v.validateTunnel("client", clientCfg); err != nil {
		t.Errorf("valid client tunnel rejected: %v", err)
	}
	if err := v.validateTunnel("client", TunnelConfig{ServerPort: 8443}); err == nil {
		t.Error("client without server_address should fail")
	}
	if err := v.validateTunnel("client", TunnelConfig{ServerAddress: "203.0.113.5"}); err == nil {
		t.Error("client without server_port should fail")
	}

	if err := v.validateTunnel("banana", serverCfg); err == nil {
		t.Error("unknown mode should fail")
	}
}

func TestValidateEnvironmentAcceptsQA(t *testing.T) {
	v := NewValidator()
	cfg := DefaultConfig()
	cfg.Metadata.Environment = "qa"
	cfg.Config.Security.TLS.MinVersion = "1.2"
	if err := v.validateEnvironmentConfig(cfg); err != nil {
		t.Errorf("qa environment should be accepted: %v", err)
	}

	cfg.Metadata.Environment = "staging"
	if err := v.validateEnvironmentConfig(cfg); err != nil {
		t.Errorf("staging environment should be accepted: %v", err)
	}

	cfg.Metadata.Environment = "narnia"
	if err := v.validateEnvironmentConfig(cfg); err == nil {
		t.Error("unknown environment should fail")
	}
}

func TestLoadDataRequiresCurrentSchema(t *testing.T) {
	l := NewConfigLoader()

	current := `
metadata:
  schema_version: "2.0.0"
  environment: qa
type: client
config:
  mode: client
  security:
    tls:
      min_version: "1.2"
`
	cfg, err := l.LoadFromString(current, "yaml")
	if err != nil {
		t.Fatalf("current schema rejected: %v", err)
	}
	if cfg.Config.Mode != ModeClient {
		t.Errorf("mode = %q, want client", cfg.Config.Mode)
	}

	legacy := `
metadata:
  schema_version: "1.0.0"
throttle:
  enabled: true
`
	if _, err := l.LoadFromString(legacy, "yaml"); err == nil {
		t.Error("legacy schema must be rejected")
	} else if !strings.Contains(err.Error(), "unsupported schema version") {
		t.Errorf("error should name the schema version problem, got: %v", err)
	}
}
