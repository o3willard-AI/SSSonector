package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("load yaml config", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "config.yaml")
		configData := `
logLevel: debug
server:
  host: "127.0.0.1"
  port: 9090
connection:
  maxConnections: 500
  keepAlive: true
  idleTimeout: 10m
rateLimit:
  enabled: true
  requestRate: 500
  burstSize: 50
circuitBreaker:
  enabled: true
  maxFailures: 3
  resetTimeout: 1m
  halfOpenMaxCalls: 5
`
		err := os.WriteFile(configPath, []byte(configData), 0644)
		require.NoError(t, err)

		var config Config
		err = loadConfig(configPath, &config)
		require.NoError(t, err)

		assert.Equal(t, "debug", config.LogLevel)
		assert.Equal(t, "127.0.0.1", config.Server.Host)
		assert.Equal(t, 9090, config.Server.Port)
		assert.Equal(t, 500, config.Connection.MaxConnections)
		assert.True(t, config.Connection.KeepAlive)
		assert.Equal(t, types.NewDuration(10*time.Minute), config.Connection.IdleTimeout)
		assert.True(t, config.RateLimit.Enabled)
		assert.Equal(t, int64(500), config.RateLimit.RequestRate)
		assert.Equal(t, 50, config.RateLimit.BurstSize)
		assert.True(t, config.CircuitBreaker.Enabled)
		assert.Equal(t, 3, config.CircuitBreaker.MaxFailures)
		assert.Equal(t, types.NewDuration(time.Minute), config.CircuitBreaker.ResetTimeout)
		assert.Equal(t, 5, config.CircuitBreaker.HalfOpenMaxCalls)
	})

	t.Run("load json config", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "config.json")
		configData := `{
			"logLevel": "debug",
			"server": {
				"host": "127.0.0.1",
				"port": 9090
			},
			"connection": {
				"maxConnections": 500,
				"keepAlive": true,
				"idleTimeout": "10m"
			},
			"rateLimit": {
				"enabled": true,
				"requestRate": 500,
				"burstSize": 50
			},
			"circuitBreaker": {
				"enabled": true,
				"maxFailures": 3,
				"resetTimeout": "1m",
				"halfOpenMaxCalls": 5
			}
		}`
		err := os.WriteFile(configPath, []byte(configData), 0644)
		require.NoError(t, err)

		var config Config
		err = loadConfig(configPath, &config)
		require.NoError(t, err)

		assert.Equal(t, "debug", config.LogLevel)
		assert.Equal(t, "127.0.0.1", config.Server.Host)
		assert.Equal(t, 9090, config.Server.Port)
		assert.Equal(t, 500, config.Connection.MaxConnections)
		assert.True(t, config.Connection.KeepAlive)
		assert.Equal(t, types.NewDuration(10*time.Minute), config.Connection.IdleTimeout)
		assert.True(t, config.RateLimit.Enabled)
		assert.Equal(t, int64(500), config.RateLimit.RequestRate)
		assert.Equal(t, 50, config.RateLimit.BurstSize)
		assert.True(t, config.CircuitBreaker.Enabled)
		assert.Equal(t, 3, config.CircuitBreaker.MaxFailures)
		assert.Equal(t, types.NewDuration(time.Minute), config.CircuitBreaker.ResetTimeout)
		assert.Equal(t, 5, config.CircuitBreaker.HalfOpenMaxCalls)
	})

	t.Run("invalid file format", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "config.txt")
		err := os.WriteFile(configPath, []byte("invalid"), 0644)
		require.NoError(t, err)

		var config Config
		err = loadConfig(configPath, &config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported config file format")
	})

	t.Run("invalid yaml", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "config.yaml")
		err := os.WriteFile(configPath, []byte("invalid: [yaml"), 0644)
		require.NoError(t, err)

		var config Config
		err = loadConfig(configPath, &config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse YAML")
	})

	t.Run("invalid json", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "config.json")
		err := os.WriteFile(configPath, []byte("invalid json"), 0644)
		require.NoError(t, err)

		var config Config
		err = loadConfig(configPath, &config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse JSON")
	})

	t.Run("file not found", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "nonexistent.yaml")
		var config Config
		err := loadConfig(configPath, &config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read config file")
	})
}

