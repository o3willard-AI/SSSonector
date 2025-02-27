# Network Forwarding Configuration

This section provides detailed information on configuring packet forwarding in SSSonector. Packet forwarding is a core functionality that allows SSSonector to forward packets between the TUN interface and the physical network interfaces, enabling communication between devices on different networks.

## Packet Forwarding Options

### Enabling Packet Forwarding

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

## Common Use Cases

### Basic Forwarding

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

### Security-Focused Forwarding

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

### Application-Specific Forwarding

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

## Troubleshooting

### Common Issues

#### Packets Not Being Forwarded

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

#### Performance Issues

If you are experiencing performance issues with packet forwarding, consider the following:

1. For HTTP traffic, ensure that `network.forwarding.http_enabled` is set to `true` to enable HTTP-specific optimizations.
2. Adjust the MTU setting to avoid fragmentation:
   ```yaml
   network:
     mtu: 1400  # Slightly smaller MTU to avoid fragmentation
   ```
3. Disable unnecessary protocol forwarding to reduce overhead.
4. Check for network congestion or bandwidth limitations.

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

## Advanced Configuration

### Combining with MTU Settings

For optimal performance, you may need to adjust the MTU setting based on the types of traffic you are forwarding:

```yaml
network:
  mtu: 1400  # Slightly smaller MTU to avoid fragmentation
  forwarding:
    enabled: true
    icmp_enabled: true
    tcp_enabled: true
    udp_enabled: true
    http_enabled: true
```

This configuration uses a slightly smaller MTU to avoid fragmentation, which can improve performance for all types of traffic.

### Combining with Logging

For troubleshooting forwarding issues, you can enable debug logging for the network category:

```yaml
network:
  forwarding:
    enabled: true
    icmp_enabled: true
    tcp_enabled: true
    udp_enabled: true
    http_enabled: true
logging:
  level: debug
  debug_categories: ["network"]
  file: /var/log/sssonector.log
```

This configuration enables debug logging for the network category, which includes packet forwarding, helping you diagnose forwarding issues.

## Environment Variables

SSSonector also supports configuration of packet forwarding through environment variables:

- `SSSONECTOR_NETWORK_FORWARDING_ENABLED`: Specifies whether packet forwarding is enabled.
- `SSSONECTOR_NETWORK_FORWARDING_ICMP_ENABLED`: Specifies whether ICMP packet forwarding is enabled.
- `SSSONECTOR_NETWORK_FORWARDING_TCP_ENABLED`: Specifies whether TCP packet forwarding is enabled.
- `SSSONECTOR_NETWORK_FORWARDING_UDP_ENABLED`: Specifies whether UDP packet forwarding is enabled.
- `SSSONECTOR_NETWORK_FORWARDING_HTTP_ENABLED`: Specifies whether HTTP packet forwarding is enabled.

Example usage:

```bash
export SSSONECTOR_NETWORK_FORWARDING_ENABLED=true
export SSSONECTOR_NETWORK_FORWARDING_ICMP_ENABLED=true
export SSSONECTOR_NETWORK_FORWARDING_TCP_ENABLED=true
export SSSONECTOR_NETWORK_FORWARDING_UDP_ENABLED=true
export SSSONECTOR_NETWORK_FORWARDING_HTTP_ENABLED=true

./sssonector
```

## Conclusion

Packet forwarding is a core functionality of SSSonector that enables communication between devices on different networks. By understanding and properly configuring the packet forwarding options, you can optimize SSSonector for your specific use case, whether it's general internet access, security-focused deployment, or application-specific connectivity.

For more information on other network configuration options, see the [Network Configuration](#network-configuration) section.
