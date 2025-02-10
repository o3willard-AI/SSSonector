# Connection Error Recovery

This document details the error recovery process for connection failures in SSSonector, particularly focusing on client-side recovery mechanisms.

## Connection Recovery Process

### 1. Initial Connection Attempt
- Client attempts to establish connection
- Default timeout: 30 seconds
- Configurable via `connection.timeout` setting

### 2. Immediate Retry Phase
- Attempts: 3 retries
- Interval: 5 seconds between attempts
- Configurable via:
  ```yaml
  connection:
    retry:
      immediate:
        attempts: 3
        interval: 5s
  ```

### 3. Gradual Retry Phase
- Attempts: 5 retries
- Interval: Exponential backoff starting at 30s
  * 30s, 60s, 120s, 240s, 480s
- Configurable via:
  ```yaml
  connection:
    retry:
      gradual:
        attempts: 5
        base_interval: 30s
        max_interval: 480s
  ```

### 4. Persistent Retry Phase
- Continues indefinitely until connection restored
- Interval: Fixed 10-minute interval
- Can be disabled via configuration
- Configurable via:
  ```yaml
  connection:
    retry:
      persistent:
        enabled: true
        interval: 10m
  ```

## Error Categories & Recovery Strategies

### 1. Network Errors
- DNS resolution failures
- TCP connection timeouts
- Network unreachable
```go
Strategy:
1. Verify DNS resolution
2. Try alternative DNS servers
3. Enter retry cycle
```

### 2. Authentication Errors
- Certificate validation failures
- Token expiration
- Invalid credentials
```go
Strategy:
1. Verify certificate validity
2. Attempt token refresh
3. If persistent, alert administrator
```

### 3. Protocol Errors
- TLS handshake failures
- Protocol version mismatch
- Malformed messages
```go
Strategy:
1. Attempt protocol negotiation
2. Try fallback protocol versions
3. Enter retry cycle
```

### 4. Resource Errors
- Server overload
- Connection limits reached
- Memory exhaustion
```go
Strategy:
1. Implement exponential backoff
2. Try connection pooling
3. Clean up resources
```

## Health Check Process

### Active Checks
- Frequency: Every 30 seconds
- Timeout: 5 seconds
- Method: TCP keep-alive + application ping
```go
func (c *Client) healthCheck() {
    // Send ping message
    // Verify response
    // Update connection state
}
```

### Passive Checks
- Monitor data flow
- Track error rates
- Measure latency
```go
metrics:
  - bytes_transferred
  - error_count
  - latency_ms
```

## Connection States

```go
type ConnectionState int

const (
    StateDisconnected
    StateConnecting
    StateConnected
    StateReconnecting
    StateFailed
)
```

### State Transitions
```
Disconnected -> Connecting -> Connected
Connected -> Reconnecting -> Connected
Reconnecting -> Failed -> Disconnected
```

## Implementation Details

### Connection Pool Management
```go
type Pool struct {
    maxRetries    int           // Maximum immediate retries
    retryInterval time.Duration // Time between retries
    maxBackoff    time.Duration // Maximum backoff time
    persistent    bool          // Enable persistent retries
}
```

### Retry Logic
```go
func (p *Pool) getConnection(ctx context.Context) (net.Conn, error) {
    // Immediate retry phase
    for i := 0; i < p.maxRetries; i++ {
        conn, err := p.connect(ctx)
        if err == nil {
            return conn, nil
        }
        time.Sleep(p.retryInterval)
    }

    // Gradual retry phase
    interval := p.retryInterval
    for i := 0; i < p.gradualRetries; i++ {
        conn, err := p.connect(ctx)
        if err == nil {
            return conn, nil
        }
        interval = min(interval*2, p.maxBackoff)
        time.Sleep(interval)
    }

    // Persistent retry phase
    if p.persistent {
        for {
            conn, err := p.connect(ctx)
            if err == nil {
                return conn, nil
            }
            time.Sleep(p.persistentInterval)
        }
    }

    return nil, ErrMaxRetriesExceeded
}
```

### Error Handling
```go
func (p *Pool) handleError(err error) {
    switch err := err.(type) {
    case *net.OpError:
        // Handle network operation errors
    case *tls.RecordHeaderError:
        // Handle TLS errors
    case *os.SyscallError:
        // Handle system call errors
    default:
        // Handle unknown errors
    }
}
```

## Monitoring & Alerting

### Metrics
```
sssonector_connection_attempts_total
sssonector_connection_failures_total
sssonector_retry_attempts_total
sssonector_connection_latency_seconds
```

### Alerts
```yaml
alerts:
  - name: ConnectionFailures
    condition: rate(sssonector_connection_failures_total[5m]) > 0.1
    severity: warning
  
  - name: PersistentConnectionFailure
    condition: time_since_last_success > 300
    severity: critical
```

## Configuration Examples

### High-Availability Setup
```yaml
connection:
  retry:
    immediate:
      attempts: 5
      interval: 2s
    gradual:
      attempts: 10
      base_interval: 15s
      max_interval: 300s
    persistent:
      enabled: true
      interval: 5m
```

### Conservative Setup
```yaml
connection:
  retry:
    immediate:
      attempts: 3
      interval: 10s
    gradual:
      attempts: 3
      base_interval: 60s
      max_interval: 600s
    persistent:
      enabled: false
```

## Best Practices

1. Error Classification
   - Categorize errors properly
   - Apply appropriate recovery strategies
   - Log detailed error information

2. Resource Management
   - Clean up failed connections
   - Monitor resource usage
   - Implement timeouts

3. Monitoring
   - Track retry attempts
   - Monitor success rates
   - Alert on persistent failures

4. Configuration
   - Set appropriate timeouts
   - Configure reasonable retry intervals
   - Enable persistent retries for critical services
