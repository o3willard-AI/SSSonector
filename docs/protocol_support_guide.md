# SSSonector Protocol Support Guide

This guide provides detailed information about the protocols supported by SSSonector and how they are handled. Understanding protocol support is essential for configuring SSSonector effectively and troubleshooting connectivity issues.

## Supported Protocols

SSSonector supports forwarding of the following protocols:

- **ICMP**: Used for ping and traceroute.
- **TCP**: Used for most application-level protocols (HTTP, SSH, etc.).
- **UDP**: Used for DNS, VoIP, and other real-time applications.
- **HTTP/HTTPS**: Specifically optimized for web traffic.

All protocols are enabled by default, but you can selectively enable or disable them using the configuration options described in the [Advanced Configuration Guide](advanced_configuration_guide.md).

## Protocol Details

### ICMP (Internet Control Message Protocol)

ICMP is used for network diagnostics and error reporting, including ping and traceroute.

#### Configuration

- **Option**: `network.forwarding.icmp_enabled`
- **Values**: `true`, `false`
- **Default**: `true`
- **Example**: `network.forwarding.icmp_enabled: true`

#### Use Cases

- **Network Diagnostics**: ICMP is commonly used for diagnosing network connectivity issues using tools like ping and traceroute.
- **Error Reporting**: ICMP is used by network devices to report errors, such as "Destination Unreachable" or "Time Exceeded."

#### Performance Considerations

- ICMP traffic is typically low-volume and has minimal impact on performance.
- ICMP packets are small and do not require fragmentation, so they are not affected by MTU settings.

#### Security Considerations

- ICMP can be used for reconnaissance, so you may want to disable it in security-sensitive environments.
- Disabling ICMP can make network troubleshooting more difficult.

#### Example Configuration

```yaml
network:
  forwarding:
    icmp_enabled: true  # Enable ICMP forwarding
```

### TCP (Transmission Control Protocol)

TCP is used for reliable, connection-oriented communication, including web browsing, email, file transfers, and many other applications.

#### Configuration

- **Option**: `network.forwarding.tcp_enabled`
- **Values**: `true`, `false`
- **Default**: `true`
- **Example**: `network.forwarding.tcp_enabled: true`

#### Use Cases

- **Web Browsing**: HTTP and HTTPS use TCP for reliable data transfer.
- **Email**: SMTP, POP3, and IMAP use TCP for reliable email delivery.
- **File Transfers**: FTP, SFTP, and SCP use TCP for reliable file transfers.
- **Remote Access**: SSH and RDP use TCP for secure remote access.
- **Database Access**: Most database protocols use TCP for reliable data access.

#### Performance Considerations

- TCP includes flow control and congestion control mechanisms that can affect performance.
- TCP performance can be affected by latency, packet loss, and MTU settings.
- For optimal TCP performance, consider the following:
  - Use a larger MTU for higher throughput (e.g., 9000 for jumbo frames).
  - Use a smaller MTU to reduce latency and avoid fragmentation (e.g., 1400).
  - Ensure that all network devices along the path support the chosen MTU size.

#### Security Considerations

- TCP connections can be vulnerable to various attacks, such as SYN floods and TCP reset attacks.
- TCP connections should be protected using TLS or other encryption mechanisms.
- Consider using firewall rules to restrict access to specific TCP ports.

#### Example Configuration

```yaml
network:
  forwarding:
    tcp_enabled: true  # Enable TCP forwarding
```

### UDP (User Datagram Protocol)

UDP is used for connectionless communication, including DNS, VoIP, video streaming, and online gaming.

#### Configuration

- **Option**: `network.forwarding.udp_enabled`
- **Values**: `true`, `false`
- **Default**: `true`
- **Example**: `network.forwarding.udp_enabled: true`

#### Use Cases

- **DNS**: Domain Name System uses UDP for name resolution.
- **VoIP**: Voice over IP uses UDP for real-time voice communication.
- **Video Streaming**: Many video streaming applications use UDP for real-time video delivery.
- **Online Gaming**: Many online games use UDP for real-time game state updates.
- **DHCP**: Dynamic Host Configuration Protocol uses UDP for IP address assignment.

#### Performance Considerations

- UDP does not include flow control or congestion control mechanisms, so it can be more efficient for real-time applications.
- UDP performance can be affected by packet loss and MTU settings.
- For optimal UDP performance, consider the following:
  - Use a smaller MTU to reduce the impact of packet loss (e.g., 1400).
  - Ensure that all network devices along the path support the chosen MTU size.

#### Security Considerations

- UDP connections can be vulnerable to various attacks, such as UDP floods and DNS amplification attacks.
- UDP connections should be protected using DTLS or other encryption mechanisms when possible.
- Consider using firewall rules to restrict access to specific UDP ports.

#### Example Configuration

```yaml
network:
  forwarding:
    udp_enabled: true  # Enable UDP forwarding
```

### HTTP/HTTPS (Hypertext Transfer Protocol)

HTTP and HTTPS are used for web browsing and many web-based applications. SSSonector includes specific optimizations for HTTP/HTTPS traffic.

#### Configuration

- **Option**: `network.forwarding.http_enabled`
- **Values**: `true`, `false`
- **Default**: `true`
- **Example**: `network.forwarding.http_enabled: true`

#### Use Cases

- **Web Browsing**: HTTP and HTTPS are used for accessing websites.
- **Web Applications**: Many web applications use HTTP and HTTPS for communication.
- **REST APIs**: Many REST APIs use HTTP and HTTPS for communication.
- **WebSockets**: WebSockets use HTTP for the initial handshake.

#### Performance Considerations

