# SSSonector

SSSonector is a high-performance, enterprise-grade communications utility designed to allow critical services to connect to and exchange data with one another over the public internet without needing a VPN.

## Overview

SSSonector creates secure tunnels between services using TLS for authentication and encryption. It operates as a standalone binary that can be run in either server or client mode, determined by its configuration file.

## Installation

SSSonector is distributed as a single binary that should be placed alongside its configuration file. No system-wide installation is required.

1. Download the latest release
2. Place the binary in your desired location
3. Create a configuration file (see Configuration section)
4. Generate certificates (first-time setup only)

## Usage

### First Time Setup

1. Generate certificates on the server:
```bash
./sssonector --generate-certs \
    --cert-dir /path/to/certs \
    --server-ip <server_ip>
```

2. Copy the client certificate and CA certificate to the client system

### Running SSSonector

Server mode:
```bash
./sssonector -config server_config.yaml
```

Client mode:
```bash
./sssonector -config client_config.yaml
```

## Configuration

SSSonector uses YAML configuration files. Example configurations:

### Server Configuration
```yaml
mode: server
listen: 0.0.0.0:443
interface: tun0
address: 10.0.0.1/24
network:
  mtu: 1500
logging:
  level: info
  file: /var/log/sssonector.log
monitoring:
  enabled: true
  port: 9090
security:
  tls:
    enabled: true
    min_version: "1.2"
    cert_file: certs/server.crt
    key_file: certs/server.key
    ca_file: certs/ca.crt
```

### Client Configuration
```yaml
mode: client
server: <server_ip>:443
interface: tun0
address: 10.0.0.2/24
network:
  mtu: 1500
logging:
  level: info
  file: /var/log/sssonector.log
security:
  tls:
    enabled: true
    min_version: "1.2"
    cert_file: certs/client.crt
    key_file: certs/client.key
    ca_file: certs/ca.crt
```

### Network Configuration
```yaml
network:
  mtu: 1500  # Optional, default is 1500
```

### Logging Configuration
```yaml
logging:
  level: info  # Optional, values: debug, info, warning, error, default is info
  file: /var/log/sssonector.log  # Optional, default is stdout
```

### Monitoring Configuration
```yaml
monitoring:
  enabled: true  # Optional, default is false
  port: 9090  # Optional, default is 9090
```

## Testing

SSSonector includes a comprehensive test framework designed to validate functionality, performance, and security. The test suite follows SSSonector's deployment model where the binary is run directly with its configuration file.

### Test Framework Structure

```
test/
├── configs/              # Test configurations
│   ├── certs/           # Generated certificates
│   ├── server.yaml      # Server configuration
│   └── client.yaml      # Client configuration
├── lib/                 # Common test utilities
│   ├── common.sh        # Shared functions
│   └── process_utils.sh # Process management utilities
├── logs/                # Test logs
├── scenarios/           # Test scenarios
│   ├── 01_cert_generation/     # Certificate generation tests
│   ├── 02_basic_connectivity/  # Basic connectivity tests
│   ├── 03_performance/        # Performance tests
│   └── 04_security/          # Security tests
└── run_tests.sh        # Main test runner
```

### Running Tests

```bash
cd test
./run_tests.sh -s <server_ip> -c <client_ip>
```

The test suite will:
1. Generate and validate certificates
2. Test basic connectivity
3. Run performance measurements
4. Verify security features

Test results are stored in `test/results/YYYYMMDD_HHMMSS/` with detailed logs and a summary report.

See [test/README.md](test/README.md) for detailed testing documentation.

## Packet Forwarding

SSSonector forwards packets between the TUN interface and the TCP connection using a bidirectional copy mechanism. This allows for transparent communication between services on either end of the tunnel.

The packet forwarding process works as follows:

1. Packets sent to the TUN interface on the client are forwarded to the server over the secure TLS connection.
2. The server receives these packets and forwards them to its TUN interface.
3. Packets sent to the TUN interface on the server are forwarded to the client over the secure TLS connection.
4. The client receives these packets and forwards them to its TUN interface.

This bidirectional forwarding allows services on either end of the tunnel to communicate as if they were on the same network.

## Security Features

- **TLS 1.2+ for authentication and encryption**: SSSonector uses TLS 1.2 or higher for secure communication between server and client. The minimum TLS version can be configured using the `security.tls.min_version` option.
- **Certificate-based mutual authentication**: Both server and client authenticate each other using certificates. Certificates can be generated using the `--generate-certs` option.
- **Process isolation through namespaces**: SSSonector uses Linux namespaces to isolate the process from the rest of the system, providing an additional layer of security.
- **Memory protections (ASLR, NX)**: SSSonector enables Address Space Layout Randomization (ASLR) and No-Execute (NX) memory protections to prevent memory-based attacks.
- **Network isolation**: SSSonector isolates network traffic to the TUN interface, preventing unauthorized access to the tunnel.
- **Minimal process capabilities**: SSSonector runs with minimal process capabilities, reducing the attack surface.

## Error Handling

SSSonector includes robust error handling to ensure reliable operation:

- **Configuration errors**: Invalid configuration options are detected and reported with clear error messages.
- **Network errors**: Network-related errors, such as connection failures or packet transmission errors, are handled gracefully with automatic retry mechanisms.
- **Certificate errors**: Certificate validation failures are reported with detailed error messages to help diagnose the issue.
- **Resource errors**: Resource-related errors, such as insufficient permissions or resource exhaustion, are detected and reported.

For detailed error messages, enable debug logging by setting `logging.level: debug` in the configuration file.

## Performance

- Low latency packet forwarding
- High throughput capacity
- Minimal CPU and memory footprint
- Efficient resource cleanup
- Configurable MTU for optimal performance

## Requirements

SSSonector has specific system and network requirements to operate correctly:

- Linux system with TUN module support
- OpenSSL for certificate operations
- Root/sudo access for network operations
- IP forwarding enabled
- Appropriate firewall rules

For detailed prerequisites and setup instructions, see [PREREQUISITES.md](docs/PREREQUISITES.md).

## Contributing

1. Fork the repository
2. Create your feature branch
3. Run the test suite with your changes
4. Submit a pull request

## License

Copyright (c) 2025 o3willard-AI. All rights reserved.
