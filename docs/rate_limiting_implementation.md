# Rate Limiting Implementation Guide

This document details the rate limiting implementation in SSSonector v2.0.0, including configuration examples and best practices.

## Overview

SSSonector implements a token bucket rate limiting algorithm with the following features:
- Independent ingress and egress rate limiting
- Configurable burst allowance
- Per-connection and global limits
- Dynamic rate adjustment
- SNMP monitoring integration

## Implementation Details

### Token Bucket Algorithm

The rate limiter uses a token bucket implementation with the following characteristics:

1. Token Generation
   - Tokens are generated at a fixed rate (rate limit)
   - Each token represents one byte of data transfer
   - Tokens accumulate up to the burst limit

2. Packet Processing
   - Each packet consumes tokens equal to its size
   - If insufficient tokens are available, the packet is delayed
   - Bursts are allowed up to the configured burst limit

### Code Structure

```go
type Limiter struct {
    rate       int64         // Tokens per second
    burst      int64         // Maximum burst size
    tokens     int64         // Current token count
    lastUpdate time.Time     // Last token update time
    mu         sync.Mutex    // Mutex for thread safety
}

type RateLimiter interface {
    Allow(n int64) bool      // Check if n bytes can be transferred
    Wait(ctx context.Context, n int64) error  // Wait for n tokens
    Update(newRate int64)    // Update rate limit
}
```

## Configuration Examples

### Example 1: Basic Rate Limiting
```yaml
throttle:
  enabled: true
  rate_limit: 1048576    # 1 MB/s
  burst_limit: 2097152   # 2 MB burst
```

### Example 2: Asymmetric Rate Limiting
```yaml
throttle:
  enabled: true
  upload_rate: 5242880   # 5 MB/s upload
  upload_burst: 10485760 # 10 MB upload burst
  download_rate: 10485760 # 10 MB/s download
  download_burst: 20971520 # 20 MB download burst
  per_connection: true
```

### Example 3: Dynamic Rate Limiting
```yaml
throttle:
  enabled: true
  rate_limit: 1048576    # Base rate: 1 MB/s
  burst_limit: 2097152   # Base burst: 2 MB
  dynamic:
    enabled: true
    min_rate: 524288     # Minimum: 512 KB/s
    max_rate: 10485760   # Maximum: 10 MB/s
    adjustment_interval: 5 # Check every 5 seconds
    increase_threshold: 0.8 # Increase if utilization > 80%
    decrease_threshold: 0.2 # Decrease if utilization < 20%
```

## Usage Examples

### Example 1: Server with Multiple Clients
```yaml
mode: "server"
tunnel:
  listen_address: "0.0.0.0"
  listen_port: 8443
  max_clients: 10
throttle:
  enabled: true
  global_rate: 52428800  # 50 MB/s total
  global_burst: 104857600 # 100 MB burst total
  per_client:
    rate: 5242880        # 5 MB/s per client
    burst: 10485760      # 10 MB burst per client
  fair_queue: true       # Enable fair queuing
monitor:
  enabled: true
  snmp_enabled: true
  metrics:
    include_rate_limits: true
```

### Example 2: High-Performance Client
```yaml
mode: "client"
tunnel:
  server_address: "server.example.com"
  server_port: 8443
throttle:
  enabled: true
  rate_limit: 104857600  # 100 MB/s
  burst_limit: 209715200 # 200 MB burst
  buffer:
    size: 65536          # 64 KB buffer
    count: 1000          # 1000 buffers in pool
  optimization:
    batch_size: 100
    max_delay: "10ms"
monitor:
  enabled: true
  update_interval: 1
```

### Example 3: Adaptive Rate Limiting
```yaml
throttle:
  enabled: true
  adaptive:
    enabled: true
    target_latency: "50ms"
    max_rate: 104857600   # 100 MB/s
    min_rate: 1048576     # 1 MB/s
    measurement_window: "1s"
    adjustment_interval: "100ms"
    increase_factor: 1.1   # 10% increase
    decrease_factor: 0.9   # 10% decrease
  monitoring:
    latency_threshold: "100ms"
    congestion_threshold: 0.8
    alert_interval: "1m"
```

## Performance Tuning

### Buffer Configuration
```yaml
buffer:
  read_size: 65536      # 64 KB read buffer
  write_size: 65536     # 64 KB write buffer
  pool_size: 1000       # Buffer pool size
  prealloc: true        # Preallocate buffers
```

### System Optimization
```bash
# Increase socket buffers
sysctl -w net.core.rmem_max=16777216
sysctl -w net.core.wmem_max=16777216

# Optimize TCP settings
sysctl -w net.ipv4.tcp_rmem="4096 87380 16777216"
sysctl -w net.ipv4.tcp_wmem="4096 87380 16777216"
```

## Monitoring Integration

### SNMP Metrics
```yaml
monitor:
  enabled: true
  snmp_enabled: true
  metrics:
    rate_limits:
      current_rate: .1.3.6.1.4.1.54321.1.3.1
      burst_rate: .1.3.6.1.4.1.54321.1.3.2
      limit_hits: .1.3.6.1.4.1.54321.1.3.3
```

### Prometheus Integration
```yaml
monitor:
  prometheus:
    enabled: true
    port: 9091
    metrics:
      - name: "sssonector_rate_current"
        help: "Current transfer rate"
        type: "gauge"
      - name: "sssonector_rate_limit_hits"
        help: "Number of rate limit hits"
        type: "counter"
```

## Troubleshooting

### Common Issues

1. Poor Performance
```bash
# Check current rate
snmpget -v2c -c public localhost:10161 .1.3.6.1.4.1.54321.1.3.1.0

# Monitor rate limit hits
watch -n 1 'snmpget -v2c -c public localhost:10161 .1.3.6.1.4.1.54321.1.3.3.0'

# View detailed metrics
sssonector -metrics
```

2. High Latency
```bash
# Monitor buffer usage
snmpwalk -v2c -c public localhost:10161 .1.3.6.1.4.1.54321.1.5

# Check system resources
top -p $(pgrep sssonector)
```

3. Inconsistent Rates
```bash
# Verify rate limit configuration
sssonector -validate-config

# Monitor rate adjustments
tail -f /var/log/sssonector/rate.log
```

## Best Practices

1. Rate Limit Selection
   - Start with conservative limits
   - Monitor actual usage patterns
   - Adjust based on network capacity
   - Consider client requirements

2. Burst Configuration
   - Set burst limit 2-3x base rate
   - Monitor burst utilization
   - Adjust based on application needs
   - Consider network latency

3. Performance Optimization
   - Use appropriate buffer sizes
   - Enable buffer pooling
   - Monitor system resources
   - Regular performance testing

4. Monitoring
   - Enable detailed metrics
   - Set up alerting
   - Regular configuration review
   - Monitor client experience

## Security Considerations

1. Rate Limit Protection
   - Implement global rate limits
   - Set per-client limits
   - Monitor for abuse
   - Regular security audits

2. Resource Protection
   - Set maximum connections
   - Implement fair queuing
   - Monitor resource usage
   - Set up alerts

3. Configuration Security
   - Validate configuration changes
   - Backup configurations
   - Document changes
   - Regular security reviews
