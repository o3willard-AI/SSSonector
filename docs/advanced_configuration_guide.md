# SSSonector Advanced Configuration Guide

This guide provides detailed information on all configuration options available in SSSonector.

## Basic Configuration

### Mode
- **Option**: `mode`
- **Description**: Specifies whether SSSonector runs in server or client mode.
- **Values**: `server`, `client`
- **Required**: Yes
- **Example**: `mode: server`

### Listen Address (Server Mode)
- **Option**: `listen`
- **Description**: Specifies the address and port on which the server listens for connections.
- **Format**: `<address>:<port>`
- **Required**: Yes (in server mode)
- **Example**: `listen: 0.0.0.0:443`

### Server Address (Client Mode)
- **Option**: `server`
- **Description**: Specifies the address and port of the server to which the client connects.
- **Format**: `<address>:<port>`
- **Required**: Yes (in client mode)
- **Example**: `server: 192.168.1.100:443`

### Interface Name
- **Option**: `interface`
- **Description**: Specifies the name of the TUN interface to create.
- **Required**: Yes
- **Example**: `interface: tun0`

### Interface Address
- **Option**: `address`
- **Description**: Specifies the IP address and subnet mask for the TUN interface.
- **Format**: `<ip>/<mask>`
- **Required**: Yes
- **Example**: `address: 10.0.0.1/24`

## Security Configuration

### TLS Enabled
- **Option**: `security.tls.enabled`
- **Description**: Specifies whether TLS is enabled for secure communication. TLS (Transport Layer Security) provides encryption, authentication, and integrity for the connection between the client and server.
- **Values**: `true`, `false`
- **Default**: `true`
- **Example**: `security.tls.enabled: true`
- **Notes**:
  - It is strongly recommended to keep TLS enabled for production environments.
  - Disabling TLS should only be done in secure, isolated networks or for testing purposes.

### TLS Minimum Version
- **Option**: `security.tls.min_version`
- **Description**: Specifies the minimum TLS version to use. Higher TLS versions provide better security but may not be supported by all clients.
- **Values**: 
  - `"1.2"`: TLS 1.2, widely supported but less secure than TLS 1.3.
  - `"1.3"`: TLS 1.3, provides better security and performance but may not be supported by older clients.
- **Default**: `"1.2"`
- **Example**: `security.tls.min_version: "1.3"`
- **Notes**:
  - TLS 1.3 offers several advantages over TLS 1.2:
    - Improved performance (faster handshakes)
    - Enhanced security (removal of obsolete and insecure features)
    - Simplified cipher suite negotiation
    - Forward secrecy by default
  - If all clients support TLS 1.3, it is recommended to set the minimum version to 1.3.
  - For backward compatibility with older clients, use TLS 1.2.

### Certificate File
- **Option**: `security.tls.cert_file`
- **Description**: Specifies the path to the certificate file. The certificate file contains the public key and is used to authenticate the server to the client.
- **Required**: Yes (if TLS is enabled)
- **Example**: `security.tls.cert_file: certs/server.crt`
- **Notes**:
  - The certificate should be in PEM format.
  - For production environments, use certificates issued by a trusted Certificate Authority (CA).
  - For testing environments, self-signed certificates can be used.

### Key File
- **Option**: `security.tls.key_file`
- **Description**: Specifies the path to the private key file. The private key is used to decrypt data encrypted with the public key.
- **Required**: Yes (if TLS is enabled)
- **Example**: `security.tls.key_file: certs/server.key`
- **Notes**:
  - The private key should be kept secure and not shared.
  - The private key should be in PEM format.
  - The private key should not be password-protected, as SSSonector does not support password-protected keys.

### CA Certificate File
- **Option**: `security.tls.ca_file`
- **Description**: Specifies the path to the CA certificate file. The CA certificate is used to verify the client certificate in mutual TLS authentication.
- **Required**: Yes (if TLS is enabled)
- **Example**: `security.tls.ca_file: certs/ca.crt`
- **Notes**:
  - The CA certificate should be in PEM format.
  - For mutual TLS authentication, the CA certificate must be the one that signed the client certificates.
  - For one-way TLS authentication, this file is still required but is not used to verify client certificates.

