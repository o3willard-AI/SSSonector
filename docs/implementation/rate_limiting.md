# Rate Limiting Implementation

This document details the rate limiting implementation in SSSonector, including recent improvements for enterprise performance requirements.

## Overview

The rate limiting system consists of several components working together to provide efficient and fair bandwidth control:

1. Token Bucket Implementation (`internal/throttle/token_bucket.go`)
2. Buffer Pool Management (`internal/throttle/io.go`)
3. Dynamic Rate Adjustment (`internal/throttle/dynamic.go`)
4. Metrics Collection (`internal/throttle/limiter.go`)

## Key Features

### TCP Overhead Compensation

- Automatically adds 5% to configured rates to account for TCP overhead
- Ensures actual application throughput matches configured limits
- Compensates for:
  * Protocol headers
  * Retransmissions
  * TCP control packets

Implementation:
```go
const tcpOverheadFactor = 1.05 // 5% overhead

func NewTokenBucket(rate float64, burst float64) *TokenBucket {
    effectiveRate := rate * tcpOverheadFactor
    // ...
}
```

### Buffer Pool Management

- Uses sync.Pool for efficient buffer reuse
- Configurable buffer sizes:
  * Minimum: 4KB
  * Maximum: 1MB
  * Default: 64KB
- Small read/write optimization
- Automatic buffer lifecycle management

Implementation:
```go
type BufferPool struct {
    pool    sync.Pool
    size    int
    maxSize int
}
```

### Dynamic Rate Adjustment

- Utilization-based rate adjustment
- Configurable thresholds:
  * Increase at 80% utilization
  * Decrease at 20% utilization
- Rate bounds enforcement:
  * Minimum: 512KB/s
  * Maximum: 100MB/s
- 5-second adjustment interval

Implementation:
```go
type DynamicAdjuster struct {
    enabled            bool
    minRate            float64
    maxRate            float64
    adjustmentInterval time.Duration
    // ...
}
```

### Burst Control

- Reduced burst window to 100ms
- Burst size calculated as: rate * 0.1
- Prevents traffic spikes while maintaining performance
- Improves fairness across connections

Implementation:
```go
burstFactor := 0.1 // 100ms
effectiveBurst := effectiveRate * burstFactor
```

## Configuration Changes

The rate limiting configuration has been extended to support new features:

```yaml
throttle:
  enabled: true
  rate: 1048576          # Base rate (1MB/s)
  dynamic:
    enabled: true
    min_rate: 524288     # Minimum rate (512KB/s)
    max_rate: 104857600  # Maximum rate (100MB/s)
    adjustment_interval: 5s
    increase_threshold: 0.8
    decrease_threshold: 0.2
  buffer:
    size: 65536          # Buffer size (64KB)
    max_pool_size: 1000  # Maximum buffers in pool
```

## Monitoring Integration

### SNMP Metrics

New metrics exposed via SNMP:
- Current rate (with TCP overhead)
- Burst rate
- Buffer pool utilization
- Rate limit hits
- Dynamic adjustment events

### Prometheus Metrics

New metrics for Prometheus:
```
sssonector_throttle_current_rate
sssonector_throttle_effective_rate
sssonector_throttle_burst_size
sssonector_throttle_limit_hits
sssonector_throttle_buffer_pool_size
sssonector_throttle_adjustment_events
```

## Performance Considerations

1. Memory Usage
   - Buffer pool prevents excessive allocations
   - Configurable pool size limits memory usage
   - Small read/write optimization reduces pool pressure

2. CPU Usage
   - Efficient token bucket implementation
   - Lock-free paths for common operations
   - Batched metric updates

3. Latency Impact
   - 100ms burst window balances throughput and latency
   - Dynamic adjustment prevents bottlenecks
   - Buffer pooling reduces GC pressure

## Testing

The rate limiting system includes comprehensive tests:

1. Unit Tests
   - Token bucket behavior
   - Buffer pool management
   - Dynamic adjustment logic
   - Configuration validation

2. Integration Tests
   - End-to-end throughput
   - TCP overhead compensation
   - Dynamic adjustment scenarios
   - Memory usage patterns

3. Performance Tests
   - High throughput scenarios
   - Concurrent connection handling
   - Memory allocation patterns
   - CPU utilization profiles

## Troubleshooting

Common issues and solutions:

1. Unexpected Throughput
   - Verify TCP overhead compensation
   - Check actual vs. configured rates
   - Monitor dynamic adjustment logs

2. Memory Usage
   - Review buffer pool configuration
   - Monitor pool utilization
   - Check for buffer leaks

3. Performance Issues
   - Analyze rate limit hits
   - Review burst patterns
   - Check dynamic adjustment logs

## Future Improvements

Planned enhancements:

1. Advanced Rate Control
   - Per-connection rate limiting
   - Priority-based throttling
   - QoS integration

2. Enhanced Monitoring
   - Detailed adjustment metrics
   - Historical rate analysis
   - Anomaly detection

3. Performance Optimization
   - Zero-copy buffer handling
   - Enhanced batching
   - Predictive rate adjustment
