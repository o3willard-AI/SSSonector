# SSSonector

SSSonector (**Secure SSL Connector**) is a high-performance, secure tunnel service designed to enable point-to-point communication over the public internet. Tunnel traffic is indistinguishable from normal HTTPS web traffic, allowing it to traverse firewalls and network infrastructure that only permits standard web browsing.

## Features

### Core Features
- Secure TLS-based tunneling with mutual certificate authentication
- **HTTPS Facade** for firewall traversal -- tunnel traffic disguised as standard HTTPS/WebSocket on port 443
- Cross-platform support (Linux, Windows, macOS)
- High-performance data transfer with optimized buffer management
- Certificate-based authentication
- Configurable MTU and buffer sizes

### HTTPS Facade (Firewall Traversal)
- Tunnel traffic appears identical to normal HTTPS to firewalls and DPI systems
- Server runs a legitimate HTTPS web server on port 443 (returns a real web page to browsers)
- Tunnel connections use standard HTTP/1.1 WebSocket upgrade protocol (RFC 6455)
- HMAC-SHA256 signed tokens prevent unauthorized tunnel establishment
- Client automatically falls back to facade when direct ports are blocked
- Zero port enumeration -- unauthorized probes see only a normal website
- No double encryption -- facade TLS layer is reused by the tunnel
- Supports multiple tunnel instances behind a single facade on port 443

### Multi-Instance Architecture
- Run multiple independent tunnel instances on a single host
- One instance per client-server pairing (point-to-point)
- Isolated configuration and certificates per instance
- Independent systemd services per instance

### Platform Support
- Linux: Full TUN interface support (primary platform)
- macOS: Basic support (future TUN implementation)
- Windows: Basic support (TAP adapter)

### Monitoring
- SNMP v2c monitoring
- Custom MIB implementation
- Real-time metrics collection
- System resource monitoring
- Prometheus integration per instance
- Grafana dashboards

### Rate Limiting
- Token bucket algorithm
- Per-connection limits
- Global rate limiting
- Dynamic rate adjustment
- Burst allowance
- Fair queuing support

## Quick Start

### Installation

```bash
# One-line install (Linux)
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/install.sh | sudo bash
```

The installer will:
1. Detect your OS and architecture
2. Download the latest release binary
3. Prompt for configuration (server/client mode, instance name, addresses)
4. Generate TLS certificates
5. Install systemd service template

#### Install Options

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `SSSONECTOR_VERSION` | Version to install | latest |
| `SSSONECTOR_MODE` | "server" or "client" | interactive |
| `SSSONECTOR_INSTANCE` | Instance name | interactive |
| `SSSONECTOR_ADDRESS` | TUN interface address (CIDR) | interactive |
| `SSSONECTOR_SERVER` | Server address (client mode) | interactive |
| `SSSONECTOR_PORT` | Listen/connect port | auto-assigned |
| `SSSONECTOR_NO_SERVICE` | Don't install systemd service | false |

```bash
# Non-interactive server install
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/install.sh | \
  sudo SSSONECTOR_MODE=server \
       SSSONECTOR_INSTANCE=tunnel-a \
       SSSONECTOR_ADDRESS=10.0.1.1/24 \
       bash

# Non-interactive client install
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
curl -LO https://github.com/o3willard-AI/SSSonector/releases/latest/download/sssonector-linux-amd64
chmod +x sssonector-linux-amd64
sudo mv sssonector-linux-amd64 /usr/local/bin/sssonector
```

## Lifecycle Management

### Upgrade

```bash
# Upgrade to latest version (restarts running instances)
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/upgrade.sh | sudo bash

# Upgrade to specific version
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/upgrade.sh | \
  sudo bash -s -- --version v1.2.0

# Upgrade without restarting services
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/upgrade.sh | \
  sudo bash -s -- --no-restart

# Non-interactive upgrade
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/upgrade.sh | \
  sudo bash -s -- -y
```

