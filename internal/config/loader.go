// Package config provides configuration loading with version detection and upgrade paths
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
)

// ConfigLoader handles loading and upgrading configuration files
type ConfigLoader struct{}

// NewConfigLoader creates a new ConfigLoader instance
func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{}
}

// LoadData loads configuration data from raw bytes with automatic version detection and upgrade
func (l *ConfigLoader) LoadData(data []byte, format string) (*types.AppConfig, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("config data is empty")
	}

	// Detect format if not specified
	if format == "" {
		format = l.detectFormat(data)
	}

	// Parse the data into a raw map for version detection
	var raw map[string]interface{}
	if err := l.parseData(data, format, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse config data: %v", err)
	}

	// Detect version
	version, err := l.detectVersion(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to detect config version: %v", err)
	}

	// If version is current, parse directly
	if version == "2.0.0" {
		var config types.AppConfig
		if err := l.parseData(data, format, &config); err != nil {
			return nil, fmt.Errorf("failed to parse current version config: %v", err)
		}
		return &config, nil
	}

	// Upgrade the configuration
	upgradedConfig, err := l.upgradeConfig(raw, version)
	if err != nil {
		return nil, fmt.Errorf("failed to upgrade config from version %s: %v", version, err)
	}

	return upgradedConfig, nil
}

// detectFormat tries to detect the config file format (JSON/YAML)
func (l *ConfigLoader) detectFormat(data []byte) string {
	trimmed := strings.TrimSpace(string(data))

	// Check for JSON indicators
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		return "json"
	}

	// Default to YAML (more common for configs)
	return "yaml"
}

