package preflight

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

// ConfigValidationCheck performs configuration validation
func ConfigValidationCheck(ctx context.Context, logger *zap.Logger) CheckResult {
	result := CheckResult{
		Name:        "Configuration Validation",
		Description: "Validating configuration file structure and values",
	}

	// Get config file path from environment or use default
	configPath := "/etc/sssonector/config.yaml"
	if envPath := os.Getenv("SSSONECTOR_CONFIG"); envPath != "" {
		configPath = envPath
	}

	// Read and parse config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		result.Status = "Failed"
		result.Error = fmt.Errorf("failed to read config file: %w", err)
		return result
	}

	var config map[string]interface{}
	if strings.HasSuffix(configPath, ".yaml") || strings.HasSuffix(configPath, ".yml") {
		err = yaml.Unmarshal(data, &config)
		if err != nil {
			result.Status = "Failed"
			result.Error = fmt.Errorf("invalid YAML syntax: %w", err)
			return result
		}
	} else if strings.HasSuffix(configPath, ".json") {
		err = json.Unmarshal(data, &config)
		if err != nil {
			result.Status = "Failed"
			result.Error = fmt.Errorf("invalid JSON syntax: %w", err)
			return result
		}
	} else {
		result.Status = "Failed"
		result.Error = fmt.Errorf("unsupported config file format: %s", configPath)
		return result
	}

	// Validate mandatory fields
	mandatoryFields := map[string]fieldSpec{
		"server": {
			Type: "object",
			Fields: map[string]fieldSpec{
				"host": {Type: "string"},
				"port": {Type: "number", Min: 1, Max: 65535},
			},
		},
		"connection": {
			Type: "object",
			Fields: map[string]fieldSpec{
				"maxConnections": {Type: "number", Min: 1},
				"keepAlive":      {Type: "boolean"},
				"idleTimeout":    {Type: "string"},
			},
		},
		"rateLimit": {
			Type: "object",
			Fields: map[string]fieldSpec{
				"enabled":     {Type: "boolean"},
				"requestRate": {Type: "number", Min: 1},
				"burstSize":   {Type: "number", Min: 1},
			},
		},
		"circuitBreaker": {
			Type: "object",
			Fields: map[string]fieldSpec{
				"enabled":          {Type: "boolean"},
				"maxFailures":      {Type: "number", Min: 1},
				"resetTimeout":     {Type: "string"},
				"halfOpenMaxCalls": {Type: "number", Min: 1},
			},
		},
	}

	var validationErrors []string
	validateFields(config, "", mandatoryFields, &validationErrors)

	if len(validationErrors) > 0 {
		result.Status = "Failed"
		result.Error = fmt.Errorf("configuration validation failed:\n%s", strings.Join(validationErrors, "\n"))
		return result
	}

	// Validate duration formats
	durationFields := []string{
		"connection.idleTimeout",
		"circuitBreaker.resetTimeout",
	}
	for _, field := range durationFields {
		value := getNestedValue(config, strings.Split(field, "."))
		if value != nil {
			if strValue, ok := value.(string); ok {
				if !isValidDuration(strValue) {
					validationErrors = append(validationErrors,
						fmt.Sprintf("invalid duration format for %s: %s (expected format: 1s, 1m, 1h, etc.)", field, strValue))
				}
			}
		}
	}

	if len(validationErrors) > 0 {
		result.Status = "Failed"
		result.Error = fmt.Errorf("configuration validation failed:\n%s", strings.Join(validationErrors, "\n"))
		return result
	}

	result.Status = "Passed"
	return result
}

type fieldSpec struct {
	Type   string
	Fields map[string]fieldSpec
	Min    float64
	Max    float64
}

func validateFields(data interface{}, path string, specs map[string]fieldSpec, errors *[]string) {
	m, ok := data.(map[string]interface{})
	if !ok {
		*errors = append(*errors, fmt.Sprintf("invalid configuration structure at %s", path))
		return
	}

	for field, spec := range specs {
		fullPath := field
		if path != "" {
			fullPath = path + "." + field
		}

		value, exists := m[field]
		if !exists {
			*errors = append(*errors, fmt.Sprintf("missing mandatory field: %s", fullPath))
			continue
		}

		switch spec.Type {
		case "object":
			if obj, ok := value.(map[string]interface{}); ok {
				validateFields(obj, fullPath, spec.Fields, errors)
			} else {
				*errors = append(*errors, fmt.Sprintf("invalid type for %s: expected object", fullPath))
			}

		case "string":
			if _, ok := value.(string); !ok {
				*errors = append(*errors, fmt.Sprintf("invalid type for %s: expected string", fullPath))
			}

		case "number":
			num, ok := getNumber(value)
			if !ok {
				*errors = append(*errors, fmt.Sprintf("invalid type for %s: expected number", fullPath))
				continue
			}
			if spec.Min != 0 && num < spec.Min {
				*errors = append(*errors, fmt.Sprintf("%s must be >= %v", fullPath, spec.Min))
			}
			if spec.Max != 0 && num > spec.Max {
				*errors = append(*errors, fmt.Sprintf("%s must be <= %v", fullPath, spec.Max))
			}

		case "boolean":
			if _, ok := value.(bool); !ok {
				*errors = append(*errors, fmt.Sprintf("invalid type for %s: expected boolean", fullPath))
			}
		}
	}
}

func getNumber(value interface{}) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	default:
		return 0, false
	}
}

func getNestedValue(data map[string]interface{}, path []string) interface{} {
	current := data
	for i, key := range path {
		if i == len(path)-1 {
			return current[key]
		}
		if next, ok := current[key].(map[string]interface{}); ok {
			current = next
		} else {
			return nil
		}
	}
	return nil
}

func isValidDuration(duration string) bool {
	// Simple duration format validation
	// Accepts: number + unit (s, m, h)
	// Examples: 30s, 5m, 2h
	parts := strings.Split(duration, "")
	if len(parts) < 2 {
		return false
	}

	// Check if all characters except the last one are digits
	for _, c := range parts[:len(parts)-1] {
		if c < "0" || c > "9" {
			return false
		}
	}

	// Check if the last character is a valid unit
	unit := parts[len(parts)-1]
	return unit == "s" || unit == "m" || unit == "h"
}
