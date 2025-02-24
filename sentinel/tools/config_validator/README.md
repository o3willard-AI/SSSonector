# Configuration Validator Tool

Part of Project SENTINEL - SSSonector ENvironment Testing & Integration Layer

## Overview

The configuration validator ensures consistent and secure configuration across all SSSonector deployments. It validates configuration files against required fields, type-specific requirements, and security policies.

## Features

- Configuration structure validation
- Security policy enforcement
- Type-specific validation (server/client)
- Certificate path verification
- Permission checking
- Value validation

## Configuration Requirements

### Common Required Fields
```yaml
type: "server|client"          # Deployment type
logging:
  level: "debug|info|warn|error"
  output: "file|stdout"
```

### Server-Specific Fields
```yaml
server:
  listen_address: "string"     # IP address or hostname
  listen_port: number         # Valid port (1-65535)
  max_connections: number     # Maximum concurrent connections
```

### Client-Specific Fields
```yaml
client:
  server_address: "string"    # Server IP or hostname
  server_port: number        # Server port (1-65535)
  retry_interval: number     # Reconnection interval
```

## Security Policies

| Resource | Required Permission |
|----------|-------------------|
| Config files | 644 |
| Certificate files (.crt) | 644 |
| Key files (.key) | 600 |

## Usage

```bash
# Make script executable
chmod +x config_validator.sh

# Validate a configuration file
./config_validator.sh -c /path/to/config.yaml

# Validate against a template
./config_validator.sh -c /path/to/config.yaml -t /path/to/template.yaml
```

## Exit Codes

- 0: Validation successful
- 1: Validation failed (missing fields, invalid values, security issues)

## Integration

This tool is typically used:
1. Before deploying configurations
2. During environment setup
3. As part of CI/CD pipelines
4. During configuration updates

## Error Handling

The tool provides detailed error messages for:
- Missing required fields
- Invalid field values
- Security policy violations
- Permission issues
- Certificate problems

## Dependencies

- bash
- yq (for YAML processing)
- standard Unix tools (stat, grep)

## Version History

- 1.0.0: Initial release
  - Basic configuration validation
  - Security policy enforcement
  - Type-specific validation
  - Certificate verification
