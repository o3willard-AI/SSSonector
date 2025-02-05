# Release Notes

## Version 1.1.0 (2025-02-05)

### New Features

#### Certificate Management
- Added built-in certificate generation with `-keygen` flag
- Implemented automatic certificate location and validation
- Added test mode with temporary certificates (`-test-without-certs`)
- Added flexible certificate path configuration with `-keyfile` flag
- Added comprehensive certificate validation and verification
- Added detailed error messages for certificate issues

#### Configuration
- Added YAML configuration support
- Added example server and client configurations
- Added configuration validation
- Added support for overriding certificate paths

#### Documentation
- Added detailed certificate management documentation
- Updated installation guides
- Added troubleshooting guide
- Improved configuration documentation

### Improvements
- Enhanced error handling for certificate operations
- Improved certificate validation checks
- Added automatic log directory creation
- Added certificate file permission checks
- Added certificate expiration monitoring

### Bug Fixes
- Fixed certificate path handling on Windows
- Fixed permission issues with private keys
- Fixed certificate validation error messages
- Fixed configuration file loading errors

## Version 1.0.0 (2025-01-15)

### Initial Release
- Basic SSL tunneling functionality
- Server and client modes
- Rate limiting support
- SNMP monitoring
- Cross-platform support
- Basic configuration options

### Known Issues
- Manual certificate management required
- Limited error reporting
- Basic configuration validation
- No test mode available

## Upcoming Features

### Version 1.2.0 (Planned)
- Certificate rotation automation
- Certificate expiration notifications
- Certificate backup and restore
- Enhanced test mode features
- Improved monitoring capabilities
- Performance optimizations

### Version 1.3.0 (Planned)
- Web-based certificate management
- Certificate revocation support
- Enhanced security features
- Performance monitoring
- Advanced rate limiting options
- Clustering support
