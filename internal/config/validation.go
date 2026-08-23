// Package validator provides configuration validation functionality
package config

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// Validator implements ConfigValidator interface
type Validator struct{}

// NewValidator creates a new Validator instance
func NewValidator() *Validator {
	return &Validator{}
}

// Validate validates the configuration
func (v *Validator) Validate(config *AppConfig) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	if config.Config == nil {
		return fmt.Errorf("config.Config cannot be nil")
	}

	// Validate version first
	if err := v.validateVersion(config); err != nil {
		return fmt.Errorf("invalid version: %v", err)
	}

	// Validate security configuration
	if err := v.validateSecurityConfig(config); err != nil {
		return fmt.Errorf("invalid security config: %v", err)
	}

	// Validate certificate file paths and extensions
	if err := v.validateCertificateFilesExist(config); err != nil {
		return fmt.Errorf("invalid certificate files: %v", err)
	}

	// Validate throttle configuration
	if err := v.validateThrottleConfig(config); err != nil {
		return fmt.Errorf("invalid throttle config: %v", err)
	}

	// Validate environment-specific configuration
	if err := v.validateEnvironmentConfig(config); err != nil {
		return fmt.Errorf("invalid environment config: %v", err)
	}

	if err := v.validateMode(config.Config.Mode); err != nil {
		return fmt.Errorf("invalid mode: %v", err)
	}

	if err := v.validateLogging(config.Config.Logging); err != nil {
		return fmt.Errorf("invalid logging config: %v", err)
	}

	if err := v.validateNetwork(config.Config.Network); err != nil {
		return fmt.Errorf("invalid network config: %v", err)
	}

	if err := v.validateTunnel(config.Config.Mode, config.Config.Tunnel); err != nil {
		return fmt.Errorf("invalid tunnel config: %v", err)
	}

	if err := v.validateSecurity(config.Config.Security); err != nil {
		return fmt.Errorf("invalid security config: %v", err)
	}

	if err := v.validateMonitor(config.Config.Monitor); err != nil {
		return fmt.Errorf("invalid monitor config: %v", err)
	}

	if err := v.validateMetrics(config.Config.Metrics); err != nil {
		return fmt.Errorf("invalid metrics config: %v", err)
	}

	if err := v.validateThrottle(config.Throttle); err != nil {
		return fmt.Errorf("invalid throttle config: %v", err)
	}

	if err := v.validateFacade(config.Config.Facade, config.Config.Mode); err != nil {
		return fmt.Errorf("invalid facade config: %v", err)
	}

	return nil
}

func (v *Validator) validateMode(mode string) error {
	switch mode {
	case ModeServer, ModeClient:
		return nil
	default:
		return fmt.Errorf("invalid mode: %s", mode)
	}
}

func (v *Validator) validateLogging(config LoggingConfig) error {
	if config.Level == "" {
		return fmt.Errorf("log level cannot be empty")
	}

	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"fatal": true,
	}

	if !validLevels[strings.ToLower(config.Level)] {
		return fmt.Errorf("invalid log level: %s", config.Level)
	}

	if config.Format != "" {
		validFormats := map[string]bool{
			"json":    true,
			"console": true,
		}
		if !validFormats[strings.ToLower(config.Format)] {
			return fmt.Errorf("invalid log format: %s", config.Format)
		}
	}

	return nil
}

func (v *Validator) validateNetwork(config NetworkConfig) error {
	if config.Interface == "" {
		return fmt.Errorf("interface cannot be empty")
	}

	if config.MTU < 576 || config.MTU > 65535 {
		return fmt.Errorf("invalid MTU: %d", config.MTU)
	}

	if config.Address != "" {
		if _, _, err := net.ParseCIDR(config.Address); err != nil {
			return fmt.Errorf("invalid IP address: %s", config.Address)
		}
	}

	for _, dns := range config.DNSServers {
		if ip := net.ParseIP(dns); ip == nil {
			return fmt.Errorf("invalid DNS server IP: %s", dns)
		}
	}

	return nil
}

