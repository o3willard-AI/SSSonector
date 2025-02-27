# SSSonector Error Handling Guide

This guide provides detailed information about error handling in SSSonector, including network failures, reconnection behavior, and best practices for handling errors.

## Error Types

SSSonector can encounter various types of errors during operation. Understanding these error types is essential for effective troubleshooting and error handling.

### Configuration Errors

Configuration errors occur when SSSonector encounters invalid or incompatible configuration options. These errors typically occur during startup and prevent SSSonector from starting.

Common configuration errors include:

- Missing required fields
- Invalid values for fields
- Incompatible combinations of options
- File paths that don't exist or aren't accessible

Example error messages:
```
Error: Required field 'mode' is missing
Error: Invalid value 'invalid_mode' for field 'mode', expected 'server' or 'client'
Error: Incompatible options: 'listen' is only valid in server mode
Error: File 'certs/server.crt' does not exist or is not accessible
```

### Network Errors

Network errors occur when SSSonector encounters issues with network operations. These errors can occur during startup or during operation.

Common network errors include:

- Interface creation failures
- IP address assignment failures
- Port binding failures
- Connection failures
- Packet forwarding failures

Example error messages:
```
Error: Failed to create TUN interface: Permission denied
Error: Failed to assign IP address: Address already in use
Error: Failed to bind to port 443: Address already in use
Error: Failed to connect to server: Connection refused
Error: Failed to forward packet: No route to host
```

### TLS Errors

TLS errors occur when SSSonector encounters issues with TLS operations. These errors can occur during startup or during operation.

Common TLS errors include:

- Certificate loading failures
- Certificate validation failures
- TLS handshake failures
- Encryption/decryption failures

Example error messages:
```
Error: Failed to load certificate: Invalid certificate format
Error: Failed to validate certificate: Certificate has expired
Error: TLS handshake failed: Protocol version not supported
Error: Encryption failed: Invalid key
```

### Tunnel Errors

Tunnel errors occur when SSSonector encounters issues with tunnel operations. These errors can occur during operation.

Common tunnel errors include:

- Tunnel establishment failures
- Data transfer failures
- Tunnel closure failures

Example error messages:
```
Error: Failed to establish tunnel: Connection refused
Error: Failed to transfer data: Connection reset by peer
Error: Failed to close tunnel: Connection already closed
```

## Error Handling Strategies

SSSonector employs various strategies to handle errors effectively. Understanding these strategies can help you configure SSSonector for optimal error handling.

### Logging

SSSonector logs all errors with detailed information to help diagnose and resolve issues. The log level determines the verbosity of the logs.

For detailed error information, set the log level to `debug`:

```yaml
logging:
  level: debug
  file: /var/log/sssonector.log
```

For error-specific debugging, enable the relevant debug categories:

```yaml
logging:
  level: debug
  debug_categories: ["network", "tls", "tunnel", "config"]
  file: /var/log/sssonector.log
```

### Retry Mechanisms

SSSonector includes retry mechanisms for certain operations to handle transient errors. These retry mechanisms help SSSonector recover from temporary issues without manual intervention.

#### Connection Retries

When a client fails to connect to a server, SSSonector will automatically retry the connection with an exponential backoff strategy. This helps handle temporary network issues or server restarts.

The connection retry behavior can be configured using the following options:

- **Option**: `client.connection.retry_enabled`
- **Description**: Specifies whether connection retries are enabled.
- **Values**: `true`, `false`
- **Default**: `true`
- **Example**: `client.connection.retry_enabled: true`

- **Option**: `client.connection.retry_max_attempts`
- **Description**: Specifies the maximum number of connection retry attempts.
- **Values**: Integer greater than 0
- **Default**: `10`
- **Example**: `client.connection.retry_max_attempts: 5`

- **Option**: `client.connection.retry_initial_delay`
- **Description**: Specifies the initial delay between connection retry attempts, in seconds.
- **Values**: Integer greater than 0
- **Default**: `1`
- **Example**: `client.connection.retry_initial_delay: 2`

- **Option**: `client.connection.retry_max_delay`
- **Description**: Specifies the maximum delay between connection retry attempts, in seconds.
- **Values**: Integer greater than 0
- **Default**: `60`
- **Example**: `client.connection.retry_max_delay: 30`

Example configuration:

