package main

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommands(t *testing.T) {
	ctx := context.Background()

	t.Run("connections command", func(t *testing.T) {
		output := captureOutput(func() {
			err := cmdConnections(ctx, nil)
			assert.NoError(t, err)
		})

		var stats ConnectionStats
		err := json.Unmarshal([]byte(output), &stats)
		assert.NoError(t, err)
		assert.Greater(t, stats.TotalConnections, int64(0))
	})

	t.Run("endpoints command", func(t *testing.T) {
		t.Run("list", func(t *testing.T) {
			output := captureOutput(func() {
				err := cmdEndpoints(ctx, []string{"list"})
				assert.NoError(t, err)
			})

			var endpoints []struct {
				Address string `json:"address"`
				Weight  int    `json:"weight"`
				Health  string `json:"health"`
			}
			err := json.Unmarshal([]byte(output), &endpoints)
			assert.NoError(t, err)
			assert.NotEmpty(t, endpoints)
		})

		t.Run("add", func(t *testing.T) {
			output := captureOutput(func() {
				err := cmdEndpoints(ctx, []string{"add", "localhost:8084", "1"})
				assert.NoError(t, err)
			})
			assert.Contains(t, output, "Added endpoint localhost:8084")
		})

		t.Run("remove", func(t *testing.T) {
			output := captureOutput(func() {
				err := cmdEndpoints(ctx, []string{"remove", "localhost:8084"})
				assert.NoError(t, err)
			})
			assert.Contains(t, output, "Removed endpoint localhost:8084")
		})

		t.Run("invalid subcommand", func(t *testing.T) {
			err := cmdEndpoints(ctx, []string{"invalid"})
			assert.Error(t, err)
		})
	})

	t.Run("circuit breaker command", func(t *testing.T) {
		output := captureOutput(func() {
			err := cmdCircuitBreaker(ctx, nil)
			assert.NoError(t, err)
		})

		var stats CircuitBreakerStats
		err := json.Unmarshal([]byte(output), &stats)
		assert.NoError(t, err)
		assert.NotEmpty(t, stats.State)
	})

	t.Run("rate limits command", func(t *testing.T) {
		output := captureOutput(func() {
			err := cmdRateLimits(ctx, nil)
			assert.NoError(t, err)
		})

		var stats RateLimitStats
		err := json.Unmarshal([]byte(output), &stats)
		assert.NoError(t, err)
		assert.True(t, stats.Enabled)
	})

	t.Run("pool command", func(t *testing.T) {
		output := captureOutput(func() {
			err := cmdPool(ctx, nil)
			assert.NoError(t, err)
		})

		var stats PoolStats
		err := json.Unmarshal([]byte(output), &stats)
		assert.NoError(t, err)
		assert.Greater(t, stats.Size, 0)
	})

	t.Run("health command", func(t *testing.T) {
		output := captureOutput(func() {
			err := cmdHealth(ctx, nil)
			assert.NoError(t, err)
		})

		var health struct {
			Status    string `json:"status"`
			Timestamp string `json:"timestamp"`
			Checks    []struct {
				Name   string `json:"name"`
				Status string `json:"status"`
			} `json:"checks"`
		}
		err := json.Unmarshal([]byte(output), &health)
		assert.NoError(t, err)
		assert.Equal(t, "healthy", health.Status)
		assert.NotEmpty(t, health.Checks)
	})

	t.Run("debug command", func(t *testing.T) {
		t.Run("enable", func(t *testing.T) {
			output := captureOutput(func() {
				err := cmdDebug(ctx, []string{"enable"})
				assert.NoError(t, err)
			})
			assert.Contains(t, output, "Debug mode enabled")
		})

		t.Run("disable", func(t *testing.T) {
			output := captureOutput(func() {
				err := cmdDebug(ctx, []string{"disable"})
				assert.NoError(t, err)
			})
			assert.Contains(t, output, "Debug mode disabled")
		})

		t.Run("invalid subcommand", func(t *testing.T) {
			err := cmdDebug(ctx, []string{"invalid"})
			assert.Error(t, err)
		})
	})

	t.Run("maintenance command", func(t *testing.T) {
		t.Run("enter", func(t *testing.T) {
			output := captureOutput(func() {
				err := cmdMaintenance(ctx, []string{"enter"})
				assert.NoError(t, err)
			})
			assert.Contains(t, output, "Entered maintenance mode")
		})

		t.Run("exit", func(t *testing.T) {
			output := captureOutput(func() {
				err := cmdMaintenance(ctx, []string{"exit"})
				assert.NoError(t, err)
			})
			assert.Contains(t, output, "Exited maintenance mode")
		})

		t.Run("invalid subcommand", func(t *testing.T) {
			err := cmdMaintenance(ctx, []string{"invalid"})
			assert.Error(t, err)
		})
	})
}

func TestConfig(t *testing.T) {
	ctx := context.Background()
	config := defaultConfig
	configFile := "test_config.json"

	t.Run("show config", func(t *testing.T) {
		output := captureOutput(func() {
			err := cmdConfig(ctx, nil, &config, configFile)
			assert.NoError(t, err)
		})

		var cfg Config
		err := json.Unmarshal([]byte(output), &cfg)
		assert.NoError(t, err)
		assert.Equal(t, config.LogLevel, cfg.LogLevel)
	})

	t.Run("set config value", func(t *testing.T) {
		err := setConfigValue(&config, "LogLevel", "debug")
		assert.NoError(t, err)
		assert.Equal(t, "debug", config.LogLevel)
	})

	t.Run("get config value", func(t *testing.T) {
		value, err := getConfigValue(&config, "LogLevel")
		assert.NoError(t, err)
		assert.Equal(t, "debug", value)
	})

	t.Run("invalid config path", func(t *testing.T) {
		_, err := getConfigValue(&config, "InvalidPath")
		assert.Error(t, err)
	})
}

// captureOutput captures stdout during the execution of a function
func captureOutput(f func()) string {
	// Create a pipe
	reader, writer, err := os.Pipe()
	if err != nil {
		panic(err)
	}

	// Save the original stdout
	stdout := os.Stdout
	// Set the pipe writer as stdout
	os.Stdout = writer

	// Run the function
	f()

	// Reset stdout
	os.Stdout = stdout

	// Close the writer
	writer.Close()

	// Read the output
	var buf strings.Builder
	_, err = io.Copy(&buf, reader)
	if err != nil {
		panic(err)
	}

	return strings.TrimSpace(buf.String())
}