func (v *Validator) ValidateIPAddress(ipStr string) error {
	// Validate IPv4 address
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return fmt.Errorf("invalid IP address format: %s", ipStr)
	}

	// Check if it's a valid unicast address
	if ip.IsUnspecified() || ip.IsMulticast() || ip.IsLoopback() {
		return fmt.Errorf("invalid IP address type: %s", ipStr)
	}

	return nil
}

func (v *Validator) ValidateCIDR(cidrStr string) error {
	_, _, err := net.ParseCIDR(cidrStr)
	if err != nil {
		return fmt.Errorf("invalid CIDR notation: %s", cidrStr)
	}
	return nil
}

func (v *Validator) validateEnvironmentConfig(config *AppConfig) error {
	// Validate environment-specific settings
	validEnvironments := map[string]bool{
		"development": true,
		"staging":     true,
		"production":  true,
		"test":        true,
		"qa":          true,
	}

	if !validEnvironments[config.Metadata.Environment] {
		return fmt.Errorf("invalid environment: %s", config.Metadata.Environment)
	}

	// Environment-specific validation
	switch config.Metadata.Environment {
	case "production":
		// Production should have stricter security
		if config.Config.Security.TLS.MinVersion != "1.3" {
			return fmt.Errorf("production environment requires TLS 1.3 minimum")
		}
		// Check if any TLS settings are configured (indicating TLS is enabled)
		if config.Config.Security.TLS.MinVersion == "" && config.Config.Security.TLS.MaxVersion == "" {
			return fmt.Errorf("TLS must be configured in production")
		}
	case "development":
		// Development can have more lenient settings
		// No additional restrictions
	}

	return nil
}

func (v *Validator) validateTunnel(mode string, config TunnelConfig) error {
	switch strings.ToLower(mode) {
	case "server":
		if config.ListenPort < 1 || config.ListenPort > 65535 {
			return fmt.Errorf("invalid listen_port: %d", config.ListenPort)
		}
	case "client":
		if config.ServerAddress == "" {
			return fmt.Errorf("server_address cannot be empty in client mode")
		}
		if config.ServerPort < 1 || config.ServerPort > 65535 {
			return fmt.Errorf("invalid server_port: %d", config.ServerPort)
		}
	default:
		return fmt.Errorf("unknown mode: %s", mode)
	}

	rc := config.Reconnect.Normalized()
	if rc.MaxAttempts > 100 {
		return fmt.Errorf("reconnect max_attempts too large: %d", rc.MaxAttempts)
	}
	if rc.InitialDelay > 5*time.Minute {
		return fmt.Errorf("reconnect initial_delay too large: %v", rc.InitialDelay)
	}
	if rc.MaxDelay < rc.InitialDelay {
		return fmt.Errorf("reconnect max_delay (%v) must be >= initial_delay (%v)", rc.MaxDelay, rc.InitialDelay)
	}
	if rc.Jitter > 0.9 {
		return fmt.Errorf("reconnect jitter must be <= 0.9, got %v", rc.Jitter)
	}

	return nil
}

func (v *Validator) validateSecurity(config SecurityConfig) error {
	if config.TLS.MinVersion == "" {
		return fmt.Errorf("TLS min version cannot be empty")
	}

	if config.TLS.MaxVersion == "" {
		return fmt.Errorf("TLS max version cannot be empty")
	}

	validVersions := map[string]bool{
		"1.2": true,
		"1.3": true,
	}

	if !validVersions[config.TLS.MinVersion] {
		return fmt.Errorf("invalid TLS min version: %s", config.TLS.MinVersion)
	}

	if !validVersions[config.TLS.MaxVersion] {
		return fmt.Errorf("invalid TLS max version: %s", config.TLS.MaxVersion)
	}

	return nil
}

func (v *Validator) validateMonitor(config MonitorConfig) error {
	if !config.Enabled {
		return nil
	}

	if config.Type == "" {
		return fmt.Errorf("monitor type cannot be empty when monitoring is enabled")
	}

	validTypes := map[string]bool{
		"prometheus": true,
		"snmp":       true,
	}

	if !validTypes[strings.ToLower(config.Type)] {
		return fmt.Errorf("invalid monitor type: %s", config.Type)
	}

	if config.Interval.Seconds() < 1 {
		return fmt.Errorf("invalid monitor interval: %v", config.Interval)
	}

	return nil
}

