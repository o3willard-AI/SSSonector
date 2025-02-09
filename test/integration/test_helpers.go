package integration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"github.com/o3willard-AI/SSSonector/internal/config/types"
)

// TestConfig represents a test configuration helper
type TestConfig struct {
	Dir     string
	Manager config.ConfigManager
	t       *testing.T
}

// NewTestConfig creates a new test configuration helper
func NewTestConfig(t *testing.T) (*TestConfig, error) {
	tempDir, err := os.MkdirTemp("", "sssonector-config-test")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %v", err)
	}

	manager := config.CreateManager(tempDir)

	return &TestConfig{
		Dir:     tempDir,
		Manager: manager,
		t:       t,
	}, nil
}

// Cleanup removes test resources
func (tc *TestConfig) Cleanup() {
	if err := tc.Manager.Close(); err != nil {
		tc.t.Errorf("Failed to close manager: %v", err)
	}
	os.RemoveAll(tc.Dir)
}

// CreateTestConfig creates a test configuration with the specified type
func (tc *TestConfig) CreateTestConfig(configType config.Type) *types.AppConfig {
	return &types.AppConfig{
		Type:    configType,
		Version: "1.0.0",
		Config: &types.Config{
			Mode: config.ModeClient,
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
				Keepalive:   "60s",
			},
			Security: types.SecurityConfig{
				MemoryProtections: types.MemoryProtectionsConfig{
					Enabled: true,
				},
				Namespace: types.NamespaceConfig{
					Enabled: true,
				},
				Capabilities: types.CapabilitiesConfig{
					Enabled: true,
				},
				Seccomp: types.SeccompConfig{
					Enabled: true,
				},
				TLS: types.TLSConfigOptions{
					MinVersion: "1.2",
					MaxVersion: "1.3",
					Ciphers:    []string{"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"},
				},
				AuthMethod: "certificate",
				CertRotation: types.CertRotation{
					Enabled:  true,
					Interval: 24 * time.Hour,
				},
			},
			Monitor: types.MonitorConfig{
				Enabled:  true,
				Type:     "prometheus",
				Interval: time.Minute,
				Prometheus: types.PrometheusConfig{
					Enabled:    true,
					Port:       9090,
					Path:       "/metrics",
					BufferSize: 1000,
				},
			},
			Metrics: types.MetricsConfig{
				Enabled:    true,
				Address:    "localhost:8125",
				Interval:   time.Second * 10,
				BufferSize: 1000,
			},
			SNMP: types.SNMPConfig{
				Enabled:   true,
				Port:      161,
				Community: "public",
			},
		},
		Metadata: types.ConfigMetadata{
			Version:     "1.0.0",
			Created:     time.Now(),
			Modified:    time.Now(),
			CreatedBy:   "test",
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Environment: "test",
			Region:      "us-west",
		},
		Throttle: types.ThrottleConfig{
			Enabled: true,
			Rate:    1024 * 1024 * 5, // 5MB/s
			Burst:   1024 * 1024,     // 1MB burst
		},
	}
}

// VerifyConfigFile verifies a configuration file exists with correct permissions
func (tc *TestConfig) VerifyConfigFile(configType config.Type, version string) error {
	expectedFilename := filepath.Join(tc.Dir, fmt.Sprintf("config-%s-%s.json", configType, version))
	info, err := os.Stat(expectedFilename)
	if err != nil {
		return fmt.Errorf("config file %s does not exist: %v", expectedFilename, err)
	}
	if info.Mode().Perm() != 0644 {
		return fmt.Errorf("expected file permissions 0644, got %v", info.Mode().Perm())
	}

	// Verify file content is valid JSON
	data, err := os.ReadFile(expectedFilename)
	if err != nil {
		return fmt.Errorf("failed to read config file: %v", err)
	}

	var config types.AppConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("invalid JSON in config file: %v", err)
	}

	return nil
}

// WaitForConfigUpdate waits for a configuration update on the watcher channel
func (tc *TestConfig) WaitForConfigUpdate(watcher <-chan *types.AppConfig, timeout time.Duration) (*types.AppConfig, error) {
	select {
	case config := <-watcher:
		return config, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timed out waiting for config update")
	}
}

// CompareConfigs compares two configurations for equality
func (tc *TestConfig) CompareConfigs(expected, actual *types.AppConfig) error {
	// Compare only the fields we care about, ignoring timestamps
	if expected.Type != actual.Type {
		return fmt.Errorf("type mismatch: expected %s, got %s", expected.Type, actual.Type)
	}
	if expected.Version != actual.Version {
		return fmt.Errorf("version mismatch: expected %s, got %s", expected.Version, actual.Version)
	}
	if expected.Config.Mode != actual.Config.Mode {
		return fmt.Errorf("mode mismatch: expected %s, got %s", expected.Config.Mode, actual.Config.Mode)
	}
	if expected.Config.Network.MTU != actual.Config.Network.MTU {
		return fmt.Errorf("MTU mismatch: expected %d, got %d", expected.Config.Network.MTU, actual.Config.Network.MTU)
	}
	if expected.Config.Security.CertRotation.Enabled != actual.Config.Security.CertRotation.Enabled {
		return fmt.Errorf("cert rotation enabled mismatch: expected %v, got %v",
			expected.Config.Security.CertRotation.Enabled,
			actual.Config.Security.CertRotation.Enabled)
	}
	if expected.Config.Security.CertRotation.Interval != actual.Config.Security.CertRotation.Interval {
		return fmt.Errorf("cert rotation interval mismatch: expected %v, got %v",
			expected.Config.Security.CertRotation.Interval,
			actual.Config.Security.CertRotation.Interval)
	}
	return nil
}
