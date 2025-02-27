# SSSonector Logging Guide

This guide provides detailed information about logging in SSSonector, including configuration options, log formats, debug categories, and best practices for effective logging.

## Logging Configuration

SSSonector provides flexible logging configuration options to help you troubleshoot issues and monitor the system's operation. The following sections describe the available logging configuration options.

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
  - `network`: Network-related debug logs, including packet forwarding, interface configuration, and network errors.
  - `tls`: TLS-related debug logs, including handshake details, certificate validation, and encryption/decryption operations.
  - `tunnel`: Tunnel-related debug logs, including tunnel establishment, data transfer, and tunnel closure.
  - `config`: Configuration-related debug logs, including configuration file parsing, validation, and application.
  - `all`: All debug categories.
- **Default**: `["all"]`
- **Example**: `logging.debug_categories: ["network", "tunnel"]`
- **Notes**:
  - Enabling specific categories can reduce log verbosity while still providing useful debugging information.
  - For comprehensive debugging, use `["all"]`.
  - When troubleshooting specific issues, enable only the relevant categories to reduce log volume.

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
  - JSON format includes additional metadata that can be useful for log analysis.

## Log Message Structure

### Text Format

In text format, log messages have the following structure:

```
YYYY-MM-DD HH:MM:SS.sss [LEVEL] [CATEGORY] Message
```

Where:
- `YYYY-MM-DD HH:MM:SS.sss` is the timestamp.
- `LEVEL` is the log level (DEBUG, INFO, WARNING, ERROR).
- `CATEGORY` is the log category (network, tls, tunnel, config).
- `Message` is the log message.

Example:
```
2025-02-26 15:30:45.123 [DEBUG] [network] Forwarding packet: src=10.0.0.1, dst=10.0.0.2, proto=TCP, size=1500
```

### JSON Format

In JSON format, log messages have the following structure:

```json
{
  "timestamp": "YYYY-MM-DD HH:MM:SS.sss",
  "level": "LEVEL",
  "category": "CATEGORY",
  "message": "Message",
  "metadata": {
    "key1": "value1",
    "key2": "value2",
    ...
  }
}
```

Where:
- `timestamp` is the timestamp.
- `level` is the log level (debug, info, warning, error).
- `category` is the log category (network, tls, tunnel, config).
- `message` is the log message.
- `metadata` is an optional object containing additional information about the log message.

Example:
```json
{
  "timestamp": "2025-02-26 15:30:45.123",
  "level": "debug",
  "category": "network",
  "message": "Forwarding packet",
  "metadata": {
    "src": "10.0.0.1",
    "dst": "10.0.0.2",
    "proto": "TCP",
    "size": 1500
  }
}
```

## Debug Categories

### Network Category

The `network` category includes logs related to network operations, such as:

- Interface configuration
- IP address assignment
- Packet forwarding
- MTU configuration
- Network errors
- Routing information

Example log messages:
```
2025-02-26 15:30:45.123 [DEBUG] [network] Creating TUN interface: tun0
2025-02-26 15:30:45.124 [DEBUG] [network] Assigning IP address: 10.0.0.1/24
2025-02-26 15:30:45.125 [DEBUG] [network] Setting MTU: 1500
2025-02-26 15:30:45.126 [DEBUG] [network] Enabling packet forwarding
2025-02-26 15:30:45.127 [DEBUG] [network] Forwarding packet: src=10.0.0.1, dst=10.0.0.2, proto=TCP, size=1500
```

### TLS Category

The `tls` category includes logs related to TLS operations, such as:

- Certificate loading
- Certificate validation
- TLS handshake
- Encryption/decryption operations
- TLS errors
- Cipher suite negotiation

Example log messages:
```
2025-02-26 15:30:45.123 [DEBUG] [tls] Loading certificate: certs/server.crt
2025-02-26 15:30:45.124 [DEBUG] [tls] Loading private key: certs/server.key
2025-02-26 15:30:45.125 [DEBUG] [tls] Loading CA certificate: certs/ca.crt
2025-02-26 15:30:45.126 [DEBUG] [tls] TLS handshake started
2025-02-26 15:30:45.127 [DEBUG] [tls] Negotiated cipher suite: TLS_AES_256_GCM_SHA384
2025-02-26 15:30:45.128 [DEBUG] [tls] TLS handshake completed
```

### Tunnel Category

The `tunnel` category includes logs related to tunnel operations, such as:

- Tunnel establishment
- Data transfer
- Tunnel closure
- Tunnel errors
- Tunnel statistics