func (v *Validator) validateMetrics(config MetricsConfig) error {
	if !config.Enabled {
		return nil
	}

	if config.Interval.Seconds() < 1 {
		return fmt.Errorf("invalid metrics interval: %v", config.Interval)
	}

	return nil
}

func (v *Validator) validateThrottle(config ThrottleConfig) error {
	if !config.Enabled {
		return nil
	}

	if config.Rate <= 0 {
		return fmt.Errorf("invalid rate: %f", config.Rate)
	}

	// burst is accepted for schema compatibility but unused: the limiter
	// derives it as 100ms of the effective rate, so it is not validated.

	return nil
}

func (v *Validator) validateVersion(config *AppConfig) error {
	if config.Metadata.SchemaVersion == "" {
		return fmt.Errorf("schema version cannot be empty")
	}
	if config.Metadata.SchemaVersion != CurrentSchemaVersion {
		return fmt.Errorf("unsupported schema version %q: only %q is supported",
			config.Metadata.SchemaVersion, CurrentSchemaVersion)
	}
	if config.Config.Security.TLS.MinVersion == "" {
		return fmt.Errorf("schema 2.0.0 requires explicit TLS minimum version configuration")
	}
	return nil
}

// validateSecurityConfig validates security-related configuration
func (v *Validator) validateSecurityConfig(config *AppConfig) error {
	// Validate certificate file paths
	if config.Config.Auth.CertFile != "" {
		// Basic path validation - should be absolute or relative path
		if strings.Contains(config.Config.Auth.CertFile, "..") {
			return fmt.Errorf("certificate file path contains invalid characters: %s", config.Config.Auth.CertFile)
		}
	}

	if config.Config.Auth.KeyFile != "" {
		if strings.Contains(config.Config.Auth.KeyFile, "..") {
			return fmt.Errorf("key file path contains invalid characters: %s", config.Config.Auth.KeyFile)
		}
	}

	if config.Config.Auth.CAFile != "" {
		if strings.Contains(config.Config.Auth.CAFile, "..") {
			return fmt.Errorf("CA file path contains invalid characters: %s", config.Config.Auth.CAFile)
		}
	}

	// Validate certificate rotation settings
	if config.Config.Auth.CertRotation.Enabled {
		if config.Config.Auth.CertRotation.Interval < time.Hour {
			return fmt.Errorf("certificate rotation interval too short: %v", config.Config.Auth.CertRotation.Interval)
		}
	}

	// Validate TLS configuration
	if config.Config.Security.TLS.MinVersion > config.Config.Security.TLS.MaxVersion {
		return fmt.Errorf("TLS min version cannot be greater than max version")
	}

	// Validate cipher suites
	if len(config.Config.Security.TLS.Ciphers) > 0 {
		validCiphers := map[string]bool{
			"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":   true,
			"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":   true,
			"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256": true,
			"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384": true,
		}

		for _, cipher := range config.Config.Security.TLS.Ciphers {
			if !validCiphers[cipher] {
				return fmt.Errorf("invalid cipher suite: %s", cipher)
			}
		}
	}

	return nil
}

func (v *Validator) validateThrottleConfig(config *AppConfig) error {
	if !config.Throttle.Enabled {
		return nil
	}

	// Validate rate vs burst relationship
	if config.Throttle.Rate > 0 && config.Throttle.Burst > 0 {
		// Burst should not be more than 10x the rate
		rateRatio := float64(config.Throttle.Burst) / config.Throttle.Rate
		if rateRatio > 10 {
			return fmt.Errorf("burst limit too large compared to rate: %.2f", rateRatio)
		}
	}

	// Validate reasonable rate limits
	if config.Throttle.Rate < 1024 { // 1KB/s minimum
		return fmt.Errorf("rate limit too low: %f", config.Throttle.Rate)
	}

	if config.Throttle.Rate > 1024*1024*1024 { // 1GB/s maximum
		return fmt.Errorf("rate limit too high: %f", config.Throttle.Rate)
	}

	return nil
}

