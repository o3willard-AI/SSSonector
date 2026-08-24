# SSSonector Configuration Management Guide

This guide provides system administrators with detailed information on managing SSSonector configurations in production environments.

## Dynamic Configuration Updates

SSSonector supports dynamic configuration updates, allowing you to modify certain settings without service interruption or connection loss.

### Supported Dynamic Changes (SIGHUP)

✅ Applied live on reload:
- `throttle.enabled`, `throttle.rate` (live and future transfers)
- `logging.level` (unless overridden by the `-log-level` startup flag)
- `auth.cert_rotation.interval`

⚠️ Detected and logged as "requires restart" warnings:
- Operating mode, network/TUN settings, tunnel ports
- Facade enablement/ports/secrets
- Logging file/format (logger sinks are fixed at startup)
- Monitor type, Prometheus port, SNMP endpoint settings, metrics interval
- Certificate paths and TLS material

See [Hot Reload](../hot_reload.md) for the complete contract.

## Configuration Methods

### 1. Direct File Edit + SIGHUP

```bash
# Edit configuration file
sudo vim /etc/sssonector/config.yaml

# Then trigger the reload - edits are NOT detected automatically
sudo systemctl reload sssonector
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
  rate: 1048576    # ~1.05 MB/s effective after TCP overhead factor

# After
throttle:
  enabled: true
  rate: 2097152    # ~2.1 MB/s effective
```

### Log Level Change

```yaml
# Before
config:
  logging:
    level: info

# After
config:
  logging:
    level: debug
```

Note: endpoint topology changes (`monitor.type`, `snmp.*`, prometheus port)
require a restart; reload logs them as warnings.

## Validation and Safety

### Pre-change Validation

```bash
# The daemon validates on every SIGHUP itself; rejected reloads are logged
# at ERROR with the offending field and the old config stays active.
journalctl -u sssonector | grep -E "reload|restart"
```

### Monitoring Changes

```bash
# Monitor logs for reload events
journalctl -u sssonector -f | grep -E "reloaded|requires restart|Log level"

# Check SNMP metrics
snmpwalk -v2c -c public localhost:10162 .1.3.6.1.4.1.54321.1

# Or scrape Prometheus directly
curl -s http://localhost:9090/metrics | grep sssonector_
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

The scripts below are **illustrative examples** — they are not shipped with
the repository. Create them in your own tooling directory if you want this
behavior, adapting paths and thresholds to your environment.

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
