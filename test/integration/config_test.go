package integration

import (
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
	"github.com/stretchr/testify/assert"
)

func TestConfigValidation(t *testing.T) {
	cfg := TestConfig()

	// Test valid configuration
	assert.NotNil(t, cfg)
	assert.Equal(t, types.TypeClient, cfg.Type)
	assert.Equal(t, "1.0.0", cfg.Version)
	assert.NotNil(t, cfg.Config)
	assert.Equal(t, types.ModeClient, cfg.Config.Mode)
}

func TestConfigDefaults(t *testing.T) {
	cfg := types.DefaultConfig()

	// Test default values
	assert.NotNil(t, cfg)
	assert.Equal(t, types.TypeServer, cfg.Type)
	assert.Equal(t, "1.0.0", cfg.Version)
	assert.NotNil(t, cfg.Config)
	assert.Equal(t, types.ModeServer, cfg.Config.Mode)
	assert.Equal(t, "/var/lib/sssonector", cfg.Config.StateDir)
	assert.Equal(t, "/var/log/sssonector", cfg.Config.LogDir)
}

func TestConfigLogging(t *testing.T) {
	cfg := TestConfig()

	// Test logging configuration
	assert.NotNil(t, cfg.Config.Logging)
	assert.Equal(t, "info", cfg.Config.Logging.Level)
	assert.Equal(t, "text", cfg.Config.Logging.Format)
	assert.Equal(t, "stdout", cfg.Config.Logging.Output)
	assert.Equal(t, "/var/log/sssonector/sssonector.log", cfg.Config.Logging.File)
}

func TestConfigNetwork(t *testing.T) {
	cfg := TestConfig()

	// Test network configuration
	assert.NotNil(t, cfg.Config.Network)
	assert.Equal(t, 1500, cfg.Config.Network.MTU)
	assert.Equal(t, "eth0", cfg.Config.Network.Interface)
	assert.Equal(t, "10.0.0.1", cfg.Config.Network.Address)
	assert.Equal(t, []string{"8.8.8.8", "8.8.4.4"}, cfg.Config.Network.DNSServers)
}

func TestConfigTunnel(t *testing.T) {
	cfg := TestConfig()

	// Test tunnel configuration
	assert.NotNil(t, cfg.Config.Tunnel)
	assert.Equal(t, "localhost", cfg.Config.Tunnel.ServerAddress)
	assert.Equal(t, 8443, cfg.Config.Tunnel.ServerPort)
	assert.Equal(t, 8443, cfg.Config.Tunnel.Port)
	assert.Equal(t, "tcp", cfg.Config.Tunnel.Protocol)
	assert.True(t, cfg.Config.Tunnel.Compression)
	assert.Equal(t, 60*time.Second, cfg.Config.Tunnel.Keepalive.Duration)
}

func TestConfigSecurity(t *testing.T) {
	cfg := TestConfig()

	// Test security configuration
	assert.NotNil(t, cfg.Config.Security)
	assert.NotNil(t, cfg.Config.Security.TLS)
	assert.Equal(t, "certificate", cfg.Config.Security.AuthMethod)
	assert.NotNil(t, cfg.Config.Security.CertRotation)
	assert.True(t, cfg.Config.Security.CertRotation.Enabled)
	assert.Equal(t, 24*time.Hour, cfg.Config.Security.CertRotation.Interval.Duration)
}

func TestConfigMonitor(t *testing.T) {
	cfg := TestConfig()

	// Test monitor configuration
	assert.NotNil(t, cfg.Config.Monitor)
	assert.True(t, cfg.Config.Monitor.Enabled)
	assert.Equal(t, time.Minute, cfg.Config.Monitor.Interval.Duration)
	assert.NotNil(t, cfg.Config.Monitor.Prometheus)
	assert.True(t, cfg.Config.Monitor.Prometheus.Enabled)
}

func TestConfigMetrics(t *testing.T) {
	cfg := TestConfig()

	// Test metrics configuration
	assert.NotNil(t, cfg.Config.Metrics)
	assert.True(t, cfg.Config.Metrics.Enabled)
	assert.Equal(t, 10*time.Second, cfg.Config.Metrics.Interval.Duration)
	assert.Equal(t, "localhost:8080", cfg.Config.Metrics.Address)
}

func TestConfigSNMP(t *testing.T) {
	cfg := TestConfig()

	// Test SNMP configuration
	assert.NotNil(t, cfg.Config.SNMP)
	assert.True(t, cfg.Config.SNMP.Enabled)
	assert.Equal(t, "public", cfg.Config.SNMP.Community)
	assert.Equal(t, 161, cfg.Config.SNMP.Port)
}

func TestConfigThrottle(t *testing.T) {
	cfg := TestConfig()

	// Test throttle configuration
	assert.NotNil(t, cfg.Throttle)
	assert.True(t, cfg.Throttle.Enabled)
	assert.Equal(t, int64(1000000), cfg.Throttle.Rate)
	assert.Equal(t, int64(100000), cfg.Throttle.Burst)
}

func TestConfigMetadata(t *testing.T) {
	cfg := TestConfig()

	// Test metadata configuration
	assert.NotNil(t, cfg.Metadata)
	assert.Equal(t, "1.0.0", cfg.Metadata.Version)
	assert.Equal(t, "test", cfg.Metadata.Environment)
	assert.Equal(t, "local", cfg.Metadata.Region)
	assert.NotEmpty(t, cfg.Metadata.LastModified)
	assert.NotEmpty(t, cfg.Metadata.Created)
	assert.NotEmpty(t, cfg.Metadata.Modified)
	assert.NotEmpty(t, cfg.Metadata.CreatedAt)
	assert.NotEmpty(t, cfg.Metadata.UpdatedAt)
}
