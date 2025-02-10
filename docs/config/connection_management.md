# Connection Management Configuration Guide

This guide explains how to configure SSSonector's connection management features for optimal performance and reliability.

## Overview

The connection management system provides:
- Connection pooling
- Automatic health checks
- Error recovery
- Resource management

## Basic Configuration

Minimal configuration example:
```yaml
connection:
  pool:
    enabled: true
    max_active: 1000
```

## Advanced Configuration

Full configuration example with all options:
```yaml
connection:
  pool:
    enabled: true
    idle_timeout: 5m
    max_idle: 100
    max_active: 1000
    retry_interval: 5s
    max_retries: 3
    
  health_check:
    enabled: true
    interval: 30s
    timeout: 1s
    
  buffer:
    size: 65536          # 64KB buffer size
    max_pool_size: 1000  # Maximum buffers in pool
    
  monitoring:
    enabled: true
    metrics_interval: 60s
```

## Configuration Options

### Pool Settings

- `enabled` (boolean, default: true)
  * Enables or disables connection pooling
  * When disabled, new connections are created for each request

- `idle_timeout` (duration, default: "5m")
  * Maximum time a connection can remain idle
  * Format: Go duration string (e.g., "5m", "1h")
  * Connections idle longer than this are removed

- `max_idle` (integer, default: 100)
  * Maximum number of idle connections
  * Excess idle connections are closed
  * Set based on typical connection needs

- `max_active` (integer, default: 1000)
  * Maximum total active connections
  * New connections blocked when limit reached
  * Size based on system resources

- `retry_interval` (duration, default: "5s")
  * Time between connection retry attempts
  * Format: Go duration string
  * Affects reconnection behavior

- `max_retries` (integer, default: 3)
  * Maximum connection retry attempts
  * Failed connections are removed
  * Balances reliability and latency

### Health Check Settings

- `enabled` (boolean, default: true)
  * Enables periodic health checks
  * Removes failed connections
  * Maintains pool health

- `interval` (duration, default: "30s")
  * Time between health checks
  * Format: Go duration string
  * Balance monitoring vs overhead

- `timeout` (duration, default: "1s")
  * Health check timeout period
  * Format: Go duration string
  * Keep short to detect issues

### Buffer Settings

- `size` (integer, default: 65536)
  * Size of individual buffers
  * Valid range: 4KB to 1MB
  * Affects memory usage

- `max_pool_size` (integer, default: 1000)
  * Maximum number of pooled buffers
  * Affects memory usage
  * Set based on connection count

### Monitoring Settings

- `enabled` (boolean, default: true)
  * Enables metrics collection
  * Provides operational insight
  * Minimal performance impact

- `metrics_interval` (duration, default: "60s")
  * Metrics collection frequency
  * Format: Go duration string
  * Balance detail vs overhead

## Best Practices

1. Pool Sizing
   - Set max_active based on system resources
   - Keep max_idle proportional to typical load
   - Consider connection establishment cost

2. Health Checks
   - Enable for production environments
   - Set interval based on stability needs
   - Keep timeouts short

3. Buffer Management
   - Use default buffer size for most cases
   - Adjust for specific workload needs
   - Monitor memory usage

4. Monitoring
   - Enable in production
   - Monitor connection utilization
   - Track error rates

## Examples

### High-Performance Server
```yaml
connection:
  pool:
    enabled: true
    max_idle: 500
    max_active: 5000
    retry_interval: 1s
  buffer:
    size: 131072      # 128KB
    max_pool_size: 2000
```

### Memory-Constrained Client
```yaml
connection:
  pool:
    enabled: true
    max_idle: 50
    max_active: 200
    idle_timeout: 1m
  buffer:
    size: 32768       # 32KB
    max_pool_size: 500
```

### High-Reliability Setup
```yaml
connection:
  pool:
    enabled: true
    max_retries: 5
    retry_interval: 2s
  health_check:
    interval: 15s
    timeout: 500ms
```

## Monitoring

### Available Metrics

1. Connection Pool
   ```
   sssonector_pool_active_connections
   sssonector_pool_idle_connections
   sssonector_pool_connection_errors
   sssonector_pool_health_check_failures
   ```

2. Buffer Pool
   ```
   sssonector_buffer_pool_size
   sssonector_buffer_pool_utilization
   sssonector_buffer_allocation_rate
   ```

3. Error Rates
   ```
   sssonector_connection_errors_total
   sssonector_connection_timeouts_total
   sssonector_health_check_failures_total
   ```

### Grafana Dashboard

A Grafana dashboard template is available at:
`monitoring/grafana/dashboards/sssonector.json`

## Troubleshooting

1. Connection Issues
   - Check health check logs
   - Verify network connectivity
   - Review retry settings
   - Monitor error rates

2. Performance Problems
   - Review pool sizing
   - Check buffer configuration
   - Monitor resource usage
   - Analyze metrics

3. Memory Usage
   - Adjust buffer pool size
   - Review idle connection count
   - Monitor system resources
   - Check for leaks

## Migration Guide

When upgrading from previous versions:

1. Enable Connection Pooling
   ```yaml
   connection:
     pool:
       enabled: true
       max_active: 1000
   ```

2. Add Health Checks
   ```yaml
   connection:
     health_check:
       enabled: true
       interval: 30s
   ```

3. Configure Buffer Pool
   ```yaml
   connection:
     buffer:
       size: 65536
       max_pool_size: 1000
   ```

4. Enable Monitoring
   ```yaml
   connection:
     monitoring:
       enabled: true
       metrics_interval: 60s
