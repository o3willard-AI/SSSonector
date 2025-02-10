# Rate Limiting Configuration Guide

This guide explains how to configure SSSonector's rate limiting features for optimal performance and resource utilization.

## Configuration Parameters

### Basic Settings

- `enabled` (boolean, default: true)
  * Enables or disables rate limiting
  * When disabled, no throttling is applied
  * Example: `enabled: true`

- `rate` (uint64, required)
  * Base rate in bytes per second
  * Used as reference for dynamic adjustments
  * Example: `rate: 1048576  # 1 MB/s`

### Dynamic Rate Limiting

The dynamic rate limiter automatically adjusts rates based on network conditions:

- Default bounds:
  * Minimum rate: 50% of base rate
  * Maximum rate: 200% of base rate
  * Burst size: 10% of current rate
  * Cooldown: 1 second between adjustments

## Configuration Examples

### 1. Basic Configuration
```yaml
throttle:
  enabled: true
  rate: 1048576    # 1 MB/s base rate
```

### 2. High-Performance Configuration
```yaml
throttle:
  enabled: true
  rate: 104857600  # 100 MB/s base rate
  buffer:
    size: 65536    # 64 KB buffer
    count: 1000    # Buffer pool size
```

### 3. Server Configuration
```yaml
throttle:
  enabled: true
  rate: 52428800   # 50 MB/s total
  per_client:
    enabled: true
    rate: 5242880  # 5 MB/s per client
```

## Monitoring Configuration

### SNMP Monitoring
```yaml
monitoring:
  enabled: true
  snmp_enabled: true
  metrics:
    include_rate_limits: true
    update_interval: 1
```

### Logging Configuration
```yaml
logging:
  level: info
  rate_limiting:
    track_individual: true
    track_adjustments: true
```

## Performance Tuning

### Buffer Configuration
```yaml
buffer:
  read_size: 65536    # 64 KB read buffer
  write_size: 65536   # 64 KB write buffer
  pool_size: 1000     # Buffer pool size
  prealloc: true      # Preallocate buffers
```

### System Settings
```bash
# Recommended sysctl settings
net.core.rmem_max=16777216
net.core.wmem_max=16777216
net.ipv4.tcp_rmem="4096 87380 16777216"
net.ipv4.tcp_wmem="4096 87380 16777216"
```

## Monitoring and Maintenance

### 1. Monitoring Commands
```bash
# Check current rates
sssonector -metrics | grep "rate"

# Monitor rate adjustments
tail -f /var/log/sssonector/rate.log

# SNMP monitoring
snmpwalk -v2c -c public localhost:10161 .1.3.6.1.4.1.54321.1.3
```

### 2. Validation
```bash
# Validate configuration
sssonector -validate-config

# Test throughput
iperf3 -c <server> -t 30 -J
```

### 3. Maintenance Tasks
- Monitor rate limit hits
- Review dynamic adjustment logs
- Check system resource usage
- Verify client experience

## Troubleshooting

### Common Issues

1. High Rate Limit Hits
   - Check current utilization
   - Verify base rate configuration
   - Monitor network conditions
   - Review adjustment patterns

2. Performance Issues
   - Check buffer configurations
   - Monitor system resources
   - Verify network capacity
   - Review TCP settings

3. Configuration Issues
   - Validate configuration file
   - Check log files for errors
   - Verify permissions
   - Test network connectivity

## Best Practices

1. Rate Configuration
   - Set appropriate base rates
   - Consider network capacity
   - Account for client needs
   - Plan for peak usage

2. Monitoring
   - Enable detailed metrics
   - Set up alerts
   - Regular log review
   - Performance monitoring

3. Security
   - Implement rate protection
   - Monitor for abuse
   - Regular audits
   - Access control

4. Maintenance
   - Regular configuration review
   - Performance testing
   - System updates
   - Backup configurations

## Advanced Configuration

### 1. Custom Adjustment Strategy
```yaml
throttle:
  enabled: true
  rate: 1048576
  dynamic:
    strategy: "adaptive"
    check_interval: "5s"
    sensitivity: 0.8
```

### 2. High Availability Setup
```yaml
throttle:
  enabled: true
  rate: 52428800
  ha:
    enabled: true
    sync_interval: "1s"
    failover: true
```

### 3. QoS Integration
```yaml
throttle:
  enabled: true
  rate: 10485760
  qos:
    enabled: true
    priority_classes: 4
    min_guaranteed: true
```

## Security Considerations

1. Rate Limit Protection
   - Set appropriate limits
   - Monitor for abuse
   - Implement alerts
   - Regular audits

2. Access Control
   - Secure configurations
   - Monitor changes
   - Audit logging
   - Role-based access

3. Network Security
   - Firewall rules
   - Traffic monitoring
   - Intrusion detection
   - Regular updates
