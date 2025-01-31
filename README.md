# SSSonector

SSSonector is a secure SSL tunnel service designed for remote office connectivity. It creates persistent TLS 1.3 tunnels with EU-exportable cipher suites enabling secure communication between remote locations without requiring inbound firewall rules.

## Features

- TLS 1.3 with EU-exportable cipher suites
- Thread-safe virtual network interfaces for all platforms
- Bandwidth throttling with upload/download controls
- Separate server and client binaries
- Persistent tunnel connections with automatic reconnection
- Cross-platform support:
  - Linux: Native TUN with ICMP handling
  - Windows: TAP with mutex protection
  - macOS: UTUN with proper byte handling
- Comprehensive logging and monitoring
- Systemd/Launchd/Windows Service integration
- Certificate management and rotation

## Requirements

### Server
- Linux/Windows/macOS
- Root/Administrator privileges
- Network access (outbound port 8443 by default)
- 100MB RAM minimum
- 50MB disk space

### Client
- Linux/Windows/macOS
- Root/Administrator privileges
- Outbound network access
- 50MB RAM minimum
- 20MB disk space

## Quick Start

### Installation

⚠️ **Important:** All installer packages are distributed through [GitHub Releases](https://github.com/o3willard-AI/SSSonector/releases/tag/v1.0.0). Always use the GitHub Releases URLs for downloading packages.

#### Linux (Debian/Ubuntu)
```bash
# Download the latest .deb package
wget https://github.com/o3willard-AI/SSSonector/releases/download/v1.0.0/sssonector_1.0.0_amd64.deb

# Install the package
sudo dpkg -i sssonector_1.0.0_amd64.deb
sudo apt-get install -f  # Install any missing dependencies
```

#### Windows
1. Download `sssonector-1.0.0-setup.exe` from the [releases page](https://github.com/o3willard-AI/SSSonector/releases/tag/v1.0.0)
2. Run the installer with administrator privileges
3. Follow the installation wizard

#### macOS
```bash
# Download the package
curl -LO https://github.com/o3willard-AI/SSSonector/releases/download/v1.0.0/sssonector-1.0.0-macos.pkg

# Install the package
sudo installer -pkg sssonector-1.0.0-macos.pkg -target /
```

### Configuration

1. Generate certificates (if not using existing ones):
```bash
sudo sssonector-cli generate-certs
```

2. Edit the configuration file:
- Server: `/etc/sssonector/config.yaml`
- Client: `/etc/sssonector/client.yaml`

3. Start the service:
```bash
# Linux
sudo systemctl start sssonector

# macOS
sudo launchctl load /Library/LaunchDaemons/com.o3willard.sssonector.plist

# Windows
net start SSSonector
```

## Configuration

### Server Mode
```yaml
mode: "server"
network:
  interface: "tun0"  # or "utun0" on macOS
  address: "10.0.0.1/24"
  mtu: 1500
tunnel:
  cert_file: "/etc/sssonector/certs/server.crt"
  key_file: "/etc/sssonector/certs/server.key"
  ca_file: "/etc/sssonector/certs/client.crt"
  listen_address: "0.0.0.0"
  listen_port: 8443
  # Bandwidth control (optional)
  upload_kbps: 10240    # 10 Mbps
  download_kbps: 10240  # 10 Mbps
```

### Client Mode
```yaml
mode: "client"
network:
  interface: "tun0"  # or "utun0" on macOS
  address: "10.0.0.2/24"
  mtu: 1500
tunnel:
  cert_file: "/etc/sssonector/certs/client.crt"
  key_file: "/etc/sssonector/certs/client.key"
  ca_file: "/etc/sssonector/certs/server.crt"
  server_address: "SERVER_IP"
  server_port: 8443
  # Bandwidth control (optional)
  upload_kbps: 10240    # 10 Mbps
  download_kbps: 10240  # 10 Mbps
```

## Building from Source

### Prerequisites
- Go 1.21 or later
- Make
- GCC
- OpenSSL development libraries

### Build Steps
```bash
# Clone the repository
git clone https://github.com/o3willard-AI/SSSonector.git
cd SSSonector

# Install dependencies
make deps

# Build the project
make build

# Create packages
make dist
```

## Monitoring

### SNMP Metrics
- Tunnel status
- Connection uptime
- Bandwidth usage
- Packet loss
- Latency
- Error counts

### Log Files
- Linux: `/var/log/sssonector/sssonector.log`
- macOS: `/var/log/sssonector/sssonector.log`
- Windows: `C:\ProgramData\SSSonector\logs\sssonector.log`

## Troubleshooting

### Common Issues

1. Connection Failures
- Check firewall rules
- Verify certificate permissions
- Ensure correct IP addresses in configs

2. Performance Issues
- Check MTU settings
- Verify bandwidth throttling configuration
- Monitor system resources

3. Certificate Problems
- Verify certificate dates
- Check certificate permissions
- Ensure proper CA chain

### Debug Mode
```bash
sudo sssonector -config /etc/sssonector/config.yaml -debug
```

## Security

- TLS 1.3 only
- EU-exportable cipher suites
- Perfect Forward Secrecy
- Certificate-based authentication
- Regular security updates

## License

MIT License - see LICENSE file for details

## Contributing

1. Fork the repository
2. Create a feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

For detailed contribution guidelines, see [CONTRIBUTING.md](CONTRIBUTING.md)

## Support

- GitHub Issues: Bug reports and feature requests
- Documentation: [docs/](docs/)
- Email: support@o3willard.com
