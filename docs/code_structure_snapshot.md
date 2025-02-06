# Code Structure Snapshot

## Directory Structure
```
SSSonector/
├── cmd/
│   └── tunnel/
│       ├── main.go     # Entry point and flag handling
│       ├── server.go   # Server implementation
│       └── client.go   # Client implementation
├── internal/
│   ├── adapter/        # TUN interface implementations
│   │   ├── interface.go
│   │   ├── interface_linux.go
│   │   ├── interface_darwin.go
│   │   └── interface_windows.go
│   ├── cert/          # Certificate management
│   │   ├── manager.go
│   │   ├── generator.go
│   │   ├── locator.go
│   │   └── validator.go
│   ├── config/        # Configuration handling
│   │   ├── loader.go
│   │   └── types.go
│   ├── connection/    # Connection management
│   │   └── manager.go
│   ├── monitor/       # Monitoring and metrics
│   │   ├── monitor.go
│   │   ├── metrics.go
│   │   ├── logging.go
│   │   └── snmp.go
│   ├── throttle/      # Rate limiting
│   │   ├── limiter.go
│   │   └── token_bucket.go
│   └── tunnel/        # Core tunnel logic
│       ├── tunnel.go
│       └── tls.go
├── test/             # Test suites
│   ├── test_temp_certs.sh
│   └── test_results.md
└── docs/             # Documentation
    ├── installation.md
    ├── certificate_management.md
    └── project_context.md
```

## Key Components

### 1. Command Layer (cmd/)
Entry points for the application:
```go
// main.go - Flag definitions
var (
    configPath        string
    generateCerts     bool
    certDir           string
    validateCerts     bool
    testMode          bool
    mode              string
    generateCertsOnly bool
)

// server.go - Server operations
type Server struct {
    config    *config.Config
    tunnel    *tunnel.Tunnel
    monitor   *monitor.Monitor
    certMgr   *cert.Manager
    testMode  bool
}

// client.go - Client operations
type Client struct {
    config    *config.Config
    tunnel    *tunnel.Tunnel
    monitor   *monitor.Monitor
    certMgr   *cert.Manager
    testMode  bool
}
```

### 2. Adapter Layer (internal/adapter/)
Platform-specific TUN interface implementations:
```go
// interface.go - Common interface
type Interface interface {
    Read([]byte) (int, error)
    Write([]byte) (int, error)
    Close() error
    GetName() string
    GetMTU() int
    GetAddress() string
    IsUp() bool
    Cleanup() error
}

// interface_linux.go - Linux implementation
type linuxInterface struct {
    name    string
    file    *os.File
    address string
    mtu     int
    isUp    bool
}
```

### 3. Certificate Management (internal/cert/)
Certificate handling and validation:
```go
// manager.go - Certificate operations
type Manager interface {
    GenerateCertificates(string) error
    ValidateCertificates(string) error
    LoadCertificates(string) error
    GetCertificatePaths() (string, string, string)
}

// generator.go - Certificate generation
func GenerateTemporaryCertificates(dir string) error
func GenerateProductionCertificates(dir string) error
```

### 4. Configuration (internal/config/)
Configuration management:
```go
// types.go - Configuration structures
type Config struct {
    Mode     string
    Network  NetworkConfig
    Tunnel   TunnelConfig
    Monitor  MonitorConfig
}

// loader.go - Configuration loading
func LoadConfig(path string) (*Config, error)
func ValidateConfig(cfg *Config) error
```

### 5. Monitoring (internal/monitor/)
System monitoring and metrics:
```go
// monitor.go - Monitoring system
type Monitor struct {
    logger     *zap.Logger
    config     *Config
    metrics    *Metrics
    snmpAgent  *SNMPAgent
    startTime  time.Time
}

// metrics.go - Performance metrics
type Metrics struct {
    BytesIn      int64
    BytesOut     int64
    PacketsIn    int64
    PacketsOut   int64
    Errors       int64
    Connections  int
    Uptime       int64
}
```

### 6. Tunnel Implementation (internal/tunnel/)
Core tunnel functionality:
```go
// tunnel.go - Tunnel operations
type Tunnel struct {
    conn      net.Conn
    adapter   adapter.Interface
    throttler *throttle.Limiter
    done      chan struct{}
    wg        sync.WaitGroup
}

// tls.go - TLS handling
func ConfigureTLS(certFile, keyFile, caFile string) (*tls.Config, error)
```

## Dependencies

### Internal Dependencies
- adapter → None
- cert → config
- config → None
- connection → tunnel, monitor
- monitor → metrics
- throttle → None
- tunnel → adapter, throttle

### External Dependencies
```go
// From go.mod
require (
    github.com/sirupsen/logrus v1.9.3
    go.uber.org/zap latest
    golang.org/x/crypto v0.17.0
    golang.org/x/sys v0.15.0
    gopkg.in/yaml.v2 v2.4.0
)
```

## Testing Structure

### Unit Tests
Each package contains its own unit tests:
- adapter_test.go
- tunnel_test.go
- monitor_test.go
- etc.

### Integration Tests
Shell scripts for end-to-end testing:
- test_temp_certs.sh
- test_cert_generation.sh
- transfer_certs.sh

### Test Utilities
Common test helpers and fixtures:
- Mock interfaces
- Test certificates
- Configuration templates

## Build System

### Makefile Targets
```makefile
build:     Build the binary
clean:     Clean build artifacts
test:      Run unit tests
install:   Install the binary
generate:  Generate certificates
validate:  Validate certificates
```

## Code Patterns

### Error Handling
- Use of custom error types
- Error wrapping with context
- Consistent error reporting

### Concurrency
- Use of goroutines for I/O
- Channel-based communication
- WaitGroup for synchronization

### Resource Management
- Proper cleanup in defer blocks
- Resource pooling where appropriate
- Graceful shutdown handling

## Future Considerations

### Planned Refactoring
- Extract common TUN logic
- Improve error handling
- Enhance monitoring system

### Technical Debt
- Certificate rotation
- Cross-platform testing
- Performance optimization

### Enhancement Opportunities
- Add clustering support
- Implement high availability
- Create management UI
