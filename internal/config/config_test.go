package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestConfigStore(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "config-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test logger
	logger := zaptest.NewLogger(t)

	// Create file store
	store := NewFileStore(tempDir, logger)

	// Create test configuration
	cfg := &AppConfig{
		Metadata: ConfigMetadata{
			Version:     "1.0.0",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Environment: "test",
			Region:      "local",
		},
		Mode: ModeServer,
		Network: NetworkConfig{
			Interface: "eth0",
			MTU:       1500,
		},
		Tunnel: TunnelConfig{
			Protocol:       "tcp",
			Encryption:     "aes-256-gcm",
			MaxConnections: 100,
		},
	}

	// Test storing configuration
	t.Run("Store", func(t *testing.T) {
		err := store.Store(cfg)
		require.NoError(t, err)

		// Verify file exists
		path := filepath.Join(tempDir, "configs", string(ModeServer), "config-1.0.0.json")
		_, err = os.Stat(path)
		assert.NoError(t, err)
	})

	// Test loading configuration
	t.Run("Load", func(t *testing.T) {
		loaded, err := store.Load(TypeServer, "1.0.0")
		require.NoError(t, err)
		assert.Equal(t, cfg.Mode, loaded.Mode)
		assert.Equal(t, cfg.Network.Interface, loaded.Network.Interface)
		assert.Equal(t, cfg.Tunnel.Protocol, loaded.Tunnel.Protocol)
	})

	// Test listing configurations
	t.Run("List", func(t *testing.T) {
		configs, err := store.List()
		require.NoError(t, err)
		assert.Len(t, configs, 1)
		assert.Equal(t, cfg.Mode, configs[0].Mode)
	})

	// Test listing by type
	t.Run("ListByType", func(t *testing.T) {
		configs, err := store.ListByType(TypeServer)
		require.NoError(t, err)
		assert.Len(t, configs, 1)
		assert.Equal(t, cfg.Mode, configs[0].Mode)
	})

	// Test getting latest version
	t.Run("GetLatest", func(t *testing.T) {
		latest, err := store.GetLatest(TypeServer)
		require.NoError(t, err)
		assert.Equal(t, cfg.Mode, latest.Mode)
	})

	// Test deleting configuration
	t.Run("Delete", func(t *testing.T) {
		err := store.Delete(TypeServer, "1.0.0")
		require.NoError(t, err)

		// Verify file is deleted
		path := filepath.Join(tempDir, "configs", string(ModeServer), "config-1.0.0.json")
		_, err = os.Stat(path)
		assert.True(t, os.IsNotExist(err))
	})
}

func TestConfigValidator(t *testing.T) {
	// Create test logger
	logger := zaptest.NewLogger(t)

	// Create validator
	validator := NewValidator(logger)

	// Test valid configuration
	t.Run("ValidConfig", func(t *testing.T) {
		cfg := &AppConfig{
			Mode: ModeServer,
			Network: NetworkConfig{
				Interface:  "eth0",
				MTU:        1500,
				IPAddress:  "192.168.1.1",
				SubnetMask: "255.255.255.0",
			},
			Tunnel: TunnelConfig{
				Protocol:       "tcp",
				Encryption:     "aes-256-gcm",
				MaxConnections: 100,
				MaxClients:     50,
				BufferSize:     65536,
			},
			Monitor: MonitorConfig{
				Enabled:  true,
				Interval: time.Minute,
				LogLevel: "info",
			},
		}

		err := validator.Validate(cfg)
		assert.NoError(t, err)
	})

	// Test invalid mode
	t.Run("InvalidMode", func(t *testing.T) {
		cfg := &AppConfig{
			Mode: "invalid",
		}
		err := validator.Validate(cfg)
		assert.Error(t, err)
	})

	// Test invalid network config
	t.Run("InvalidNetwork", func(t *testing.T) {
		cfg := &AppConfig{
			Mode: ModeServer,
			Network: NetworkConfig{
				MTU: 100, // Too small
			},
		}
		err := validator.Validate(cfg)
		assert.Error(t, err)
	})

	// Test invalid tunnel config
	t.Run("InvalidTunnel", func(t *testing.T) {
		cfg := &AppConfig{
			Mode: ModeServer,
			Network: NetworkConfig{
				MTU: 1500,
			},
			Tunnel: TunnelConfig{
				Protocol: "invalid",
			},
		}
		err := validator.Validate(cfg)
		assert.Error(t, err)
	})

	// Test schema validation
	t.Run("SchemaValidation", func(t *testing.T) {
		cfg := &AppConfig{
			Mode: ModeServer,
		}

		schema := []byte(`{
			"type": "object",
			"required": ["mode"],
			"properties": {
				"mode": {
					"type": "string",
					"enum": ["server", "client"]
				}
			}
		}`)

		err := validator.ValidateSchema(cfg, schema)
		assert.NoError(t, err)
	})
}

func TestConfigManager(t *testing.T) {
	// Create temporary directory for test files
	tempDir, err := os.MkdirTemp("", "config-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create test logger
	logger := zaptest.NewLogger(t)

	// Create components
	store := NewFileStore(tempDir, logger)
	validator := NewValidator(logger)
	configPath := filepath.Join(tempDir, "config.json")

	// Create manager
	manager := NewManager(configPath, store, validator, logger)

	// Create test configuration
	cfg := &AppConfig{
		Metadata: ConfigMetadata{
			Version:     "1.0.0",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Environment: "test",
		},
		Mode: ModeServer,
		Network: NetworkConfig{
			Interface: "eth0",
			MTU:       1500,
		},
	}

	// Test applying configuration
	t.Run("Apply", func(t *testing.T) {
		err := manager.Apply(cfg)
		require.NoError(t, err)
	})

	// Test exporting configuration
	t.Run("Export", func(t *testing.T) {
		data, err := manager.Export(cfg, FormatJSON)
		require.NoError(t, err)

		var exported AppConfig
		err = json.Unmarshal(data, &exported)
		require.NoError(t, err)
		assert.Equal(t, cfg.Mode, exported.Mode)
	})

	// Test importing configuration
	t.Run("Import", func(t *testing.T) {
		data := []byte(`{
			"metadata": {
				"version": "1.0.0",
				"environment": "test"
			},
			"mode": "server",
			"network": {
				"interface": "eth0",
				"mtu": 1500
			}
		}`)

		imported, err := manager.Import(data, FormatJSON)
		require.NoError(t, err)
		assert.Equal(t, ModeServer, imported.Mode)
	})

	// Test rollback
	t.Run("Rollback", func(t *testing.T) {
		err := manager.Rollback(TypeServer, "1.0.0")
		require.NoError(t, err)
	})
}
