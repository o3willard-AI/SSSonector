# SSSonector Project Context

## Project Overview
SSSonector is a cross-platform SSL tunneling application that provides secure network connectivity between systems. It supports both server and client modes, with configurable bandwidth controls and comprehensive monitoring capabilities.

## Core Components

### 1. Network Interface Management
- Platform-specific virtual network interfaces (TUN/TAP)
- Implementations for Linux, Windows, and macOS
- Automatic interface creation and cleanup
- MTU configuration support

### 2. SSL/TLS Implementation
- Certificate-based authentication
- Support for CA, server, and client certificates
- Secure key management
- Certificate validation and verification

### 3. Bandwidth Control
- Configurable upload and download throttling
- Per-connection bandwidth limits
- Real-time bandwidth monitoring
- Default limits of 10 Mbps for both directions

### 4. Configuration System
- YAML-based configuration
- Support for both server and client modes
- Flexible certificate path configuration
- Monitoring and logging settings

## Recent Updates (as of January 31, 2025)

### Core Functionality Implementation
1. Server and Client Modes
   - Fully implemented server mode with client acceptance
   - Client mode with automatic reconnection
   - Support for multiple simultaneous clients
   - Proper connection handling and cleanup

2. Cross-Platform Support
   - Linux: Full TUN/TAP support with ip command integration
   - Windows: TAP driver integration
   - macOS: TUN device support
   - Platform-specific network configuration

3. Performance Features
   - Bandwidth throttling implementation
   - Connection monitoring
   - Statistics collection
   - Resource cleanup

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
- Cross-platform build support
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

## Development Guidelines

### Code Structure
1. Main Components:
   - cmd/tunnel: Main application entry point
   - internal/adapter: Platform-specific network interfaces
   - internal/cert: Certificate management
   - internal/config: Configuration handling
   - internal/tunnel: Core tunneling logic

2. Platform-Specific Code:
   - interface_linux.go
   - interface_windows.go
   - interface_darwin.go

### Best Practices
1. Error Handling:
   - Proper error wrapping
   - Detailed error messages
   - Graceful failure handling

2. Resource Management:
   - Proper cleanup of network interfaces
   - Certificate file handling
   - Memory management

3. Security:
   - Secure certificate handling
   - Proper file permissions
   - Network security best practices

## Testing

### Test Environment
- VirtualBox-based testing environment
- Cross-platform testing requirements
- Network isolation for security testing

### Test Cases
1. Basic Functionality:
   - Server/client connection
   - Certificate validation
   - Bandwidth control

2. Error Conditions:
   - Network failures
   - Invalid certificates
   - Resource exhaustion

3. Performance:
   - Bandwidth limits
   - Multiple clients
   - Resource usage

## Known Issues and Limitations
1. macOS Package:
   - Final package must be built on macOS system
   - Requires additional signing steps

2. Windows TAP Driver:
   - Requires administrator privileges
   - May need manual driver installation

## Future Improvements
1. Planned Features:
   - UDP tunnel support
   - IPv6 support
   - Dynamic reconfiguration
   - Web-based management interface

2. Performance Optimizations:
   - Connection pooling
   - Improved buffer management
   - Protocol optimizations

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

## Documentation
- Installation guides for all platforms
- Configuration reference
- API documentation
- Troubleshooting guide
- Release notes

This context document should be updated with any significant changes to the project structure, features, or best practices.
