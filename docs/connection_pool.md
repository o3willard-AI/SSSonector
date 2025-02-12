# Connection Pool

The connection pool provides efficient management of network connections with support for health checks, connection retries, and proper context handling.

## Configuration

```go
type Config struct {
    // InitialSize is the initial number of connections to create
    InitialSize int `yaml:"initialSize" json:"initialSize"`

    // MaxSize is the maximum number of connections allowed in the pool
    MaxSize int `yaml:"maxSize" json:"maxSize"`

    // MinSize is the minimum number of connections to maintain
    MinSize int `yaml:"minSize" json:"minSize"`

    // MaxIdleTime is how long a connection can remain idle before being closed
    MaxIdleTime types.Duration `yaml:"maxIdleTime" json:"maxIdleTime"`

    // ConnectTimeout is the maximum time to wait for a new connection
    ConnectTimeout types.Duration `yaml:"connectTimeout" json:"connectTimeout"`

    // HealthCheck is a function to verify connection health
    HealthCheck func(net.Conn) error

    // Factory is used to create new connections
    Factory Factory

    // OnClose is called when a connection is closed
    OnClose func(net.Conn)
}
```

## Usage

### Creating a Pool

```go
import (
    "time"
    "github.com/o3willard-AI/SSSonector/internal/config/types"
)

config := pool.Config{
    InitialSize:    5,
    MaxSize:        10,
    MinSize:        2,
    MaxIdleTime:    types.NewDuration(time.Minute),
    ConnectTimeout: types.NewDuration(5 * time.Second),
    Factory: func(ctx context.Context) (net.Conn, error) {
        var d net.Dialer
        return d.DialContext(ctx, "tcp", "example.com:80")
    },
    HealthCheck: func(conn net.Conn) error {
        // Implement health check logic
        return nil
    },
}

p, err := pool.NewPool(config)
if err != nil {
    log.Fatal(err)
}
defer p.Close()
```

### Getting a Connection

```go
// With context timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

conn, err := p.Get(ctx)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        // Handle timeout
    }
    // Handle other errors
    return err
}
defer p.Put(conn)
```

### Retry Strategy

The pool includes a retry manager that implements three retry strategies:

1. Immediate Retries
   - Quick retries with minimal delay
   - Useful for transient failures

2. Gradual Retries (Exponential Backoff)
   - Increasing delays between attempts
   - Prevents overwhelming the system

3. Persistent Retries
   - Long-interval retries
   - Used for extended outages

```go
config := pool.RetryConfig{
    MaxImmediateRetries:  3,
    ImmediateDelay:       time.Millisecond,
    MaxGradualRetries:    5,
    InitialInterval:      time.Millisecond,
    MaxInterval:          10 * time.Millisecond,
    BackoffMultiplier:    2.0,
    MaxPersistentRetries: 2,
    PersistentDelay:      5 * time.Second,
}
```

## Error Handling

The pool handles several types of errors:

- `context.DeadlineExceeded`: When the context timeout is reached
- `context.Canceled`: When the context is canceled
- `ErrMaxRetriesExceeded`: When all retry attempts fail
- Health check failures: When a connection fails its health check

## Metrics

The pool provides metrics through the `GetMetrics()` method:

```go
success, failures := pool.GetMetrics()
```

## Best Practices

1. Always use context timeouts to prevent indefinite blocking
2. Implement appropriate health checks
3. Configure pool size based on your application's needs
4. Use metrics to monitor pool health
5. Properly handle connection cleanup with `defer p.Put(conn)`

## Breaking Changes in v2.0.0

1. Duration type changes:
   - Now using centralized types.Duration from internal/config/types
   - Provides consistent JSON/YAML marshaling
   - Better validation support

2. Configuration updates:
   - Use types.NewDuration() for duration fields
   - Updated validation rules
   - Enhanced error messages

3. Import changes:
   ```go
   // Old
   import "github.com/o3willard-AI/SSSonector/internal/pool"
   
   // New
   import (
       "github.com/o3willard-AI/SSSonector/internal/pool"
       "github.com/o3willard-AI/SSSonector/internal/config/types"
   )
   ```

Please update your code accordingly if you're upgrading from a previous version.