- HTTP/HTTPS traffic can benefit from specific optimizations, such as header compression and connection reuse.
- HTTP/HTTPS performance can be affected by latency, packet loss, and MTU settings.
- For optimal HTTP/HTTPS performance, consider the following:
  - Enable HTTP forwarding to take advantage of SSSonector's HTTP-specific optimizations.
  - Use a smaller MTU to reduce latency and avoid fragmentation (e.g., 1400).
  - Ensure that all network devices along the path support the chosen MTU size.

#### Security Considerations

- HTTP traffic is unencrypted and can be intercepted or modified by attackers.
- HTTPS traffic is encrypted using TLS and provides confidentiality, integrity, and authentication.
- Consider using HTTPS instead of HTTP whenever possible.
- Consider using firewall rules to restrict access to specific HTTP/HTTPS ports.

#### Example Configuration

```yaml
network:
  forwarding:
    tcp_enabled: true     # HTTP/HTTPS requires TCP
    http_enabled: true    # Enable HTTP-specific optimizations
```

## Protocol Combinations

### Web Browsing

For optimal web browsing performance, enable TCP and HTTP forwarding:

```yaml
network:
  forwarding:
    enabled: true
    tcp_enabled: true     # Required for HTTP/HTTPS
    http_enabled: true    # Enable HTTP-specific optimizations
    udp_enabled: true     # Required for DNS
    icmp_enabled: true    # Useful for troubleshooting
```

### VoIP and Video Conferencing

For optimal VoIP and video conferencing performance, enable UDP forwarding:

```yaml
network:
  forwarding:
    enabled: true
    udp_enabled: true     # Required for real-time media
    tcp_enabled: true     # Required for signaling
    http_enabled: true    # Required for web interfaces
    icmp_enabled: true    # Useful for troubleshooting
```

### Remote Access

For optimal remote access performance, enable TCP forwarding:

```yaml
network:
  forwarding:
    enabled: true
    tcp_enabled: true     # Required for SSH, RDP, etc.
    udp_enabled: true     # Required for DNS
    http_enabled: true    # Required for web interfaces
    icmp_enabled: true    # Useful for troubleshooting
```

### Gaming

For optimal gaming performance, enable UDP forwarding:

```yaml
network:
  forwarding:
    enabled: true
    udp_enabled: true     # Required for real-time game state updates
    tcp_enabled: true     # Required for game downloads and updates
    http_enabled: true    # Required for web interfaces
    icmp_enabled: true    # Useful for troubleshooting
```

## Protocol Troubleshooting

### ICMP Issues

If ping or traceroute is not working across the tunnel:

1. Verify that `network.forwarding.icmp_enabled` is set to `true`.
2. Check for firewall rules that might be blocking ICMP traffic.
3. Ensure that the remote host is configured to respond to ICMP requests.

### TCP Issues

If TCP applications (web, SSH, etc.) are not working across the tunnel:

1. Verify that `network.forwarding.tcp_enabled` is set to `true`.
2. Check for firewall rules that might be blocking the specific TCP ports you are using.
3. Verify that the remote service is running and accessible.
4. Try using a different MTU setting to avoid fragmentation issues.

### UDP Issues

If UDP applications (DNS, VoIP, etc.) are not working across the tunnel:

1. Verify that `network.forwarding.udp_enabled` is set to `true`.
2. Check for firewall rules that might be blocking the specific UDP ports you are using.
3. Verify that the remote service is running and accessible.
4. For DNS specifically, check the DNS server configuration.
5. Try using a different MTU setting to avoid fragmentation issues.

### HTTP/HTTPS Issues

If web browsing is slow or not working across the tunnel:

1. Verify that both `network.forwarding.tcp_enabled` and `network.forwarding.http_enabled` are set to `true`.
2. Check for firewall rules that might be blocking HTTP (port 80) or HTTPS (port 443) traffic.
3. Verify that the web server is accessible.
4. Check for DNS resolution issues if you are using domain names.
5. Try using a different MTU setting to avoid fragmentation issues.

## Advanced Protocol Configuration

### MTU Optimization

The Maximum Transmission Unit (MTU) setting can significantly affect protocol performance. Here are some recommendations for different protocols:

- **ICMP**: ICMP packets are typically small and not affected by MTU settings.
- **TCP**: For optimal TCP performance, use a larger MTU for higher throughput (e.g., 9000 for jumbo frames) or a smaller MTU to reduce latency and avoid fragmentation (e.g., 1400).
- **UDP**: For optimal UDP performance, use a smaller MTU to reduce the impact of packet loss (e.g., 1400).
- **HTTP/HTTPS**: For optimal HTTP/HTTPS performance, use a smaller MTU to reduce latency and avoid fragmentation (e.g., 1400).

Example configuration:

```yaml
network:
  mtu: 1400  # Smaller MTU to reduce latency and avoid fragmentation
  forwarding:
    enabled: true
    icmp_enabled: true
    tcp_enabled: true
    udp_enabled: true
    http_enabled: true
```

### Protocol-Specific Logging

For troubleshooting protocol-specific issues, you can enable debug logging for the network category:

```yaml
logging:
  level: debug
  debug_categories: ["network"]
  file: /var/log/sssonector.log
```

This configuration enables debug logging for the network category, which includes protocol-specific logs, helping you diagnose protocol-related issues.

## Conclusion

SSSonector supports a wide range of protocols, including ICMP, TCP, UDP, and HTTP/HTTPS. By understanding how these protocols are handled and configuring SSSonector appropriately, you can optimize performance and troubleshoot connectivity issues effectively.

For more information on configuring protocol forwarding, see the [Advanced Configuration Guide](advanced_configuration_guide.md).
