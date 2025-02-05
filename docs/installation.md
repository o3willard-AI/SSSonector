# SSSonector Installation Guide

This guide covers the installation and initial setup of SSSonector.

## Prerequisites

- Linux, Windows, or macOS
- Administrator/root access for installation
- Go 1.22 or later (for building from source)

## Installation Methods

### 1. Pre-built Packages

#### Debian/Ubuntu
```bash
# Download the package
wget https://github.com/o3willard-AI/SSSonector/releases/download/v1.1.0/sssonector_1.0.0_amd64.deb

# Install the package
sudo dpkg -i sssonector_1.0.0_amd64.deb
```

#### Windows
1. Download `sssonector-1.0.0-setup.exe` from the [releases page](https://github.com/o3willard-AI/SSSonector/releases)
2. Run the installer
3. Follow the installation wizard

#### Source Archives
- Download `sssonector-1.0.0.tar.gz` or `sssonector-1.0.0.zip`
- Extract and follow the build instructions below

## Initial Setup

### 1. Certificate Generation
SSSonector now includes built-in certificate generation:

```bash
# Generate certificates in the current directory
sssonector -keygen

# Generate certificates in a specific directory
sssonector -keygen -keyfile /path/to/certs
```

This will create:
- `ca.crt`: CA certificate
- `ca.key`: CA private key
- `server.crt`: Server certificate
- `server.key`: Server private key
- `client.crt`: Client certificate
- `client.key`: Client private key

### 2. Configuration

Create configuration files in `/etc/sssonector/`:

#### Server Configuration
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
  max_clients: 10
```

#### Client Configuration
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
```

### 3. Testing the Installation

SSSonector includes a test mode for quick connectivity verification:

```bash
# On the server
sssonector -mode server -test-without-certs

# On the client
sssonector -mode client -test-without-certs
```

This creates temporary 15-second certificates for testing.

## Building from Source

1. Clone the repository:
```bash
git clone https://github.com/o3willard-AI/SSSonector.git
cd SSSonector
```

2. Build the project:
```bash
make
```

3. Install:
```bash
sudo make install
```

## Troubleshooting

### Certificate Issues
- Use `-keyfile` to specify certificate location
- Check file permissions (private keys should be 600)
- Use test mode (`-test-without-certs`) to verify basic connectivity
- See [Certificate Management](certificate_management.md) for detailed guidance

### Network Issues
- Verify server is accessible
- Check firewall rules
- Ensure TUN/TAP interface is available
- Verify port 8443 is open

## Next Steps

- Review [Certificate Management](certificate_management.md) for detailed certificate handling
- Configure monitoring (optional)
- Set up rate limiting (optional)
- Configure automatic startup

## Support

For issues and questions:
- GitHub Issues: [SSSonector Issues](https://github.com/o3willard-AI/SSSonector/issues)
- Documentation: See [docs/](../docs/) directory
