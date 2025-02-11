# SSSonector

SSSonector is a high-performance, secure tunnel service with advanced monitoring and rate limiting capabilities.

## Features

### Core Features
- Secure TLS-based tunneling
- Cross-platform support (Linux, Windows, macOS)
- High-performance data transfer
- Certificate-based authentication
- Configurable MTU and buffer sizes

### Platform Support
- Linux: Full TUN interface support
- Windows: Basic support (TAP adapter)
- macOS: Basic support (future TUN implementation)

### Monitoring
- Basic and advanced monitoring modes
- Real-time metrics collection
- System resource monitoring
- Prometheus integration
- Grafana dashboards

### Security
- TLS 1.2/1.3 support
- Strong cipher suites
- Memory protection features
- Namespace isolation
- Capability restrictions

### Performance
- Optimized buffer management
- Connection pooling
- Async I/O operations
- Resource usage optimization
- Performance metrics tracking

## Quick Start

### Installation

#### Linux
```bash
# Download latest release
wget https://github.com/o3willard-AI/SSSonector/releases/download/v2.0.0/sssonector_2.0.0_linux_amd64
chmod +x sssonector_2.0.0_linux_amd64
sudo mv sssonector_2.0.0_linux_amd64 /usr/local/bin/sssonector
```

#### Windows
```powershell
# Download and install
Invoke-WebRequest -Uri "https://github.com/o3willard-AI/SSSonector/releases/download/v2.0.0/sssonector_2.0.0_windows_amd64.exe" -OutFile "sssonector.exe"
```

#### macOS
```bash
# Intel Mac
curl -LO https://github.com/o3willard-AI/SSSonector/releases/download/v2.0.0/sssonector_2.0.0_darwin_amd64

# Apple Silicon
curl -LO https://github.com/o3willard-AI/SSSonector/releases/download/v2.0.0/sssonector_2.0.0_darwin_arm64

chmod +x sssonector_2.0.0_darwin_*
sudo mv sssonector_2.0.0_darwin_* /usr/local/bin/sssonector
```

### Basic Configuration

#### Server Setup
```yaml
type: server
config:
  mode: server
  network:
    name: tun0
    interface: tun0
    address: 10.0.0.1/24
    mtu: 1500
  tunnel:
    listen_port: 8443
    protocol: tcp
    cert_file: /etc/sssonector/certs/server.crt
    key_file: /etc/sssonector/certs/server.key
    ca_file: /etc/sssonector/certs/ca.crt
    listen_address: 0.0.0.0
    max_clients: 10
  security:
    memory_protections:
      enabled: true
    namespace:
      enabled: true
    capabilities:
      enabled: true
    tls:
      min_version: "1.2"
      max_version: "1.3"
      ciphers:
        - TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
        - TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
  monitor:
    enabled: true
    type: basic
    interval: 1s
  metrics:
    enabled: true
    address: 127.0.0.1:9091
    interval: 5s
    buffer_size: 1000
version: 1.0.0
metadata:
  version: 1.0.0
  environment: development
  region: local
throttle:
  enabled: false
  rate: 1048576
  burst: 1048576
```

#### Client Setup
```yaml
type: client
config:
  mode: client
  network:
    name: tun0
    interface: tun0
    address: 10.0.0.2/24
    mtu: 1500
  tunnel:
    server_port: 8443
    protocol: tcp
    cert_file: /etc/sssonector/certs/client.crt
    key_file: /etc/sssonector/certs/client.key
    ca_file: /etc/sssonector/certs/ca.crt
    server_address: 192.168.50.210
  security:
    memory_protections:
      enabled: true
    namespace:
      enabled: true
    capabilities:
      enabled: true
    tls:
      min_version: "1.2"
      max_version: "1.3"
      ciphers:
        - TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384
        - TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
  monitor:
    enabled: true
    type: basic
    interval: 1s
  metrics:
    enabled: true
    address: 127.0.0.1:9091
    interval: 5s
    buffer_size: 1000
version: 1.0.0
metadata:
  version: 1.0.0
  environment: development
  region: local
throttle:
  enabled: false
  rate: 1048576
  burst: 1048576
```

### Running the Service

#### Linux/macOS
```bash
# Start service
sudo systemctl start sssonector  # Linux
sudo launchctl load /Library/LaunchDaemons/com.sssonector.plist  # macOS

# View status
sudo systemctl status sssonector  # Linux
sudo launchctl list | grep sssonector  # macOS

# View logs
journalctl -u sssonector -f  # Linux
tail -f /var/log/sssonector/output.log  # macOS
```

#### Windows
```powershell
# Start service
Start-Service SSonector

# View status
Get-Service SSonector

# View logs
Get-EventLog -LogName Application -Source "SSonector"
```

## Documentation

- [Installation Guide](docs/installation.md)
- [Configuration Guide](docs/configuration_guide.md)
- Platform-specific guides:
  - [Linux Installation](docs/linux_install.md)
  - [Windows Installation](docs/windows_install.md)
  - [macOS Installation](docs/macos_install.md)
- [Monitoring Guide](docs/monitoring_guide.md)
- [Rate Limiting Implementation](docs/rate_limiting_implementation.md)
- [Release Notes](docs/RELEASE_NOTES.md)

## Building from Source

### Prerequisites
- Go 1.21 or later
- Make
- GCC (Linux/macOS)
- Visual Studio Build Tools (Windows)

### Build Steps
```bash
# Clone repository
git clone https://github.com/o3willard-AI/SSSonector.git
cd SSSonector

# Build
make build

# Install
sudo make install
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- Documentation: https://docs.sssonector.io
- Issues: https://github.com/o3willard-AI/SSSonector/issues
- Community: https://community.sssonector.io
- Security: https://security.sssonector.io
