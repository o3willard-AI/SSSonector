# SSSonector API Reference

## Overview
This document provides comprehensive API documentation for SSSonector, including interface definitions, protocols, error handling, and integration guidelines.

## Table of Contents
1. [Core Interfaces](#core-interfaces)
2. [Protocol Specification](#protocol-specification)
3. [Error Handling](#error-handling)
4. [Rate Limiting](#rate-limiting)
5. [Authentication](#authentication)
6. [Integration Guidelines](#integration-guidelines)

## Core Interfaces

### Connection Manager
```go
// Manager handles connection pooling and lifecycle
type Manager interface {
    // Accept adds a new connection to the pool
    Accept(conn net.Conn) error
    
    // Close gracefully closes all connections
    Close() error
    
    // GetStats returns current connection statistics
    GetStats() Stats
    
    // SetConfig updates manager configuration
    SetConfig(cfg *Config) error
}

// Config defines connection manager configuration
type Config struct {
    MaxConnections  int
    KeepAlive      bool
    KeepAliveIdle  time.Duration
    RetryAttempts  int
    RetryInterval  time.Duration
    ConnectTimeout time.Duration
}

// Stats provides connection pool statistics
type Stats struct {
    ActiveConnections   int
    IdleConnections    int
    TotalConnections   int
    FailedConnections  int
    AverageLatency    time.Duration
}
```

### Load Balancer
```go
// Balancer provides load balancing functionality
type Balancer interface {
    // AddEndpoint adds a new endpoint with optional weight
    AddEndpoint(addr string, weight int) error
    
    // RemoveEndpoint removes an endpoint
    RemoveEndpoint(addr string) error
    
    // Connect returns a connection to a selected endpoint
    Connect(ctx context.Context) (net.Conn, error)
    
    // UpdateWeight updates endpoint weight
    UpdateWeight(addr string, weight int) error
}

// BalancerConfig defines load balancer configuration
type BalancerConfig struct {
    Strategy           Strategy
    HealthCheck        time.Duration
    RetryTimeout       time.Duration
    MaxRetries         int
    HealthyThreshold   int
    UnhealthyThreshold int
}

// Strategy defines load balancing algorithms
type Strategy string

const (
    RoundRobin         Strategy = "round_robin"
    LeastConnections   Strategy = "least_connections"
    WeightedRoundRobin Strategy = "weighted_round_robin"
)
```

### Rate Limiter
```go
// RateLimiter provides rate limiting functionality
type RateLimiter interface {
    // Allow checks if request is allowed under rate limit
    Allow(key string) bool
    
    // AllowN checks if N requests are allowed
    AllowN(key string, n int) bool
    
    // GetLimit returns current rate limit for key
    GetLimit(key string) Rate
    
    // SetLimit updates rate limit for key
    SetLimit(key string, rate Rate) error
}

// Rate defines rate limit parameters
type Rate struct {
    Requests int
    Interval time.Duration
    Burst    int
}
```

### Circuit Breaker
```go
// CircuitBreaker provides circuit breaking functionality
type CircuitBreaker interface {
    // Execute runs function with circuit breaking
    Execute(ctx context.Context, fn func() error) error
    
    // State returns current circuit breaker state
    State() State
    
    // Reset forces circuit breaker to closed state
    Reset()
    
    // Trip forces circuit breaker to open state
    Trip()
}

// State represents circuit breaker states
type State string

const (
    StateClosed    State = "closed"
    StateHalfOpen  State = "half_open"
    StateOpen      State = "open"
)
```

## Protocol Specification

### Connection Establishment
1. TCP connection established
2. TLS handshake with mutual authentication
3. Protocol version negotiation
4. Connection parameters exchange
5. Ready for data transfer

```sequence
Client->Server: TCP SYN
Server->Client: TCP SYN-ACK
Client->Server: TCP ACK
Client->Server: TLS ClientHello
Server->Client: TLS ServerHello, Certificate, ...
Client->Server: TLS Certificate, Finished
Server->Client: TLS Finished
Client->Server: Version: 2.0.0
Server->Client: Parameters: {...}
Client->Server: Ready
```

### Data Transfer
- Maximum packet size: 16KB
- Binary protocol
- Length-prefixed messages
- Compression support
- Flow control

### Packet Format
```
+----------------+----------------+----------------+
|    Length     |     Type      |    Payload    |
|    4 bytes    |    1 byte     |    N bytes    |
+----------------+----------------+----------------+
```

### Message Types
```go
const (
    TypeData       byte = 0x01
    TypeControl    byte = 0x02
    TypeHeartbeat  byte = 0x03
    TypeError      byte = 0x04
)
```

## Error Handling

### Error Codes
```go
const (
    // Connection errors
    ErrConnRefused    = "CONN_REFUSED"
    ErrConnTimeout    = "CONN_TIMEOUT"
    ErrConnClosed     = "CONN_CLOSED"
    
    // Authentication errors
    ErrAuthFailed     = "AUTH_FAILED"
    ErrCertInvalid    = "CERT_INVALID"
    ErrCertExpired    = "CERT_EXPIRED"
    
    // Rate limiting errors
    ErrRateLimited    = "RATE_LIMITED"
    ErrBurstExceeded  = "BURST_EXCEEDED"
    
    // Protocol errors
    ErrInvalidVersion = "INVALID_VERSION"
    ErrInvalidPacket  = "INVALID_PACKET"
    ErrInvalidType    = "INVALID_TYPE"
)
```

### Error Response Format
```json
{
    "code": "ERROR_CODE",
    "message": "Human readable message",
    "details": {
        "field1": "value1",
        "field2": "value2"
    },
    "timestamp": "2025-02-22T21:54:32Z"
}
```

### Error Handling Best Practices
1. Always check error codes
2. Implement appropriate retry logic
3. Log error details
4. Monitor error rates
5. Handle timeouts properly

## Rate Limiting

### Token Bucket Algorithm
```go
type TokenBucket struct {
    rate       float64     // Tokens per second
    burst      float64     // Maximum burst size
    tokens     float64     // Current tokens
    lastUpdate time.Time   // Last update time
}
```

### Rate Limit Configuration
```yaml
rate_limit:
  enabled: true
  default_rate: 1000
  default_burst: 100
  cleanup_interval: 60s
```

### Dynamic Rate Adjustment
```yaml
dynamic_rate:
  enabled: true
  min_rate: 100
  max_rate: 5000
  increase_factor: 1.5
  decrease_factor: 0.5
  cooldown: 1s
```

## Authentication

### Certificate Requirements
- X.509 certificates
- RSA 2048 bits minimum
- ECDSA P-256 minimum
- SHA-256 signatures
- Valid CA signature

### Authentication Flow
1. TLS handshake
2. Certificate validation
3. Role verification
4. Permission check
5. Connection accepted

### Configuration Example
```yaml
auth:
  cert_file: "/path/to/cert.pem"
  key_file: "/path/to/key.pem"
  ca_file: "/path/to/ca.pem"
  verify_client: true
  verify_depth: 2
```

## Integration Guidelines

### Connection Management
```go
// Create connection manager
cfg := &connection.Config{
    MaxConnections: 100,
    KeepAlive:     true,
    KeepAliveIdle: 30 * time.Second,
    RetryAttempts: 3,
}
manager := connection.NewManager(cfg)

// Accept connections
listener, err := net.Listen("tcp", ":8080")
for {
    conn, err := listener.Accept()
    if err != nil {
        log.Error(err)
        continue
    }
    manager.Accept(conn)
}
```

### Load Balancing
```go
// Create load balancer
cfg := &balancer.Config{
    Strategy:    balancer.RoundRobin,
    MaxRetries:  3,
}
lb := balancer.NewBalancer(cfg)

// Add endpoints
lb.AddEndpoint("10.0.0.1:8080", 1)
lb.AddEndpoint("10.0.0.2:8080", 1)

// Get connection
conn, err := lb.Connect(context.Background())
```

### Rate Limiting
```go
// Create rate limiter
cfg := &ratelimit.Config{
    DefaultRate:  1000,
    DefaultBurst: 100,
}
limiter := ratelimit.NewRateLimiter(cfg)

// Check rate limit
if !limiter.Allow("client-1") {
    return ErrRateLimited
}
```

### Circuit Breaking
```go
// Create circuit breaker
cfg := &breaker.Config{
    MaxFailures:  5,
    ResetTimeout: 10 * time.Second,
}
cb := breaker.NewCircuitBreaker(cfg)

// Execute with circuit breaker
err := cb.Execute(context.Background(), func() error {
    return someOperation()
})
```

## Best Practices

### Connection Management
1. Configure appropriate timeouts
2. Implement retry logic
3. Monitor connection health
4. Clean up resources properly
5. Handle errors gracefully

### Load Balancing
1. Regular health checks
2. Proper failover configuration
3. Monitor endpoint health
4. Adjust weights based on performance
5. Handle endpoint failures

### Rate Limiting
1. Configure appropriate limits
2. Monitor rate limit hits
3. Implement backoff strategy
4. Handle burst traffic
5. Clean up old entries

### Circuit Breaking
1. Configure proper thresholds
2. Monitor circuit state
3. Log state changes
4. Handle half-open state
5. Implement fallbacks

## Monitoring

### Metrics
```go
type Metrics struct {
    ActiveConnections   prometheus.Gauge
    RequestsTotal      prometheus.Counter
    RequestLatency     prometheus.Histogram
    ErrorsTotal        prometheus.Counter
    RateLimitHits      prometheus.Counter
    CircuitBreaks      prometheus.Counter
}
```

### Health Checks
```go
type Health struct {
    Status    string            `json:"status"`
    Details   map[string]string `json:"details"`
    Timestamp time.Time         `json:"timestamp"`
}
```

## Appendix

### Error Codes Reference
| Code | Description | Recovery Action |
|------|-------------|----------------|
| CONN_REFUSED | Connection refused | Retry with backoff |
| CONN_TIMEOUT | Connection timeout | Check network/firewall |
| AUTH_FAILED | Authentication failed | Verify certificates |
| RATE_LIMITED | Rate limit exceeded | Implement backoff |

### Configuration Reference
| Parameter | Default | Description |
|-----------|---------|-------------|
| max_connections | 1000 | Maximum concurrent connections |
| keep_alive | true | Enable TCP keepalive |
| retry_attempts | 3 | Number of retry attempts |
| timeout | 30s | Connection timeout |

### Protocol Constants
| Constant | Value | Description |
|----------|-------|-------------|
| MAX_PACKET_SIZE | 16384 | Maximum packet size |
| VERSION | 2.0.0 | Protocol version |
| COMPRESSION | true | Enable compression |
