# Configuration Package API

Reference for the `internal/config` package. This documents the actual
interfaces and types as they exist in the code today.

> This is an **internal** Go package. It is not a stable public API and may
> change without notice. For user-facing configuration, see
> [configuration_guide.md](../configuration_guide.md).

## Interfaces (`ifaces.go`)

```go
type ConfigStore interface {
    Load() (*AppConfig, error)
    Store(*AppConfig) error
    ListVersions(configType Type) ([]string, error)
}
```

- `Load`: loads the latest configuration from storage.
- `Store`: persists the configuration to storage.
- `ListVersions`: lists available version identifiers for a given config type.

```go
type ConfigValidator interface {
    Validate(*AppConfig) error
}
```

- `Validate`: validates a configuration (value checks, relationships,
  environment-specific constraints).

```go
type ConfigManager interface {
    Get() (*AppConfig, error)
    Set(*AppConfig) error
    Update(*AppConfig) error
    Watch() (<-chan *AppConfig, error)
    Close() error
}
```

- `Get`: returns the current configuration (lazily loads if not yet held; a
  missing file falls back to defaults, other load errors fail loudly).
- `Set`: validates, stores, and notifies watchers.
- `Update`: validates, updates in-memory config, stores, and notifies.
- `Watch`: returns a channel that receives configuration updates.
- `Close`: closes all watcher channels.

## Implementations

### FileStore

```go
func NewFileStore(configDir string) *FileStore
```

File-based `ConfigStore`. Reads/writes `config.yaml` inside `configDir`
(permissions `0750` dir / `0600` file) and lists versions by scanning `.yaml`
files whose `Type` matches.

### Validator

```go
func NewValidator() *Validator
```

`ConfigValidator` implementation. Also exposes exported helpers
`ValidateIPAddress` / `ValidateCIDR`.

### Manager

```go
func NewManager(store ConfigStore, validator ConfigValidator) *Manager
```

`ConfigManager` implementation coordinating store + validator + watchers.

### Factory helpers (`config.go`)

```go
const DefaultConfigDir = "/etc/sssonector"

func CreateManager(configDir string) ConfigManager
func CreateManagerWithOptions(store ConfigStore, validator ConfigValidator) ConfigManager
func CreateDefaultConfig() *AppConfig
func CreateAppConfig(configType Type) *AppConfig
func CreateConfigLoader() *ConfigLoader
func LoadConfigFile(filename string) (*AppConfig, error)
func LoadConfigString(content, format string) (*AppConfig, error)
```

## Types

### AppConfig

```go
type AppConfig struct {
    Type     Type           // server | client
    Config   *Config        // nested section configuration
    Version  string
    Metadata ConfigMetadata
    Throttle ThrottleConfig
}
```

### ConfigMetadata

```go
type ConfigMetadata struct {
    Version       string
    Created, Modified time.Time
    CreatedBy     string
    CreatedAt, UpdatedAt time.Time
    Environment   string
    Region        string
    SchemaVersion string
}
```

### Type constants

```go
type Type string
const (
    TypeServer Type = "server"
    TypeClient Type = "client"
)
```

There is no `ConfigFormat` type and no `Delete/List/ListByType/GetLatest`
methods on `ConfigStore`; those do not exist in this codebase.

## Usage

```go
mgr := config.CreateManager("/etc/sssonector")
cfg, err := mgr.Get()
if err != nil {
    return err
}
if err := mgr.Set(cfg); err != nil {
    return err
}
updates, err := mgr.Watch()
```