func (v *Validator) validateFacade(config FacadeConfig, mode string) error {
	if !config.Enabled {
		return nil
	}

	if mode == ModeServer {
		// Server-side facade validation
		if config.ListenPort < 1 || config.ListenPort > 65535 {
			return fmt.Errorf("invalid facade listen port: %d", config.ListenPort)
		}

		if len(config.TunnelPorts) == 0 {
			return fmt.Errorf("facade requires at least one tunnel port in tunnel_ports")
		}

		for _, port := range config.TunnelPorts {
			if port < 1 || port > 65535 {
				return fmt.Errorf("invalid tunnel port in facade config: %d", port)
			}
		}

		// Facade listen port must not collide with any tunnel port it routes to
		for _, port := range config.TunnelPorts {
			if port == config.ListenPort {
				return fmt.Errorf("facade listen port %d conflicts with tunnel port", port)
			}
		}
	}

	if mode == ModeClient {
		// Client-side facade validation
		if config.ServerPort < 1 || config.ServerPort > 65535 {
			return fmt.Errorf("invalid facade server port: %d", config.ServerPort)
		}

		if config.DirectTimeout > 0 && config.DirectTimeout < time.Second {
			return fmt.Errorf("facade direct_timeout too short: %v (minimum 1s)", config.DirectTimeout)
		}
	}

	// Token TTL validation (applies to both server and client)
	if config.TokenTTL > 0 {
		if config.TokenTTL < 5*time.Second {
			return fmt.Errorf("facade token_ttl too short: %v (minimum 5s)", config.TokenTTL)
		}
		if config.TokenTTL > 120*time.Second {
			return fmt.Errorf("facade token_ttl too long: %v (maximum 120s)", config.TokenTTL)
		}
	}

	return nil
}

func (v *Validator) validateCertificateFilesExist(config *AppConfig) error {
	// Enhanced certificate file validation
	if config.Config.Auth.CertFile != "" {
		// Check for common path issues
		if strings.HasPrefix(config.Config.Auth.CertFile, "..") {
			return fmt.Errorf("certificate file path contains parent directory reference: %s", config.Config.Auth.CertFile)
		}

		if strings.Contains(config.Config.Auth.CertFile, "//") {
			return fmt.Errorf("certificate file path contains duplicate slashes: %s", config.Config.Auth.CertFile)
		}

		// Check file extension
		if !strings.HasSuffix(config.Config.Auth.CertFile, ".crt") &&
			!strings.HasSuffix(config.Config.Auth.CertFile, ".pem") {
			return fmt.Errorf("certificate file should have .crt or .pem extension: %s", config.Config.Auth.CertFile)
		}
	}

	if config.Config.Auth.KeyFile != "" {
		if strings.HasPrefix(config.Config.Auth.KeyFile, "..") {
			return fmt.Errorf("key file path contains parent directory reference: %s", config.Config.Auth.KeyFile)
		}

		if strings.Contains(config.Config.Auth.KeyFile, "//") {
			return fmt.Errorf("key file path contains duplicate slashes: %s", config.Config.Auth.KeyFile)
		}

		// Check file extension
		if !strings.HasSuffix(config.Config.Auth.KeyFile, ".key") &&
			!strings.HasSuffix(config.Config.Auth.KeyFile, ".pem") {
			return fmt.Errorf("key file should have .key or .pem extension: %s", config.Config.Auth.KeyFile)
		}
	}

	if config.Config.Auth.CAFile != "" {
		if strings.HasPrefix(config.Config.Auth.CAFile, "..") {
			return fmt.Errorf("CA file path contains parent directory reference: %s", config.Config.Auth.CAFile)
		}

		if strings.Contains(config.Config.Auth.CAFile, "//") {
			return fmt.Errorf("CA file path contains duplicate slashes: %s", config.Config.Auth.CAFile)
		}

		// Check file extension
		if !strings.HasSuffix(config.Config.Auth.CAFile, ".crt") &&
			!strings.HasSuffix(config.Config.Auth.CAFile, ".pem") {
			return fmt.Errorf("CA file should have .crt or .pem extension: %s", config.Config.Auth.CAFile)
		}
	}

	return nil
}
