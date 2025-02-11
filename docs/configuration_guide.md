# SSSonector Configuration Guide

This guide provides detailed information about configuring SSSonector, including complete examples and explanations for both server and client setups.

## Configuration File Structure

SSSonector uses YAML configuration files with the following main sections:
- `type`: Server or client type
- `config`: Main configuration section containing all settings
  - `mode`: Determines if running as server or client
  - `network`: Network interface configuration
  - `tunnel`: SSL tunnel and certificate settings
  - `monitor`: Monitoring and SNMP configuration
  - `security`: Security settings including TLS configuration
  - `metrics`: Metrics collection settings
- `version`: Configuration version
- `metadata`: Configuration metadata
- `throttle`: Rate limiting settings

## Basic Configuration Examples

### Server Configuration
```yaml
type: server
config:
  mode: server
  network:
    name: tun0
    interface: tun0
    address: 10.0.0.1/24  # Server uses first address in tunnel network
    mtu: 1500             # Standard MTU, adjust if needed

  tunnel:
    listen_port: 8443
    protocol: tcp
    cert_file: /etc/sssonector/certs/server.crt
    key_file: /etc/sssonector/certs/server.key
    ca_file: /etc/sssonector/certs/ca.crt
    listen_address: 0.0.0.0  # Listen on all interfaces
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
  rate: 1048576    # 1 MB/s
  burst: 1048576   # 1 MB burst
```

### Client Configuration
```yaml
type: client
config:
  mode: client
  network:
    name: tun0
    interface: tun0
    address: 10.0.0.2/24  # Client uses unique address in tunnel network
    mtu: 1500

  tunnel:
    server_port: 8443
    protocol: tcp
    cert_file: /etc/sssonector/certs/client.crt
    key_file: /etc/sssonector/certs/client.key
    ca_file: /etc/sssonector/certs/ca.crt
    server_address: 192.168.50.210  # Server's public IP

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

## Section Details

### Network Configuration
- `name`: Name of the TUN interface to create
- `interface`: Name of the TUN interface (should match name)
- `address`: IP address/netmask for the tunnel interface
- `mtu`: Maximum Transmission Unit (default: 1500)

### Tunnel Configuration
- `cert_file`, `key_file`, `ca_file`: Paths to SSL certificates
  * Can be absolute paths or relative to config directory
  * Default location: /etc/sssonector/certs/
- `listen_address`, `listen_port`: Server listening settings
- `server_address`, `server_port`: Client connection settings
- `max_clients`: Maximum concurrent client connections (server only)
- `protocol`: Transport protocol (tcp/udp)

### Security Configuration
- `memory_protections`: Enable/disable memory protection features
- `namespace`: Enable/disable namespace isolation
- `capabilities`: Enable/disable capability restrictions
- `tls`: TLS configuration
  * `min_version`: Minimum TLS version
  * `max_version`: Maximum TLS version
  * `ciphers`: List of allowed cipher suites

### Monitor Configuration
- `enabled`: Enable/disable monitoring
- `type`: Monitoring type (basic/advanced)
- `interval`: Metrics collection interval

### Metrics Configuration
- `enabled`: Enable/disable metrics collection
- `address`: Metrics server address
- `interval`: Collection interval
- `buffer_size`: Metrics buffer size

### Throttle Configuration
- `enabled`: Enable/disable rate limiting
- `rate`: Sustained rate limit in bytes/sec
- `burst`: Burst rate limit in bytes/sec

## Path Resolution Rules

1. Certificate paths:
   - Absolute paths are used as-is
   - Relative paths are resolved from config file location
   - Default paths use /etc/sssonector/certs/

2. Log files:
   - Absolute paths are used as-is
   - Relative paths are resolved from current directory
   - Default location: /var/log/sssonector/

## Common Issues and Solutions

1. Certificate Loading Fails
   ```
   Problem: "failed to load certificate"
   Solution: Ensure paths are correct and files have proper permissions (600)
   ```

2. Network Interface Creation Fails
   ```
   Problem: "failed to create tun device"
   Solution: Run with sufficient privileges (root/sudo)
   ```

3. TLS Configuration Fails
   ```
   Problem: "invalid security config: TLS min version cannot be empty"
   Solution: Ensure TLS configuration is complete with min_version and max_version
   ```

## Configuration Testing

Before deploying, validate your configuration:

1. Test configuration syntax:
   ```bash
   sssonector -validate-config /etc/sssonector/config.yaml
   ```

2. Test with debug logging:
   ```bash
   sssonector -config /etc/sssonector/config.yaml -debug
   ```

3. Verify TLS configuration:
   ```bash
   sssonector -validate-tls -config /etc/sssonector/config.yaml
   ```

## Best Practices

1. Security
   - Use absolute paths for certificates
   - Set restrictive file permissions (600 for keys)
   - Configure TLS versions and ciphers explicitly
   - Enable all security features (memory_protections, namespace, capabilities)

2. Performance
   - Set appropriate MTU for your network
   - Configure rate limits based on available bandwidth
   - Adjust metrics buffer size based on memory availability
   - Monitor system metrics for optimization

3. Monitoring
   - Enable metrics collection in production
   - Set appropriate collection intervals
   - Monitor memory usage with buffer sizes
   - Use structured logging format (json)

4. Testing
   - Validate configurations before deployment
   - Test with debug logging enabled
   - Verify TLS settings
   - Check security feature effectiveness