```yaml
client:
  connection:
    retry_enabled: true
    retry_max_attempts: 5
    retry_initial_delay: 2
    retry_max_delay: 30
```

With this configuration, SSSonector will retry the connection up to 5 times, with an initial delay of 2 seconds between attempts. The delay will increase exponentially with each attempt, up to a maximum of 30 seconds.

#### Data Transfer Retries

When a data transfer operation fails, SSSonector will automatically retry the operation with an exponential backoff strategy. This helps handle temporary network issues or congestion.

The data transfer retry behavior can be configured using the following options:

- **Option**: `tunnel.transfer.retry_enabled`
- **Description**: Specifies whether data transfer retries are enabled.
- **Values**: `true`, `false`
- **Default**: `true`
- **Example**: `tunnel.transfer.retry_enabled: true`

- **Option**: `tunnel.transfer.retry_max_attempts`
- **Description**: Specifies the maximum number of data transfer retry attempts.
- **Values**: Integer greater than 0
- **Default**: `3`
- **Example**: `tunnel.transfer.retry_max_attempts: 5`

- **Option**: `tunnel.transfer.retry_initial_delay`
- **Description**: Specifies the initial delay between data transfer retry attempts, in milliseconds.
- **Values**: Integer greater than 0
- **Default**: `100`
- **Example**: `tunnel.transfer.retry_initial_delay: 200`

- **Option**: `tunnel.transfer.retry_max_delay`
- **Description**: Specifies the maximum delay between data transfer retry attempts, in milliseconds.
- **Values**: Integer greater than 0
- **Default**: `1000`
- **Example**: `tunnel.transfer.retry_max_delay: 500`

Example configuration:

```yaml
tunnel:
  transfer:
    retry_enabled: true
    retry_max_attempts: 5
    retry_initial_delay: 200
    retry_max_delay: 500
```

With this configuration, SSSonector will retry data transfer operations up to 5 times, with an initial delay of 200 milliseconds between attempts. The delay will increase exponentially with each attempt, up to a maximum of 500 milliseconds.

### Reconnection Behavior

SSSonector includes reconnection mechanisms to handle network failures and server restarts. These mechanisms help SSSonector maintain connectivity without manual intervention.

#### Client Reconnection

When a client loses connection to a server, SSSonector will automatically attempt to reconnect with an exponential backoff strategy. This helps handle temporary network issues or server restarts.

The client reconnection behavior can be configured using the following options:

- **Option**: `client.reconnection.enabled`
- **Description**: Specifies whether client reconnection is enabled.
- **Values**: `true`, `false`
- **Default**: `true`
- **Example**: `client.reconnection.enabled: true`

- **Option**: `client.reconnection.max_attempts`
- **Description**: Specifies the maximum number of reconnection attempts.
- **Values**: Integer greater than 0, or `-1` for unlimited attempts
- **Default**: `-1` (unlimited)
- **Example**: `client.reconnection.max_attempts: 10`

- **Option**: `client.reconnection.initial_delay`
- **Description**: Specifies the initial delay between reconnection attempts, in seconds.
- **Values**: Integer greater than 0
- **Default**: `1`
- **Example**: `client.reconnection.initial_delay: 2`

- **Option**: `client.reconnection.max_delay`
- **Description**: Specifies the maximum delay between reconnection attempts, in seconds.
- **Values**: Integer greater than 0
- **Default**: `60`
- **Example**: `client.reconnection.max_delay: 30`

Example configuration:

```yaml
client:
  reconnection:
    enabled: true
    max_attempts: 10
    initial_delay: 2
    max_delay: 30
```

With this configuration, SSSonector will attempt to reconnect up to 10 times, with an initial delay of 2 seconds between attempts. The delay will increase exponentially with each attempt, up to a maximum of 30 seconds.

#### Server Restart Handling

When a server restarts, clients will automatically attempt to reconnect using the reconnection mechanism described above. This helps maintain connectivity without manual intervention.

To ensure smooth server restarts, consider the following best practices:

1. Configure clients with appropriate reconnection settings.
2. Use a persistent TUN interface on the server to maintain the same interface name and IP address across restarts.
3. Use a fixed server address and port to ensure clients can reconnect to the same endpoint.
4. Use a service manager (e.g., systemd) to automatically restart the server if it crashes.