Upgrade process:
1. Checks current installed version
2. Downloads new binary from GitHub releases
3. Stops running instances
4. Replaces the binary
5. Restarts previously running instances

### Uninstall

```bash
# Remove binary and systemd service (preserves instances)
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/uninstall.sh | sudo bash

# Remove everything (binary, service, configs, logs, certificates)
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/uninstall.sh | \
  sudo bash -s -- --purge

# Remove all instances but keep binary
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/uninstall.sh | \
  sudo bash -s -- --all

# Non-interactive uninstall
curl -fsSL https://raw.githubusercontent.com/o3willard-AI/SSSonector/main/uninstall.sh | \
  sudo bash -s -- -y --purge
```

Uninstall options:

| Option | Description |
|--------|-------------|
| (default) | Removes binary and systemd service template, preserves instance configs |
| `--purge` | Removes everything: binary, service, configs, logs, certificates |
| `--all` | Removes all instance configurations |
| `-y, --yes` | Skip confirmation prompt |

### Instance Management

SSSonector supports multiple independent tunnel instances per host:

```bash
# List all instances
ls /etc/sssonector/instances/

# Start a specific instance
sudo systemctl start sssonector@<instance-name>

# Stop an instance
sudo systemctl stop sssonector@<instance-name>

# View logs
journalctl -u sssonector@<instance-name> -f

# Enable at boot
sudo systemctl enable sssonector@<instance-name>

# Remove an instance
sudo rm -rf /etc/sssonector/instances/<instance-name>
```

## Configuration

### Instance Directory Structure

```
/etc/sssonector/
├── instances/
│   ├── client-a/
│   │   ├── config.yaml      # Instance configuration
│   │   └── certs/
│   │       ├── ca.crt       # Certificate Authority
│   │       ├── ca.key       # CA private key
│   │       ├── server.crt   # Server certificate
│   │       ├── server.key   # Server private key
│   │       ├── client.crt   # Client certificate
│   │       └── client.key   # Client private key
│   └── client-b/
│       ├── config.yaml
│       └── certs/
└── templates/               # Configuration templates
```

### Server Configuration

```yaml
type: server
config:
  mode: server
  network:
    name: tun1
    interface: tun1
    address: 10.0.1.1/24
    mtu: 1500
  tunnel:
    listen_address: 0.0.0.0
    listen_port: 8443
    protocol: tcp
  auth:
    cert_file: /etc/sssonector/instances/client-a/certs/server.crt
    key_file: /etc/sssonector/instances/client-a/certs/server.key
    ca_file: /etc/sssonector/instances/client-a/certs/ca.crt
  security:
    tls:
      min_version: "1.2"
      max_version: "1.3"
    auth_method: certificate
  # HTTPS Facade -- enable to allow clients to connect via port 443
  facade:
    enabled: true
    listen_address: 0.0.0.0
    listen_port: 443
    token_ttl: 30s
    tunnel_ports:
      - 8443
  monitor:
    enabled: true
    type: prometheus
    prometheus:
      enabled: true
      port: 9090
      path: /metrics
throttle:
  enabled: true
  rate: 1048576    # ~1.1 MB/s effective after TCP overhead adjustment
  burst: 2097152   # accepted but unused: limiter derives burst as 100ms of effective rate
```

### Client Configuration

```yaml
type: client
config:
  mode: client
  network:
    name: tun1
    interface: tun1
    address: 10.0.1.2/24
    mtu: 1500
  tunnel:
    server_address: 192.168.1.10
    server_port: 8443
    protocol: tcp
  auth:
    cert_file: /etc/sssonector/instances/client-a/certs/client.crt
    key_file: /etc/sssonector/instances/client-a/certs/client.key
    ca_file: /etc/sssonector/instances/client-a/certs/ca.crt
  security:
    tls:
      min_version: "1.2"
      max_version: "1.3"
    auth_method: certificate
  # HTTPS Facade -- enable to fall back to port 443 when direct port is blocked
  facade:
    enabled: true
    server_port: 443
    direct_timeout: 3s
throttle:
  enabled: true
  rate: 1048576    # ~1.1 MB/s effective after TCP overhead adjustment
  burst: 2097152   # accepted but unused: limiter derives burst as 100ms of effective rate
```