### Mutual TLS Authentication
- **Option**: `security.tls.mutual_auth`
- **Description**: Specifies whether mutual TLS authentication is enabled. When enabled, both the server and client authenticate each other using certificates.
- **Values**: `true`, `false`
- **Default**: `false`
- **Example**: `security.tls.mutual_auth: true`
- **Notes**:
  - Mutual TLS authentication provides an additional layer of security by requiring clients to present valid certificates.
  - When enabled, clients must have valid certificates signed by the CA specified in the `security.tls.ca_file` option.
  - For environments where client authentication is not required, this option can be set to `false`.

### Certificate Verification
- **Option**: `security.tls.verify_cert`
- **Description**: Specifies whether certificate verification is enabled. When enabled, the server verifies the client certificate against the CA certificate.
- **Values**: `true`, `false`
- **Default**: `true`
- **Example**: `security.tls.verify_cert: true`
- **Notes**:
  - It is strongly recommended to keep certificate verification enabled for production environments.
  - Disabling certificate verification should only be done in secure, isolated networks or for testing purposes.
  - This option is only relevant when `security.tls.mutual_auth` is set to `true`.

## Network Configuration

### MTU
- **Option**: `network.mtu`
- **Description**: Specifies the Maximum Transmission Unit (MTU) for the TUN interface. Adjusting the MTU can improve performance in certain network conditions. A larger MTU can increase throughput but may cause fragmentation issues on some networks. A smaller MTU can reduce latency and avoid fragmentation.
- **Default**: `1500`
- **Range**: `576` to `9000`
- **Example**: `network.mtu: 1500`
- **Notes**: 
  - For high-performance scenarios, consider using a larger MTU (e.g., `9000` for jumbo frames).
  - For low-latency scenarios, consider using a slightly smaller MTU (e.g., `1400`) to avoid fragmentation.
  - When using a non-standard MTU, ensure that all network devices along the path support the chosen MTU size.

### Packet Forwarding

- **Option**: `network.forwarding.enabled`
- **Description**: Specifies whether packet forwarding is enabled. When enabled, SSSonector will forward packets between the TUN interface and the physical network interfaces.
- **Values**: `true`, `false`
- **Default**: `true`
- **Example**: `network.forwarding.enabled: true`
- **Notes**: 
  - Packet forwarding is required for most use cases of SSSonector.
  - Disabling packet forwarding can be useful for testing or debugging purposes.
  - When packet forwarding is disabled, packets will still be received by the TUN interface but will not be forwarded to the physical network interfaces.

### Protocol-Specific Forwarding

SSSonector allows you to selectively enable or disable forwarding for specific protocols. This can be useful for security purposes or to optimize performance for specific use cases.

#### ICMP Forwarding

- **Option**: `network.forwarding.icmp_enabled`
- **Description**: Specifies whether ICMP packets are forwarded. ICMP (Internet Control Message Protocol) is used for network diagnostics and error reporting, including ping and traceroute.
- **Values**: `true`, `false`
- **Default**: `true`
- **Example**: `network.forwarding.icmp_enabled: true`
- **Notes**: 
  - ICMP forwarding is required for ping and traceroute to work across the tunnel.
  - Disabling ICMP forwarding can improve security by preventing ping-based reconnaissance, but may make network troubleshooting more difficult.
  - Even with ICMP forwarding disabled, other protocols (TCP, UDP, etc.) can still be forwarded if their respective options are enabled.

#### TCP Forwarding

- **Option**: `network.forwarding.tcp_enabled`
- **Description**: Specifies whether TCP packets are forwarded. TCP (Transmission Control Protocol) is used for reliable, connection-oriented communication, including web browsing, email, file transfers, and many other applications.
- **Values**: `true`, `false`
- **Default**: `true`
- **Example**: `network.forwarding.tcp_enabled: true`
- **Notes**: 
  - TCP forwarding is required for most application-level protocols (HTTP, SSH, etc.) to work across the tunnel.
  - Disabling TCP forwarding will prevent most internet applications from working across the tunnel.
  - TCP forwarding includes all TCP ports, including well-known ports (1-1023), registered ports (1024-49151), and dynamic/private ports (49152-65535).

