# Hot Reload Configuration

SSSonector supports hot reloading of configuration, allowing you to modify settings without restarting the service or disrupting active connections.

## Supported Configuration Changes

The following settings can be modified at runtime:

1. Rate Limiting:
   - Upload and download bandwidth limits
   - Rate limiting enable/disable
   - Burst size adjustments

2. Monitoring:
   - SNMP settings
   - Logging levels
   - Update intervals

3. Connection Management:
   - Maximum client limits
   - Buffer sizes
   - Timeouts

## Unchangeable Settings

Some settings cannot be modified during runtime and require a service restart:

1. Core Settings:
   - Operating mode (server/client)
   - Network interface
   - TUN device settings

2. Security:
   - Certificate paths
   - TLS settings
   - Authentication methods

## How to Use Hot Reload

### Method 1: SIGHUP Signal

Send a SIGHUP signal to the SSSonector process to trigger a configuration reload:

```bash
# Using process ID
kill -HUP $(pidof sssonector)

# Using systemd
systemctl reload sssonector
```

### Method 2: File Monitoring

SSSonector automatically monitors its configuration file for changes. Simply modify and save the configuration file:

```bash
# Edit configuration
vim /etc/sssonector/config.yaml

# Changes will be detected and applied automatically
```

## Configuration File Example

```yaml
# Rate limiting settings (hot reloadable)
throttle:
  enabled: true
  rate_limit: 1048576    # 1 MB/s
  burst_limit: 104858    # 100ms worth of data
  dynamic:
    enabled: true
    min_rate: 524288     # 512 KB/s
    max_rate: 10485760   # 10 MB/s

# Monitoring settings (hot reloadable)
monitor:
  enabled: true
  snmp_enabled: true
  snmp_port: 161
  update_interval: 30

# Core settings (requires restart)
mode: "server"
network:
  interface: "tun0"
  address: "10.0.0.1/24"
  mtu: 1500
```

## Validation

When configuration changes are applied:

1. The new configuration is validated before application
2. Changes are applied atomically
3. Invalid changes are rejected with error logging
4. Current configuration remains active if validation fails

## Monitoring Changes

Monitor the application logs to track configuration changes:

```bash
# View configuration reload events
journalctl -u sssonector | grep "configuration"

# Check current settings via SNMP
snmpwalk -v2c -c public localhost:161 .1.3.6.1.4.1.54321.1
```

## Best Practices

1. Testing:
   - Test configuration changes in a staging environment
   - Verify settings after reload
   - Monitor system performance during changes

2. Rate Limiting:
   - Make gradual adjustments to rate limits
   - Monitor connection quality after changes
   - Consider peak usage times

3. Error Handling:
   - Check logs for validation errors
   - Verify configuration syntax before saving
   - Keep backup of working configuration

4. Security:
   - Restrict configuration file permissions
   - Use version control for tracking changes
   - Document all configuration modifications

## Troubleshooting

### Common Issues

1. Changes Not Applied:
   - Verify file permissions
   - Check syntax validity
   - Review error logs
   - Ensure changes are supported for hot reload

2. Connection Issues:
   - Monitor connection metrics
   - Check rate limiting logs
   - Verify client configurations
   - Review resource usage

3. Performance Impact:
   - Monitor system resources
   - Check connection latency
   - Verify throughput metrics
   - Review error rates

### Error Messages

1. Invalid Configuration:
```
Failed to validate configuration: invalid rate limit value
```
- Ensure values are within acceptable ranges
- Check configuration syntax

2. State Errors:
```
Cannot modify [setting] while connections are active
```
- Some settings require no active connections
- Consider scheduling changes during low usage

3. Permission Issues:
```
Failed to reload configuration: permission denied
```
- Check file permissions
- Verify process privileges

## Command Reference

### Check Current Configuration
```bash
# View effective configuration
sssonector -show-config

# Validate configuration file
sssonector -validate-config /etc/sssonector/config.yaml
```

### Monitor Changes
```bash
# Watch configuration status
watch -n 1 'sssonector -status'

# Monitor rate limiting
sssonector -metrics | grep "rate"
```

### Force Reload
```bash
# Force configuration reload
sssonector -reload

# Reload with validation
sssonector -reload -validate
```

## Example Scenarios

### 1. Adjusting Rate Limits

Original configuration:
```yaml
throttle:
  enabled: true
  rate_limit: 1048576  # 1 MB/s
```

Modified configuration:
```yaml
throttle:
  enabled: true
  rate_limit: 2097152  # 2 MB/s
```

Effect: Bandwidth limit will be adjusted without disrupting connections.

### 2. Updating Monitoring

Original configuration:
```yaml
monitor:
  update_interval: 30
  snmp_enabled: false
```

Modified configuration:
```yaml
monitor:
  update_interval: 10
  snmp_enabled: true
  snmp_port: 161
```

Effect: Monitoring settings will be updated immediately.

## Support

For issues with hot reload functionality:

1. Check the troubleshooting guide above
2. Review application logs
3. Contact support with:
   - Configuration file
   - Error messages
   - System information
   - Recent changes made
