package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
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
	newConfig := &types.AppConfig{
		Type:    TypeClient,
		Version: "1.0.0",
		Config: &types.Config{
			Mode: ModeClient,
			Logging: types.LoggingConfig{
				Level:  "debug",
				File:   "/var/log/sssonector.log",
				Format: "json",
			},
			Network: types.NetworkConfig{
				Interface:  "tun0",
				MTU:        1500,
				Address:    "10.0.0.1",
				DNSServers: []string{"8.8.8.8"},
			},
			Tunnel: types.TunnelConfig{
				Port:        8080,
				Protocol:    "tcp",
				Compression: true,
			},
		},
		Metadata: types.ConfigMetadata{
			Version:     "1.0.0",
			Created:     time.Now(),
			Modified:    time.Now(),
			CreatedBy:   "test",
			Environment: "test",
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
	expectedFilename := filepath.Join(tempDir, "config-client-1.0.0.json")
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
