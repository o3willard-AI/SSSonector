# SSSonector Documentation Index

## Core Documentation

1. `README.md`
   - Main project documentation
   - Overview of features including TLS tunneling, cross-platform support, monitoring, and rate limiting
   - Quick start guide with installation and configuration examples
   - Build instructions and contribution guidelines

## Implementation Documentation

1. `docs/implementation/ARCHITECTURE.md`
   - Comprehensive system architecture documentation
   - Details of core components: Service Layer, Control System, Configuration Management
   - Network Layer and Security System design
   - Monitoring and Rate Limiting system architecture
   - Communication and Control Flow diagrams
   - Error handling and platform support details
   - Future considerations for scalability and improvements

2. `docs/rate_limiting_implementation.md`
   - Detailed guide for rate limiting implementation in v2.1.0
   - TCP overhead compensation and burst control mechanisms
   - Configuration examples for various scenarios
   - Performance tuning guidelines and buffer configuration
   - SNMP monitoring integration details
   - Troubleshooting guide and best practices
   - Security considerations for rate limiting

## Installation Documentation

1. `docs/linux_install.md`
   - Comprehensive Linux installation guide
   - System requirements and pre-installation steps
   - Binary and source installation methods
   - Configuration examples and systemd service setup
   - Firewall configuration and performance tuning
   - Troubleshooting guide and monitoring integration
   - Support resources and documentation

2. `docs/windows_install.md`
   - Windows installation and configuration guide
   - TAP adapter setup and system requirements
   - Binary and source installation procedures
   - Windows service configuration
   - Performance tuning and firewall setup
   - Known limitations and troubleshooting
   - Windows-specific monitoring setup

3. `docs/macos_install.md`
   - macOS installation and setup guide
   - System requirements and development prerequisites
   - Intel and Apple Silicon support
   - Launch daemon configuration
   - Performance optimization
   - Platform-specific limitations
   - macOS monitoring integration

## Configuration Documentation

1. `docs/config/README.md`
   - Comprehensive configuration management system documentation
   - Type-safe configuration structures and validation
   - Configuration versioning and hot reload capabilities
   - Multiple format support (JSON, YAML, TOML)
   - Storage backend implementation details
   - Configuration examples and best practices
   - Integration with other system components
   - Error handling and troubleshooting

## Platform-Specific Documentation

1. Linux Support
   - Full TUN interface support
   - Comprehensive systemd integration
   - Advanced network configuration
   - Performance optimization options

2. Windows Support
   - Basic TAP adapter implementation
   - Windows service management
   - Performance counter integration
   - Known limitations and workarounds

3. macOS Support
   - Basic network interface support
   - Launch daemon integration
   - System Integrity Protection considerations
   - Platform-specific limitations

## Testing and QA Documentation

1. `docs/qa_guide.md`
   - Comprehensive QA testing procedures and environment setup
   - Detailed test cases for installation, service management, and rate limiting
   - Network interface and performance testing guidelines
   - Stress testing procedures and documentation requirements
   - Common issues and troubleshooting guide
   - Bug reporting guidelines and requirements

2. `test/qa_docs/qa_environment.md`
   - Detailed QA environment configuration and setup
   - Information about Server (192.168.50.210), Client (192.168.50.211), and Monitor (192.168.50.212) systems
   - Access credentials and SSH certificate details
   - Test resources and FTP server configurations
   - Build and deployment process
   - Environment maintenance procedures
   - Network configuration and required services

## Security Documentation

1. `docs/security/architecture.md`
   - Detailed security architecture diagrams and flows
   - Process isolation and resource control
   - Mandatory access control implementation
   - Security boundaries and implementation details
   - Component interaction diagrams

2. `docs/security/README.md`
   - Comprehensive security features overview
   - Linux security features (Namespaces, Cgroups, Seccomp)
   - Memory protection and resource limits
   - SELinux and AppArmor policy configuration
   - Security best practices and monitoring
   - Installation and troubleshooting guides
   - Security update procedures and references

## Monitoring Documentation

1. `docs/snmp_monitoring.md`
   - Comprehensive SNMP monitoring setup and usage guide
   - Detailed MIB structure and metric categories
   - Configuration examples for different use cases
   - Usage examples for metrics collection
   - Integration with Prometheus and Grafana
   - Troubleshooting guide and best practices
   - Security and performance recommendations
