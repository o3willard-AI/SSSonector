# SSSonector Deployment Patterns Guide

## Core Deployment Model

SSSonector follows a simple, standalone deployment model:
- Single binary executable
- Configuration file driven
- No system service integration required
- Direct filesystem deployment

### Basic Structure
```
/path/to/sssonector/
├── sssonector           # The binary executable
├── config/
│   └── config.yaml      # Configuration file
├── certs/               # Generated certificates
│   ├── ca.crt
│   ├── server.crt
│   ├── server.key
│   ├── client.crt
│   └── client.key
└── log/                 # Optional log directory
    └── sssonector.log
```

### Deployment Steps

1. Binary Placement
   - Copy the appropriate platform binary to desired location
   - Set executable permissions (chmod 755)
   - No installation or registration required

2. Certificate Generation
   ```bash
   # Generate certificates without starting the tunnel
   ./sssonector -generate-certs -cert-dir ./certs
   ```

3. Configuration
   - Create config.yaml in an accessible location
   - Specify all operational parameters in config
   - Config determines server/client mode
   - Config controls foreground/background operation

4. Operation
   ```bash
   # Start SSSonector with config file
   ./sssonector -config ./config/config.yaml [-debug] [-v]
   ```

### Version Management

Each SSSonector binary includes:
- Version number (from git tag)
- Build timestamp
- Git commit hash

Version information can be queried:
```bash
./sssonector --version
```

### Binary Distribution
```
versions/
├── v2.0.0-82-ge5bd185/
│   ├── linux-amd64/
│   │   ├── sssonector
│   │   └── sssonector.sha256
│   ├── linux-arm64/
│   ├── darwin-amd64/
│   └── windows-amd64/
└── README.md
```

### Version Verification
```bash
# Check binary version
./sssonector --version

# Verify checksum
sha256sum -c sssonector.sha256

# Check binary type
file sssonector
```

## Startup Logging

### Basic Configuration
```yaml
# Basic startup logging configuration
logging:
  startup_logs: true
  level: info
  format: json
  output: file
  file: ./log/sssonector.log
```

### Development Configuration
```yaml
logging:
  startup_logs: true
  level: debug
  format: json
  output: stdout
  version_info: true  # Include version in all logs
```

### Production Configuration
```yaml
logging:
  startup_logs: true
  level: info
  format: json
  output: file
  file: ./log/sssonector.log
  version_info: true  # Include version in all logs
  rotation:
    max_size: 100MB
    max_age: 30d
    max_backups: 10
    compress: true
```

## Best Practices

### Deployment
1. File Organization
   - Keep binary and config in known location
   - Maintain consistent directory structure
   - Use absolute paths in config
   - Secure certificate storage

2. Version Management
   - Track deployed versions
   - Maintain checksums
   - Document configuration changes
   - Keep previous versions available

3. Operation
   - Test configuration before deployment
   - Use debug logging during setup
   - Monitor log output
   - Maintain backup configurations

4. Security
   - Secure certificate permissions
   - Restrict config file access
   - Use absolute paths
   - Validate configurations before use

### Configuration
1. Use clear, documented config files
2. Validate config before deployment
3. Maintain separate configs for different environments
4. Document all custom settings

### Monitoring
1. Check log files regularly
2. Monitor tunnel status
3. Track version information
4. Verify certificate validity

### Maintenance
1. Keep binary updated
2. Rotate logs appropriately
3. Backup configurations
4. Document changes

## Common Patterns

### Development Setup
```bash
./sssonector -config ./config/dev.yaml -debug -v
```

### Production Setup
```bash
./sssonector -config /opt/sssonector/config/prod.yaml
```

### Testing Setup
```bash
./sssonector -config ./config/test.yaml -debug
```

## Troubleshooting

### Common Issues
1. Certificate Problems
   - Check certificate locations
   - Verify permissions
   - Validate certificate dates

2. Configuration Issues
   - Validate config syntax
   - Check file permissions
   - Verify paths are absolute

3. Network Issues
   - Check TUN interface
   - Verify IP forwarding
   - Test network connectivity

### Debug Steps
1. Run with -debug flag
2. Check log output
3. Verify configuration
4. Test network setup
