# Rate Limiting Implementation Guide

This document details the rate limiting implementation in SSSonector v2.1.0, including configuration examples and best practices.

## Overview

SSSonector implements an enhanced rate limiting system with the following features:
- Independent ingress and egress rate limiting
- TCP overhead compensation
- Token bucket-based rate limiting
- Dynamic rate adjustment with cooldown
- SNMP monitoring integration
- Buffer pool optimization

## Implementation Details

### Rate Limiting Implementation

The rate limiter uses a custom token bucket implementation with the following enhancements:

1. Token Bucket Algorithm
   - Precise rate limiting with token accumulation
   - Configurable burst size for traffic spikes
   - Thread-safe token management
   - Accurate timing for token replenishment

2. TCP Overhead Compensation
   - Automatically adds 10% to configured rates for TCP overhead
   - Ensures actual throughput matches configured limits
   - Compensates for protocol headers and retransmissions

3. Dynamic Rate Adjustment
   - Automatic rate adjustment based on usage patterns
   - Configurable min/max rate bounds
   - Cooldown period between adjustments
   - Thread-safe rate changes

### Code Structure

```go
type TokenBucket struct {
    rate       float64     // Tokens per second
    burst      float64     // Maximum token burst
    tokens     float64     // Current token count
    lastUpdate time.Time   // Last token update time
    mu         sync.Mutex  // Thread safety
}

type DynamicLimiter struct {
    *Limiter
    minRate     float64
    maxRate     float64
    adjustMu    sync.RWMutex
    lastAdjust  time.Time
    adjustCount int
    cooldown    types.Duration
    logger      *zap.Logger
}

// NewTokenBucket creates a new token bucket rate limiter
func NewTokenBucket(rate, burst float64) *TokenBucket {
    return &TokenBucket{
        rate:       rate,
        burst:      burst,
        tokens:     0,           // Start with 0 tokens
        lastUpdate: time.Now(),
    }
}

// NewDynamicLimiter creates a dynamic rate limiter
func NewDynamicLimiter(cfg *types.AppConfig, limiter *Limiter, logger *zap.Logger) *DynamicLimiter {
    baseRate := float64(cfg.Throttle.Rate)
    return &DynamicLimiter{
        Limiter:    limiter,
        minRate:    baseRate / 2,  // Default min rate
        maxRate:    baseRate * 2,  // Default max rate
        cooldown:   types.NewDuration(time.Second),
        logger:     logger,
    }
}
```

## Configuration Examples

### Example 1: Basic Rate Limiting
```yaml
throttle:
  enabled: true
  rate_limit: 1048576    # 1 MB/s (actual: ~1.1 MB/s with TCP overhead)
  burst_limit: 104858    # 10% of rate for burst
```

### Example 2: Dynamic Rate Limiting
```yaml
throttle:
  enabled: true
  rate_limit: 1048576    # Base rate: 1 MB/s
  burst_limit: 104858    # 10% of rate for burst
  dynamic:
    enabled: true
    min_rate: 524288     # Minimum: 512 KB/s
    max_rate: 2097152    # Maximum: 2 MB/s
    cooldown: "1s"       # 1 second between adjustments
    adjustment_percent: 50 # Adjust by 50% up/down
```

### Example 3: High-Performance Configuration
```yaml
throttle:
  enabled: true
  rate_limit: 104857600  # 100 MB/s
  burst_limit: 10485760  # 10% burst
  buffer:
    size: 32768         # 32 KB buffer size
    pool_size: 1024     # Buffer pool size
    prealloc: true      # Preallocate buffers
  dynamic:
    enabled: true
    min_rate: 52428800  # 50 MB/s minimum
    max_rate: 209715200 # 200 MB/s maximum
    cooldown: "500ms"   # Fast adjustments
```

## Performance Tuning

### Buffer Pool Configuration
```yaml
buffer:
  read_size: 32768      # 32 KB read buffer
  write_size: 32768     # 32 KB write buffer
  pool_size: 1024       # Buffer pool size
  prealloc: true        # Preallocate buffers
```

### Dynamic Rate Adjustment
```yaml
dynamic:
  enabled: true
  cooldown: "1s"        # Adjustment cooldown
  min_rate: "512KB/s"   # Minimum rate
  max_rate: "10MB/s"    # Maximum rate
  metrics:
    enabled: true       # Track adjustments
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
      adjust_count: .1.3.6.1.4.1.54321.1.3.4
```

## Troubleshooting

### Common Issues

1. Rate Limiting Behavior
```bash
# Check current rate (includes TCP overhead)
snmpget -v2c -c public localhost:10161 .1.3.6.1.4.1.54321.1.3.1.0

# Monitor rate adjustments
snmpget -v2c -c public localhost:10161 .1.3.6.1.4.1.54321.1.3.4.0

# Verify actual throughput
iperf3 -c <server> -t 30 -J
```

2. Performance Issues
```bash
# Check buffer pool metrics
snmpwalk -v2c -c public localhost:10161 .1.3.6.1.4.1.54321.1.4

# Monitor system resources
top -p $(pgrep sssonector)
```

3. Dynamic Rate Issues
```bash
# Check adjustment logs
tail -f /var/log/sssonector/rate.log | grep "rate limiter"

# Verify cooldown periods
snmpget -v2c -c public localhost:10161 .1.3.6.1.4.1.54321.1.3.5.0
```

## Best Practices

1. Rate Limit Configuration
   - Set appropriate base rates
   - Configure reasonable min/max bounds
   - Use appropriate cooldown periods
   - Monitor adjustment patterns

2. Buffer Configuration
   - Use power-of-two buffer sizes
   - Enable buffer pooling
   - Preallocate for performance
   - Monitor pool utilization

3. Performance Optimization
   - Balance burst size with latency
   - Tune dynamic adjustment parameters
   - Monitor system resources
   - Regular performance testing

4. Monitoring
   - Track rate adjustments
   - Monitor buffer pool usage
   - Set up alerts for limit hits
   - Regular metric analysis

## Security Considerations

1. Rate Limit Protection
   - Set appropriate rate bounds
   - Configure reasonable cooldowns
   - Monitor adjustment patterns
   - Alert on excessive adjustments

2. Resource Protection
   - Configure appropriate buffer sizes
   - Monitor pool utilization
   - Set up resource alerts
   - Regular capacity planning

3. Configuration Security
   - Validate rate configurations
   - Document adjustment policies
   - Regular security reviews
   - Monitor for abuse

## Testing Requirements

1. Rate Limiting Tests
   - Verify base rate enforcement
   - Test burst handling
   - Validate TCP overhead compensation
   - Check timing accuracy

2. Dynamic Adjustment Tests
   - Test rate increase/decrease
   - Verify cooldown periods
   - Check adjustment bounds
   - Monitor adjustment counts

3. Performance Tests
   - Measure throughput accuracy
   - Test buffer pool efficiency
   - Verify resource usage
   - Check latency impact

4. Integration Tests
   - Test with monitoring systems
   - Verify metric accuracy
   - Check alert triggers
   - Validate logging
