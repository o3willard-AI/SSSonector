# Rate Limiting Configuration Guide

This guide explains how to configure SSSonector's rate limiting features for optimal performance and resource utilization.

## Configuration Overview

Rate limiting configuration is specified in the `throttle` section of your configuration file. The system now includes advanced features like TCP overhead compensation, dynamic rate adjustment, and buffer pooling.

## Basic Configuration

Minimal configuration example:
```yaml
throttle:
  enabled: true
  rate: 1048576  # 1MB/s base rate
```

**Note:** The actual throughput will be approximately 5% higher than the configured rate to account for TCP overhead. For example, a rate of 1MB/s will result in an effective rate of 1.05MB/s.

## Advanced Configuration

Full configuration example with all options:
```yaml
throttle:
  enabled: true
  rate: 1048576          # Base rate (1MB/s)
  
  # Dynamic rate adjustment
  dynamic:
    enabled: true
    min_rate: 524288     # Minimum rate (512KB/s)
    max_rate: 104857600  # Maximum rate (100MB/s)
    adjustment_interval: 5s
    increase_threshold: 0.8  # Increase rate above 80% utilization
    decrease_threshold: 0.2  # Decrease rate below 20% utilization
  
  # Buffer pool settings
  buffer:
    size: 65536          # Buffer size (64KB)
    max_pool_size: 1000  # Maximum buffers in pool
  
  # Monitoring settings
  monitoring:
    enabled: true
    snmp_enabled: true
    metrics_interval: 60s
```

## Configuration Options

### Basic Settings

- `enabled` (boolean, default: true)
  * Enables or disables rate limiting
  * When disabled, no throttling is applied

- `rate` (integer, required)
  * Base rate in bytes per second
  * Actual throughput will be 5% higher due to TCP overhead compensation
  * Example: 1048576 for 1MB/s

### Dynamic Rate Adjustment

- `dynamic.enabled` (boolean, default: false)
  * Enables or disables dynamic rate adjustment
  * When enabled, rates automatically adjust based on utilization

- `dynamic.min_rate` (integer, default: 524288)
  * Minimum allowed rate in bytes per second
  * Rate will never drop below this value
  * Example: 524288 for 512KB/s

- `dynamic.max_rate` (integer, default: 104857600)
  * Maximum allowed rate in bytes per second
  * Rate will never exceed this value
  * Example: 104857600 for 100MB/s

- `dynamic.adjustment_interval` (duration, default: "5s")
  * How often to check and adjust rates
  * Format: Go duration string (e.g., "5s", "1m")

- `dynamic.increase_threshold` (float, default: 0.8)
  * Utilization threshold to trigger rate increase
  * Range: 0.0 to 1.0
  * Example: 0.8 means increase when utilization > 80%

- `dynamic.decrease_threshold` (float, default: 0.2)
  * Utilization threshold to trigger rate decrease
  * Range: 0.0 to 1.0
  * Example: 0.2 means decrease when utilization < 20%

### Buffer Pool Settings

- `buffer.size` (integer, default: 65536)
  * Size of individual buffers in bytes
  * Valid range: 4KB to 1MB
  * Example: 65536 for 64KB buffers

- `buffer.max_pool_size` (integer, default: 1000)
  * Maximum number of buffers to keep in pool
  * Affects memory usage
  * Example: 1000 buffers

### Monitoring Settings

- `monitoring.enabled` (boolean, default: true)
  * Enables collection of rate limiting metrics

- `monitoring.snmp_enabled` (boolean, default: true)
  * Enables SNMP metrics exposure

- `monitoring.metrics_interval` (duration, default: "60s")
  * How often to update metrics
  * Format: Go duration string

## Best Practices

1. Rate Selection
   - Start with a base rate slightly below your target throughput
   - Enable dynamic adjustment for optimal performance
   - Set min_rate to your minimum acceptable throughput
   - Set max_rate based on your network capacity

2. Buffer Configuration
   - Use default buffer size (64KB) for most cases
   - Increase for high-throughput scenarios
   - Decrease for memory-constrained environments
   - Set max_pool_size based on concurrent connections

3. Dynamic Adjustment
   - Enable for variable workloads
   - Use default thresholds (80%/20%) initially
   - Adjust based on monitoring data
   - Consider network characteristics

4. Monitoring
   - Enable SNMP for integration with monitoring systems
   - Monitor actual vs. configured rates
   - Track dynamic adjustment events
   - Watch for rate limit hits

## Examples

### High-Performance Server
```yaml
throttle:
  enabled: true
  rate: 10485760         # 10MB/s base
  dynamic:
    enabled: true
    min_rate: 1048576    # 1MB/s minimum
    max_rate: 104857600  # 100MB/s maximum
  buffer:
    size: 131072         # 128KB buffers
    max_pool_size: 2000  # Larger pool
```

### Memory-Constrained Client
```yaml
throttle:
  enabled: true
  rate: 1048576         # 1MB/s base
  dynamic:
    enabled: true
    min_rate: 262144    # 256KB/s minimum
    max_rate: 5242880   # 5MB/s maximum
  buffer:
    size: 32768         # 32KB buffers
    max_pool_size: 500  # Smaller pool
```

### Fixed-Rate Connection
```yaml
throttle:
  enabled: true
  rate: 1048576         # 1MB/s fixed rate
  dynamic:
    enabled: false      # Disable dynamic adjustment
  buffer:
    size: 65536        # Default buffer size
    max_pool_size: 1000
```

## Troubleshooting

1. Throughput Issues
   - Check actual vs. configured rates in metrics
   - Verify TCP overhead compensation
   - Review dynamic adjustment logs
   - Monitor rate limit hits

2. Memory Usage
   - Reduce buffer size and/or pool size
   - Monitor buffer pool utilization
   - Check for memory leaks
   - Review GC patterns

3. Performance Problems
   - Enable dynamic adjustment
   - Increase buffer size for high throughput
   - Adjust rate thresholds
   - Monitor CPU usage

## Migration Guide

When upgrading from previous versions:

1. Add TCP overhead compensation
   - Reduce configured rates by 5%
   - System will automatically add overhead

2. Enable dynamic adjustment
   - Start with default thresholds
   - Monitor adjustment patterns
   - Fine-tune based on results

3. Configure buffer pools
   - Use default values initially
   - Adjust based on monitoring data
   - Consider memory constraints

4. Update monitoring
   - Configure new metrics
   - Update dashboards
   - Set up alerts
