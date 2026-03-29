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
- SNMP v2c monitoring
- Custom MIB implementation
- Real-time metrics collection
- System resource monitoring
- Prometheus integration
- Grafana dashboards

### Rate Limiting
- Token bucket algorithm
- Per-connection limits
- Global rate limiting
- Dynamic rate adjustment
- Burst allowance
- Fair queuing support

### Performance
- Optimized buffer management
- Connection pooling
- Async I/O operations
- Resource usage optimization
- Performance metrics tracking

## Quick Start

### Installation

#### One-line Install (Linux)
```bash
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/install.sh | sudo bash
```

The installer will:
1. Detect your OS and architecture
2. Download the latest release binary
3. Prompt for configuration (server/client mode, instance name, addresses)
4. Generate TLS certificates
5. Install systemd service template

#### Non-interactive Install
```bash
# Server mode
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/install.sh | \
  sudo SSSONECTOR_MODE=server \
       SSSONECTOR_INSTANCE=tunnel-a \
       SSSONECTOR_ADDRESS=10.0.1.1/24 \
       bash

# Client mode
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/install.sh | \
  sudo SSSONECTOR_MODE=client \
       SSSONECTOR_INSTANCE=tunnel-a \
       SSSONECTOR_ADDRESS=10.0.1.2/24 \
       SSSONECTOR_SERVER=your-server-ip \
       bash
```

#### Manual Download

| Platform | Architecture | Binary |
|----------|-------------|--------|
| Linux | amd64 | `sssonector-linux-amd64` |
| Linux | arm64 | `sssonector-linux-arm64` |
| macOS | amd64 | `sssonector-darwin-amd64` |
| macOS | arm64 | `sssonector-darwin-arm64` |
| Windows | amd64 | `sssonector-windows-amd64.exe` |

```bash
# Download from releases page
curl -LO https://github.com/o3willard-AI/SSSonector/releases/latest/download/sssonector-linux-amd64
chmod +x sssonector-linux-amd64
sudo mv sssonector-linux-amd64 /usr/local/bin/sssonector
```

#### Uninstall
```bash
# Remove binary and service only (keeps configs)
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/uninstall.sh | sudo bash

# Remove everything including configurations
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/uninstall.sh | sudo bash -s -- --purge
```

### Basic Configuration

#### Server Setup
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
monitor:
  enabled: true
  snmp_enabled: true
  snmp_port: 10161
throttle:
  enabled: true
  rate_limit: 10485760  # 10 MB/s
  burst_limit: 20971520 # 20 MB burst
```

#### Client Setup
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
  server_address: "server.example.com"
  server_port: 8443
monitor:
  enabled: true
  log_file: "/var/log/sssonector/client.log"
throttle:
  enabled: true
  rate_limit: 5242880   # 5 MB/s
  burst_limit: 10485760 # 10 MB burst
```

### Running the Service

SSSonector uses a multi-instance architecture - each tunnel connection runs as a separate systemd service.

#### Linux
```bash
# List instances
ls /etc/sssonector/instances/

# Start a specific instance
sudo systemctl start sssonector@<instance-name>

# View status
sudo systemctl status sssonector@<instance-name>

# View logs
journalctl -u sssonector@<instance-name> -f

# Enable at boot
sudo systemctl enable sssonector@<instance-name>
```

#### Managing Multiple Instances
```bash
# Create a new server instance
sudo /usr/local/bin/sssonector instance create-server \
  --name client-b \
  --address 10.0.2.1/24 \
  --port 8444

# Create a new client instance  
sudo /usr/local/bin/sssonector instance create-client \
  --name client-b \
  --address 10.0.2.2/24 \
  --server your-server-ip \
  --port 8444

# List all instances
sudo /usr/local/bin/sssonector instance list
```

#### macOS
```bash
# Start service
sudo launchctl load /Library/LaunchDaemons/com.o3willard.sssonector.plist

# View status
sudo launchctl list | grep sssonector

# View logs
tail -f /var/log/sssonector/service.log
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
- [Multi-Instance Deployment](docs/multi_instance_deployment.md)
- [Configuration Guide](docs/configuration_guide.md)
- Platform-specific guides:
  - [Linux Installation](docs/linux_install.md)
  - [macOS Installation](docs/macos_install.md)
- [SNMP Monitoring](docs/snmp_monitoring.md)
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
