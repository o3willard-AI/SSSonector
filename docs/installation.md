# SSSonector Installation Guide

For detailed configuration examples and best practices, please refer to our [Configuration Guide](configuration_guide.md).

## Prerequisites

### System Requirements
- Go 1.21 or later
- TUN/TAP kernel module support
- iproute2 package (Linux only)
- Administrative privileges for network interface creation

### Platform-specific Requirements

#### Linux
- Ubuntu 20.04+, CentOS 7+, or RHEL 8+
- `sudo` privileges
- Development tools: `build-essential`
- TUN/TAP kernel module loaded

#### macOS
- macOS 10.15 (Catalina) or later
- Xcode Command Line Tools
- Network Extension entitlements

#### Windows
- Windows 10 or Server 2016+
- TAP-Windows Adapter V9
- Administrator privileges
- Visual Studio Build Tools (optional)

## Installation

### From Binary Releases
1. Download the latest release for your platform from the releases page
2. Extract the archive to your desired location
3. Run the installation script:
   ```bash
   # Linux/macOS
   sudo ./install.sh

   # Windows (Run as Administrator)
   install.bat
   ```

### From Source

1. Clone the repository:
   ```bash
   git clone https://github.com/o3willard-AI/SSSonector.git
   cd SSSonector
   ```

2. Build the project:
   ```bash
   make build
   ```

3. Install the binary:
   ```bash
   sudo make install
   ```

## Configuration

For complete configuration examples and best practices, see the [Configuration Guide](configuration_guide.md).

### Certificate Setup

SSSonector provides several certificate management options through command-line flags:

```bash
# Generate production certificates
sssonector -keygen

# Run with temporary certificates (testing only)
sssonector -test-without-certs

# Generate certificates without starting service
sssonector -generate-certs-only

# Specify custom certificate directory
sssonector -keyfile /path/to/certs

# Validate existing certificates
sssonector -validate-certs
```

### Performance Tuning

Recent optimizations have improved data transfer reliability and performance. Consider these settings:

1. Buffer Size Configuration:
   ```yaml
   buffer:
     read_size: 65536    # Optimized for large transfers
     write_size: 65536   # Matched with read size
     pool_size: 1024     # Increased for better performance
   ```

2. Rate Limiting (optional):
   ```yaml
   throttle:
     enabled: true
     rate_mbps: 100      # Adjust based on needs
     burst_size: 1048576 # Optimized for bursty traffic
   ```

3. Retry Settings:
   ```yaml
   retry:
     max_attempts: 3     # Number of retry attempts
     base_delay_ms: 50   # Base delay between retries
     max_delay_ms: 1000  # Maximum backoff delay
   ```

### Monitoring Configuration

Enhanced monitoring capabilities are now available:

```yaml
monitor:
  enabled: true
  metrics:
    interval_seconds: 10
    detailed_logging: true
  snmp:
    enabled: true
    port: 161
    community: "public"
  error_tracking:
    enabled: true
    detailed: true
```

## Post-Installation

### Verify Installation
```bash
# Check version and build info
sssonector -version

# Validate configuration
sssonector -validate-config

# Test certificate setup
sssonector -validate-certs
```

### Start the Service

#### Linux (systemd)
```bash
sudo systemctl start sssonector
sudo systemctl enable sssonector  # Start on boot
```

#### macOS
```bash
sudo launchctl load /Library/LaunchDaemons/com.sssonector.plist
```

#### Windows
```powershell
Start-Service SSonector
Set-Service SSonector -StartupType Automatic
```

### Verify Operation
```bash
# Check service status
sssonector -status

# View metrics
sssonector -metrics

# Test connection
sssonector -test
```

## Troubleshooting

### Common Issues

1. Connection Drops
   - Check network stability
   - Verify certificate validity
   - Review error logs for retry patterns

2. Performance Issues
   - Adjust buffer sizes
   - Check rate limiting configuration
   - Monitor system resources

3. Certificate Problems
   - Verify certificate paths
   - Check certificate expiration
   - Validate certificate chain

### Logging

Enhanced logging is now available:
```bash
# Enable detailed logging
sssonector -log-level debug

# View performance metrics
sssonector -metrics-detail

# Monitor SNMP data
snmpwalk -v2c -c public localhost .1.3.6.1.4.1.X
```

## Upgrading

### From Previous Versions

1. Backup configuration:
   ```bash
   cp /etc/sssonector/config.yaml /etc/sssonector/config.yaml.bak
   ```

2. Update the software:
   ```bash
   # Using package manager
   sudo apt-get update && sudo apt-get upgrade sssonector

   # Or from source
   git pull
   make clean && make build
   sudo make install
   ```

3. Update configuration:
   ```bash
   sssonector -update-config
   ```

4. Restart service:
   ```bash
   sudo systemctl restart sssonector
   ```

## Support

For additional help:
- Documentation: https://docs.sssonector.io
- Issues: https://github.com/o3willard-AI/SSSonector/issues
- Community: https://community.sssonector.io
