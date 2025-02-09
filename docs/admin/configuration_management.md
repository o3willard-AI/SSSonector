# SSSonector Configuration Management Guide

This guide provides system administrators with detailed information on managing SSSonector configurations in production environments.

## Dynamic Configuration Updates

SSSonector supports dynamic configuration updates, allowing you to modify certain settings without service interruption or connection loss.

### Supported Dynamic Changes

✅ Can be changed without restart:
- Rate limiting settings
- Monitoring parameters
- Connection limits
- Buffer sizes
- SNMP settings

❌ Requires restart:
- Operating mode (server/client)
- Network interface settings
- TLS/Certificate configurations
- Authentication methods

## Configuration Methods

### 1. Direct File Edit

```bash
# Edit configuration file
sudo vim /etc/sssonector/config.yaml

# Changes are automatically detected and applied
```

### 2. SIGHUP Signal

```bash
# Get process ID
pid=$(systemctl show --property MainPID --value sssonector)

# Send reload signal
sudo kill -HUP $pid

# Or using systemctl
sudo systemctl reload sssonector
```

## Configuration Examples

### Rate Limiting Adjustment

```yaml
# Before
throttle:
  enabled: true
  rate_limit: 1048576    # 1 MB/s
  burst_limit: 104858    # 100ms worth of data

# After
throttle:
  enabled: true
  rate_limit: 2097152    # 2 MB/s
  burst_limit: 209716    # 100ms worth of data
```

### Monitoring Changes

```yaml
# Before
monitor:
  enabled: true
  snmp_enabled: false
  update_interval: 30

# After
monitor:
  enabled: true
  snmp_enabled: true
  snmp_port: 161
  update_interval: 10
```

## Validation and Safety

### Pre-change Validation

```bash
# Validate configuration before applying
sssonector -validate-config /path/to/new/config.yaml

# Show current effective configuration
sssonector -show-config
```

### Monitoring Changes

```bash
# Watch configuration status
watch -n 1 'sssonector -status'

# Monitor logs for changes
journalctl -u sssonector -f | grep "configuration"

# Check SNMP metrics
snmpwalk -v2c -c public localhost:161 .1.3.6.1.4.1.54321.1
```

## Best Practices

### 1. Change Management

✅ Do:
- Test changes in staging first
- Make one change at a time
- Monitor system during changes
- Keep backup of working config
- Document all changes

❌ Don't:
- Make multiple changes simultaneously
- Edit configs during peak hours
- Forget to validate changes
- Ignore monitoring alerts

### 2. Rate Limiting Changes

✅ Do:
- Make gradual adjustments
- Monitor connection quality
- Consider peak usage times
- Validate actual throughput

❌ Don't:
- Make drastic changes
- Ignore user feedback
- Forget TCP overhead
- Skip validation

### 3. Monitoring Updates

✅ Do:
- Verify metric collection
- Check log rotation
- Test alert systems
- Monitor resource usage

❌ Don't:
- Disable critical metrics
- Ignore error rates
- Skip validation
- Remove audit logs

## Troubleshooting

### Common Issues

1. Changes Not Applied
```
# Check file permissions
ls -l /etc/sssonector/config.yaml

# Verify syntax
sssonector -validate-config /etc/sssonector/config.yaml

# Check process status
systemctl status sssonector
```

2. Performance Issues
```
# Monitor system resources
top -p $(pidof sssonector)

# Check connection metrics
sssonector -metrics

# View detailed logs
journalctl -u sssonector -n 100
```

3. Connection Problems
```
# Check rate limiting
sssonector -metrics | grep "rate"

# Monitor interfaces
ip -s link show dev tun0

# View error logs
tail -f /var/log/sssonector/error.log
```

### Error Messages

1. Configuration Errors
```
Failed to validate configuration: invalid rate limit value
- Check value ranges
- Verify syntax
- Review documentation
```

2. Permission Issues
```
Failed to reload configuration: permission denied
- Check file ownership
- Verify process privileges
- Review SELinux context
```

3. Resource Issues
```
Failed to apply changes: insufficient resources
- Check system resources
- Monitor memory usage
- Review connection limits
```

## Monitoring Tools

### 1. Built-in Tools

```bash
# Show current status
sssonector -status

# View metrics
sssonector -metrics

# Check configuration
sssonector -show-config
```

### 2. System Tools

```bash
# Monitor process
ps aux | grep sssonector

# Check resource usage
top -p $(pidof sssonector)

# View network stats
netstat -anp | grep sssonector
```

### 3. Log Analysis

```bash
# View service logs
journalctl -u sssonector

# Check error logs
tail -f /var/log/sssonector/error.log

# Monitor config changes
grep "configuration" /var/log/sssonector/audit.log
```

## Automation Examples

### 1. Monitoring Script

```bash
#!/bin/bash
# monitor_config.sh

while true; do
    # Get current metrics
    metrics=$(sssonector -metrics)
    
    # Check rate limits
    current_rate=$(echo "$metrics" | grep "rate_limit")
    
    # Log changes
    echo "$(date): $current_rate" >> /var/log/sssonector/rate_history.log
    
    sleep 60
done
```

### 2. Validation Script

```bash
#!/bin/bash
# validate_changes.sh

config_file="/etc/sssonector/config.yaml"
backup_file="/etc/sssonector/config.yaml.bak"

# Backup current config
cp "$config_file" "$backup_file"

# Validate new config
if ! sssonector -validate-config "$config_file"; then
    echo "Validation failed, restoring backup"
    cp "$backup_file" "$config_file"
    exit 1
fi

# Trigger reload
systemctl reload sssonector
```

### 3. Health Check

```bash
#!/bin/bash
# health_check.sh

# Check process
if ! pgrep sssonector > /dev/null; then
    echo "Process not running"
    exit 1
fi

# Check metrics
if ! sssonector -metrics > /dev/null; then
    echo "Metrics unavailable"
    exit 1
fi

# Verify configuration
if ! sssonector -validate-config /etc/sssonector/config.yaml; then
    echo "Configuration invalid"
    exit 1
fi

echo "Health check passed"
exit 0
```

## Support and Resources

### Getting Help

1. Check Documentation:
   - Implementation guide
   - Troubleshooting guide
   - FAQ section

2. System Information:
   - Configuration file
   - Log files
   - Error messages
   - System metrics

3. Contact Support:
   - Configuration file
   - Error messages
   - Recent changes
   - System logs

### Additional Resources

1. Documentation:
   - [Implementation Guide](../implementation/hot_reload_design.md)
   - [Rate Limiting Guide](../rate_limiting_implementation.md)
   - [Monitoring Guide](../monitoring/README.md)

2. Tools:
   - Configuration validator
   - Metric collectors
   - Health checks
   - Automation scripts

3. Community:
   - Issue tracker
   - Discussion forums
   - Knowledge base
   - Best practices