Example log messages:
```
2025-02-26 15:30:45.123 [DEBUG] [tunnel] Tunnel establishment started
2025-02-26 15:30:45.124 [DEBUG] [tunnel] Tunnel established: local=10.0.0.1, remote=10.0.0.2
2025-02-26 15:30:45.125 [DEBUG] [tunnel] Data transfer started
2025-02-26 15:30:45.126 [DEBUG] [tunnel] Data transferred: bytes=1500, direction=outbound
2025-02-26 15:30:45.127 [DEBUG] [tunnel] Data transferred: bytes=1000, direction=inbound
2025-02-26 15:30:45.128 [DEBUG] [tunnel] Tunnel closed
```

### Config Category

The `config` category includes logs related to configuration operations, such as:

- Configuration file loading
- Configuration validation
- Configuration application
- Environment variable processing
- Configuration errors

Example log messages:
```
2025-02-26 15:30:45.123 [DEBUG] [config] Loading configuration file: /etc/sssonector/sssonector.yaml
2025-02-26 15:30:45.124 [DEBUG] [config] Validating configuration
2025-02-26 15:30:45.125 [DEBUG] [config] Processing environment variables
2025-02-26 15:30:45.126 [DEBUG] [config] Applying configuration
2025-02-26 15:30:45.127 [DEBUG] [config] Configuration applied successfully
```

## Log Rotation

SSSonector does not handle log rotation internally. It is recommended to use external tools like `logrotate` to manage log rotation. Here's an example `logrotate` configuration for SSSonector:

```
/var/log/sssonector.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 0640 root root
    postrotate
        systemctl reload sssonector
    endscript
}
```

This configuration rotates the log file daily, keeps 7 days of logs, compresses old logs, and reloads SSSonector after rotation.

## Common Logging Scenarios

### Debugging Connection Issues

To debug connection issues, enable debug logging for the `network` and `tunnel` categories:

```yaml
logging:
  level: debug
  debug_categories: ["network", "tunnel"]
  file: /var/log/sssonector.log
```

This configuration will provide detailed information about network and tunnel operations, helping you diagnose connection issues.

### Debugging TLS Issues

To debug TLS issues, enable debug logging for the `tls` category:

```yaml
logging:
  level: debug
  debug_categories: ["tls"]
  file: /var/log/sssonector.log
```

This configuration will provide detailed information about TLS operations, helping you diagnose TLS issues.

### Debugging Configuration Issues

To debug configuration issues, enable debug logging for the `config` category:

```yaml
logging:
  level: debug
  debug_categories: ["config"]
  file: /var/log/sssonector.log
```

This configuration will provide detailed information about configuration operations, helping you diagnose configuration issues.

### Comprehensive Debugging

For comprehensive debugging, enable debug logging for all categories:

```yaml
logging:
  level: debug
  debug_categories: ["all"]
  file: /var/log/sssonector.log
```

This configuration will provide detailed information about all operations, helping you diagnose complex issues.

### Production Logging

For production environments, use a more conservative logging configuration:

```yaml
logging:
  level: info
  file: /var/log/sssonector.log
  format: json
```

This configuration logs only informational, warning, and error messages, reducing log volume while still providing useful information. The JSON format makes it easier to process logs with automated tools.

## Environment Variables

SSSonector also supports configuration of logging through environment variables:

- `SSSONECTOR_LOGGING_LEVEL`: Specifies the logging level.
- `SSSONECTOR_LOGGING_FILE`: Specifies the log file.
- `SSSONECTOR_LOGGING_DEBUG_CATEGORIES`: Specifies the debug categories (comma-separated list).
- `SSSONECTOR_LOGGING_FORMAT`: Specifies the log format (`text` or `json`).

Example usage:

```bash
export SSSONECTOR_LOGGING_LEVEL=debug
export SSSONECTOR_LOGGING_FILE=/var/log/sssonector.log
export SSSONECTOR_LOGGING_DEBUG_CATEGORIES=network,tunnel
export SSSONECTOR_LOGGING_FORMAT=text

./sssonector
```

## Best Practices

### Log Level Selection

- Use `debug` level only for troubleshooting specific issues.
- Use `info` level for normal operation in production environments.
- Use `warning` level for minimal logging in production environments.
- Use `error` level only if you want to log only errors.

### Log File Management

- Use a dedicated log file for SSSonector logs.
- Configure log rotation to prevent disk space issues.
- Monitor log file size and growth rate.
- Archive old logs for historical analysis.

### Debug Category Selection

- Enable only the relevant debug categories for specific troubleshooting.
- Use `all` category only for comprehensive debugging.
- Disable debug logging when not needed to reduce log volume.

### Log Format Selection

- Use `text` format for manual inspection and troubleshooting.
- Use `json` format for automated log processing and analysis.
- Consider using log processing tools like ELK (Elasticsearch, Logstash, Kibana) for advanced log analysis.

## Conclusion

SSSonector provides flexible logging configuration options to help you troubleshoot issues and monitor the system's operation. By understanding and properly configuring the logging options, you can effectively diagnose and resolve issues with SSSonector.

For more information on other configuration options, see the [Advanced Configuration Guide](advanced_configuration_guide.md).
