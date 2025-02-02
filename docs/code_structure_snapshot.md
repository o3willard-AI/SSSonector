# SSSonector Code Structure Snapshot

This document provides a snapshot of the Go source code structure as of Version 1.0.0 (January 31, 2025).

## Source Code Organization

### Command Layer (`cmd/tunnel/`)
- `main.go`: Application entry point
- `mode.go`: Mode selection and configuration
- `client.go`: Client mode implementation
- `server.go`: Server mode implementation

### Internal Packages

#### Adapter (`internal/adapter/`)
- `interface.go`: Common interface definitions
- `interface_darwin.go`: macOS-specific implementation
- `interface_linux.go`: Linux-specific implementation
- `interface_windows.go`: Windows-specific implementation
- `manager.go`: Interface management
- `types.go`: Shared type definitions

#### Certificate Management (`internal/cert/`)
- `manager.go`: Certificate and key management

#### Configuration (`internal/config/`)
- `loader.go`: Configuration file loading
- `types.go`: Configuration structure definitions

#### Connection Management (`internal/connection/`)
- `manager.go`: Connection handling
- `manager_test.go`: Connection tests

#### Monitoring (`internal/monitor/`)
- `logging.go`: Logging implementation
- `metrics.go`: Metrics collection
- `monitor.go`: Monitoring system
- `snmp.go`: SNMP support
- `snmp_pdu.go`: SNMP protocol data units

#### Bandwidth Control (`internal/throttle/`)
- `limiter.go`: Bandwidth limiting implementation
- `limiter_test.go`: Bandwidth control tests

#### Tunnel Core (`internal/tunnel/`)
- `cert.go`: Certificate operations
- `errors.go`: Error definitions
- `tls.go`: TLS implementation
- `transfer.go`: Data transfer
- `tunnel.go`: Core tunnel logic

## Package Dependencies

```
cmd/tunnel/
  ├─ internal/adapter
  ├─ internal/cert
  ├─ internal/config
  ├─ internal/connection
  ├─ internal/monitor
  ├─ internal/throttle
  └─ internal/tunnel
```

## Critical Components

1. Interface Management
   - Platform-specific implementations in adapter package
   - Thread-safe operations
   - Resource cleanup

2. Security
   - TLS implementation in tunnel package
   - Certificate management in cert package
   - Secure transfer protocols

3. Performance
   - Bandwidth control in throttle package
   - Connection management
   - Resource optimization

4. Monitoring
   - Logging system
   - Metrics collection
   - SNMP support

## Test Coverage

Unit tests are present for critical components:
- Connection management (`manager_test.go`)
- Bandwidth control (`limiter_test.go`)

This snapshot should be updated whenever significant changes are made to the code structure.
