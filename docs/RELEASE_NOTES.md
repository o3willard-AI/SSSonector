# Release Notes

## Version 1.0.0 (January 31, 2025)

### New Features
- Implemented server and client modes for SSL tunneling
- Added bandwidth control with upload and download throttling
- Added cross-platform support for Linux, Windows, and macOS
- Implemented platform-specific virtual network interfaces
- Added TLS certificate management

### Configuration Changes
- Added `uploadKbps` and `downloadKbps` settings for bandwidth control
- Improved certificate configuration with proper validation
- Added support for CA certificates

### Bug Fixes
- Fixed interface initialization issues
- Improved error handling and resource cleanup
- Fixed cross-platform compatibility issues

### Documentation
- Updated configuration documentation with new bandwidth settings
- Added platform-specific installation guides
- Improved troubleshooting documentation

### Known Issues
- macOS package requires final build on macOS system