#### UDP Forwarding

- **Option**: `network.forwarding.udp_enabled`
- **Description**: Specifies whether UDP packets are forwarded. UDP (User Datagram Protocol) is used for connectionless communication, including DNS, VoIP, video streaming, and online gaming.
- **Values**: `true`, `false`
- **Default**: `true`
- **Example**: `network.forwarding.udp_enabled: true`
- **Notes**: 
  - UDP forwarding is required for DNS, VoIP, and many real-time applications to work across the tunnel.
  - Disabling UDP forwarding will prevent DNS resolution and many real-time applications from working across the tunnel.
  - UDP forwarding includes all UDP ports, including well-known ports (1-1023), registered ports (1024-49151), and dynamic/private ports (49152-65535).

#### HTTP Forwarding

- **Option**: `network.forwarding.http_enabled`
- **Description**: Specifies whether HTTP traffic is forwarded. HTTP (Hypertext Transfer Protocol) is used for web browsing and many web-based applications.
- **Values**: `true`, `false`
- **Default**: `true`
- **Example**: `network.forwarding.http_enabled: true`
- **Notes**: 
  - HTTP forwarding is specifically optimized for web traffic, providing better performance than generic TCP forwarding for HTTP traffic.
  - Disabling HTTP forwarding will cause HTTP traffic to be handled by the generic TCP forwarding mechanism, which may result in lower performance.
  - HTTP forwarding includes both HTTP (port 80) and HTTPS (port 443) traffic.
  - This option is only relevant if `network.forwarding.tcp_enabled` is also set to `true`.

### Forwarding Use Cases

#### Basic Forwarding

For most use cases, the default forwarding configuration is sufficient:

```yaml
network:
  forwarding:
    enabled: true
    icmp_enabled: true
    tcp_enabled: true
    udp_enabled: true
    http_enabled: true
```

This configuration enables forwarding for all supported protocols, allowing all types of traffic to pass through the tunnel.

#### Security-Focused Forwarding

For security-sensitive environments, you may want to disable certain protocols:

```yaml
network:
  forwarding:
    enabled: true
    icmp_enabled: false  # Disable ping to prevent reconnaissance
    tcp_enabled: true
    udp_enabled: true
    http_enabled: true
```

This configuration disables ICMP forwarding, preventing ping-based reconnaissance while still allowing other types of traffic.

#### Application-Specific Forwarding

For environments where only specific applications need to be accessible through the tunnel, you can disable unnecessary protocols:

```yaml
network:
  forwarding:
    enabled: true
    icmp_enabled: false
    tcp_enabled: true    # Enable TCP for web and SSH
    udp_enabled: false   # Disable UDP if not needed
    http_enabled: true   # Optimize HTTP traffic
```

This configuration enables only TCP and HTTP traffic, which is sufficient for web browsing and SSH access while disabling other protocols.

### Forwarding Troubleshooting

#### Common Issues

If packets are not being forwarded as expected, check the following:

1. Verify that `network.forwarding.enabled` is set to `true`.
2. Verify that the protocol-specific forwarding option is set to `true` for the protocol you are using.
3. Check the system's IP forwarding setting:
   ```bash
   # On Linux
   sysctl net.ipv4.ip_forward
   ```
   The output should be `net.ipv4.ip_forward = 1`. If not, enable IP forwarding:
   ```bash
   # On Linux
   sudo sysctl -w net.ipv4.ip_forward=1
   ```
4. Check for firewall rules that might be blocking the traffic:
   ```bash
   # On Linux
   sudo iptables -L
   ```

#### Protocol-Specific Issues

##### ICMP Issues

If ping or traceroute is not working across the tunnel:

