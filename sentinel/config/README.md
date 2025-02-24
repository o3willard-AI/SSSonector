# Configuration Templates

Part of Project SENTINEL - SSSonector ENvironment Testing & Integration Layer

## Overview

These configuration templates provide standardized, secure defaults for SSSonector deployments. They serve as both reference configurations and validation templates for the configuration validator tool.

## Templates

### Server Template (`server_template.yaml`)
Base configuration for SSSonector server deployments:
- Network configuration
- TLS/Certificate settings
- Performance tuning
- Monitoring setup
- Security policies
- State management

### Client Template (`client_template.yaml`)
Base configuration for SSSonector client deployments:
- Server connection settings
- TLS/Certificate settings
- Performance tuning
- Connection recovery
- Monitoring setup
- Security policies

## Usage

1. Copy the appropriate template:
```bash
# For server deployment
cp server_template.yaml /opt/sssonector/config/server.yaml

# For client deployment
cp client_template.yaml /opt/sssonector/config/client.yaml
```

2. Modify the configuration:
```bash
# Edit with appropriate values
vim /opt/sssonector/config/server.yaml
```

3. Validate the configuration:
```bash
# Using config validator
../tools/config_validator/config_validator.sh -c /opt/sssonector/config/server.yaml -t server_template.yaml
```

## Security Considerations

1. File Permissions
   - Configuration files: 644
   - Certificate files (.crt): 644
   - Private key files (.key): 600

2. Sensitive Information
   - Never commit configurations with real credentials
   - Use environment variables for sensitive values
   - Keep private keys secure

3. Network Security
   - Use appropriate network restrictions
   - Enable TLS verification
   - Configure allowed networks/fingerprints

## Template Structure

### Common Sections
```yaml
# Basic identification
type: "server|client"

# Logging configuration
logging:
  level: "info"
  output: "file"
  ...

# TLS/Certificate configuration
certs:
  ca_cert: "/path/to/ca.crt"
  cert: "/path/to/cert.crt"
  key: "/path/to/key.key"
  ...

# Performance tuning
performance:
  read_buffer_size: 4096
  write_buffer_size: 4096
  ...

# Monitoring configuration
monitoring:
  enabled: true
  metrics_port: 9090|9091
  ...
```

## Customization Guidelines

1. Required Changes
   - Server addresses and ports
   - Certificate paths
   - Log file locations
   - Network restrictions

2. Optional Tuning
   - Buffer sizes
   - Worker pool sizes
   - Rate limits
   - Monitoring settings

3. Environment-Specific Settings
   - Debug options
   - Performance tuning
   - Monitoring configuration
   - Log levels

## Version Control

- Templates are versioned with SSSonector
- Breaking changes increment major version
- Additions increment minor version
- Documentation updates increment patch version

## Dependencies

- yq (for YAML processing)
- config_validator tool
- setup_directory_structure tool

## Integration

These templates integrate with:
1. Configuration validator tool
2. Directory structure setup
3. Deployment scripts
4. Monitoring systems

## Testing

Templates should be tested:
1. With config_validator tool
2. In development environments
3. With security scanners
4. Against known good configurations
