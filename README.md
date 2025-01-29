# SSSonector - Secure Scalable SSL Connector

SSSonector is a cross-platform SSL tunneling application that creates secure connections between remote networks. It supports both client and server modes, with features like bandwidth throttling, SNMP monitoring, and automatic reconnection.

## Features

- TLS 1.3 with EU-exportable cipher suites
- Cross-platform support (Windows, Linux, macOS)
- Bandwidth throttling
- SNMP monitoring
- Automatic reconnection
- Certificate-based authentication
- Virtual network interface creation
- Persistent SSL tunnels
- Connection monitoring and logging

## Installation

### Platform-Specific Installation Guides

For detailed installation instructions, please refer to the appropriate guide for your platform:

- [Ubuntu Installation Guide](docs/ubuntu_install.md)
- [Red Hat/Rocky Linux Installation Guide](docs/linux_install.md)
- [macOS Installation Guide](docs/macos_install.md)
- [Windows Installation Guide](docs/windows_install.md)

### Quick Start

#### Linux (Ubuntu/Debian)
```bash
# Download the latest release
wget https://github.com/o3willard-AI/SSSonector/releases/latest/download/sssonector_amd64.deb

# Install the package
sudo dpkg -i sssonector_amd64.deb
sudo apt-get install -f  # Install dependencies if needed
```

#### Linux (RHEL/Rocky)
```bash
# Download the latest release
wget https://github.com/o3willard-AI/SSSonector/releases/latest/download/sssonector.el8.x86_64.rpm

# Install the package
sudo dnf install sssonector.el8.x86_64.rpm
```

#### macOS
```bash
# Download the latest release
curl -LO https://github.com/o3willard-AI/SSSonector/releases/latest/download/SSSonector.pkg

# Install the package
sudo installer -pkg SSSonector.pkg -target /
```

#### Windows
1. Download the latest release from [GitHub Releases](https://github.com/o3willard-AI/SSSonector/releases/latest)
2. Run the installer as administrator
3. Follow the installation wizard

### Building from Source

Requirements:
- Go 1.21 or later
- Make
- OpenSSL (for certificate generation)
- Platform-specific build tools (see installation guides)

```bash
# Clone the repository
git clone https://github.com/o3willard-AI/SSSonector.git
cd SSSonector

# Build and install
make
sudo make install
```

## Basic Configuration

Configuration files are stored in:
- Linux: `/etc/sssonector/`
- macOS: `/etc/sssonector/`
- Windows: `C:\ProgramData\SSSonector\`

For detailed configuration examples and use cases, see the platform-specific installation guides.

## Service Management

### Linux (systemd)
```bash
sudo systemctl start sssonector    # Start service
sudo systemctl enable sssonector   # Enable at boot
sudo systemctl status sssonector   # Check status
```

### macOS (launchd)
```bash
sudo launchctl load /Library/LaunchDaemons/com.o3willard.sssonector.plist    # Start
sudo launchctl unload /Library/LaunchDaemons/com.o3willard.sssonector.plist  # Stop
```

### Windows
```powershell
Start-Service SSSonector    # Start service
Stop-Service SSSonector     # Stop service
Get-Service SSSonector     # Check status
```

## Documentation

- [Ubuntu Installation Guide](docs/ubuntu_install.md) - Ubuntu-specific installation and configuration
- [Linux Installation Guide](docs/linux_install.md) - Red Hat/Rocky Linux installation and configuration
- [macOS Installation Guide](docs/macos_install.md) - macOS-specific installation and configuration
- [Windows Installation Guide](docs/windows_install.md) - Windows-specific installation and configuration
- [QA Guide](docs/qa_guide.md) - Testing procedures and troubleshooting

## Development

### Building Installers
```bash
# Install dependencies
make installer-deps

# Build installers for all platforms
make installers
```

### Running Tests
```bash
make test
```

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

MIT License - see LICENSE file for details