1. Verify that `network.forwarding.icmp_enabled` is set to `true`.
2. Check for firewall rules that might be blocking ICMP traffic.
3. Ensure that the remote host is configured to respond to ICMP requests.

##### TCP Issues

If TCP applications (web, SSH, etc.) are not working across the tunnel:

1. Verify that `network.forwarding.tcp_enabled` is set to `true`.
2. Check for firewall rules that might be blocking the specific TCP ports you are using.
3. Verify that the remote service is running and accessible.

##### UDP Issues

If UDP applications (DNS, VoIP, etc.) are not working across the tunnel:

1. Verify that `network.forwarding.udp_enabled` is set to `true`.
2. Check for firewall rules that might be blocking the specific UDP ports you are using.
3. Verify that the remote service is running and accessible.
4. For DNS specifically, check the DNS server configuration.

##### HTTP Issues

If web browsing is slow or not working across the tunnel:

1. Verify that both `network.forwarding.tcp_enabled` and `network.forwarding.http_enabled` are set to `true`.
2. Check for firewall rules that might be blocking HTTP (port 80) or HTTPS (port 443) traffic.
3. Verify that the web server is accessible.
4. Check for DNS resolution issues if you are using domain names.

## Logging Configuration

### Log Level
- **Option**: `logging.level`
- **Description**: Specifies the logging level. The log level determines the verbosity of the logs.
- **Values**: 
  - `debug`: Most verbose level, includes detailed information useful for debugging.
  - `info`: Standard level, includes general operational information.
  - `warning`: Includes only warning and error messages.
  - `error`: Includes only error messages.
- **Default**: `info`
- **Example**: `logging.level: debug`
- **Notes**:
  - Debug logging can significantly increase log file size and may impact performance.
  - For production environments, `info` or `warning` is recommended.
  - For troubleshooting issues, `debug` provides the most detailed information.

### Log File
- **Option**: `logging.file`
- **Description**: Specifies the path to the log file. If not specified, logs are written to stdout.
- **Default**: `stdout`
- **Example**: `logging.file: /var/log/sssonector.log`
- **Notes**:
  - For debugging, logging to stdout can be more convenient.
  - For production environments, logging to a file is recommended.
  - Log files should be rotated regularly to prevent disk space issues.

### Debug Categories
- **Option**: `logging.debug_categories`
- **Description**: Specifies which categories of debug logs to enable. This option is only relevant when `logging.level` is set to `debug`.
- **Values**: Array of strings, each representing a debug category.
- **Available Categories**:
  - `network`: Network-related debug logs.
  - `tls`: TLS-related debug logs.
  - `tunnel`: Tunnel-related debug logs.
  - `config`: Configuration-related debug logs.
  - `all`: All debug categories.
- **Default**: `["all"]`
- **Example**: `logging.debug_categories: ["network", "tunnel"]`
- **Notes**:
  - Enabling specific categories can reduce log verbosity while still providing useful debugging information.
  - For comprehensive debugging, use `["all"]`.

### Log Format
- **Option**: `logging.format`
- **Description**: Specifies the format of log messages.
- **Values**: 
  - `text`: Human-readable text format.
  - `json`: JSON format, useful for log processing tools.
- **Default**: `text`
- **Example**: `logging.format: json`
- **Notes**:
  - JSON format is recommended for environments where logs are processed by automated tools.
  - Text format is more readable for manual inspection.

## Monitoring Configuration

### Monitoring Enabled
- **Option**: `monitoring.enabled`
- **Description**: Specifies whether monitoring is enabled.
- **Values**: `true`, `false`
- **Default**: `false`
- **Example**: `monitoring.enabled: true`

### Monitoring Port
- **Option**: `monitoring.port`
- **Description**: Specifies the port on which the monitoring server listens.
- **Default**: `9090`
- **Example**: `monitoring.port: 9090`

## Complete Configuration Examples

