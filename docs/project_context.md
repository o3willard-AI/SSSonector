SSSonector Project Context

## Project Overview

SSSonector is a cross-platform SSL tunneling application written in Go that enables secure communication between remote office locations through firewalls. The application can operate in both client and server modes creating persistent SSL tunnels for data transport.

### Key Features

- TLS 1.3 with EU-exportable cipher suites
- Virtual network interfaces for transparent routing
- Persistent tunnel connections with automatic reconnection
- Bandwidth throttling capabilities
- SNMP monitoring and telemetry
- Cross-platform support (Linux Windows macOS)
- Comprehensive logging and monitoring
- Systemd/Launchd/Windows Service integration
- Certificate management and rotation

## Technical Architecture

### Core Components

1. Network Adapters
- Virtual network interface creation and management
- Platform-specific implementations (Linux Windows macOS)
- Private IP space configuration (10.x.x.x 192.168.x.x)

2. Tunnel Management
- TLS 1.3 implementation
- Certificate handling
- Connection persistence
- Automatic reconnection logic

3. Configuration Management
- YAML-based configuration
- Server/Client mode settings
- Network interface settings
- Certificate paths
- Bandwidth throttling settings

4. Monitoring & Telemetry
- SNMP integration
- Logging system
- Performance metrics
- Connection status tracking

### Directory Structure

```
SSSonector/
├── cmd/
│   └── tunnel/           # Main application entry point
├── internal/
│   ├── adapter/          # Virtual network interface implementations
│   ├── cert/            # Certificate management
│   ├── config/          # Configuration handling
│   ├── monitor/         # Monitoring and telemetry
│   ├── throttle/        # Bandwidth throttling
│   └── tunnel/          # Core tunneling logic
├── configs/             # Configuration templates
├── dist/                # Release artifacts
│   └── v1.0.0/         # Version-specific release files
├── docs/               # Documentation
│   ├── installation.md # Installation guide
│   └── macos_build.md # macOS build instructions
├── installers/         # Platform-specific installers
│   └── windows.nsi    # NSIS installer script
└── scripts/            # Build and maintenance scripts
```

## Build System

### Prerequisites
- Go 1.21 or later (required for atomic operations and newer stdlib features)
- Platform-specific build tools:
  - Windows: NSIS (for creating Windows installers)
  - Linux: dpkg-deb (for .deb packages), rpmbuild (for .rpm packages)
  - macOS: pkgbuild (for .pkg installers)

### Build Process
1. Generate certificates (if needed)
2. Compile platform-specific binaries using the Makefile:
   ```bash
   make build  # Builds binaries for all platforms
   ```
3. Create installers:
   ```bash
   make package-deb     # Create Debian package
   make package-rpm     # Create RPM package
   make package-windows # Create Windows installer
   make package-macos   # Create macOS package (requires macOS)
   # Or build all:
   make package
   ```
4. Package documentation and artifacts

### Release Process
1. Build all installers using the Makefile
2. Generate SHA256 checksums for verification
3. Create GitHub release with version tag
4. Upload installer packages to GitHub release
5. Update installation documentation with new download links

## GitHub Integration

### Release Structure
- All releases are tagged (e.g., v1.0.0)
- Release assets include:
  - Linux: .deb and .rpm packages
  - Windows: NSIS installer (.exe)
  - macOS: Build instructions (pending community contribution)
  - SHA256 checksums file
  - Release notes

### File Handling
- Large binary files (installers) are distributed through GitHub Releases
- Build artifacts are excluded via .gitignore
- Documentation and source files are version controlled
- Release assets have consistent naming:
  - sssonector_${VERSION}_amd64.deb
  - sssonector-${VERSION}-1.x86_64.rpm
  - sssonector-${VERSION}-setup.exe
  - sssonector-${VERSION}.pkg (macOS, pending)

## Deployment

### Installation Methods

1. Linux (Debian/Ubuntu)
```bash
wget https://github.com/o3willard-AI/SSSonector/releases/download/v${VERSION}/sssonector_${VERSION}_amd64.deb
sudo dpkg -i sssonector_${VERSION}_amd64.deb
```

2. Linux (RHEL/CentOS)
```bash
wget https://github.com/o3willard-AI/SSSonector/releases/download/v${VERSION}/sssonector-${VERSION}-1.x86_64.rpm
sudo yum install sssonector-${VERSION}-1.x86_64.rpm
```

3. Windows
- Download installer from GitHub releases page
- Run with administrator privileges

4. macOS
- Follow build instructions in docs/macos_build.md
- Community contributions welcome for official packages

### Configuration