Example systemd service configuration:

```ini
[Unit]
Description=SSSonector Server
After=network.target

[Service]
ExecStart=/usr/local/bin/sssonector -config /etc/sssonector/server.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

This configuration will automatically restart the SSSonector server if it crashes, with a 5-second delay between restart attempts.

### Network Failure Handling

SSSonector includes mechanisms to handle various types of network failures. These mechanisms help SSSonector maintain connectivity and recover from network issues.

#### Temporary Network Outages

SSSonector can handle temporary network outages using the reconnection mechanism described above. When a network outage occurs, the client will attempt to reconnect to the server when the network becomes available again.

To optimize handling of temporary network outages, consider the following best practices:

1. Configure clients with appropriate reconnection settings.
2. Use a longer maximum delay for reconnection attempts to avoid excessive reconnection attempts during extended outages.
3. Use a service manager (e.g., systemd) to automatically restart the client if it crashes.

Example configuration:

```yaml
client:
  reconnection:
    enabled: true
    max_attempts: -1  # Unlimited attempts
    initial_delay: 5
    max_delay: 300    # 5 minutes
```

With this configuration, SSSonector will attempt to reconnect indefinitely, with an initial delay of 5 seconds between attempts. The delay will increase exponentially with each attempt, up to a maximum of 5 minutes.

#### Network Interface Changes

SSSonector can handle network interface changes, such as switching from Wi-Fi to Ethernet or changing IP addresses. When a network interface change occurs, the client will attempt to reconnect to the server using the new interface.

To optimize handling of network interface changes, consider the following best practices:

1. Configure clients with appropriate reconnection settings.
2. Use a DNS name instead of an IP address for the server address to handle IP address changes.
3. Use a service manager (e.g., systemd) to automatically restart the client if it crashes.

Example configuration:

```yaml
mode: client
server: sssonector.example.com:443  # Use DNS name instead of IP address
interface: tun0
address: 10.0.0.2/24
client:
  reconnection:
    enabled: true
    max_attempts: -1  # Unlimited attempts
    initial_delay: 1
    max_delay: 60     # 1 minute
```

With this configuration, SSSonector will attempt to reconnect indefinitely, with an initial delay of 1 second between attempts. The delay will increase exponentially with each attempt, up to a maximum of 1 minute.

#### VPN Conflicts

SSSonector can handle conflicts with other VPN software by using a unique TUN interface name and IP address range. When a conflict occurs, SSSonector will log an error and attempt to use the configured interface and address.

To avoid VPN conflicts, consider the following best practices:

1. Use a unique TUN interface name (e.g., `sssonector0` instead of `tun0`).
2. Use a unique IP address range (e.g., `10.100.0.0/24` instead of `10.0.0.0/24`).
3. Configure SSSonector to start after other VPN software to ensure it can detect and handle conflicts.

Example configuration:

```yaml
mode: client
server: 192.168.1.100:443
interface: sssonector0  # Unique interface name
address: 10.100.0.2/24  # Unique IP address range
```

### Error Recovery

SSSonector includes mechanisms to recover from errors and restore normal operation. These mechanisms help SSSonector maintain connectivity and recover from issues without manual intervention.

#### Tunnel Recovery

When a tunnel fails, SSSonector will automatically attempt to re-establish the tunnel using the reconnection mechanism described above. This helps recover from temporary network issues or server restarts.

To optimize tunnel recovery, consider the following best practices:

1. Configure clients with appropriate reconnection settings.
2. Use a persistent TUN interface on both the server and client to maintain the same interface name and IP address across restarts.
3. Use a fixed server address and port to ensure clients can reconnect to the same endpoint.
4. Use a service manager (e.g., systemd) to automatically restart SSSonector if it crashes.

#### Data Transfer Recovery

When a data transfer operation fails, SSSonector will automatically retry the operation using the retry mechanism described above. This helps recover from temporary network issues or congestion.

To optimize data transfer recovery, consider the following best practices:

1. Configure appropriate data transfer retry settings.
2. Use a smaller MTU to reduce the likelihood of fragmentation and packet loss.
3. Use TCP-based protocols for critical data transfers to leverage TCP's built-in reliability mechanisms.

Example configuration:

```yaml
network:
  mtu: 1400  # Smaller MTU to reduce fragmentation
