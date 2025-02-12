# SSSonector

SSSonector is a high-performance, enterprise-grade communications utility designed to allow critical services to connect to and exchange data with one another over the public internet without needing a VPN.

## Features

- **Connection Pooling**
  - Dynamic pool size management
  - Connection health checks
  - Automatic cleanup of stale connections
  - Resource utilization monitoring

- **Rate Limiting**
  - Token bucket algorithm
  - Dynamic rate adjustment
  - Per-connection rate limiting
  - Burst allowance
  - Automatic cleanup

- **Circuit Breaking**
  - State machine implementation
  - Failure detection and tracking
  - Half-open state testing
  - Automatic recovery
  - Circuit breaker metrics

- **Load Balancing**
  - Multiple balancing strategies
    - Round Robin
    - Least Connections
    - Weighted Round Robin
  - Health checking
  - Automatic failover
  - Connection distribution
  - Endpoint management

- **Connection Tracking**
  - Connection statistics
  - Performance metrics
  - Resource monitoring
  - Duration tracking
  - Peak usage tracking

## Installation

```bash
go get github.com/o3willard-AI/SSSonector
```

## Usage

### Basic Connection Management

```go
// Create connection manager
cfg := &connection.Config{
    MaxConnections: 100,
    KeepAlive:      true,
    KeepAliveIdle:  30 * time.Second,
    RetryAttempts:  3,
    RetryInterval:  time.Second,
    ConnectTimeout: 5 * time.Second,
}
manager := connection.NewManager(logger, cfg)

// Accept connections
conn, err := net.Accept()
if err != nil {
    log.Fatal(err)
}
manager.Accept(conn)

// Get connection stats
stats := manager.GetStats()
```

### Load Balancing

```go
// Create load balancer
cfg := &balancer.Config{
    Strategy:           balancer.RoundRobin,
    HealthCheck:        time.Second,
    RetryTimeout:       time.Second,
    MaxRetries:         3,
    HealthyThreshold:   2,
    UnhealthyThreshold: 3,
}
lb := balancer.NewBalancer(logger, cfg)

// Add endpoints
lb.AddEndpoint("localhost:8081", 1)
lb.AddEndpoint("localhost:8082", 1)

// Get connection
conn, err := lb.Connect(context.Background())
if err != nil {
    log.Fatal(err)
}
```

### Rate Limiting

```go
// Create rate limiter
cfg := &ratelimit.Config{
    DefaultRate:  1000,
    DefaultBurst: 100,
    CleanupTime:  time.Minute,
}
limiter := ratelimit.NewRateLimiter(cfg, logger)

// Check rate limit
if !limiter.Allow("client-1") {
    return fmt.Errorf("rate limit exceeded")
}
```

### Circuit Breaking

```go
// Create circuit breaker
cfg := &breaker.Config{
    MaxFailures:      5,
    ResetTimeout:     10 * time.Second,
    HalfOpenMaxCalls: 2,
    FailureWindow:    time.Minute,
}
cb := breaker.NewCircuitBreaker(cfg, logger)

// Execute with circuit breaker
err := cb.Execute(context.Background(), func() error {
    return someOperation()
})
```

### Performance Testing

```bash
# Run benchmark with default settings
go run cmd/benchmark/main.go

# Run benchmark with custom settings
go run cmd/benchmark/main.go \
  -connections 100 \
  -duration 1m \
  -payload 2048 \
  -interval 50ms \
  -warmup 10s
```

## Configuration

### Connection Manager
- `MaxConnections`: Maximum number of concurrent connections
- `KeepAlive`: Enable TCP keepalive
- `KeepAliveIdle`: Time before sending keepalive probes
- `RetryAttempts`: Number of connection retry attempts
- `RetryInterval`: Time between retry attempts
- `ConnectTimeout`: Connection timeout duration

### Load Balancer
- `Strategy`: Load balancing strategy (RoundRobin, LeastConnections, WeightedRoundRobin)
- `HealthCheck`: Health check interval
- `RetryTimeout`: Time between retry attempts
- `MaxRetries`: Maximum number of retry attempts
- `HealthyThreshold`: Number of successes to mark endpoint healthy
- `UnhealthyThreshold`: Number of failures to mark endpoint unhealthy

### Rate Limiter
- `DefaultRate`: Default requests per second
- `DefaultBurst`: Maximum burst size
- `CleanupTime`: Interval for cleaning up inactive rate limiters

### Circuit Breaker
- `MaxFailures`: Maximum failures before opening
- `ResetTimeout`: Time before attempting reset
- `HalfOpenMaxCalls`: Maximum calls in half-open state
- `FailureWindow`: Time window for counting failures

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the LICENSE file for details.
