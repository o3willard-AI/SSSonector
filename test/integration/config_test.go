package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"github.com/o3willard-AI/SSSonector/internal/tunnel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestConfigurationIntegration(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "config-integration-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test logger
	logger := zaptest.NewLogger(t)

	// Create test configuration
	cfg := &config.AppConfig{
		Metadata: config.ConfigMetadata{
			Version:     "1.0.0",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Environment: "test",
			Region:      "local",
		},
		Mode: config.ModeServer,
		Network: config.NetworkConfig{
			Interface:  "eth0",
			MTU:        1500,
			IPAddress:  "192.168.1.1",
			SubnetMask: "255.255.255.0",
			Gateway:    "192.168.1.254",
			DNSServers: []string{"8.8.8.8", "8.8.4.4"},
		},
		Tunnel: config.TunnelConfig{
			Protocol:       "tcp",
			Encryption:     "aes-256-gcm",
			Compression:    "none",
			MaxConnections: 100,
			MaxClients:     50,
			BufferSize:     65536,
			CertFile:       "cert.pem",
			KeyFile:        "key.pem",
			CAFile:         "ca.pem",
		},
		Monitor: config.MonitorConfig{
			Enabled:       true,
			Interval:      time.Minute,
			LogLevel:      "info",
			SNMPEnabled:   true,
			SNMPPort:      161,
			SNMPCommunity: "public",
			Prometheus: config.PrometheusConfig{
				Enabled: true,
				Port:    9090,
				Path:    "/metrics",
			},
		},
		Security: config.SecurityConfig{
			TLS: config.TLSConfig{
				MinVersion: "1.2",
				MaxVersion: "1.3",
				Ciphers: []string{
					"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
					"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
				},
			},
			AuthMethod: "certificate",
			CertRotation: config.CertRotationConfig{
				Enabled:  true,
				Interval: 720 * time.Hour,
			},
		},
	}

	// Create configuration components
	store := config.NewFileStore(tempDir, logger)
	validator := config.NewValidator(logger)
	configPath := filepath.Join(tempDir, "config.json")

	// Create configuration manager
	manager := config.NewManager(configPath, store, validator, logger)

	// Test configuration integration with tunnel
	t.Run("TunnelIntegration", func(t *testing.T) {
		// Update certificate paths
		err := tunnel.UpdateCertificatePaths(cfg, tempDir)
		require.NoError(t, err)

		// Verify certificate paths are absolute
		assert.True(t, filepath.IsAbs(cfg.Tunnel.CertFile))
		assert.True(t, filepath.IsAbs(cfg.Tunnel.KeyFile))
		assert.True(t, filepath.IsAbs(cfg.Tunnel.CAFile))

		// Create tunnel server
		server := tunnel.NewServer(cfg, manager, logger)
		require.NotNil(t, server)

		// Create tunnel client
		client := tunnel.NewClient(cfg, manager, logger)
		require.NotNil(t, client)
	})

	// Test configuration versioning
	t.Run("Versioning", func(t *testing.T) {
		// Store initial version
		err := manager.Apply(cfg)
		require.NoError(t, err)

		// Update configuration
		cfg.Network.MTU = 9000
		cfg.Metadata.Version = "1.1.0"

		// Store new version
		err = manager.Apply(cfg)
		require.NoError(t, err)

		// List versions
		versions, err := store.ListVersions(config.TypeServer)
		require.NoError(t, err)
		assert.Len(t, versions, 2)

		// Get latest version
		latest, err := store.GetLatest(config.TypeServer)
		require.NoError(t, err)
		assert.Equal(t, "1.1.0", latest.Metadata.Version)

		// Roll back to previous version
		err = manager.Rollback(config.TypeServer, "1.0.0")
		require.NoError(t, err)

		// Verify rollback
		current, err := store.GetLatest(config.TypeServer)
		require.NoError(t, err)
		assert.Equal(t, 1500, current.Network.MTU)
	})

	// Test configuration export/import
	t.Run("ExportImport", func(t *testing.T) {
		// Export configuration
		data, err := manager.Export(cfg, config.FormatJSON)
		require.NoError(t, err)

		// Verify JSON structure
		var exported map[string]interface{}
		err = json.Unmarshal(data, &exported)
		require.NoError(t, err)
		assert.Contains(t, exported, "metadata")
		assert.Contains(t, exported, "network")
		assert.Contains(t, exported, "tunnel")

		// Import configuration
		imported, err := manager.Import(data, config.FormatJSON)
		require.NoError(t, err)
		assert.Equal(t, cfg.Mode, imported.Mode)
		assert.Equal(t, cfg.Network.MTU, imported.Network.MTU)
	})

	// Test configuration validation
	t.Run("Validation", func(t *testing.T) {
		// Test valid configuration
		err := validator.Validate(cfg)
		assert.NoError(t, err)

		// Test invalid MTU
		invalidCfg := *cfg
		invalidCfg.Network.MTU = 100
		err = validator.Validate(&invalidCfg)
		assert.Error(t, err)

		// Test invalid protocol
		invalidCfg = *cfg
		invalidCfg.Tunnel.Protocol = "invalid"
		err = validator.Validate(&invalidCfg)
		assert.Error(t, err)

		// Test invalid TLS version
		invalidCfg = *cfg
		invalidCfg.Security.TLS.MinVersion = "invalid"
		err = validator.Validate(&invalidCfg)
		assert.Error(t, err)
	})

	// Test configuration watching
	t.Run("Watching", func(t *testing.T) {
		// Subscribe to changes
		ch, err := manager.Subscribe(config.TypeServer)
		require.NoError(t, err)

		// Apply configuration change
		cfg.Network.MTU = 9000
		err = manager.Apply(cfg)
		require.NoError(t, err)

		// Wait for notification
		select {
		case event := <-ch:
			assert.Equal(t, config.TypeServer, event.Type)
			assert.Equal(t, "update", event.ChangeType)
		case <-time.After(time.Second):
			t.Fatal("Timeout waiting for configuration change event")
		}

		// Unsubscribe
		err = manager.Unsubscribe(config.TypeServer)
		require.NoError(t, err)
	})
}
