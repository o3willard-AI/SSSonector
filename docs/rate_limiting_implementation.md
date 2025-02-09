# Rate Limiting Implementation Guide

This document details the rate limiting implementation in SSSonector v2.1.0, including configuration examples and best practices.

## Overview

SSSonector implements an enhanced rate limiting system with the following features:
- Independent ingress and egress rate limiting
- TCP overhead compensation
- Optimized burst control
- Per-connection and global limits
- Dynamic rate adjustment
- SNMP monitoring integration

## Implementation Details

### Rate Limiting Implementation

The rate limiter uses golang.org/x/time/rate with the following enhancements:

1. TCP Overhead Compensation
   - Automatically adds 5% to configured rates for TCP overhead
   - Ensures actual throughput matches configured limits
   - Compensates for protocol headers and retransmissions

2. Burst Control
   - Burst size set to 100ms worth of data (reduced from 1s)
   - Provides better latency control and fairness
   - Prevents large traffic spikes while maintaining performance

3. Timeout Handling
   - 5-second timeout on rate limit waits
   - Prevents indefinite blocking on congestion
   - Returns recoverable errors for retry logic

### Code Structure

```go
type Limiter struct {
    reader   io.Reader
    writer   io.Writer
    inLimit  *rate.Limiter  // Download rate limiter
    outLimit *rate.Limiter  // Upload rate limiter
    mu       sync.RWMutex   // Thread safety for updates
}

// NewLimiter creates a rate limiter with TCP overhead compensation
func NewLimiter(reader io.Reader, writer io.Writer, uploadKbps, downloadKbps int64) *Limiter {
    const (
        tcpOverhead = 1.05  // 5% overhead
        burstFactor = 0.1   // 100ms worth of data
    )
    return &Limiter{
        reader:   reader,
        writer:   writer,
        inLimit:  rate.NewLimiter(rate.Limit(downloadKbps*1024*tcpOverhead), 
                                 int(downloadKbps*1024*burstFactor)),
        outLimit: rate.NewLimiter(rate.Limit(uploadKbps*1024*tcpOverhead), 
                                 int(uploadKbps*1024*burstFactor)),
    }
}
```

## Configuration Examples

### Example 1: Basic Rate Limiting
```yaml
throttle:
  enabled: true
  rate_limit: 1048576    # 1 MB/s (actual: ~1.05 MB/s with TCP overhead)
  burst_limit: 104858    # 100ms worth of data
```

### Example 2: Asymmetric Rate Limiting
```yaml
throttle:
  enabled: true
  upload_rate: 5242880   # 5 MB/s upload (actual: ~5.25 MB/s)
  upload_burst: 524288   # 100ms worth of data
  download_rate: 10485760 # 10 MB/s download (actual: ~10.5 MB/s)
  download_burst: 1048576 # 100ms worth of data
  per_connection: true
```

### Example 3: Dynamic Rate Limiting
```yaml
throttle:
  enabled: true
  rate_limit: 1048576    # Base rate: 1 MB/s
  burst_limit: 104858    # 100ms worth of data
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
  global_rate: 52428800  # 50 MB/s total (actual: ~52.5 MB/s)
  global_burst: 5242880  # 100ms worth of data
  per_client:
    rate: 5242880        # 5 MB/s per client
    burst: 524288        # 100ms worth of data
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
  rate_limit: 104857600  # 100 MB/s (actual: ~105 MB/s)
  burst_limit: 10485760  # 100ms worth of data
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

## Troubleshooting

### Common Issues

1. Rate Limiting Behavior
```bash
# Check current rate (includes TCP overhead)
snmpget -v2c -c public localhost:10161 .1.3.6.1.4.1.54321.1.3.1.0

# Monitor actual throughput
iperf3 -c <server> -t 30 -J

# Verify TCP overhead compensation
# Actual throughput should be ~5% higher than configured rate
sssonector -metrics
```

2. High Latency
```bash
# Monitor burst usage
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
   - Account for 5% TCP overhead in planning
   - Monitor actual usage patterns
   - Adjust based on network capacity
   - Consider client requirements

2. Burst Configuration
   - Use 100ms worth of data for burst size
   - Smaller bursts provide better latency control
   - TCP overhead is automatically compensated
   - Monitor actual throughput with SNMP

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
