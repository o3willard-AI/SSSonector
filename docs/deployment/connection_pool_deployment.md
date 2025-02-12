# Connection Pool Deployment Guide

This guide covers deployment considerations and best practices for the connection pool component.

## Pre-deployment Checklist

1. Configuration Review
   - Verify pool size settings match system resources
   - Confirm timeout values are appropriate for network conditions
   - Review retry strategy configuration
   - Validate health check implementation
   - Verify Duration type usage in configuration files

2. System Requirements
   - Memory: Sufficient for max connections * average connection size
   - Network: Stable connectivity to target services
   - CPU: Adequate for connection handling and health checks
   - Disk: Space for logs and metrics

3. Dependencies
   - Go 1.21 or higher
   - Required packages in go.mod
   - Logging infrastructure (zap)
   - Metrics collection system

## Configuration Examples

### Basic Production Setup
```yaml
pool:
  initialSize: 10
  maxSize: 50
  minSize: 5
  maxIdleTime: 5m      # Uses types.Duration
  connectTimeout: 10s   # Uses types.Duration
  healthCheckInterval: 30s

retry:
  maxImmediateRetries: 3
  immediateDelay: 1ms
  maxGradualRetries: 5
  initialInterval: 1ms
  maxInterval: 10ms
  backoffMultiplier: 2.0
  maxPersistentRetries: 2
  persistentDelay: 5s
```

### High-Load Setup
```yaml
pool:
  initialSize: 50
  maxSize: 200
  minSize: 20
  maxIdleTime: 2m      # Uses types.Duration
  connectTimeout: 5s    # Uses types.Duration
  healthCheckInterval: 15s

retry:
  maxImmediateRetries: 5
  immediateDelay: 1ms
  maxGradualRetries: 7
  initialInterval: 1ms
  maxInterval: 20ms
  backoffMultiplier: 1.5
  maxPersistentRetries: 3
  persistentDelay: 3s
```

## Duration Type Configuration

1. YAML Configuration
   ```yaml
   # Valid duration formats
   maxIdleTime: 5m
   connectTimeout: 10s
   healthCheckInterval: 30s500ms
   ```

2. JSON Configuration
   ```json
   {
     "maxIdleTime": "5m",
     "connectTimeout": "10s",
     "healthCheckInterval": "30.5s"
   }
   ```

3. Environment Variables
   ```bash
   export POOL_MAX_IDLE_TIME="5m"
   export POOL_CONNECT_TIMEOUT="10s"
   ```

## Monitoring Setup

1. Metrics to Track
   ```go
   // Success/Failure counts
   success, failures := pool.GetMetrics()

   // Pool size
   currentSize := len(pool.conns)

   // Connection latency
   timeToAcquire := time.Since(start)
   ```

2. Logging Configuration
   ```go
   logger := zap.NewProduction(
       zap.Fields(
           zap.String("component", "connection_pool"),
           zap.String("version", "2.0.0"),
       ),
   )
   ```

3. Health Check Implementation
   ```go
   healthCheck := func(conn net.Conn) error {
       deadline := time.Now().Add(time.Second)
       if err := conn.SetDeadline(deadline); err != nil {
           return fmt.Errorf("failed to set deadline: %w", err)
       }
       // Implement health check logic
       return nil
   }
   ```

## Deployment Steps

1. Pre-deployment
   ```bash
   # Run tests
   go test ./internal/pool/... -v -race

   # Run benchmarks
   go test ./internal/benchmark/... -v

   # Validate configuration
   go run ./cmd/admin validate-config config.yaml
   ```

2. Deployment
   ```bash
   # Build
   go build -o ssonector ./cmd/...

   # Deploy configuration
   cp config/pool.yaml /etc/ssonector/

   # Start service
   systemctl start ssonector
   ```

3. Post-deployment
   ```bash
   # Verify metrics
   curl http://localhost:8080/metrics

   # Check logs
   journalctl -u ssonector
   ```

## Scaling Considerations

1. Vertical Scaling
   - Increase maxSize based on available memory
   - Adjust retry intervals for higher load
   - Optimize health check frequency

2. Horizontal Scaling
   - Deploy multiple instances
   - Use load balancer
   - Configure per-instance pool sizes

3. Resource Calculation
   ```
   Memory per connection = ~10KB
   Total memory = maxSize * memory per connection
   CPU cores needed = maxSize / 100
   ```

## Troubleshooting

1. Duration Type Issues
   ```go
   // Enable debug logging for duration parsing
   logger.Debug("parsing duration",
       zap.String("raw", rawValue),
       zap.Any("parsed", duration))

   // Validate duration values
   if err := validator.ValidateDuration(duration); err != nil {
       logger.Error("invalid duration", zap.Error(err))
   }
   ```

2. Performance Issues
   - Check connection acquisition times
   - Monitor pool size
   - Review retry patterns
   - Analyze health check duration

3. Resource Issues
   - Monitor memory usage
   - Track goroutine count
   - Check network utilization

## Maintenance

1. Regular Tasks
   - Review metrics daily
   - Analyze error patterns
   - Update configurations as needed
   - Check resource usage

2. Updates
   - Test in staging environment
   - Deploy during low-traffic periods
   - Monitor closely after updates
   - Keep rollback plan ready

3. Backup
   - Configuration backup
   - Metrics history
   - Error logs retention

## Security

1. Network Security
   - Use TLS for connections
   - Implement connection timeouts
   - Configure firewall rules

2. Resource Protection
   - Set connection limits
   - Implement rate limiting
   - Monitor for abuse

3. Error Handling
   - Sanitize error messages
   - Log security events
   - Implement circuit breakers

## Breaking Changes in v2.0.0

1. Duration Type Changes
   - Now using types.Duration from internal/config/types
   - Updated configuration format
   - New validation rules

2. Configuration Updates
   - YAML/JSON format changes
   - Environment variable changes
   - Validation updates

3. Deployment Updates
   - Review configuration files
   - Update monitoring dashboards
   - Modify alert thresholds
   - Update deployment scripts

Please ensure all configuration files are updated to use the new Duration type format before deploying.
