# SSSonector

A high-performance SSL tunneling application for secure network connectivity.

## Features

- Secure SSL/TLS encrypted tunneling
- Linux TUN device support
- Bandwidth throttling
- Connection management
- Real-time monitoring with SNMP support
- YAML-based configuration
- Cross-platform support (Linux, macOS*, Windows*)

*Platform support in development

## Quick Start

### Installation

#### From Packages

##### Debian/Ubuntu:
```bash
sudo dpkg -i sssonector_1.0.0_amd64.deb
```

##### RHEL/CentOS:
```bash
sudo rpm -i sssonector-1.0.0-1.x86_64.rpm
```

##### macOS:
```bash
sudo installer -pkg sssonector-1.0.0.pkg -target /
```

##### Windows:
Run the installer: `sssonector-1.0.0-setup.exe`

#### From Source

1. Install Go 1.21 or later
2. Clone the repository:
   ```bash
   git clone https://github.com/o3willard-AI/SSSonector.git
   cd SSSonector
   ```
3. Build and install:
   ```bash
   make
   sudo make install
   ```

### Configuration

1. Create configuration directory:
   ```bash
   sudo mkdir -p /etc/sssonector/certs
   ```

2. Copy sample configurations:
   ```bash
   sudo cp configs/server.yaml /etc/sssonector/config.yaml  # For server
   sudo cp configs/client.yaml /etc/sssonector/config.yaml  # For client
   ```

3. Generate SSL certificates:
   ```bash
   openssl req -x509 -newkey rsa:4096 -keyout /etc/sssonector/certs/server.key \
     -out /etc/sssonector/certs/server.crt -days 365 -nodes
   ```

### Usage

#### Server Mode
```bash
sudo sssonector -config /etc/sssonector/config.yaml
```

#### Client Mode
```bash
sudo sssonector -config /etc/sssonector/config.yaml
```

## Documentation

- [Installation Guide](docs/installation.md)
- [Configuration Guide](docs/configuration.md)
- [API Documentation](docs/api.md)
- [Troubleshooting](docs/troubleshooting.md)

## Development

### Requirements

- Go 1.21+
- Make
- OpenSSL
- GCC

### Building

```bash
# Build binary
make build

# Build packages
make dist

# Run tests
make test

# Install locally
sudo make install
```

### Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- The Go team for the excellent networking libraries
- OpenSSL for the cryptographic foundations
- The Linux kernel team for the TUN/TAP implementation
