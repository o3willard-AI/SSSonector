# Rate Limiting Implementation Guide

This document details the rate limiting implementation in SSSonector, including recent improvements for enterprise performance requirements.

## Architecture Overview

The rate limiting system consists of several components working together to provide efficient and fair bandwidth control:

1. Base Rate Limiter (`Limiter`)
   - Handles basic rate limiting using token bucket algorithm
   - Manages both ingress and egress traffic
   - Provides metrics collection
   - Applies TCP overhead compensation

2. Dynamic Rate Limiter (`DynamicLimiter`)
   - Extends base limiter with dynamic rate adjustment
   - Automatically adapts to network conditions
   - Maintains rate bounds for stability
   - Implements cooldown periods between adjustments

3. Token Bucket Implementation
   - Uses golang.org/x/time/rate under the hood
   - Enhanced with TCP overhead compensation
   - Optimized burst handling
   - Thread-safe operations

## Implementation Details

### Dynamic Rate Limiting

The dynamic rate limiter provides automatic rate adjustment capabilities:

1. Rate Bounds
   ```go
   minRate = baseRate / 2  // 50% of configured rate
   maxRate = baseRate * 2  // 200% of configured rate
   ```

2. Rate Adjustments
   - Percentage-based increases/decreases
   - Bounded by min/max rates
   - 1-second cooldown between adjustments
   - Thread-safe operations with mutex protection

3. TCP Overhead Compensation
   ```go
   adjustedRate = rate * tcpOverheadFactor  // 10% overhead
   adjustedBurst = burst * tcpOverheadFactor
   ```

4. Burst Control
   ```go
   burst = rate * 0.1  // 10% of current rate
   ```

### Metrics and Monitoring

The rate limiter provides detailed metrics through several channels:

1. Direct Metrics
   ```go
   type LimiterMetrics struct {
       Rate      float64  // Current rate in bytes/sec
       Burst     float64  // Current burst size
       LimitHits uint64   // Number of rate limit hits
   }
   ```

2. SNMP Integration
   - Current rates (OID: .1.3.6.1.4.1.54321.1.3.1)
   - Burst rates (OID: .1.3.6.1.4.1.54321.1.3.2)
   - Limit hits (OID: .1.3.6.1.4.1.54321.1.3.3)

3. Logging
   - Rate adjustments
   - Limit hits
   - Configuration changes
   - Error conditions

## Troubleshooting Guide

### Common Issues

1. High Rate Limit Hits
   - Check current utilization with metrics
   - Verify min/max rate configuration
   - Monitor network conditions
   - Review adjustment patterns in logs

2. Unstable Rates
   - Check cooldown period (default: 1 second)
   - Verify rate bounds are appropriate
   - Monitor adjustment frequency
   - Review TCP overhead compensation

3. Performance Impact
   - Monitor CPU usage during adjustments
   - Check burst size calculations
   - Verify buffer configurations
   - Review token bucket performance

### Diagnostic Commands

1. Check Current Rates
   ```bash
   # SNMP query for current rates
   snmpget -v2c -c public localhost:10161 .1.3.6.1.4.1.54321.1.3.1.0

   # Monitor rate adjustments
   tail -f /var/log/sssonector/rate.log | grep "rate"
   ```

2. Verify Configuration
   ```bash
   # Validate configuration
   sssonector -validate-config

   # Check metrics
   sssonector -metrics | grep "rate"
   ```

3. Performance Testing
   ```bash
   # Test throughput
   iperf3 -c <server> -t 30 -J

   # Monitor system impact
   top -p $(pgrep sssonector)
   ```

## Extension Points

The rate limiting system is designed for extensibility:

1. Custom Rate Adjustment Strategies
   - Implement new adjustment algorithms in DynamicLimiter
   - Add custom metrics for decision making
   - Extend configuration options

2. Additional Metrics
   - Add new fields to LimiterMetrics struct
   - Implement new SNMP OIDs
   - Extend logging capabilities

3. Enhanced Control
   - Add new configuration parameters
   - Implement additional adjustment methods
   - Extend monitoring capabilities

### Example: Adding Custom Adjustment Strategy

```go
// Add new configuration
type AdjustmentConfig struct {
    Strategy       string  // Adjustment strategy name
    CheckInterval  time.Duration
    Sensitivity    float64
}

// Implement new strategy
func (dl *DynamicLimiter) AdjustRate(strategy string) {
    switch strategy {
    case "adaptive":
        dl.adaptiveAdjust()
    case "gradual":
        dl.gradualAdjust()
    default:
        dl.defaultAdjust()
    }
}
```

## Best Practices

1. Rate Configuration
   - Set appropriate base rates
   - Configure reasonable min/max bounds
   - Consider network capacity
   - Account for TCP overhead

2. Monitoring
   - Enable detailed metrics
   - Set up alerting
   - Monitor adjustment patterns
   - Track system impact

3. Performance
   - Use appropriate burst sizes
   - Configure adequate cooldown periods
   - Monitor resource usage
   - Regular performance testing

4. Security
   - Implement rate limit protection
   - Monitor for abuse
   - Regular security audits
   - Validate configuration changes

## Future Improvements

1. Enhanced Algorithms
   - Machine learning-based adjustments
   - Network condition awareness
   - Predictive rate adaptation
   - QoS integration

2. Monitoring
   - Extended metrics collection
   - Advanced visualization
   - Automated analysis
   - Anomaly detection

3. Configuration
   - Dynamic parameter tuning
   - Profile-based settings
   - Context-aware adjustments
   - Integration with external systems