tunnel:
  transfer:
    retry_enabled: true
    retry_max_attempts: 5
    retry_initial_delay: 200
    retry_max_delay: 500
```

## Common Error Scenarios

### Server Unreachable

When a client cannot reach the server, it will log an error and attempt to reconnect using the reconnection mechanism described above.

Example error message:
```
Error: Failed to connect to server: Connection refused
```

Troubleshooting steps:

1. Verify that the server is running.
2. Verify that the server address and port are correct.
3. Verify that there are no firewall rules blocking the connection.
4. Verify that the network is functioning properly.
5. Check the server logs for any errors.

### Certificate Validation Failure

When certificate validation fails, SSSonector will log an error and refuse the connection.

Example error message:
```
Error: Failed to validate certificate: Certificate has expired
```

Troubleshooting steps:

1. Verify that the certificates are valid and not expired.
2. Verify that the CA certificate is correct and trusted.
3. Verify that the certificate is issued for the correct domain or IP address.
4. Verify that the system time is correct on both the server and client.
5. Consider regenerating the certificates if necessary.

### Network Interface Conflict

When a network interface conflict occurs, SSSonector will log an error and attempt to use the configured interface.

Example error message:
```
Error: Failed to create TUN interface: Device or resource busy
```

Troubleshooting steps:

1. Verify that no other software is using the same TUN interface name.
2. Try using a different TUN interface name.
3. Check if any other VPN software is running and potentially causing conflicts.
4. Verify that you have the necessary permissions to create TUN interfaces.

### Packet Forwarding Failure

When packet forwarding fails, SSSonector will log an error and continue operation, but packets may not be forwarded correctly.

Example error message:
```
Error: Failed to forward packet: No route to host
```

Troubleshooting steps:

1. Verify that packet forwarding is enabled in the system.
2. Verify that the routing tables are configured correctly.
3. Verify that there are no firewall rules blocking the forwarded packets.
4. Check the network configuration on both the server and client.

## Best Practices

### Error Handling Configuration

To optimize error handling in SSSonector, consider the following best practices:

1. Configure appropriate log levels and debug categories to capture detailed error information.
2. Configure appropriate retry and reconnection settings to handle transient errors.
3. Use a service manager (e.g., systemd) to automatically restart SSSonector if it crashes.
4. Monitor logs for recurring errors and address the root causes.
5. Implement proper error handling in client applications that use SSSonector.

Example configuration:

```yaml
logging:
  level: info
  file: /var/log/sssonector.log
client:
  connection:
    retry_enabled: true
    retry_max_attempts: 5
    retry_initial_delay: 2
    retry_max_delay: 30
  reconnection:
    enabled: true
    max_attempts: -1  # Unlimited attempts
    initial_delay: 5
    max_delay: 300    # 5 minutes
tunnel:
  transfer:
    retry_enabled: true
    retry_max_attempts: 5
    retry_initial_delay: 200
    retry_max_delay: 500
```

### Monitoring and Alerting

To detect and respond to errors effectively, consider implementing monitoring and alerting for SSSonector:

1. Monitor SSSonector logs for error messages.
2. Set up alerts for critical errors or recurring issues.
3. Monitor system resources (CPU, memory, network) to detect performance issues.
4. Monitor tunnel status and connectivity to detect outages.
5. Implement health checks to verify that SSSonector is functioning properly.

Example monitoring configuration:

```yaml
monitoring:
  enabled: true
  port: 9090
```

With this configuration, SSSonector will expose monitoring metrics on port 9090, which can be scraped by monitoring tools like Prometheus.

### Disaster Recovery

To prepare for and recover from major failures, consider implementing a disaster recovery plan for SSSonector:

1. Regularly back up SSSonector configuration files and certificates.
2. Document the installation and configuration process for quick recovery.
3. Implement redundancy for critical components (e.g., multiple servers).
4. Test the recovery process regularly to ensure it works as expected.
5. Have a fallback communication method in case SSSonector is unavailable.

## Conclusion

SSSonector includes robust error handling mechanisms to handle various types of errors and maintain connectivity without manual intervention. By understanding these mechanisms and following best practices, you can configure SSSonector for optimal error handling and recovery.

For more information on other configuration options, see the [Advanced Configuration Guide](advanced_configuration_guide.md).
