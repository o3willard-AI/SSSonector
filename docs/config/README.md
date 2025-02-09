# Configuration Management

The configuration management system provides a robust and flexible way to manage SSSonector's configuration. It supports versioning, validation, hot reloading, and multiple storage backends.

## Features

- **Type-Safe Configuration**: Strongly typed configuration structures for all components
- **Version Control**: Track and manage configuration versions
- **Validation**: Schema-based and component-specific validation rules
- **Hot Reload**: Dynamic configuration updates without restart
- **Multiple Formats**: Support for JSON, YAML, and TOML
- **Storage Backends**: File-based storage with extensible interface
- **Audit Logging**: Track configuration changes and validations

## Components

### Configuration Types

The system defines several configuration types:

```go
// AppConfig represents the complete application configuration
type AppConfig struct {
    Metadata ConfigMetadata  // Configuration metadata
    Mode     Mode           // Operating mode
    Network  NetworkConfig  // Network configuration
    Tunnel   TunnelConfig  // Tunnel configuration
    Monitor  MonitorConfig // Monitor configuration
    Throttle ThrottleConfig // Rate limiting configuration
    Security SecurityConfig // Security configuration
}
```

### Configuration Store

The configuration store interface provides methods for storing and retrieving configurations:

```go
type ConfigStore interface {
    Store(cfg *AppConfig) error
    Load(cfgType ConfigType, version string) (*AppConfig, error)
    Delete(cfgType ConfigType, version string) error
    List() ([]*AppConfig, error)
    ListByType(cfgType ConfigType) ([]*AppConfig, error)
    ListVersions(cfgType ConfigType) ([]string, error)
    GetLatest(cfgType ConfigType) (*AppConfig, error)
}
```

### Configuration Validator

The validator ensures configuration correctness:

```go
type ConfigValidator interface {
    Validate(cfg *AppConfig) error
    ValidateSchema(cfg *AppConfig, schema []byte) error
}
```

### Configuration Manager

The manager provides high-level configuration management:

```go
type ConfigManager interface {
    GetStore() ConfigStore
    GetValidator() ConfigValidator
    GetWatcher() ConfigWatcher
    Apply(cfg *AppConfig) error
    Rollback(cfgType ConfigType, version string) error
    Diff(cfgType ConfigType, version1, version2 string) (string, error)
    Export(cfg *AppConfig, format ConfigFormat) ([]byte, error)
    Import(data []byte, format ConfigFormat) (*AppConfig, error)
}
```

## Usage Examples

### Loading Configuration

```go
// Create configuration loader
loader := config.NewLoader(logger)

// Load configuration from file
cfg, err := loader.LoadFromFile("config.json")
if err != nil {
    log.Fatal(err)
}
```

### Storing Configuration

```go
// Create file store
store := config.NewFileStore("/etc/sssonector/configs", logger)

// Store configuration
err := store.Store(cfg)
if err != nil {
    log.Fatal(err)
}
```

### Validating Configuration

```go
// Create validator
validator := config.NewValidator(logger)

// Validate configuration
err := validator.Validate(cfg)
if err != nil {
    log.Fatal(err)
}
```

### Managing Configuration

```go
// Create manager
manager := config.NewManager(configPath, store, validator, logger)

// Apply configuration
err := manager.Apply(cfg)
if err != nil {
    log.Fatal(err)
}

// Export configuration
data, err := manager.Export(cfg, config.FormatJSON)
if err != nil {
    log.Fatal(err)
}
```

## Best Practices

1. **Version Control**: Always version your configurations and maintain a history of changes.
2. **Validation**: Use schema validation to catch configuration errors early.
3. **Rollback Support**: Keep previous versions available for rollback.
4. **Audit Trail**: Log configuration changes with metadata.
5. **Security**: Secure sensitive configuration data.

## Configuration File Example

```json
{
    "metadata": {
        "version": "1.0.0",
        "environment": "production",
        "region": "us-west-1"
    },
    "mode": "server",
    "network": {
        "interface": "eth0",
        "mtu": 1500,
        "ip_address": "192.168.1.1",
        "subnet_mask": "255.255.255.0"
    },
    "tunnel": {
        "protocol": "tcp",
        "encryption": "aes-256-gcm",
        "compression": "none",
        "max_connections": 100,
        "buffer_size": 65536
    },
    "monitor": {
        "enabled": true,
        "interval": "60s",
        "log_level": "info",
        "prometheus": {
            "enabled": true,
            "port": 9090,
            "path": "/metrics"
        }
    },
    "security": {
        "tls": {
            "min_version": "1.2",
            "max_version": "1.3",
            "ciphers": [
                "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
                "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384"
            ]
        },
        "auth_method": "certificate",
        "cert_rotation": {
            "enabled": true,
            "interval": "720h"
        }
    }
}
```

## Error Handling

The system provides detailed error messages for common issues:

- Invalid configuration format
- Schema validation failures
- Missing required fields
- Invalid field values
- Storage errors
- Version conflicts

## Integration

The configuration management system integrates with other components:

- **Certificate Management**: Handles certificate paths and rotation
- **Monitoring**: Configures metrics collection
- **Security**: Manages TLS and authentication settings
- **Network**: Controls interface and tunnel settings

## Contributing

When adding new configuration options:

1. Update the appropriate configuration type
2. Add validation rules
3. Update documentation
4. Add tests
5. Update example configurations
