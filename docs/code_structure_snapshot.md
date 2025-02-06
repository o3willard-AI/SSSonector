# Code Structure Overview

## Core Components

### Tunnel Implementation (`internal/tunnel/`)
- `tunnel.go`: Core tunnel implementation with optimized data transfer
  * Bidirectional data streaming with EOF handling
  * MTU-aware chunked transfers
  * Exponential backoff retry mechanism
  * Buffer overflow prevention
  * Metrics integration
- `tls.go`: TLS configuration and security settings
- `tls_config.go`: TLS profile management
- `buffer.go`: Buffer management and pooling
- `tunnel_test.go`: Comprehensive test suite

### Certificate Management (`internal/cert/`)
- `manager.go`: Certificate lifecycle management
- `generator.go`: Certificate generation utilities
- `validator.go`: Certificate validation logic
- `locator.go`: Certificate discovery and loading
- `generator/`: Certificate generation implementation
- `validator/`: Detailed validation rules

### Network Adapters (`internal/adapter/`)
- `interface.go`: Common adapter interface
- `interface_linux.go`: Linux TUN implementation
- `interface_darwin.go`: macOS network extension
- `interface_windows.go`: Windows TAP adapter

### Monitoring System (`internal/monitor/`)
- `monitor.go`: Core monitoring functionality
- `metrics.go`: Performance metrics collection
- `logging.go`: Structured logging
- `snmp.go`: SNMP integration
- `mib.go`: SNMP MIB definitions
- `snmp_message.go`: SNMP message handling
- `snmp_asn1.go`: ASN.1 encoding/decoding

### Rate Limiting (`internal/throttle/`)
- `limiter.go`: Rate limiting implementation
- `token_bucket.go`: Token bucket algorithm
- `io.go`: I/O rate control
- `limiter_test.go`: Rate limiting tests

### Connection Management (`internal/connection/`)
- `manager.go`: Connection lifecycle
- `manager_test.go`: Connection tests

### Configuration (`internal/config/`)
- `types.go`: Configuration structures
- `loader.go`: Configuration loading

### Command Line Interface (`cmd/tunnel/`)
- `main.go`: Entry point and flag handling
- `server.go`: Server mode implementation
- `client.go`: Client mode implementation

## Test Suite

### Integration Tests (`test/`)
- `test_temp_certs.sh`: Certificate testing
- `test_cert_generation.sh`: Certificate generation tests
- `transfer_certs.sh`: Certificate transfer tests
- `run_cert_tests.sh`: Certificate test suite runner
- `test_results.md`: Test results documentation

### Unit Tests
- Comprehensive test coverage across all packages
- Mock implementations for testing
- Performance benchmarks
- Error condition testing

## Documentation

### User Documentation
- `README.md`: Project overview and quick start
- `docs/installation.md`: Installation instructions
- `docs/certificate_management.md`: Certificate guide
- `docs/project_context.md`: Project overview
- `docs/code_structure_snapshot.md`: Code organization
- `docs/RELEASE_NOTES.md`: Version history

### Platform-specific Guides
- `docs/linux_install.md`: Linux setup
- `docs/macos_build.md`: macOS build guide
- `docs/windows_install.md`: Windows setup
- `docs/ubuntu_install.md`: Ubuntu-specific guide

### Development Documentation
- `docs/virtualbox_testing.md`: Testing environment
- `docs/ai_context_restoration.md`: Development history
- `docs/ai_recovery_prompt.md`: Recovery procedures

## Build System

### Build Configuration
- `Makefile`: Build automation
- `go.mod`: Go module definition
- `configs/`: Configuration templates
  * `server.yaml`: Server configuration
  * `client.yaml`: Client configuration

### Installation
- `installers/`: Platform installers
  * `windows.nsi`: Windows installer script

## Recent Improvements

### Tunnel Optimizations
- Enhanced EOF handling in data transfer
- Improved buffer management
- Added retry mechanisms
- Optimized chunked transfers
- Better error recovery

### Monitoring Enhancements
- More detailed metrics collection
- Improved error tracking
- Enhanced SNMP integration
- Better performance analysis

### Testing Improvements
- Extended test coverage
- More comprehensive error testing
- Better mock implementations
- Enhanced performance benchmarks
