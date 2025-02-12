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
		Config:  testBaseConfig(),
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
		Throttle: testThrottleConfig(),
	}
}

// testBaseConfig returns a test base configuration
func testBaseConfig() *types.Config {
	return &types.Config{
		Mode:     types.ModeClient,
		StateDir: "/var/lib/sssonector",
		LogDir:   "/var/log/sssonector",
		Logging:  testLoggingConfig(),
		Network:  testNetworkConfig(),
		Tunnel:   testTunnelConfig(),
		Security: testSecurityConfig(),
		Monitor:  testMonitorConfig(),
		Metrics:  testMetricsConfig(),
		SNMP:     testSNMPConfig(),
	}
}

// testLoggingConfig returns a test logging configuration
func testLoggingConfig() *types.LoggingConfig {
	cfg := types.NewLoggingConfig()
	cfg.File = "/var/log/sssonector/sssonector.log"
	return cfg
}

// testNetworkConfig returns a test network configuration
func testNetworkConfig() *types.NetworkConfig {
	cfg := types.NewNetworkConfig()
	cfg.Interface = "eth0"
	cfg.Address = "10.0.0.1"
	cfg.DNSServers = []string{"8.8.8.8", "8.8.4.4"}
	return cfg
}

// testTunnelConfig returns a test tunnel configuration
func testTunnelConfig() *types.TunnelConfig {
	cfg := types.NewTunnelConfig()
	cfg.ServerAddress = "localhost"
	cfg.ServerPort = 8443
	cfg.Port = 8443
	cfg.Protocol = "tcp"
	cfg.Compression = true
	cfg.Keepalive = types.NewDuration(60 * time.Second)
	return cfg
}

// testSecurityConfig returns a test security configuration
func testSecurityConfig() *types.SecurityConfig {
	cfg := types.NewSecurityConfig()
	cfg.TLS = testTLSConfig()
	cfg.AuthMethod = "certificate"
	cfg.CertRotation = testCertRotation()
	return cfg
}

// testTLSConfig returns a test TLS configuration
func testTLSConfig() *types.TLSConfig {
	cfg := types.NewTLSConfig()
	cfg.Options = testTLSConfigOptions()
	cfg.CertRotation = testCertRotation()
	return cfg
}

// testTLSConfigOptions returns a test TLS options configuration
func testTLSConfigOptions() *types.TLSConfigOptions {
	cfg := types.NewTLSConfigOptions()
	cfg.CipherSuites = []string{"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"}
	return cfg
}

// testCertRotation returns a test certificate rotation configuration
func testCertRotation() *types.CertRotation {
	cfg := types.NewCertRotation()
	cfg.Enabled = true
	cfg.Interval = types.NewDuration(24 * time.Hour)
	return cfg
}

// testMonitorConfig returns a test monitor configuration
func testMonitorConfig() *types.MonitorConfig {
	cfg := types.NewMonitorConfig()
	cfg.Enabled = true
	cfg.Interval = types.NewDuration(time.Minute)
	cfg.Prometheus = testPrometheusConfig()
	return cfg
}

// testPrometheusConfig returns a test Prometheus configuration
func testPrometheusConfig() *types.PrometheusConfig {
	cfg := types.NewPrometheusConfig()
	cfg.Enabled = true
	return cfg
}

// testMetricsConfig returns a test metrics configuration
func testMetricsConfig() *types.MetricsConfig {
	cfg := types.NewMetricsConfig()
	cfg.Enabled = true
	cfg.Interval = types.NewDuration(10 * time.Second)
	return cfg
}

// testSNMPConfig returns a test SNMP configuration
func testSNMPConfig() *types.SNMPConfig {
	cfg := types.NewSNMPConfig()
	cfg.Enabled = true
	return cfg
}

// testThrottleConfig returns a test throttle configuration
func testThrottleConfig() *types.ThrottleConfig {
	cfg := types.NewThrottleConfig()
	cfg.Enabled = true
	cfg.Rate = 1000000
	cfg.Burst = 100000
	return cfg
}

// CompareConfigs compares two configurations
func CompareConfigs(expected, actual *types.AppConfig) error {
	if expected.Type != actual.Type {
		return fmt.Errorf("type mismatch: expected %s, got %s", expected.Type, actual.Type)
	}

	if expected.Version != actual.Version {
		return fmt.Errorf("version mismatch: expected %s, got %s", expected.Version, actual.Version)
	}

	if err := compareConfig(expected.Config, actual.Config); err != nil {
		return fmt.Errorf("config mismatch: %v", err)
	}

	return nil
}

// compareConfig compares two base configurations
func compareConfig(expected, actual *types.Config) error {
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
