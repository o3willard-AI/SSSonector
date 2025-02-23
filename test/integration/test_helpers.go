package integration

import (
	"fmt"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
)

// TestConfig returns a test configuration
func TestConfig() *types.AppConfig {
	return &types.AppConfig{
		Type:    types.TypeClient,
		Version: "1.0.0",
		Config:  testServiceConfig(),
		Metadata: &types.ConfigMetadata{
			Version:      "1.0.0",
			LastModified: time.Now().Format(time.RFC3339),
			Created:      time.Now().Format(time.RFC3339),
			Modified:     time.Now().Format(time.RFC3339),
			CreatedAt:    time.Now().Format(time.RFC3339),
			UpdatedAt:    time.Now().Format(time.RFC3339),
			Environment:  "test",
			Region:       "local",
		},
		Throttle: &types.ThrottleConfig{
			Enabled: true,
			Rate:    1000000,
			Burst:   100000,
		},
	}
}

// testServiceConfig returns a test service configuration
func testServiceConfig() *types.ServiceConfig {
	return &types.ServiceConfig{
		Mode:     types.ModeClient,
		StateDir: "/var/lib/sssonector",
		LogDir:   "/var/log/sssonector",
		Logging: &types.LoggingConfig{
			Level:  "info",
			Format: "json",
			Output: "stdout",
			File:   "/var/log/sssonector/sssonector.log",
		},
		Network: &types.NetworkConfig{
			Interface: "eth0",
			Address:   "10.0.0.1",
			MTU:       1500,
			DNS:       []string{"8.8.8.8", "8.8.4.4"},
		},
		Tunnel: &types.TunnelConfig{
			ServerAddress: "localhost",
			ServerPort:    8443,
			Port:          8443,
			Protocol:      "tcp",
			Compression:   true,
			Keepalive:     types.NewDuration(60 * time.Second),
		},
		Security: &types.SecurityConfig{
			MemoryProtections: types.MemoryProtectionConfig{
				Enabled: true,
			},
			Namespace: types.NamespaceConfig{
				Enabled: true,
			},
			Capabilities: types.CapabilitiesConfig{
				Enabled: true,
			},
			TLS: types.TLSConfig{
				MinVersion: "1.2",
				MaxVersion: "1.3",
				Options: types.TLSConfigOptions{
					CipherSuites: []string{"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"},
				},
				CertRotation: types.CertRotation{
					Enabled:  true,
					Interval: types.NewDuration(24 * time.Hour),
				},
			},
			AuthMethod: "certificate",
		},
		Monitor: &types.MonitorConfig{
			Enabled:  true,
			Type:     "basic",
			Interval: types.NewDuration(time.Minute),
			Prometheus: types.PrometheusConfig{
				Enabled: true,
				Port:    9090,
				Path:    "/metrics",
			},
		},
		Metrics: &types.MetricsConfig{
			Enabled:    true,
			Address:    "localhost:9090",
			Interval:   types.NewDuration(10 * time.Second),
			BufferSize: 1000,
		},
		SNMP: &types.SNMPConfig{
			Enabled:   true,
			Community: "public",
			Port:      161,
		},
	}
}

// CompareConfigs compares two configurations
func CompareConfigs(expected, actual *types.AppConfig) error {
	if expected.Type != actual.Type {
		return fmt.Errorf("type mismatch: expected %s, got %s", expected.Type, actual.Type)
	}

	if expected.Version != actual.Version {
		return fmt.Errorf("version mismatch: expected %s, got %s", expected.Version, actual.Version)
	}

	if err := compareServiceConfig(expected.Config, actual.Config); err != nil {
		return fmt.Errorf("config mismatch: %v", err)
	}

	return nil
}

// compareServiceConfig compares two service configurations
func compareServiceConfig(expected, actual *types.ServiceConfig) error {
	if expected.Mode != actual.Mode {
		return fmt.Errorf("mode mismatch: expected %s, got %s", expected.Mode, actual.Mode)
	}

	if expected.StateDir != actual.StateDir {
		return fmt.Errorf("state dir mismatch: expected %s, got %s", expected.StateDir, actual.StateDir)
	}

	if expected.LogDir != actual.LogDir {
		return fmt.Errorf("log dir mismatch: expected %s, got %s", expected.LogDir, actual.LogDir)
	}

	return nil
}