### Server Configuration
```yaml
mode: server
listen: 0.0.0.0:443
interface: tun0
address: 10.0.0.1/24
network:
  mtu: 1500
  forwarding:
    enabled: true
    icmp_enabled: true
    tcp_enabled: true
    udp_enabled: true
    http_enabled: true
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
server: 192.168.1.100:443  # Replace with your server's IP address and port
interface: tun0
address: 10.0.0.2/24
network:
  mtu: 1500
  forwarding:
    enabled: true
    icmp_enabled: true
    tcp_enabled: true
    udp_enabled: true
    http_enabled: true
logging:
  level: info
  file: /var/log/sssonector.log
monitoring:
  enabled: false
security:
  tls:
    enabled: true
    min_version: "1.2"
    cert_file: certs/client.crt
    key_file: certs/client.key
    ca_file: certs/ca.crt
```

> **Note**: Replace `192.168.1.100:443` with the actual IP address and port of your SSSonector server.

## Advanced Configuration Scenarios

### High-Performance Configuration

For high-performance scenarios, consider the following configuration:

```yaml
mode: server
listen: 0.0.0.0:443
interface: tun0
address: 10.0.0.1/24
network:
  mtu: 9000  # Jumbo frames for higher throughput
  forwarding:
    enabled: true
    icmp_enabled: true
    tcp_enabled: true
    udp_enabled: true
    http_enabled: true
logging:
  level: warning  # Reduce logging overhead
  file: /var/log/sssonector.log
monitoring:
  enabled: true
  port: 9090
security:
  tls:
    enabled: true
    min_version: "1.3"  # TLS 1.3 for better performance
    cert_file: certs/server.crt
    key_file: certs/server.key
    ca_file: certs/ca.crt
```

### Debugging Configuration

For debugging issues, consider the following configuration:

```yaml
mode: server
listen: 0.0.0.0:443
interface: tun0
address: 10.0.0.1/24
network:
  mtu: 1500
  forwarding:
    enabled: true
    icmp_enabled: true
    tcp_enabled: true
    udp_enabled: true
    http_enabled: true
logging:
  level: debug  # Enable debug logging
  debug_categories: ["all"]  # Enable all debug categories
  file: stdout  # Log to stdout for immediate feedback
  format: text  # Human-readable format for easier debugging
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

### Low-Latency Configuration

For low-latency scenarios, consider the following configuration:

```yaml
mode: server
listen: 0.0.0.0:443
interface: tun0
address: 10.0.0.1/24
network:
  mtu: 1400  # Slightly smaller MTU to avoid fragmentation
  forwarding:
    enabled: true
    icmp_enabled: true
    tcp_enabled: true
    udp_enabled: true
    http_enabled: true
logging:
  level: warning  # Reduce logging overhead
  file: /var/log/sssonector.log
monitoring:
  enabled: false  # Disable monitoring to reduce overhead
security:
  tls:
    enabled: true
    min_version: "1.3"  # TLS 1.3 for better performance
    cert_file: certs/server.crt
    key_file: certs/server.key
    ca_file: certs/ca.crt
