# SSSonector

A secure, cross-platform SSL tunneling solution with comprehensive monitoring capabilities. This application creates persistent TLS 1.3 tunnels between remote locations, supporting both client and server modes with configurable bandwidth throttling and SNMP monitoring.

## Features

- **Cross-Platform Support**
  - Linux, macOS, and Windows compatibility
  - Platform-specific virtual network interfaces
  - Native service integration (systemd, launchd, Windows Services)

- **Security**
  - TLS 1.3 with EU-exportable cipher suites
  - Automatic certificate management
  - Certificate auto-renewal
  - Secure private key handling

- **Network Features**
  - Virtual network interface creation (TUN/TAP)
  - Configurable bandwidth throttling
  - QoS support
  - Connection persistence with auto-reconnect

- **Monitoring & Telemetry**
  - SNMP v2c support
  - Prometheus metrics
  - Grafana dashboards
  - Detailed logging
  - Connection statistics

## Quick Start

### Prerequisites

- Go 1.21 or later
- Root/Administrator privileges
- Docker and Docker Compose (for monitoring)

### Installation

1. Download the latest release for your platform:
   ```bash
   # Linux/macOS
   curl -LO https://github.com/yourusername/SSSonector/releases/latest/download/SSSonector-$(uname -s | tr '[:upper:]' '[:lower:]')-amd64.tar.gz
   tar xzf SSSonector-*.tar.gz
   cd SSSonector

   # Windows (PowerShell)
   Invoke-WebRequest -Uri "https://github.com/yourusername/SSSonector/releases/latest/download/SSSonector-windows-amd64.zip" -OutFile "SSSonector.zip"
   Expand-Archive SSSonector.zip
   cd SSSonector
   ```

2. Configure the application:
   ```bash
   # Server mode
   cp configs/server.yaml /etc/SSSonector/config.yaml
   vim /etc/SSSonector/config.yaml

   # Client mode
   cp configs/client.yaml /etc/SSSonector/config.yaml
   vim /etc/SSSonector/config.yaml
   ```

3. Install as a service:
   ```bash
   # Linux
   make install-linux

   # macOS
   make install-darwin

   # Windows (PowerShell as Administrator)
   make install-windows
   ```

### Running

```bash
# Server mode
sudo SSSonector --config /etc/SSSonector/config.yaml --mode server

# Client mode
sudo SSSonector --config /etc/SSSonector/config.yaml --mode client
```

### Monitoring Setup

1. Start the monitoring stack:
   ```bash
   cd monitoring
   docker-compose up -d
   ```

2. Access monitoring interfaces:
   - Grafana: http://localhost:3000 (admin/admin)
   - Prometheus: http://localhost:9090

## Development

### Building from Source

```bash
# Clone repository
git clone https://github.com/yourusername/SSSonector.git
cd SSSonector

# Build
make build

# Run tests
make test

# Create distribution packages
make dist
```

### Available Make Commands

- `make build` - Build the binary
- `make test` - Run tests
- `make dist` - Create distribution packages
- `make run-server` - Run in server mode
- `make run-client` - Run in client mode
- `make monitoring-up` - Start monitoring stack
- `make test-certs` - Test certificate generation
- `make test-snmp` - Test SNMP functionality

## Configuration

### Server Configuration

```yaml
mode: "server"
network:
  interface: "tun0"
  address: "10.0.0.1"
  listen_address: "0.0.0.0"
  listen_port: 5000
  max_clients: 5

tls:
  cert_file: "certs/server.crt"
  key_file: "certs/server.key"
  auto_generate: true
  validity_days: 90

throttle:
  enabled: true
  upload_kbps: 1024
  down_kbps: 1024

monitor:
  snmp_enabled: true
  snmp_port: 161
  community: "public"
```

### Client Configuration

```yaml
mode: "client"
network:
  interface: "tun0"
  address: "10.0.0.2"
  server_address: "tunnel.example.com"
  server_port: 5000
  retry_attempts: 10
  retry_interval: 30

tls:
  cert_file: "certs/client.crt"
  key_file: "certs/client.key"
  auto_generate: true
  validity_days: 90

throttle:
  enabled: true
  upload_kbps: 1024
  down_kbps: 1024
```

## Monitoring

### SNMP Metrics

- Bytes Received/Sent (Counter64)
- Packets Lost (Counter64)
- Latency (Integer32, microseconds)
- Connection Status
- CPU/Memory Usage
- Uptime

### Grafana Dashboard

The included Grafana dashboard provides:
- Real-time bandwidth graphs
- Latency monitoring
- Packet loss tracking
- System resource utilization
- Connection status

## Troubleshooting

See [QA Guide](docs/qa_guide.md) for detailed troubleshooting steps and common issues.

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

For issues and support, please visit:
https://github.com/yourusername/SSSonector/issues
