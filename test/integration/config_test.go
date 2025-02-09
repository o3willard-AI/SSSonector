package integration

import (
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config"
)

func TestConfigurationLifecycle(t *testing.T) {
	// Create test helper
	tc, err := NewTestConfig(t)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}
	defer tc.Cleanup()

	// Test default config
	cfg, err := tc.Manager.Get()
	if err != nil {
		t.Fatalf("Failed to get default config: %v", err)
	}
	if cfg == nil {
		t.Fatal("Expected non-nil default config")
	}
	if cfg.Type != config.TypeServer {
		t.Errorf("Expected default type %s, got %s", config.TypeServer, cfg.Type)
	}

	// Test setting new config
	newConfig := tc.CreateTestConfig(config.TypeClient)
	if err := tc.Manager.Set(newConfig); err != nil {
		t.Fatalf("Failed to set config: %v", err)
	}

	// Test configuration persistence
	savedConfig, err := tc.Manager.Get()
	if err != nil {
		t.Fatalf("Failed to get saved config: %v", err)
	}

	// Verify saved config matches expected
	if err := tc.CompareConfigs(newConfig, savedConfig); err != nil {
		t.Errorf("Config mismatch: %v", err)
	}

	// Verify config file was created correctly
	if err := tc.VerifyConfigFile(config.TypeClient, "1.0.0"); err != nil {
		t.Error(err)
	}

	// Test configuration watching
	watcher, err := tc.Manager.Watch()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	// Update configuration and verify watcher receives it
	newConfig.Config.Network.MTU = 1400
	if err := tc.Manager.Update(newConfig); err != nil {
		t.Fatalf("Failed to update config: %v", err)
	}

	// Wait for update with timeout
	updatedConfig, err := tc.WaitForConfigUpdate(watcher, time.Second)
	if err != nil {
		t.Error(err)
	} else if updatedConfig.Config.Network.MTU != 1400 {
		t.Errorf("Expected MTU 1400, got %d", updatedConfig.Config.Network.MTU)
	}

	// Test configuration versioning by creating a new version
	newConfig.Version = "1.0.1"
	if err := tc.Manager.Set(newConfig); err != nil {
		t.Fatalf("Failed to set new version: %v", err)
	}

	// Verify both version files exist
	if err := tc.VerifyConfigFile(config.TypeClient, "1.0.0"); err != nil {
		t.Error(err)
	}
	if err := tc.VerifyConfigFile(config.TypeClient, "1.0.1"); err != nil {
		t.Error(err)
	}
}

func TestConfigurationValidation(t *testing.T) {
	// Create test helper
	tc, err := NewTestConfig(t)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}
	defer tc.Cleanup()

	// Test invalid MTU
	invalidConfig := tc.CreateTestConfig(config.TypeClient)
	invalidConfig.Config.Network.MTU = 100 // Too small

	if err := tc.Manager.Set(invalidConfig); err == nil {
		t.Error("Expected validation error for invalid MTU")
	}

	// Test invalid TLS version
	invalidConfig = tc.CreateTestConfig(config.TypeClient)
	invalidConfig.Config.Security.TLS.MinVersion = "1.1" // Unsupported version

	if err := tc.Manager.Set(invalidConfig); err == nil {
		t.Error("Expected validation error for invalid TLS version")
	}

	// Test invalid protocol
	invalidConfig = tc.CreateTestConfig(config.TypeClient)
	invalidConfig.Config.Tunnel.Protocol = "invalid" // Unsupported protocol

	if err := tc.Manager.Set(invalidConfig); err == nil {
		t.Error("Expected validation error for invalid protocol")
	}

	// Test invalid monitor type
	invalidConfig = tc.CreateTestConfig(config.TypeClient)
	invalidConfig.Config.Monitor.Type = "invalid" // Unsupported type

	if err := tc.Manager.Set(invalidConfig); err == nil {
		t.Error("Expected validation error for invalid monitor type")
	}

	// Test invalid metrics interval
	invalidConfig = tc.CreateTestConfig(config.TypeClient)
	invalidConfig.Config.Metrics.Interval = 0 // Invalid interval

	if err := tc.Manager.Set(invalidConfig); err == nil {
		t.Error("Expected validation error for invalid metrics interval")
	}
}

func TestConfigurationHotReload(t *testing.T) {
	// Create test helper
	tc, err := NewTestConfig(t)
	if err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}
	defer tc.Cleanup()

	// Create initial config
	initialConfig := tc.CreateTestConfig(config.TypeClient)
	if err := tc.Manager.Set(initialConfig); err != nil {
		t.Fatalf("Failed to set initial config: %v", err)
	}

	// Create watcher
	watcher, err := tc.Manager.Watch()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}

	// Test multiple updates
	updates := []struct {
		field string
		apply func()
	}{
		{
			field: "MTU",
			apply: func() {
				initialConfig.Config.Network.MTU = 1400
			},
		},
		{
			field: "Protocol",
			apply: func() {
				initialConfig.Config.Tunnel.Protocol = "udp"
			},
		},
		{
			field: "Log Level",
			apply: func() {
				initialConfig.Config.Logging.Level = "info"
			},
		},
		{
			field: "Rate Limit",
			apply: func() {
				initialConfig.Throttle.Rate = 1024 * 1024 * 10 // 10MB/s
			},
		},
	}

	for _, update := range updates {
		t.Run(update.field, func(t *testing.T) {
			// Apply update
			update.apply()
			if err := tc.Manager.Update(initialConfig); err != nil {
				t.Fatalf("Failed to update %s: %v", update.field, err)
			}

			// Wait for update
			updatedConfig, err := tc.WaitForConfigUpdate(watcher, time.Second)
			if err != nil {
				t.Error(err)
			} else if err := tc.CompareConfigs(initialConfig, updatedConfig); err != nil {
				t.Errorf("Config mismatch after %s update: %v", update.field, err)
			}
		})
	}
}