```

## Environment Variables

SSSonector also supports configuration through environment variables. Environment variables take precedence over configuration file values.

### Available Environment Variables

- `SSSONECTOR_MODE`: Specifies the mode (`server` or `client`).
- `SSSONECTOR_LISTEN`: Specifies the listen address (server mode).
- `SSSONECTOR_SERVER`: Specifies the server address (client mode).
- `SSSONECTOR_INTERFACE`: Specifies the interface name.
- `SSSONECTOR_ADDRESS`: Specifies the interface address.
- `SSSONECTOR_NETWORK_MTU`: Specifies the MTU.
- `SSSONECTOR_NETWORK_FORWARDING_ENABLED`: Specifies whether packet forwarding is enabled.
- `SSSONECTOR_NETWORK_FORWARDING_ICMP_ENABLED`: Specifies whether ICMP packet forwarding is enabled.
- `SSSONECTOR_NETWORK_FORWARDING_TCP_ENABLED`: Specifies whether TCP packet forwarding is enabled.
- `SSSONECTOR_NETWORK_FORWARDING_UDP_ENABLED`: Specifies whether UDP packet forwarding is enabled.
- `SSSONECTOR_NETWORK_FORWARDING_HTTP_ENABLED`: Specifies whether HTTP packet forwarding is enabled.
- `SSSONECTOR_LOGGING_LEVEL`: Specifies the logging level.
- `SSSONECTOR_LOGGING_FILE`: Specifies the log file.
- `SSSONECTOR_LOGGING_DEBUG_CATEGORIES`: Specifies the debug categories (comma-separated list).
- `SSSONECTOR_LOGGING_FORMAT`: Specifies the log format (`text` or `json`).
- `SSSONECTOR_MONITORING_ENABLED`: Specifies whether monitoring is enabled.
- `SSSONECTOR_MONITORING_PORT`: Specifies the monitoring port.
- `SSSONECTOR_SECURITY_TLS_ENABLED`: Specifies whether TLS is enabled.
- `SSSONECTOR_SECURITY_TLS_MIN_VERSION`: Specifies the minimum TLS version.
- `SSSONECTOR_SECURITY_TLS_CERT_FILE`: Specifies the certificate file.
- `SSSONECTOR_SECURITY_TLS_KEY_FILE`: Specifies the key file.
- `SSSONECTOR_SECURITY_TLS_CA_FILE`: Specifies the CA certificate file.
- `SSSONECTOR_SECURITY_TLS_MUTUAL_AUTH`: Specifies whether mutual TLS authentication is enabled.
- `SSSONECTOR_SECURITY_TLS_VERIFY_CERT`: Specifies whether certificate verification is enabled.

### Example Usage

```bash
# Basic configuration
export SSSONECTOR_MODE=server
export SSSONECTOR_LISTEN=0.0.0.0:443
export SSSONECTOR_INTERFACE=tun0
export SSSONECTOR_ADDRESS=10.0.0.1/24

# Network configuration
export SSSONECTOR_NETWORK_MTU=1500
export SSSONECTOR_NETWORK_FORWARDING_ENABLED=true
export SSSONECTOR_NETWORK_FORWARDING_ICMP_ENABLED=true
export SSSONECTOR_NETWORK_FORWARDING_TCP_ENABLED=true
export SSSONECTOR_NETWORK_FORWARDING_UDP_ENABLED=true
export SSSONECTOR_NETWORK_FORWARDING_HTTP_ENABLED=true

# Logging configuration
export SSSONECTOR_LOGGING_LEVEL=info
export SSSONECTOR_LOGGING_FILE=/var/log/sssonector.log
export SSSONECTOR_LOGGING_FORMAT=text

# Monitoring configuration
export SSSONECTOR_MONITORING_ENABLED=true
export SSSONECTOR_MONITORING_PORT=9090

# Security configuration
export SSSONECTOR_SECURITY_TLS_ENABLED=true
export SSSONECTOR_SECURITY_TLS_MIN_VERSION=1.2
export SSSONECTOR_SECURITY_TLS_CERT_FILE=certs/server.crt
export SSSONECTOR_SECURITY_TLS_KEY_FILE=certs/server.key
export SSSONECTOR_SECURITY_TLS_CA_FILE=certs/ca.crt
export SSSONECTOR_SECURITY_TLS_MUTUAL_AUTH=false
export SSSONECTOR_SECURITY_TLS_VERIFY_CERT=true

# Run SSSonector
./sssonector
```

## Configuration File Locations

SSSonector searches for configuration files in the following locations, in order:

1. The path specified with the `-config` flag
2. `./sssonector.yaml` in the current directory
3. `./sssonector.yml` in the current directory
4. `/etc/sssonector/sssonector.yaml`
5. `/etc/sssonector/sssonector.yml`

## Configuration Validation

SSSonector validates the configuration file at startup and reports any errors. Common validation errors include:

- Missing required fields
- Invalid values for fields
- Incompatible combinations of options
- File paths that don't exist or aren't accessible

For detailed error messages, enable debug logging by setting `logging.level: debug` in the configuration file or using the `SSSONECTOR_LOGGING_LEVEL=debug` environment variable.
