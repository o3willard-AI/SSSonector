package preflight

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestConfigValidationCheck(t *testing.T) {
	logger := zaptest.NewLogger(t)
	ctx := context.Background()
	tempDir := t.TempDir()

	t.Run("valid yaml config", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "config.yaml")
		configData := `
server:
  host: "127.0.0.1"
  port: 8080
connection:
  maxConnections: 1000
  keepAlive: true
  idleTimeout: "5m"
rateLimit:
  enabled: true
  requestRate: 1000
  burstSize: 100
circuitBreaker:
  enabled: true
  maxFailures: 5
  resetTimeout: "30s"
  halfOpenMaxCalls: 2
`
		err := os.WriteFile(configPath, []byte(configData), 0644)
		require.NoError(t, err)

		os.Setenv("SSSONECTOR_CONFIG", configPath)
		defer os.Unsetenv("SSSONECTOR_CONFIG")

		result := ConfigValidationCheck(ctx, logger)
		assert.Equal(t, "Passed", result.Status)
		assert.NoError(t, result.Error)
	})

	t.Run("valid json config", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "config.json")
		configData := `{
			"server": {
				"host": "127.0.0.1",
				"port": 8080
			},
			"connection": {
				"maxConnections": 1000,
				"keepAlive": true,
				"idleTimeout": "5m"
			},
			"rateLimit": {
				"enabled": true,
				"requestRate": 1000,
				"burstSize": 100
			},
			"circuitBreaker": {
				"enabled": true,
				"maxFailures": 5,
				"resetTimeout": "30s",
				"halfOpenMaxCalls": 2
			}
		}`
		err := os.WriteFile(configPath, []byte(configData), 0644)
		require.NoError(t, err)

		os.Setenv("SSSONECTOR_CONFIG", configPath)
		defer os.Unsetenv("SSSONECTOR_CONFIG")

		result := ConfigValidationCheck(ctx, logger)
		assert.Equal(t, "Passed", result.Status)
		assert.NoError(t, result.Error)
	})

	t.Run("missing mandatory field", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "config.yaml")
		configData := `
server:
  host: "127.0.0.1"
  # port is missing
connection:
  maxConnections: 1000
  keepAlive: true
  idleTimeout: "5m"
rateLimit:
  enabled: true
  requestRate: 1000
  burstSize: 100
circuitBreaker:
  enabled: true
  maxFailures: 5
  resetTimeout: "30s"
  halfOpenMaxCalls: 2
`
		err := os.WriteFile(configPath, []byte(configData), 0644)
		require.NoError(t, err)

		os.Setenv("SSSONECTOR_CONFIG", configPath)
		defer os.Unsetenv("SSSONECTOR_CONFIG")

		result := ConfigValidationCheck(ctx, logger)
		assert.Equal(t, "Failed", result.Status)
		assert.ErrorContains(t, result.Error, "missing mandatory field: server.port")
	})

	t.Run("invalid field type", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "config.yaml")
		configData := `
server:
  host: "127.0.0.1"
  port: "8080"  # should be number
connection:
  maxConnections: 1000
  keepAlive: true
  idleTimeout: "5m"
rateLimit:
  enabled: true
  requestRate: 1000
  burstSize: 100
circuitBreaker:
  enabled: true
  maxFailures: 5
  resetTimeout: "30s"
  halfOpenMaxCalls: 2
`
		err := os.WriteFile(configPath, []byte(configData), 0644)
		require.NoError(t, err)

		os.Setenv("SSSONECTOR_CONFIG", configPath)
		defer os.Unsetenv("SSSONECTOR_CONFIG")

		result := ConfigValidationCheck(ctx, logger)
		assert.Equal(t, "Failed", result.Status)
		assert.ErrorContains(t, result.Error, "invalid type for server.port: expected number")
	})

	t.Run("invalid duration format", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "config.yaml")
		configData := `
server:
  host: "127.0.0.1"
  port: 8080
connection:
  maxConnections: 1000
  keepAlive: true
  idleTimeout: "invalid"  # invalid duration
rateLimit:
  enabled: true
  requestRate: 1000
  burstSize: 100
circuitBreaker:
  enabled: true
  maxFailures: 5
  resetTimeout: "30s"
  halfOpenMaxCalls: 2
`
		err := os.WriteFile(configPath, []byte(configData), 0644)
		require.NoError(t, err)

		os.Setenv("SSSONECTOR_CONFIG", configPath)
		defer os.Unsetenv("SSSONECTOR_CONFIG")

		result := ConfigValidationCheck(ctx, logger)
		assert.Equal(t, "Failed", result.Status)
		assert.ErrorContains(t, result.Error, "invalid duration format for connection.idleTimeout")
	})

	t.Run("value out of range", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "config.yaml")
		configData := `
server:
  host: "127.0.0.1"
  port: 70000  # port too high
connection:
  maxConnections: 1000
  keepAlive: true
  idleTimeout: "5m"
rateLimit:
  enabled: true
  requestRate: 1000
  burstSize: 100
circuitBreaker:
  enabled: true
  maxFailures: 5
  resetTimeout: "30s"
  halfOpenMaxCalls: 2
`
		err := os.WriteFile(configPath, []byte(configData), 0644)
		require.NoError(t, err)

		os.Setenv("SSSONECTOR_CONFIG", configPath)
		defer os.Unsetenv("SSSONECTOR_CONFIG")

		result := ConfigValidationCheck(ctx, logger)
		assert.Equal(t, "Failed", result.Status)
		assert.ErrorContains(t, result.Error, "server.port must be <= 65535")
	})

	t.Run("invalid file format", func(t *testing.T) {
		configPath := filepath.Join(tempDir, "config.txt")
		configData := "invalid config format"
		err := os.WriteFile(configPath, []byte(configData), 0644)
		require.NoError(t, err)

		os.Setenv("SSSONECTOR_CONFIG", configPath)
		defer os.Unsetenv("SSSONECTOR_CONFIG")

		result := ConfigValidationCheck(ctx, logger)
		assert.Equal(t, "Failed", result.Status)
		assert.ErrorContains(t, result.Error, "unsupported config file format")
	})

	t.Run("file not found", func(t *testing.T) {
		os.Setenv("SSSONECTOR_CONFIG", "/nonexistent/config.yaml")
		defer os.Unsetenv("SSSONECTOR_CONFIG")

		result := ConfigValidationCheck(ctx, logger)
		assert.Equal(t, "Failed", result.Status)
		assert.ErrorContains(t, result.Error, "failed to read config file")
	})
}

func TestDurationValidation(t *testing.T) {
	tests := []struct {
		name     string
		duration string
		valid    bool
	}{
		{"valid seconds", "30s", true},
		{"valid minutes", "5m", true},
		{"valid hours", "2h", true},
		{"invalid unit", "5x", false},
		{"no unit", "5", false},
		{"empty", "", false},
		{"invalid format", "5mm", false},
		{"negative", "-5s", false},
		{"decimal", "5.5s", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.valid, isValidDuration(tt.duration))
		})
	}
}