## Multi-Instance Deployment

SSSonector uses a point-to-point architecture: one instance per client-server pairing.

### Direct Connection (ports open)

```
┌─────────────────────────────────────────────────────────────────┐
│                        SERVER (192.168.1.10)                     │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐           │
│  │  Instance    │  │  Instance    │  │  Instance    │           │
│  │  client-a    │  │  client-b    │  │  client-c    │           │
│  │  tun1        │  │  tun2        │  │  tun3        │           │
│  │  10.0.1.1    │  │  10.0.2.1    │  │  10.0.3.1    │           │
│  │  port 8443   │  │  port 8444   │  │  port 8445   │           │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘           │
└─────────┼────────────────┼────────────────┼────────────────────┘
          │                │                │
          │ TLS            │ TLS            │ TLS
          ▼                ▼                ▼
     ┌─────────┐      ┌─────────┐      ┌─────────┐
     │Client A │      │Client B │      │Client C │
     │10.0.1.2 │      │10.0.2.2 │      │10.0.3.2 │
     └─────────┘      └─────────┘      └─────────┘
```

### With HTTPS Facade (firewall only allows port 443)

```
┌─────────────────────────────────────────────────────────────────┐
│                        SERVER (192.168.1.10)                     │
│                                                                  │
│  ┌─────────────────────────────────────────────────────────┐     │
│  │              HTTPS Facade (:443)                         │     │
│  │  GET / → "Hello, World" web page                         │     │
│  │  WebSocket Upgrade + HMAC Token → proxy to tunnel port   │     │
│  └──────┬────────────────┬────────────────┬────────────────┘     │
│         │ proxy           │ proxy           │ proxy               │
│  ┌──────▼───────┐  ┌──────▼───────┐  ┌──────▼───────┐           │
│  │  Instance    │  │  Instance    │  │  Instance    │           │
│  │  client-a    │  │  client-b    │  │  client-c    │           │
│  │  port 8443   │  │  port 8444   │  │  port 8445   │           │
│  └──────────────┘  └──────────────┘  └──────────────┘           │
└─────────────────────────────────────────────────────────────────┘
          ▲                ▲                ▲
          │ HTTPS (:443)   │ HTTPS (:443)   │ HTTPS (:443)
          │                │                │
     ┌─────────┐      ┌─────────┐      ┌─────────┐
     │Client A │      │Client B │      │Client C │
     │(behind  │      │(behind  │      │(behind  │
     │firewall)│      │firewall)│      │firewall)│
     └─────────┘      └─────────┘      └─────────┘
```

Clients automatically try the direct port first, then fall back to the facade.

See [Multi-Instance Deployment Guide](docs/multi_instance_deployment.md) for details.

## Documentation

- [Installation Guide](docs/installation.md)
- [Multi-Instance Deployment](docs/multi_instance_deployment.md)
- [Configuration Guide](docs/configuration_guide.md)
- [HTTPS Facade Design](docs/implementation/https_facade.md)
- [Certificate Management](docs/certificate_management.md)
- Platform-specific guides:
  - [Linux Installation](docs/linux_install.md)
  - [macOS Installation](docs/macos_install.md)
- [SNMP Monitoring](docs/snmp_monitoring.md)
- [Rate Limiting Implementation](docs/rate_limiting_implementation.md)
- [Architecture](docs/implementation/ARCHITECTURE.md)
- [Release Notes](docs/RELEASE_NOTES.md)

## Building from Source

### Prerequisites
- Go 1.21 or later
- Make
- GCC (Linux/macOS)

### Build Steps
```bash
git clone https://github.com/o3willard-AI/SSSonector.git
cd SSSonector
make build
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

- Issues: https://github.com/o3willard-AI/SSSonector/issues
