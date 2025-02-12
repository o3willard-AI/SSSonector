package main

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config/types"
)

func cmdConfig(ctx context.Context, args []string, config *Config, configFile string) error {
	if len(args) == 0 {
		// Show current configuration
		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal config: %w", err)
		}
		fmt.Println(string(data))
		return nil
	}

	switch args[0] {
	case "save":
		if configFile == "" {
			return fmt.Errorf("no config file specified")
		}
		if err := saveConfig(configFile, config); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		fmt.Printf("Configuration saved to %s\n", configFile)

	case "set":
		if len(args) != 3 {
			return fmt.Errorf("usage: config set <key> <value>")
		}
		if err := setConfigValue(config, args[1], args[2]); err != nil {
			return fmt.Errorf("failed to set config value: %w", err)
		}
		fmt.Printf("Set %s = %s\n", args[1], args[2])

	case "get":
		if len(args) != 2 {
			return fmt.Errorf("usage: config get <key>")
		}
		value, err := getConfigValue(config, args[1])
		if err != nil {
			return fmt.Errorf("failed to get config value: %w", err)
		}
		fmt.Printf("%s = %v\n", args[1], value)

	default:
		return fmt.Errorf("unknown config subcommand: %s", args[0])
	}

	return nil
}

func setConfigValue(config *Config, key, value string) error {
	parts := strings.Split(key, ".")
	current := reflect.ValueOf(config).Elem()

	// Navigate through the struct fields
	for i, part := range parts {
		if current.Kind() != reflect.Struct {
			return fmt.Errorf("invalid path: %s is not a struct", strings.Join(parts[:i], "."))
		}

		field := current.FieldByName(part)
		if !field.IsValid() {
			return fmt.Errorf("invalid path: field %s not found", part)
		}

		if i == len(parts)-1 {
			// Set the value
			switch field.Kind() {
			case reflect.String:
				field.SetString(value)
			case reflect.Int, reflect.Int64:
				v, err := strconv.ParseInt(value, 10, 64)
				if err != nil {
					return fmt.Errorf("invalid integer value: %s", value)
				}
				field.SetInt(v)
			case reflect.Bool:
				v, err := strconv.ParseBool(value)
				if err != nil {
					return fmt.Errorf("invalid boolean value: %s", value)
				}
				field.SetBool(v)
			case reflect.Float64:
				v, err := strconv.ParseFloat(value, 64)
				if err != nil {
					return fmt.Errorf("invalid float value: %s", value)
				}
				field.SetFloat(v)
			case reflect.Struct:
				if field.Type() == reflect.TypeOf(types.Duration{}) {
					d, err := time.ParseDuration(value)
					if err != nil {
						return fmt.Errorf("invalid duration value: %s", value)
					}
					field.Set(reflect.ValueOf(types.NewDuration(d)))
				} else {
					return fmt.Errorf("unsupported type: %s", field.Type())
				}
			default:
				return fmt.Errorf("unsupported type: %s", field.Type())
			}
		} else {
			current = field
		}
	}

	return nil
}

func getConfigValue(config *Config, key string) (interface{}, error) {
	parts := strings.Split(key, ".")
	current := reflect.ValueOf(config).Elem()

	// Navigate through the struct fields
	for i, part := range parts {
		if current.Kind() != reflect.Struct {
			return nil, fmt.Errorf("invalid path: %s is not a struct", strings.Join(parts[:i], "."))
		}

		field := current.FieldByName(part)
		if !field.IsValid() {
			return nil, fmt.Errorf("invalid path: field %s not found", part)
		}

		if i == len(parts)-1 {
			// Return the value
			switch field.Kind() {
			case reflect.String:
				return field.String(), nil
			case reflect.Int, reflect.Int64:
				return field.Int(), nil
			case reflect.Bool:
				return field.Bool(), nil
			case reflect.Float64:
				return field.Float(), nil
			case reflect.Struct:
				if field.Type() == reflect.TypeOf(types.Duration{}) {
					return field.Interface().(types.Duration).String(), nil
				}
				return nil, fmt.Errorf("unsupported type: %s", field.Type())
			default:
				return nil, fmt.Errorf("unsupported type: %s", field.Type())
			}
		} else {
			current = field
		}
	}

	return nil, fmt.Errorf("invalid path: %s", key)
}
