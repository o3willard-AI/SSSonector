# Configuration Package API Documentation

## Overview

The `config` package provides a comprehensive configuration management system for SSSonector. This document details the API for each component.

## Types

### AppConfig

```go
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

The `AppConfig` is the root configuration type that contains all application settings.

### ConfigMetadata

```go
type ConfigMetadata struct {
    Version       string            // Configuration version
    CreatedAt     time.Time        // Creation timestamp
    UpdatedAt     time.Time        // Last update timestamp
    LastValidator string           // Last validator identifier
    ValidatedAt   *time.Time       // Last validation timestamp
    Environment   string           // Deployment environment
    Region        string           // Geographic region
    Tags          map[string]string // Custom metadata tags
}
```

Metadata provides tracking and auditing information for configurations.

## Interfaces

### ConfigStore

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

Methods:
- `Store`: Stores a configuration
- `Load`: Loads a configuration by type and version
- `Delete`: Deletes a configuration
- `List`: Lists all configurations
- `ListByType`: Lists configurations by type
- `ListVersions`: Lists versions for a type
- `GetLatest`: Gets the latest version

### ConfigValidator

```go
type ConfigValidator interface {
    Validate(cfg *AppConfig) error
    ValidateSchema(cfg *AppConfig, schema []byte) error
}
```

Methods:
- `Validate`: Validates configuration values
- `ValidateSchema`: Validates against JSON Schema

### ConfigManager

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

Methods:
- `GetStore`: Returns the configuration store
- `GetValidator`: Returns the validator
- `GetWatcher`: Returns the watcher
- `Apply`: Applies a configuration
- `Rollback`: Rolls back to a version
- `Diff`: Shows differences between versions
- `Export`: Exports to a format
- `Import`: Imports from a format

## Implementations

### FileStore

```go
func NewFileStore(baseDir string, logger *zap.Logger) *FileStore
```

File-based implementation of `ConfigStore`:
- Uses JSON files for storage
- Organizes by type and version
- Maintains version history
- Supports atomic operations

### Validator

```go
func NewValidator(logger *zap.Logger) *Validator
```

Default implementation of `ConfigValidator`:
- Validates field values
- Checks relationships
- Enforces constraints
- Supports JSON Schema

### Manager

```go
func NewManager(configPath string, store ConfigStore, validator ConfigValidator, logger *zap.Logger) *Manager
```

Default implementation of `ConfigManager`:
- Coordinates operations
- Manages transactions
- Handles notifications
- Provides utilities

## Error Types

```go
var (
    ErrInvalidConfig    = errors.New("invalid configuration")
    ErrVersionNotFound  = errors.New("version not found")
    ErrTypeNotFound     = errors.New("type not found")
    ErrInvalidFormat    = errors.New("invalid format")
    ErrValidationFailed = errors.New("validation failed")
)
```

## Constants

```go
const (
    FormatJSON ConfigFormat = "json"
    FormatYAML ConfigFormat = "yaml"
    FormatTOML ConfigFormat = "toml"
)

const (
    TypeServer     ConfigType = "server"
    TypeClient     ConfigType = "client"
    TypeTunnel     ConfigType = "tunnel"
    TypeSecurity   ConfigType = "security"
    TypeMonitoring ConfigType = "monitoring"
)
```

## Usage Examples

### Creating a Manager

```go
// Create components
store := config.NewFileStore("/etc/sssonector/configs", logger)
validator := config.NewValidator(logger)

// Create manager
manager := config.NewManager(configPath, store, validator, logger)
```

### Loading and Validating

```go
// Load configuration
cfg, err := manager.GetStore().Load(config.TypeServer, "1.0.0")
if err != nil {
    return err
}

// Validate configuration
if err := manager.GetValidator().Validate(cfg); err != nil {
    return err
}
```

### Applying Changes

```go
// Update configuration
cfg.Network.MTU = 1500

// Apply changes
if err := manager.Apply(cfg); err != nil {
    return err
}
```

### Rolling Back

```go
// Roll back to previous version
if err := manager.Rollback(config.TypeServer, "1.0.0"); err != nil {
    return err
}
```

### Exporting and Importing

```go
// Export configuration
data, err := manager.Export(cfg, config.FormatJSON)
if err != nil {
    return err
}

// Import configuration
cfg, err = manager.Import(data, config.FormatJSON)
if err != nil {
    return err
}
```

## Best Practices

1. **Error Handling**
   - Check all errors
   - Use type assertions
   - Provide context
   - Log failures

2. **Validation**
   - Validate early
   - Use schemas
   - Check relationships
   - Verify constraints

3. **Versioning**
   - Use semantic versions
   - Keep history
   - Document changes
   - Support rollback

4. **Security**
   - Secure storage
   - Validate input
   - Audit changes
   - Control access

5. **Performance**
   - Cache results
   - Batch operations
   - Use transactions
   - Handle concurrency
