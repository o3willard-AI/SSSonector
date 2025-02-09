# SSSonector Development Guide

## Development Environment Setup

### 1. Prerequisites

```bash
# Required tools
go 1.21 or later
git
make
gcc
openssl

# Optional tools
dlv (debugging)
golangci-lint (linting)
goimports (code formatting)
```

### 2. Repository Setup

```bash
# Clone repository
git clone https://github.com/o3willard-AI/SSSonector.git
cd SSSonector

# Install dependencies
go mod download

# Build project
make build

# Run tests
make test
```

### 3. Development Tools

```bash
# Install development tools
make dev-tools

# Run linter
make lint

# Run code formatter
make fmt

# Generate mocks
make generate
```

## Project Structure

```
.
├── cmd/                    # Command-line tools
│   ├── sssonector/        # Main service
│   └── sssonectorctl/     # Control utility
├── docs/                   # Documentation
│   ├── admin/             # Administration guides
│   ├── config/            # Configuration docs
│   ├── implementation/    # Implementation details
│   └── security/          # Security docs
├── internal/              # Internal packages
│   ├── adapter/           # Network adapters
│   ├── config/            # Configuration
│   ├── monitor/           # Monitoring
│   ├── security/          # Security
│   ├── service/           # Service management
│   ├── throttle/          # Rate limiting
│   └── tunnel/            # Tunnel implementation
├── scripts/               # Build/install scripts
├── security/              # Security policies
└── test/                  # Test suites
```

## Development Workflow

### 1. Making Changes

1. Create feature branch:
```bash
git checkout -b feature/my-feature
```

2. Make changes following guidelines:
- Use interfaces for abstraction
- Write tests first
- Follow Go best practices
- Update documentation

3. Run tests:
```bash
# Run unit tests
make test

# Run integration tests
make test-integration

# Run all tests with coverage
make test-coverage
```

4. Submit changes:
```bash
# Format code
make fmt

# Run linter
make lint

# Commit changes
git commit -s -m "feat: Add new feature"

# Push changes
git push origin feature/my-feature
```

### 2. Testing Guidelines

#### Unit Tests

```go
func TestFeature(t *testing.T) {
    // Arrange
    cfg := &config.AppConfig{...}
    logger := zaptest.NewLogger(t)
    
    // Act
    result, err := NewFeature(cfg, logger)
    
    // Assert
    require.NoError(t, err)
    assert.NotNil(t, result)
}
```

#### Integration Tests

```go
func TestIntegration(t *testing.T) {
    // Skip in short mode
    if testing.Short() {
        t.Skip("Skipping integration test")
    }
    
    // Setup test environment
    cleanup := setupTest(t)
    defer cleanup()
    
    // Run test
    ...
}
```

#### Performance Tests

```go
func BenchmarkFeature(b *testing.B) {
    // Setup
    cfg := setupBenchmark(b)
    
    // Run benchmark
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        result := feature(cfg)
        require.NotNil(b, result)
    }
}
```

### 3. Documentation Guidelines

#### Code Documentation

```go
// Package throttle provides rate limiting functionality.
package throttle

// TokenBucket implements the token bucket algorithm for rate limiting.
// It is thread-safe and supports dynamic rate updates.
type TokenBucket struct {
    // rate is the token replenishment rate per second
    rate float64
    
    // burst is the maximum token capacity
    burst float64
}

// Take attempts to take n tokens from the bucket.
// It returns true if successful, false if insufficient tokens.
func (b *TokenBucket) Take(n float64) bool {
    ...
}
```

#### API Documentation

```markdown
## API Endpoint

### POST /api/v1/tunnel/start

Start a new tunnel connection.

**Request:**
```json
{
    "mode": "server",
    "address": "192.168.1.1",
    "port": 8080
}
```

**Response:**
```json
{
    "id": "tunnel-123",
    "status": "running"
}
```
```

### 4. Performance Optimization

#### Profiling

```bash
# Enable profiling
export SSSONECTOR_PROFILE=1

# Run with CPU profiling
go tool pprof http://localhost:6060/debug/pprof/profile

# Run with memory profiling
go tool pprof http://localhost:6060/debug/pprof/heap
```

#### Benchmarking

```bash
# Run benchmarks
go test -bench=. -benchmem ./...

# Run specific benchmark
go test -bench=BenchmarkFeature -benchmem ./...

# Profile benchmark
go test -bench=. -cpuprofile=cpu.prof
```

## Common Tasks

### 1. Adding a New Feature

1. Design the interface:
```go
type NewFeature interface {
    Start() error
    Stop() error
    Status() Status
}
```

2. Implement the interface:
```go
type feature struct {
    cfg    *config.AppConfig
    logger *zap.Logger
}

func NewFeature(cfg *config.AppConfig, logger *zap.Logger) (NewFeature, error) {
    ...
}
```

3. Add configuration:
```go
type FeatureConfig struct {
    Enabled bool    `json:"enabled"`
    Rate    float64 `json:"rate"`
}
```

4. Write tests:
```go
func TestNewFeature(t *testing.T) {
    ...
}
```

### 2. Adding a New Protocol

1. Define protocol interface:
```go
type Protocol interface {
    Connect() error
    Send(data []byte) error
    Receive() ([]byte, error)
    Close() error
}
```

2. Implement protocol:
```go
type protocol struct {
    conn    net.Conn
    config  *config.ProtocolConfig
    logger  *zap.Logger
}
```

3. Add protocol factory:
```go
func NewProtocol(cfg *config.ProtocolConfig) (Protocol, error) {
    ...
}
```

### 3. Adding Metrics

1. Define metrics:
```go
var (
    connectionCounter = prometheus.NewCounter(prometheus.CounterOpts{
        Name: "connections_total",
        Help: "Total number of connections",
    })
)
```

2. Register metrics:
```go
func init() {
    prometheus.MustRegister(connectionCounter)
}
```

3. Use metrics:
```go
func (s *Server) handleConnection() {
    connectionCounter.Inc()
    ...
}
```

## Debugging Tips

### 1. Logging

```go
// Use structured logging
logger.Info("Starting server",
    zap.String("addr", addr),
    zap.Int("port", port),
)

// Include context
logger.Error("Connection failed",
    zap.Error(err),
    zap.String("remote", remote),
)
```

### 2. Debugging

```go
// Use delve
dlv debug ./cmd/sssonector

// Set breakpoints
b internal/tunnel/server.go:123

// Inspect variables
p variable
```

### 3. Tracing

```go
// Enable tracing
ctx, span := tracer.Start(ctx, "operation")
defer span.End()

// Add attributes
span.SetAttributes(
    attribute.String("key", "value"),
)
```

## Release Process

1. Version bump:
```bash
make version VERSION=v1.2.0
```

2. Update changelog:
```bash
make changelog
```

3. Build release:
```bash
make release
```

4. Create release:
```bash
make publish
```

## Resources

- [Go Documentation](https://golang.org/doc/)
- [Project Wiki](https://github.com/o3willard-AI/SSSonector/wiki)
- [Style Guide](https://golang.org/doc/effective_go)
- [Security Guidelines](../security/README.md)