// parseData parses configuration data based on format
func (l *ConfigLoader) parseData(data []byte, format string, target interface{}) error {
	switch strings.ToLower(format) {
	case "json":
		return json.Unmarshal(data, target)
	case "yaml", "yml":
		return yaml.Unmarshal(data, target)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// detectVersion detects the schema version from raw configuration data
func (l *ConfigLoader) detectVersion(raw map[string]interface{}) (string, error) {
	// Check for schema_version in metadata
	if metadata, ok := raw["metadata"].(map[string]interface{}); ok {
		if schemaVersion, ok := metadata["schema_version"].(string); ok && schemaVersion != "" {
			return schemaVersion, nil
		}
	}

	// Check for version field directly
	if version, ok := raw["version"].(string); ok && version != "" {
		return version, nil
	}

	// Check for version field in metadata with different naming
	if metadata, ok := raw["metadata"].(map[string]interface{}); ok {
		if version, ok := metadata["version"].(string); ok && version != "" {
			return version, nil
		}
	}

	// Check for configuration patterns that indicate version
	if _, hasThrottle := raw["throttle"]; hasThrottle {
		// Version 1.x+ has throttle section
		if config, ok := raw["config"].(map[string]interface{}); ok {
			if monitor, ok := config["monitor"].(map[string]interface{}); ok {
				if _, hasType := monitor["type"]; hasType {
					// Version 1.1.0+ has monitor.type
					return "1.1.0", nil
				}
			}
			// Version 1.0.0 has main config sections
			return "1.0.0", nil
		}
	}

	// If no version indicators found, assume legacy version 0.0.0
	return "0.0.0", nil
}

// upgradeConfig upgrades configuration from an older version to current version 2.0.0
func (l *ConfigLoader) upgradeConfig(raw map[string]interface{}, fromVersion string) (*types.AppConfig, error) {
	var config types.AppConfig

	switch fromVersion {
	case "0.0.0":
		config = l.upgradeFromLegacy(raw)
	case "1.0.0":
		config = l.upgradeFrom10(raw)
	case "1.1.0":
		config = l.upgradeFrom11(raw)
	default:
		return nil, fmt.Errorf("unsupported upgrade from version %s", fromVersion)
	}

	// Record the migration
	l.recordMigration(&config, fromVersion, "2.0.0", "Automatic upgrade via loader")

	return &config, nil
}

// upgradeFromLegacy upgrades from unversioned/legacy configuration
func (l *ConfigLoader) upgradeFromLegacy(raw map[string]interface{}) types.AppConfig {
	config := types.NewAppConfig(types.TypeServer)

	// Copy basic fields if they exist
	if mode, ok := raw["mode"].(string); ok {
		config.Config.Mode = mode
	}

	// Default values for legacy upgrades
	config.Metadata.SchemaVersion = "2.0.0"
	config.Metadata.Environment = "production" // Legacy configs assumed production

	return *config
}

// upgradeFrom10 upgrades from version 1.0.0 to 2.0.0
func (l *ConfigLoader) upgradeFrom10(raw map[string]interface{}) types.AppConfig {
	config := types.NewAppConfig(types.TypeServer)

	// Copy config section
	if rawConfig, ok := raw["config"].(map[string]interface{}); ok {
		// Copy mode
		if mode, ok := rawConfig["mode"].(string); ok {
			config.Config.Mode = mode
		}

		// Copy network settings
		if network, ok := rawConfig["network"].(map[string]interface{}); ok {
			if iface, ok := network["interface"].(string); ok {
				config.Config.Network.Interface = iface
			}
			if mtu, ok := network["mtu"].(int); ok {
				config.Config.Network.MTU = mtu
			}
			if addr, ok := network["address"].(string); ok {
				config.Config.Network.Address = addr
			}
			if dnsServers, ok := network["dns_servers"].([]interface{}); ok {
				for _, dns := range dnsServers {
					if dnsStr, ok := dns.(string); ok {
						config.Config.Network.DNSServers = append(config.Config.Network.DNSServers, dnsStr)
					}
				}
			}
		}

		// Copy logging settings
		if logging, ok := rawConfig["logging"].(map[string]interface{}); ok {
			if level, ok := logging["level"].(string); ok {
				config.Config.Logging.Level = level
			}
			if file, ok := logging["file"].(string); ok {
				config.Config.Logging.File = file
			}
		}

		// Copy tunnel settings
		if tunnel, ok := rawConfig["tunnel"].(map[string]interface{}); ok {
			if portRaw, ok := tunnel["port"].(int); ok {
				config.Config.Tunnel.Port = portRaw
			}
			if protocol, ok := tunnel["protocol"].(string); ok {
				config.Config.Tunnel.Protocol = protocol
			}
		}
	}

	// Set TLS defaults for version 2.0.0 compatibility
	config.Config.Security.TLS.MinVersion = "1.2"
	config.Config.Security.TLS.MaxVersion = "1.3"
	config.Metadata.SchemaVersion = "2.0.0"

	return *config
}

// upgradeFrom11 upgrades from version 1.1.0 to 2.0.0
func (l *ConfigLoader) upgradeFrom11(raw map[string]interface{}) types.AppConfig {
	// Start from 1.0.0 upgrade as base
	config := l.upgradeFrom10(raw)

	// Add version 1.1.0 specific changes that need handling for 2.0.0
	if rawConfig, ok := raw["config"].(map[string]interface{}); ok {
		if monitor, ok := rawConfig["monitor"].(map[string]interface{}); ok {
			if mtype, ok := monitor["type"].(string); ok {
				config.Config.Monitor.Type = mtype
			}
			if enabledRaw, ok := monitor["enabled"].(bool); ok {
				config.Config.Monitor.Enabled = enabledRaw
			}
			if intervalRaw, ok := monitor["interval"].(float64); ok {
				config.Config.Monitor.Interval = time.Duration(intervalRaw * 1000000000) // Convert seconds to nanoseconds
			}
		}

		// Copy metrics settings if present
		if metrics, ok := rawConfig["metrics"].(map[string]interface{}); ok {
			if enabledRaw, ok := metrics["enabled"].(bool); ok {
				config.Config.Metrics.Enabled = enabledRaw
			}
			if addr, ok := metrics["address"].(string); ok {
				config.Config.Metrics.Address = addr
			}
		}
	}

	// Ensure TLS settings for 2.0.0
	config.Config.Security.TLS.MinVersion = "1.2"
	config.Config.Security.TLS.MaxVersion = "1.3"
	config.Metadata.SchemaVersion = "2.0.0"

	return config
}

// recordMigration adds a migration record to the configuration
func (l *ConfigLoader) recordMigration(config *types.AppConfig, fromVersion, toVersion, notes string) {
	record := types.MigrationRecord{
		FromVersion: fromVersion,
		ToVersion:   toVersion,
		Timestamp:   time.Now(),
		Status:      "completed",
		Notes:       notes,
	}

	config.Metadata.MigrationHistory = append(config.Metadata.MigrationHistory, record)
}

// LoadFile loads configuration from a file with automatic format detection and version upgrading
func (l *ConfigLoader) LoadFile(filename string) (*types.AppConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %v", filename, err)
	}

	format := l.detectFormat(data)
	return l.LoadData(data, format)
}

// LoadFromString loads configuration from a string with specified format
func (l *ConfigLoader) LoadFromString(content, format string) (*types.AppConfig, error) {
	return l.LoadData([]byte(content), format)
}
