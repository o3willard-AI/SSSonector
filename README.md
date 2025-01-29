# SSSonector

A lightweight, cross-platform SSL tunnel application for secure network connectivity between remote locations.

## Features

- Cross-platform support (Windows, Linux, macOS)
- TLS 1.3 with EU-exportable cipher suites
- Virtual network interface creation
- Persistent SSL tunnels
- Automatic reconnection
- Bandwidth throttling
- SNMP monitoring support
- Detailed logging and telemetry
- Client and server modes
- Connection statistics and monitoring

## Requirements

- Go 1.21 or later
- OpenSSL for certificate generation
- Root/Administrator privileges for virtual interface creation

## Installation

### From Source

1. Clone the repository:
```bash
git clone https://github.com/o3willard-AI/SSSonector.git
cd SSSonector
```

2. Build the project:
```bash
make
```

3. Install the application:
```bash
sudo make install
```

### Using Pre-built Packages

- Linux (DEB): `sudo dpkg -i sssonector_<version>_amd64.deb`
- Linux (RPM): `sudo rpm -i sssonector-<version>.x86_64.rpm`
- macOS: Install the provided .pkg file
- Windows: Run the provided installer .exe

## Configuration

Configuration files are stored in `/etc/sssonector/` (Linux/macOS) or `C:\Program Files\SSSonector\` (Windows).

### Server Mode

1. Edit `/etc/sssonector/config.yaml`:
```yaml
mode: "server"
network:
  interface: "tun0"
  address: "10.0.0.1/24"
  mtu: 1500
tunnel:
  cert_file: "/etc/sssonector/certs/server.crt"
  key_file: "/etc/sssonector/certs/server.key"
  ca_file: "/etc/sssonector/certs/client.crt"
  listen_address: "0.0.0.0"
  listen_port: 8443
```

2. Run the server:
```bash
sudo sssonector -config /etc/sssonector/config.yaml
```

### Client Mode

1. Edit `/etc/sssonector/config-client.yaml`:
```yaml
mode: "client"
network:
  interface: "tun0"
  address: "10.0.0.2/24"
  mtu: 1500
tunnel:
  cert_file: "/etc/sssonector/certs/client.crt"
  key_file: "/etc/sssonector/certs/client.key"
  ca_file: "/etc/sssonector/certs/server.crt"
  server_address: "server.example.com"
  server_port: 8443
```

2. Run the client:
```bash
sudo sssonector -config /etc/sssonector/config-client.yaml
```

## Certificate Management

Generate certificates for server and client:
```bash
make generate-certs
```

Certificates will be placed in the `certs` directory:
- `server.crt` and `server.key`: Server certificate and private key
- `client.crt` and `client.key`: Client certificate and private key

## Monitoring

### SNMP Monitoring

SNMP monitoring is available on port 161 by default. Configure your SNMP manager to connect using the community string specified in the configuration file.

Available metrics:
- Bytes sent/received
- Connection status
- Tunnel uptime
- Current bandwidth usage
- Number of connected clients (server mode)

### Logging

Logs are written to:
- Linux/macOS: `/var/log/sssonector/`
- Windows: `C:\Program Files\SSSonector\logs\`

## Security

- TLS 1.3 with EU-exportable cipher suites only
- Mutual certificate authentication
- Private IP space for virtual interfaces
- No inbound connections required at remote sites

## Building from Source

### All Platforms
```bash
make build-all
```

### Platform-Specific
```bash
make build-linux
make build-darwin
make build-windows
```

## License

MIT License - see LICENSE file for details
