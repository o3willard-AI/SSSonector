package config

import (
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/stretchr/testify/assert"
)

func TestLoadConfig(t *testing.T) {
	// Test loading default configuration
	cfg := types.DefaultConfig()
	assert.NotNil(t, cfg)
	assert.Equal(t, types.TypeServer, cfg.Type)
	assert.Equal(t, "1.0.0", cfg.Version)
	assert.NotNil(t, cfg.Config)
	assert.Equal(t, types.ModeServer, cfg.Config.Mode)
	assert.Equal(t, "/var/lib/sssonector", cfg.Config.StateDir)
	assert.Equal(t, "/var/log/sssonector", cfg.Config.LogDir)
}

func TestConfigValidation(t *testing.T) {
	// Test valid configuration
	cfg := &types.AppConfig{
		Type:    types.TypeClient,
		Version: "1.0.0",
		Config: &types.ServiceConfig{
			Mode:     types.ModeClient,
			StateDir: "/var/lib/sssonector",
			LogDir:   "/var/log/sssonector",
			Logging:  &types.LoggingConfig{},
			Network:  &types.NetworkConfig{},
			Tunnel:   &types.TunnelConfig{},
		},
		Metadata: &types.ConfigMetadata{},
	}

	assert.NotNil(t, cfg)
	assert.Equal(t, types.TypeClient, cfg.Type)
	assert.Equal(t, "1.0.0", cfg.Version)
	assert.Equal(t, types.ModeClient, cfg.Config.Mode)
}

func TestConfigDefaults(t *testing.T) {
	// Test default values
	cfg := types.DefaultConfig()

	assert.NotNil(t, cfg)
	assert.Equal(t, types.TypeServer, cfg.Type)
	assert.Equal(t, "1.0.0", cfg.Version)
	assert.NotNil(t, cfg.Config)
	assert.Equal(t, types.ModeServer, cfg.Config.Mode)
	assert.Equal(t, "/var/lib/sssonector", cfg.Config.StateDir)
	assert.Equal(t, "/var/log/sssonector", cfg.Config.LogDir)
}

func TestConfigLogging(t *testing.T) {
	// Test logging configuration
	cfg := types.DefaultConfig()

	assert.NotNil(t, cfg.Config.Logging)
	assert.Equal(t, "info", cfg.Config.Logging.Level)
	assert.Equal(t, "text", cfg.Config.Logging.Format)
	assert.Equal(t, "stdout", cfg.Config.Logging.Output)
}

func TestConfigNetwork(t *testing.T) {
	// Test network configuration
	cfg := types.DefaultConfig()

	assert.NotNil(t, cfg.Config.Network)
	assert.Equal(t, 1500, cfg.Config.Network.MTU)
	assert.Empty(t, cfg.Config.Network.DNS)
	assert.Empty(t, cfg.Config.Network.Routes)
}

func TestConfigTunnel(t *testing.T) {
	// Test tunnel configuration
	cfg := types.DefaultConfig()

	assert.NotNil(t, cfg.Config.Tunnel)
	assert.Equal(t, 1500, cfg.Config.Tunnel.MTU)
	assert.Equal(t, "tcp", cfg.Config.Tunnel.Protocol)
	assert.False(t, cfg.Config.Tunnel.Compression)
	assert.Equal(t, time.Minute, cfg.Config.Tunnel.Keepalive.Duration)
}

func TestConfigSecurity(t *testing.T) {
	// Test security configuration
	cfg := types.DefaultConfig()

	assert.NotNil(t, cfg.Config.Security)
	assert.True(t, cfg.Config.Security.MemoryProtections.Enabled)
	assert.True(t, cfg.Config.Security.Namespace.Enabled)
	assert.True(t, cfg.Config.Security.Capabilities.Enabled)
	assert.Equal(t, "1.2", cfg.Config.Security.TLS.MinVersion)
	assert.Equal(t, "1.3", cfg.Config.Security.TLS.MaxVersion)
}

func TestConfigMonitor(t *testing.T) {
	// Test monitor configuration
	cfg := types.DefaultConfig()

	assert.NotNil(t, cfg.Config.Monitor)
	assert.False(t, cfg.Config.Monitor.Enabled)
	assert.Equal(t, time.Minute, cfg.Config.Monitor.Interval.Duration)
	assert.Equal(t, "basic", cfg.Config.Monitor.Type)
}

func TestConfigMetrics(t *testing.T) {
	// Test metrics configuration
	cfg := types.DefaultConfig()

	assert.NotNil(t, cfg.Config.Metrics)
	assert.False(t, cfg.Config.Metrics.Enabled)
	assert.Equal(t, 10*time.Second, cfg.Config.Metrics.Interval.Duration)
	assert.Equal(t, "localhost:8080", cfg.Config.Metrics.Address)
	assert.Equal(t, 1000, cfg.Config.Metrics.BufferSize)
}

func TestConfigThrottle(t *testing.T) {
	// Test throttle configuration
	cfg := types.DefaultConfig()

	assert.NotNil(t, cfg.Throttle)
	assert.False(t, cfg.Throttle.Enabled)
	assert.Equal(t, int64(0), cfg.Throttle.Rate)
	assert.Equal(t, int64(0), cfg.Throttle.Burst)
}

func TestConfigMetadata(t *testing.T) {
	// Test metadata configuration
	cfg := types.DefaultConfig()

	assert.NotNil(t, cfg.Metadata)
	assert.Equal(t, "1.0.0", cfg.Metadata.Version)
	assert.Equal(t, "development", cfg.Metadata.Environment)
	assert.Equal(t, "local", cfg.Metadata.Region)
}

func TestConfigAdapter(t *testing.T) {
	// Test adapter configuration
	cfg := types.DefaultConfig()

	assert.NotNil(t, cfg.Adapter)
	assert.Equal(t, 3, cfg.Adapter.RetryAttempts)
	assert.Equal(t, time.Second, cfg.Adapter.RetryDelay)
	assert.Equal(t, 30*time.Second, cfg.Adapter.CleanupTimeout)
}
