# SSSonector Project Context

## Project Overview
SSSonector is a cross-platform SSL tunneling application that provides secure network connectivity between systems. It supports both server and client modes with configurable bandwidth controls, thread-safe operations, and comprehensive monitoring capabilities.

## Core Components

### 1. Network Interface Management
- Platform-specific virtual network interfaces with thread safety:
  - Linux: TUN with ICMP handling and packet routing
  - Windows: TAP with mutex-protected operations
  - macOS: UTUN with proper byte handling
- Automatic interface creation and cleanup
- MTU configuration support
- Resource leak prevention

### 2. SSL/TLS Implementation
- TLS 1.3 with certificate-based authentication
- Support for CA, server, and client certificates
- Secure key management and validation
- Certificate rotation support
- EU-exportable cipher suites

### 3. Bandwidth Control
- Configurable upload and download throttling
- Per-connection bandwidth limits
- Real-time bandwidth monitoring
- Default limits of 10 Mbps for both directions
- Thread-safe bandwidth management

### 4. Configuration System
- YAML-based configuration
- Support for both server and client modes
- Flexible certificate path configuration
- Monitoring and logging settings
- Bandwidth control settings

## Recent Updates (Version 1.0.0 - January 31, 2025)

### Core Functionality Implementation
1. Server and Client Modes
   - Thread-safe server mode with client acceptance
   - Client mode with automatic reconnection
   - Support for multiple simultaneous clients
   - Proper connection handling and cleanup
   - Resource leak prevention

2. Cross-Platform Support
   - Linux: Full TUN support with ICMP handling
   - Windows: Enhanced TAP driver integration
   - macOS: Improved UTUN device support
   - Platform-specific network configuration
   - Thread-safe operations

3. Performance Features
   - Bandwidth throttling implementation
   - Connection monitoring
   - Statistics collection
   - Resource cleanup
   - Race condition prevention

### Configuration Updates
1. New Configuration Options
   ```yaml
   tunnel:
     uploadKbps: 10240    # 10 Mbps
     downloadKbps: 10240  # 10 Mbps
   ```

2. Enhanced Certificate Management
   - Support for CA certificates
   - Proper certificate validation
   - Secure key file handling
   - Certificate rotation support

3. Monitoring Configuration
   ```yaml
   monitor:
     logFile: "/var/log/sssonector/monitor.log"
     snmpEnabled: false
     snmpPort: 161
     snmpCommunity: "public"
   ```

## Package Management

### Build System
- Cross-platform build support with build tags
- Automated package generation
- Platform-specific installers:
  - Linux: .deb and .rpm packages
  - Windows: NSIS-based installer
  - macOS: pkg installer structure

### Installation
- Automated service installation
- Configuration file deployment
- Certificate directory setup
- Proper permissions management
- Resource cleanup on uninstall

## Development Guidelines

### Code Structure
1. Main Components:
   - cmd/tunnel: Main application entry point with build tags
   - internal/adapter: Thread-safe platform-specific interfaces
   - internal/cert: Certificate management
   - internal/config: Configuration handling
   - internal/tunnel: Core tunneling logic

2. Platform-Specific Code:
   - interface_linux.go: TUN with ICMP
   - interface_windows.go: TAP with mutex
   - interface_darwin.go: UTUN with byte handling

### Best Practices
1. Error Handling:
   - Proper error wrapping
   - Detailed error messages
   - Graceful failure handling
   - Resource cleanup

2. Resource Management:
   - Thread-safe interface operations
   - Proper mutex usage
   - Memory leak prevention
   - Graceful cleanup

3. Security:
   - Secure certificate handling
   - Proper file permissions
   - Network security best practices
   - Resource isolation

## Testing

### Test Environment
- VirtualBox-based testing environment
- Cross-platform testing requirements
- Network isolation for security testing
- Thread safety validation

### Test Cases
1. Basic Functionality:
   - Server/client connection
   - Certificate validation
   - Bandwidth control
   - Thread safety

2. Error Conditions:
   - Network failures
   - Invalid certificates
   - Resource exhaustion
   - Race conditions

3. Performance:
   - Bandwidth limits
   - Multiple clients
   - Resource usage
   - Memory leaks

## Known Issues and Limitations
1. macOS Package:
   - Final package must be built on macOS system
   - Requires additional signing steps

2. Windows TAP Driver:
   - Requires administrator privileges
   - May need manual driver installation
   - Some antivirus software may require manual whitelisting

3. Interface Management:
   - Race conditions in tunnel initialization have been fixed
   - Improved error handling for interface operations
   - Enhanced packet handling in tunnel transfer

## Future Improvements
1. Planned Features:
   - Web-based management interface
   - Enhanced monitoring capabilities
   - Additional platform support
   - Performance optimizations

2. Performance Optimizations:
   - Connection pooling
   - Improved buffer management
   - Protocol optimizations
   - Resource usage optimization

## Maintenance
1. Regular Tasks:
   - Certificate rotation
   - Log rotation
   - Performance monitoring
   - Security updates

2. Troubleshooting:
   - Log analysis
   - Network diagnostics
   - Performance profiling
   - Resource monitoring

## Documentation
- Installation guides for all platforms
- Configuration reference
- API documentation
- Troubleshooting guide
- Release notes

This context document should be updated with any significant changes to the project structure, features, or best practices.
