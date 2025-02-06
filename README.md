# SSSonector

A high-performance, secure SSL tunnel implementation with TUN interface support.

## Features

### Core Functionality
- TUN interface-based networking
- Certificate-based authentication
- Rate limiting and monitoring
- Cross-platform support (Linux/macOS/Windows)

### Performance Optimizations
- Optimized buffer management
- MTU-aware chunked transfers
- Intelligent retry mechanisms
- Connection pooling
- Exponential backoff for error recovery

### Security
- Strong certificate validation
- Automated certificate rotation
- Secure key management
- Regular security updates
- Comprehensive audit logging

### Monitoring
- Real-time performance metrics
- SNMP integration
- Detailed error tracking
- Resource utilization monitoring
- Connection status reporting

## Quick Start

### Installation

```bash
# From binary
curl -L https://github.com/o3willard-AI/SSSonector/releases/latest/download/sssonector-$(uname -s)-$(uname -m) -o sssonector
chmod +x sssonector
sudo mv sssonector /usr/local/bin/

# From source
git clone https://github.com/o3willard-AI/SSSonector.git
cd SSSonector
make build
sudo make install
```

### Basic Usage

```bash
# Generate certificates
sssonector -keygen

# Start server
sssonector -server -config server.yaml

# Start client
sssonector -client -config client.yaml
```

## Certificate Management

SSSonector provides five certificate management flags:

```bash
# Run with temporary certificates (testing only)
sssonector -test-without-certs

# Generate certificates without starting service
sssonector -generate-certs-only

# Specify certificate directory
sssonector -keyfile /path/to/certs

# Generate production certificates
sssonector -keygen

# Validate existing certificates
sssonector -validate-certs
```

## Configuration

### Performance Tuning

```yaml
buffer:
  read_size: 65536    # Optimized for large transfers
  write_size: 65536   # Matched with read size
  pool_size: 1024     # Increased for better performance

retry:
  max_attempts: 3     # Number of retry attempts
  base_delay_ms: 50   # Base delay between retries
  max_delay_ms: 1000  # Maximum backoff delay

throttle:
  enabled: true
  rate_mbps: 100      # Adjust based on needs
  burst_size: 1048576 # Optimized for bursty traffic
```

### Monitoring Setup

```yaml
monitor:
  enabled: true
  metrics:
    interval_seconds: 10
    detailed_logging: true
  snmp:
    enabled: true
    port: 161
    community: "public"
  error_tracking:
    enabled: true
    detailed: true
```

## Platform Support

### Linux
- Ubuntu 20.04+
- CentOS 7+
- RHEL 8+
- Full TUN/TAP support
- Systemd integration

### macOS
- macOS 10.15+
- Network Extension support
- Automatic permission handling

### Windows
- Windows 10
- Server 2016+
- TAP-Windows adapter support
- Windows service integration

## Development

### Prerequisites
- Go 1.21 or later
- TUN/TAP kernel module
- iproute2 package (Linux)
- Administrative privileges

### Building
```bash
# Build binary
make build

# Run tests
make test

# Generate documentation
make docs
```

### Testing
```bash
# Run unit tests
go test ./...

# Run integration tests
make test-integration

# Test certificate generation
./test/test_cert_generation.sh
```

## Documentation

- [Installation Guide](docs/installation.md)
- [Certificate Management](docs/certificate_management.md)
- [Project Overview](docs/project_context.md)
- [Code Structure](docs/code_structure_snapshot.md)
- Platform Guides:
  * [Linux](docs/linux_install.md)
  * [macOS](docs/macos_build.md)
  * [Windows](docs/windows_install.md)
  * [Ubuntu](docs/ubuntu_install.md)

## Recent Improvements

### v1.1.0
- Enhanced tunnel data transfer reliability
- Improved buffer management
- Added retry mechanisms
- Optimized chunked transfers
- Better error recovery
- Enhanced monitoring capabilities
- Improved certificate management
- Better cross-platform support

## Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- Documentation: https://docs.sssonector.io
- Issues: https://github.com/o3willard-AI/SSSonector/issues
- Community: https://community.sssonector.io