func TestSaveConfig(t *testing.T) {
	tempDir := t.TempDir()
	config := Config{
		LogLevel: "debug",
		Server: struct {
			Host string `yaml:"host" json:"host"`
			Port int    `yaml:"port" json:"port"`
		}{
			Host: "127.0.0.1",
			Port: 9090,
		},
		Connection: struct {
			MaxConnections int            `yaml:"maxConnections" json:"maxConnections"`
			KeepAlive      bool           `yaml:"keepAlive" json:"keepAlive"`
			IdleTimeout    types.Duration `yaml:"idleTimeout" json:"idleTimeout"`
		}{
			MaxConnections: 500,
			KeepAlive:      true,
			IdleTimeout:    types.NewDuration(10 * time.Minute),
		},
		RateLimit: struct {
			Enabled     bool  `yaml:"enabled" json:"enabled"`
			RequestRate int64 `yaml:"requestRate" json:"requestRate"`
			BurstSize   int   `yaml:"burstSize" json:"burstSize"`
		}{
			Enabled:     true,
			RequestRate: 500,
			BurstSize:   50,
		},
		CircuitBreaker: struct {
			Enabled          bool           `yaml:"enabled" json:"enabled"`
			MaxFailures      int            `yaml:"maxFailures" json:"maxFailures"`
			ResetTimeout     types.Duration `yaml:"resetTimeout" json:"resetTimeout"`
			HalfOpenMaxCalls int            `yaml:"halfOpenMaxCalls" json:"halfOpenMaxCalls"`
		}{
			Enabled:          true,
			MaxFailures:      3,
			ResetTimeout:     types.NewDuration(time.Minute),
			HalfOpenMaxCalls: 5,
		},
	}

	t.Run("save yaml config", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "config.yaml")
		err := saveConfig(configPath, &config)
		require.NoError(t, err)

		var loadedConfig Config
		err = loadConfig(configPath, &loadedConfig)
		require.NoError(t, err)

		assert.Equal(t, config.LogLevel, loadedConfig.LogLevel)
		assert.Equal(t, config.Server.Host, loadedConfig.Server.Host)
		assert.Equal(t, config.Server.Port, loadedConfig.Server.Port)
		assert.Equal(t, config.Connection.MaxConnections, loadedConfig.Connection.MaxConnections)
		assert.Equal(t, config.Connection.KeepAlive, loadedConfig.Connection.KeepAlive)
		assert.Equal(t, config.Connection.IdleTimeout, loadedConfig.Connection.IdleTimeout)
		assert.Equal(t, config.RateLimit.Enabled, loadedConfig.RateLimit.Enabled)
		assert.Equal(t, config.RateLimit.RequestRate, loadedConfig.RateLimit.RequestRate)
		assert.Equal(t, config.RateLimit.BurstSize, loadedConfig.RateLimit.BurstSize)
		assert.Equal(t, config.CircuitBreaker.Enabled, loadedConfig.CircuitBreaker.Enabled)
		assert.Equal(t, config.CircuitBreaker.MaxFailures, loadedConfig.CircuitBreaker.MaxFailures)
		assert.Equal(t, config.CircuitBreaker.ResetTimeout, loadedConfig.CircuitBreaker.ResetTimeout)
		assert.Equal(t, config.CircuitBreaker.HalfOpenMaxCalls, loadedConfig.CircuitBreaker.HalfOpenMaxCalls)
	})

	t.Run("save json config", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "config.json")
		err := saveConfig(configPath, &config)
		require.NoError(t, err)

		var loadedConfig Config
		err = loadConfig(configPath, &loadedConfig)
		require.NoError(t, err)

		assert.Equal(t, config.LogLevel, loadedConfig.LogLevel)
		assert.Equal(t, config.Server.Host, loadedConfig.Server.Host)
		assert.Equal(t, config.Server.Port, loadedConfig.Server.Port)
		assert.Equal(t, config.Connection.MaxConnections, loadedConfig.Connection.MaxConnections)
		assert.Equal(t, config.Connection.KeepAlive, loadedConfig.Connection.KeepAlive)
		assert.Equal(t, config.Connection.IdleTimeout, loadedConfig.Connection.IdleTimeout)
		assert.Equal(t, config.RateLimit.Enabled, loadedConfig.RateLimit.Enabled)
		assert.Equal(t, config.RateLimit.RequestRate, loadedConfig.RateLimit.RequestRate)
		assert.Equal(t, config.RateLimit.BurstSize, loadedConfig.RateLimit.BurstSize)
		assert.Equal(t, config.CircuitBreaker.Enabled, loadedConfig.CircuitBreaker.Enabled)
		assert.Equal(t, config.CircuitBreaker.MaxFailures, loadedConfig.CircuitBreaker.MaxFailures)
		assert.Equal(t, config.CircuitBreaker.ResetTimeout, loadedConfig.CircuitBreaker.ResetTimeout)
		assert.Equal(t, config.CircuitBreaker.HalfOpenMaxCalls, loadedConfig.CircuitBreaker.HalfOpenMaxCalls)
	})

	t.Run("invalid file format", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "config.txt")
		err := saveConfig(configPath, &config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported config file format")
	})

	t.Run("invalid directory", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "nonexistent", "config.yaml")
		err := saveConfig(configPath, &config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to write config file")
	})
}
