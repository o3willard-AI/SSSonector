// Package config provides configuration loading for the current schema.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// CurrentSchemaVersion is the only configuration schema this build loads
const CurrentSchemaVersion = "2.0.0"

// ConfigLoader handles loading configuration files
type ConfigLoader struct{}

// NewConfigLoader creates a new ConfigLoader instance
func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{}
}

// LoadData parses configuration data. Only the current schema
// ("2.0.0") is supported; older layouts are rejected with an explicit
// message instead of being silently upgraded.
func (l *ConfigLoader) LoadData(data []byte, format string) (*AppConfig, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("config data is empty")
	}

	if format == "" {
		format = l.detectFormat(data)
	}

	var raw map[string]interface{}
	if err := l.parseData(data, format, &raw); err != nil {
		return nil, fmt.Errorf("failed to parse config data: %v", err)
	}

	schemaVersion := ""
	if metadata, ok := raw["metadata"].(map[string]interface{}); ok {
		if sv, ok := metadata["schema_version"].(string); ok {
			schemaVersion = sv
		}
	}
	if schemaVersion != CurrentSchemaVersion {
		return nil, fmt.Errorf(
			"unsupported schema version %q: this build loads schema_version %q configs only",
			schemaVersion, CurrentSchemaVersion)
	}

	var config AppConfig
	if err := l.parseData(data, format, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config data: %v", err)
	}

	return &config, nil
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

// LoadFile loads configuration from a file with automatic format detection and version upgrading
func (l *ConfigLoader) LoadFile(filename string) (*AppConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %v", filename, err)
	}

	format := l.detectFormat(data)
	return l.LoadData(data, format)
}

// LoadFromString loads configuration from a string with specified format
func (l *ConfigLoader) LoadFromString(content, format string) (*AppConfig, error) {
	return l.LoadData([]byte(content), format)
}
