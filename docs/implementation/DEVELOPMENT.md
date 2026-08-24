# SSSonector Development Guide

This document provides information for developers working on the SSSonector codebase.

## Project Structure

```
internal/
  ├── adapter/         # Network interface adapters
  ├── buffer/          # Multi-tier buffer pool management
  ├── cert/            # Certificate lifecycle and rotation
  ├── config/          # Configuration management
  │   ├── types/       # Configuration type definitions
  │   ├── validator/   # Configuration validation rules
  │   ├── store/       # File-based config storage
  │   ├── manager/     # Thread-safe config manager
  │   └── loader.go    # Version detection and upgrade
  ├── connection/      # Connection lifecycle management
  ├── facade/          # HTTPS facade for firewall traversal
  │   ├── server.go    # HTTPS server with WebSocket upgrade
  │   ├── client.go    # Direct-connect with facade fallback
  │   ├── token.go     # HMAC-SHA256 token generation/validation
  │   └── proxy.go     # Bidirectional TCP proxy
  ├── memory/          # Memory management and GC control
  ├── monitor/         # Monitoring and metrics
  ├── pool/            # Connection pooling
  ├── security/        # Security and authentication
  ├── service/         # Core service implementation
  │   ├── control/     # Service control interface
  │   ├── daemon/      # Daemon management
  │   ├── platform/    # Platform-specific code
  │   ├── base.go      # Base service implementation
  │   └── types.go     # Service type definitions
  ├── throttle/        # Rate limiting (token bucket + dynamic)
  └── tunnel/          # Tunnel implementation
```

## Service Control System

The service control system provides a unified interface for managing and monitoring the SSSonector service. It consists of several components:

### Service Interface

The core service interface (`internal/service/types.go`) defines the contract for service operations:

```go
type Service interface {
    Start() error
    Stop() error
    Reload() error
    Status() (*ServiceStatus, error)
    Metrics() (*ServiceMetrics, error)
    Health() error
}
```

Service states are represented by the `ServiceState` type:
- `StateStarting`: Service is initializing
- `StateRunning`: Service is operational
- `StateStopping`: Service is shutting down
- `StateStopped`: Service is not running
- `StateReloading`: Service is reloading configuration
- `StateFailed`: Service encountered an error

### Control Commands

Service control commands (`ServiceCommand` type) include:
- `CmdStatus`: Get service status
- `CmdMetrics`: Get service metrics
- `CmdHealth`: Perform health check
- `CmdStart`: Start the service
- `CmdStop`: Stop the service
- `CmdReload`: Reload configuration

### Control Server

The control server (`internal/service/control/interface.go`) provides:
- Unix socket-based communication
- Command handling and routing
- Response serialization
- Error handling

Configuration:
```go
type ControlServer struct {
    service    service.Service
    socket     net.Listener
    socketPath string
}
```

The socket path can be configured via:
```go
func (c *ControlServer) SetSocketPath(path string)
```

### Control Client

The control client (`internal/service/control/client.go`) provides:
- Socket connection management
- Command execution
- Response deserialization
- Error handling

Usage:
```go
client, err := control.NewClient(cfg, logger)
client.SetSocketPath("/var/run/sssonector.sock")
response, err := client.ExecuteCommand(service.CmdStatus, nil)
```

### Base Service Implementation

The base service (`internal/service/base.go`) provides:
- Core service lifecycle management
- Status and metrics tracking
- Configuration management
- Error handling

### CLI Interface

The command-line interface (`cmd/sssonectorctl/main.go`) supports:
- All service commands
- Socket path configuration
- JSON output formatting
- Error reporting

Example usage:
```bash
sssonectorctl status
sssonectorctl --socket=/var/run/custom.sock metrics
sssonectorctl reload
```

## Error Handling

Service errors are represented by the `ServiceError` type with specific error codes:
- `ErrNotRunning`: Service is not running
- `ErrAlreadyRunning`: Service is already running
- `ErrInvalidCommand`: Invalid command received
- `ErrInvalidConfig`: Invalid configuration
- `ErrInternal`: Internal service error

## Configuration

The service control system uses the following configuration:

```go
type ServiceOptions struct {
    Name      string // Service name
    ConfigDir string // Configuration directory
    DataDir   string // Data directory
    LogDir    string // Log directory
}
```

Default socket path: `/var/run/sssonector.sock`

## Development Guidelines

1. Error Handling
   - Use `ServiceError` for service-specific errors
   - Include error codes for categorization
   - Provide descriptive error messages

2. Command Implementation
   - Add new commands in `service/types.go`
   - Implement command handling in control server
   - Update CLI interface for new commands

3. Testing
   - Write unit tests for new commands
   - Test error conditions
   - Verify socket communication
   - Test platform-specific behavior

4. Documentation
   - Update this document for new features
   - Document command parameters
   - Include usage examples
   - Document error conditions

## Building and Testing

Build the service:
```bash
make build
```

Run tests:
```bash
make test
```

## Contributing

1. Follow Go coding standards
2. Add tests for new features
3. Update documentation
4. Run full test suite before submitting
