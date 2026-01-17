// Package validator provides configuration validation functionality
package validator

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
)

// Validator implements ConfigValidator interface
type Validator struct{}

// NewValidator creates a new Validator instance
func NewValidator() *Validator {
	return &Validator{}
}

// Validate validates the configuration
func (v *Validator) Validate(config *types.AppConfig) error {
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

	if err := v.validateMigrationHistory(config); err != nil {
		return fmt.Errorf("invalid migration history: %v", err)
	}

	if err := v.validateMigration(config); err != nil {
		return fmt.Errorf("invalid migration path: %v", err)
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

	if err := v.validateTunnel(config.Config.Tunnel); err != nil {
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

	return nil
}

// validateMigration validates configuration migration paths and upgrade logic
func (v *Validator) validateMigration(config *types.AppConfig) error {
	history := config.Metadata.MigrationHistory

	if len(history) == 0 {
		// No migration history - this is expected for fresh configs
		return nil
	}

	// Define valid migration paths (from -> to)
	validMigrations := map[string]map[string]bool{
		"1.0.0": {
			"1.1.0": true,
			"2.0.0": true,
		},
		"1.1.0": {
			"2.0.0": true,
		},
		"2.0.0": {
			// No migrations from 2.0.0 yet
		},
	}

	// Validate each migration record sequentially
	for i, record := range history {
		// Check if this migration path is valid
		if !v.isValidMigrationPath(record.FromVersion, record.ToVersion, validMigrations) {
			return fmt.Errorf("invalid migration path: %s -> %s", record.FromVersion, record.ToVersion)
		}

		// Check chronological order (later migrations should be after earlier ones)
		if i > 0 {
			if record.Timestamp.Before(history[i-1].Timestamp) {
				return fmt.Errorf("migration record %d has timestamp before previous record", i+1)
			}
		}

		// Validate final migration leads to current schema version
		if i == len(history)-1 {
			if record.ToVersion != config.Metadata.SchemaVersion {
				return fmt.Errorf("final migration destination (%s) does not match current schema version (%s)",
					record.ToVersion, config.Metadata.SchemaVersion)
			}
		}

		// Validate starting point for first migration
		if i == 0 {
			// First migration should either be from 0.0.0 (fresh install) or 1.0.0
			if record.FromVersion != "0.0.0" && record.FromVersion != "1.0.0" {
				return fmt.Errorf("first migration must start from 0.0.0 or 1.0.0, got %s", record.FromVersion)
			}
		} else {
			// Subsequent migrations should start where the previous one ended
			if record.FromVersion != history[i-1].ToVersion {
				return fmt.Errorf("migration %d starts from %s but previous migration ended at %s",
					i+1, record.FromVersion, history[i-1].ToVersion)
			}
		}

		// Version 2.0.0 breaking changes require special validation
		if record.ToVersion == "2.0.0" {
			if err := v.validateVersionTwoMigration(config); err != nil {
				return fmt.Errorf("version 2.0.0 migration validation failed: %v", err)
			}
		}
	}

	// Validate that there are no circular dependencies or invalid jumps
	if err := v.validateMigrationSequence(history); err != nil {
		return fmt.Errorf("migration sequence validation failed: %v", err)
	}

	return nil
}

// isValidMigrationPath checks if a migration from one version to another is valid
func (v *Validator) isValidMigrationPath(from, to string, validMigrations map[string]map[string]bool) bool {
	if fromTargets, exists := validMigrations[from]; exists {
		return fromTargets[to]
	}
	return false
}

// validateVersionTwoMigration validates specific requirements for migrating to version 2.0.0
func (v *Validator) validateVersionTwoMigration(config *types.AppConfig) error {
	// Version 2.0.0 requires explicit TLS configuration
	if config.Config.Security.TLS.MinVersion == "" {
		return fmt.Errorf("version 2.0.0 migration requires TLS minimum version to be configured")
	}

	// For production environment, require TLS 1.2 minimum
	if config.Metadata.Environment == "production" && config.Config.Security.TLS.MinVersion < "1.2" {
		return fmt.Errorf("production environment migration to 2.0.0 requires TLS 1.2 minimum")
	}

	return nil
}

// validateMigrationSequence validates the overall migration sequence for consistency
func (v *Validator) validateMigrationSequence(history []types.MigrationRecord) error {
	// Check for duplicate migrations (same from->to pair)
	migrationPaths := make(map[string]bool)
	for _, record := range history {
		path := record.FromVersion + "->" + record.ToVersion
		if migrationPaths[path] {
			return fmt.Errorf("duplicate migration path detected: %s", path)
		}
		migrationPaths[path] = true
	}

	// Validate that migration status makes sense chronologically
	statuses := []string{}
	for _, record := range history {
		statuses = append(statuses, record.Status)
	}

	// Check for "failed" followed by successful migrations (would indicate inconsistency)
	for i, status := range statuses {
		if status == "failed" {
			// After a failed migration, subsequent records should be retries or rollbacks
			for j := i + 1; j < len(statuses); j++ {
				if statuses[j] == "completed" {
					return fmt.Errorf("migration %d failed but later migration %d completed - inconsistent migration status", i+1, j+1)
				}
			}
		}
	}

	return nil
}

func (v *Validator) validateMode(mode string) error {
	switch mode {
	case types.ModeServer, types.ModeClient:
		return nil
	default:
		return fmt.Errorf("invalid mode: %s", mode)
	}
}

func (v *Validator) validateLogging(config types.LoggingConfig) error {
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

	return nil
}

func (v *Validator) validateNetwork(config types.NetworkConfig) error {
	if config.Interface == "" {
		return fmt.Errorf("interface cannot be empty")
	}

	if config.MTU < 576 || config.MTU > 65535 {
		return fmt.Errorf("invalid MTU: %d", config.MTU)
	}

	if config.Address != "" {
		if ip := net.ParseIP(config.Address); ip == nil {
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

func (v *Validator) validateEnvironmentConfig(config *types.AppConfig) error {
	// Validate environment-specific settings
	validEnvironments := map[string]bool{
		"development": true,
		"staging":     true,
		"production":  true,
		"test":        true,
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

func (v *Validator) validateTunnel(config types.TunnelConfig) error {
	if config.Port < 1 || config.Port > 65535 {
		return fmt.Errorf("invalid port: %d", config.Port)
	}

	validProtocols := map[string]bool{
		"tcp":  true,
		"udp":  true,
		"quic": true,
	}

	if !validProtocols[strings.ToLower(config.Protocol)] {
		return fmt.Errorf("invalid protocol: %s", config.Protocol)
	}

	return nil
}

func (v *Validator) validateSecurity(config types.SecurityConfig) error {
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

func (v *Validator) validateMonitor(config types.MonitorConfig) error {
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

func (v *Validator) validateMetrics(config types.MetricsConfig) error {
	if !config.Enabled {
		return nil
	}

	if config.Address == "" {
		return fmt.Errorf("metrics address cannot be empty when metrics are enabled")
	}

	if config.Interval.Seconds() < 1 {
		return fmt.Errorf("invalid metrics interval: %v", config.Interval)
	}

	if config.BufferSize < 1 {
		return fmt.Errorf("invalid metrics buffer size: %d", config.BufferSize)
	}

	return nil
}

func (v *Validator) validateThrottle(config types.ThrottleConfig) error {
	if !config.Enabled {
		return nil
	}

	if config.Rate <= 0 {
		return fmt.Errorf("invalid rate: %f", config.Rate)
	}

	if config.Burst <= 0 {
		return fmt.Errorf("invalid burst: %d", config.Burst)
	}

	return nil
}

func (v *Validator) validateVersion(config *types.AppConfig) error {
	if config.Metadata.SchemaVersion == "" {
		return fmt.Errorf("schema version cannot be empty")
	}

	// Validate schema version format (semantic versioning)
	if err := v.validateSemanticVersion(config.Metadata.SchemaVersion); err != nil {
		return fmt.Errorf("invalid schema version format: %v", err)
	}

	// Define supported schema versions with compatibility rules
	supportedVersions := map[string]struct {
		major           int
		minor           int
		patch           int
		breakingChanges bool
		deprecated      bool
	}{
		"1.0.0": {1, 0, 0, false, false},
		"1.1.0": {1, 1, 0, false, false},
		"2.0.0": {2, 0, 0, true, false},
	}

	versionInfo, supported := supportedVersions[config.Metadata.SchemaVersion]
	if !supported {
		return fmt.Errorf("unsupported schema version: %s (supported: 1.0.0, 1.1.0, 2.0.0)", config.Metadata.SchemaVersion)
	}

	if versionInfo.deprecated {
		return fmt.Errorf("schema version %s is deprecated and no longer supported", config.Metadata.SchemaVersion)
	}

	// Validate version compatibility with current components
	if err := v.validateVersionCompatibility(config, versionInfo); err != nil {
		return fmt.Errorf("version compatibility check failed: %v", err)
	}

	return nil
}

// validateSemanticVersion validates semantic version format (MAJOR.MINOR.PATCH)
func (v *Validator) validateSemanticVersion(version string) error {
	parts := strings.Split(version, ".")
	if len(parts) != 3 {
		return fmt.Errorf("version must be in MAJOR.MINOR.PATCH format")
	}

	for i, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil || num < 0 {
			return fmt.Errorf("version part %d must be a non-negative integer: %s", i+1, part)
		}
		// Major version should not be 0 for production software, but allow for early versions
		if i == 0 && num > 99 {
			return fmt.Errorf("major version too high: %d", num)
		}
	}

	return nil
}

// validateVersionCompatibility checks version-specific compatibility rules
func (v *Validator) validateVersionCompatibility(config *types.AppConfig, versionInfo struct {
	major           int
	minor           int
	patch           int
	breakingChanges bool
	deprecated      bool
}) error {
	// Version 2.0.0+ requires TLS configuration
	if versionInfo.major >= 2 {
		if config.Config.Security.TLS.MinVersion == "" {
			return fmt.Errorf("version 2.0.0+ requires explicit TLS minimum version configuration")
		}
		if versionInfo.breakingChanges {
			// Additional breaking change validations for major version bumps
			if config.Metadata.Environment == "production" && config.Config.Security.TLS.MinVersion < "1.2" {
				return fmt.Errorf("production environment with version 2.0.0+ requires TLS 1.2 minimum")
			}
		}
	}

	// Version 1.1.0+ requires monitoring configuration if metrics enabled
	if versionInfo.major >= 1 && versionInfo.minor >= 1 {
		if config.Config.Metrics.Enabled && config.Config.Monitor.Type == "" {
			return fmt.Errorf("version 1.1.0+ requires monitor type configuration when metrics are enabled")
		}
	}

	// Version 1.0.0 has basic requirements
	if versionInfo.major == 1 && versionInfo.minor == 0 && versionInfo.patch == 0 {
		if config.Metadata.Environment == "" {
			return fmt.Errorf("version 1.0.0 requires environment to be specified")
		}
	}

	return nil
}

func (v *Validator) validateMigrationHistory(config *types.AppConfig) error {
	for _, record := range config.Metadata.MigrationHistory {
		if record.FromVersion == "" {
			return fmt.Errorf("migration record missing from_version")
		}

		if record.ToVersion == "" {
			return fmt.Errorf("migration record missing to_version")
		}

		if record.Status == "" {
			return fmt.Errorf("migration record missing status")
		}

		validStatuses := map[string]bool{
			"completed":  true,
			"failed":     true,
			"pending":    true,
			"rolledback": true,
		}

		if !validStatuses[record.Status] {
			return fmt.Errorf("invalid migration status: %s", record.Status)
		}

		if record.Timestamp.IsZero() {
			return fmt.Errorf("migration record missing timestamp")
		}
	}

	return nil
}

func (v *Validator) validateSecurityConfig(config *types.AppConfig) error {
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

func (v *Validator) validateThrottleConfig(config *types.AppConfig) error {
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

func (v *Validator) validateCertificateFilesExist(config *types.AppConfig) error {
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