#### Server Mode
```yaml
mode: "server"
network:
  interface: "tun0"
  address: "10.0.0.1/24"
  mtu: 1500
tunnel:
  cert_file: "/etc/sssonector/certs/server.crt"
  key_file: "/etc/sssonector/certs/server.key"
  ca_file: "/etc/sssonector/certs/ca.crt"
  listen_address: "0.0.0.0"
  listen_port: 8443
```

#### Client Mode
```yaml
mode: "client"
network:
  interface: "tun0"
  address: "10.0.0.2/24"
  mtu: 1500
tunnel:
  cert_file: "/etc/sssonector/certs/client.crt"
  key_file: "/etc/sssonector/certs/client.key"
  ca_file: "/etc/sssonector/certs/ca.crt"
  server_address: "tunnel.example.com"
  server_port: 8443
```

## Testing

### VirtualBox Testing Environment
- Two VMs: server and client
- Host-only network for inter-VM communication
- NAT for internet access
- Detailed setup in docs/virtualbox_testing.md

### Performance Testing
```bash
# Install iperf3 on both VMs
sudo apt install iperf3

# On server
iperf3 -s

# On client
iperf3 -c 10.0.0.1 -t 30
```

## Development Guidelines

### Code Organization
1. Platform-specific code in separate files
2. Interface-based design for modularity
3. Clear separation of concerns
4. Comprehensive error handling
5. Detailed logging

### Testing Requirements
1. Unit tests for core components
2. Integration tests for tunnel functionality
3. Platform-specific testing
4. Performance benchmarks

### Documentation Standards
1. Clear code comments
2. Up-to-date API documentation
3. Comprehensive user guides
4. Detailed installation instructions

## Future Enhancements

1. Additional Platform Support
- ARM architecture support
- Container-based deployment
- Cloud platform integration

2. Feature Enhancements
- Multiple tunnel support
- Advanced routing capabilities
- Enhanced monitoring dashboard
- Automated certificate rotation

3. Performance Improvements
- Optimized packet handling
- Reduced memory footprint
- Improved reconnection logic

## Using This Context

When working with this project in AI-assisted development:

1. Initial Setup
- Share this context document at the start of each new task
- Highlight specific sections relevant to the task

2. Task Structure
- Begin with clear task objectives
- Reference relevant sections from this document
- Include any additional context specific to the task

3. Implementation
- Follow existing patterns and conventions
- Maintain consistency with current architecture
- Update documentation as needed

4. Testing
- Follow established testing patterns
- Use VirtualBox environment for validation
- Verify cross-platform compatibility

5. Documentation
- Update this context document for significant changes
- Maintain platform-specific documentation
- Keep installation guides current

Example Task Format:
```markdown
Task: [Brief description]

Relevant Context:
- [Section from this document]
- [Additional specific details]

Requirements:
1. [Requirement 1]
2. [Requirement 2]
...

Expected Outcome:
- [What success looks like]
```

This context document should be treated as a living document and updated as the project evolves.

## Recent Changes and Improvements

### Download and Distribution Changes (January 2025)

1. GitHub Releases Integration
- All installer packages are now exclusively distributed through GitHub Releases
- Direct repository downloads (via /blob/main/dist/...) have been deprecated
- Binary files are no longer stored in the repository's dist directory
- SHA256 checksums are included with each release

2. Download URL Format
- Correct format: `https://github.com/o3willard-AI/SSSonector/releases/download/v${VERSION}/[package-name]`
- Example: `https://github.com/o3willard-AI/SSSonector/releases/download/v1.0.0/sssonector_1.0.0_amd64.deb`
- This format ensures:
  * Correct filename preservation when downloading
  * Compatibility with wget and other download tools
  * Proper version tracking and distribution

3. Documentation Updates
- All documentation now uses consistent GitHub Releases URLs
- Added download verification steps
- Added warnings about correct URL format usage
- Updated installation guides with proper download instructions

### File Organization

1. Repository Structure
- Source code and documentation in version control
- Binary files (installers) in GitHub Releases
- Build artifacts excluded via .gitignore
- Clear separation between source and distribution files

2. Best Practices
- Always use GitHub Releases URLs for downloads
- Verify downloads using provided SHA256 checksums
- Follow platform-specific installation guides
- Reference this context document for project standards

### Important Notes for AI Development

1. URL Handling
- Always use GitHub Releases URLs in documentation and scripts
- Include download verification steps
- Add clear warnings about URL format when documenting downloads

2. File Management
- Do not commit binary files to the repository
- Use GitHub Releases for distributing installers
- Keep documentation in sync with current practices
- Update this context document when making significant changes

3. Testing Considerations
- Verify download URLs work with wget and curl
- Check SHA256 checksums after downloads
- Test installation procedures on all supported platforms
- Ensure documentation reflects actual user experience

This section will be updated as new changes and improvements are made to the project.
