# Connection Pool Testing Guide

This document outlines the test coverage and QA procedures for the connection pool component.

## Test Categories

### 1. Basic Pool Operations

#### Test: TestPool/basic_pool_operations
- Verifies basic connection acquisition and return
- Checks connection reuse functionality
- Validates pool initialization
```go
import (
    "time"
    "github.com/o3willard-AI/SSSonector/internal/config/types"
)

config := pool.Config{
    InitialSize:    1,
    MaxSize:        2,
    MinSize:        1,
    MaxIdleTime:    types.NewDuration(time.Minute),
    ConnectTimeout: types.NewDuration(time.Second),
    Factory:        factory,
}

pool, err := NewPool(config)
conn1, err := pool.Get(context.Background())
// Use conn1
pool.Put(conn1)
conn2, err := pool.Get(context.Background()) // Should get the same connection
```

### 2. Context Handling

#### Test: TestPool/pool_exhaustion
- Validates context timeout when pool is exhausted
- Ensures proper cleanup on context cancellation
- Verifies deadlock prevention
```go
config := pool.Config{
    InitialSize:    1,
    MaxSize:        1,
    MinSize:        1,
    MaxIdleTime:    types.NewDuration(time.Minute),
    ConnectTimeout: types.NewDuration(time.Second),
    Factory:        factory,
}

ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
defer cancel()
_, err := pool.Get(ctx) // Should return context.DeadlineExceeded
```

#### Test: TestPool/connection_timeout
- Tests connection timeout during creation
- Verifies context propagation to factory
- Validates timeout error handling
```go
factory := func(ctx context.Context) (net.Conn, error) {
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    case <-time.After(50 * time.Millisecond):
        return &mockConn{}, nil
    }
}
```

### 3. Duration Type Handling

#### Test: TestDurationMarshaling
- Validates JSON marshaling/unmarshaling
- Tests YAML marshaling/unmarshaling
- Verifies string representation
```go
config := pool.Config{
    MaxIdleTime: types.NewDuration(5 * time.Minute),
}
data, err := json.Marshal(config)
// Verify marshaled format
var decoded pool.Config
err = json.Unmarshal(data, &decoded)
// Verify decoded duration
```

#### Test: TestDurationValidation
- Tests duration bounds validation
- Verifies negative duration handling
- Validates zero duration cases
```go
cases := []struct {
    name     string
    duration types.Duration
    valid    bool
}{
    {"valid", types.NewDuration(time.Minute), true},
    {"zero", types.NewDuration(0), false},
    {"negative", types.NewDuration(-time.Second), false},
}
```

### 4. Health Checks

#### Test: TestPool/health_check_failure
- Validates health check functionality
- Tests connection replacement on health check failure
- Verifies metrics update
```go
failingHealthCheck := func(conn net.Conn) error {
    return errors.New("health check failed")
}
```

### 5. Retry Behavior

#### Test: TestRetryManagerImmediateSuccess
- Tests immediate retry strategy
- Verifies quick recovery from transient failures
- Validates retry metrics

#### Test: TestRetryManagerGradualSuccess
- Tests exponential backoff strategy
- Verifies increasing delay between attempts
- Validates backoff limits

#### Test: TestRetryManagerPersistentSuccess
- Tests persistent retry strategy
- Verifies long-term recovery behavior
- Validates retry chain completion

## Test Coverage Requirements

1. Core Functionality (100% coverage required)
   - Connection acquisition/return
   - Pool initialization
   - Resource cleanup

2. Error Handling (100% coverage required)
   - Context cancellation
   - Timeout handling
   - Health check failures

3. Duration Type (100% coverage required)
   - Marshaling/Unmarshaling
   - Validation
   - String representation

4. Edge Cases (90% coverage minimum)
   - Pool exhaustion
   - Maximum retries
   - Concurrent operations

## Running Tests

```bash
# Run all pool tests
go test ./internal/pool/... -v

# Run with race detector
go test ./internal/pool/... -race

# Run benchmarks
go test ./internal/benchmark/... -v
```

## Test Environment Requirements

1. System Resources
   - Minimum 1GB RAM
   - Network access for connection tests
   - Stable system clock for timing tests

2. Network Conditions
   - Local network interface available
   - No firewall blocking test ports
   - Stable network connection

3. Test Dependencies
   - Go 1.21 or higher
   - `testify` package for assertions
   - `zaptest` for logging in tests

## Common Test Scenarios

1. Duration Type Testing
   ```go
   // Test JSON marshaling
   duration := types.NewDuration(5 * time.Minute)
   data, err := json.Marshal(duration)
   
   // Test YAML marshaling
   data, err := yaml.Marshal(duration)
   
   // Test validation
   err = validator.Validate(duration)
   ```

2. Error Handling
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), timeout)
   defer cancel()
   conn, err := pool.Get(ctx)
   if err != nil {
       // Handle error
   }
   ```

3. Retry Behavior
   ```go
   manager := NewRetryManager(logger, factory)
   manager.WithConfig(retryConfig)
   conn, err := manager.GetConnection(ctx)
   ```

## Test Maintenance

1. Regular Updates
   - Review test coverage monthly
   - Update test cases for new features
   - Maintain test documentation

2. Performance Monitoring
   - Track test execution times
   - Monitor resource usage
   - Update benchmarks as needed

3. Quality Checks
   - Run tests with race detector
   - Verify cleanup in tests
   - Check for test isolation

## Breaking Changes in v2.0.0

1. Duration Type Updates
   - Now using types.Duration from internal/config/types
   - Updated test cases for marshaling/validation
   - New test coverage requirements

2. Configuration Testing
   - Updated validation test cases
   - New duration-specific tests
   - Enhanced error message verification

Please ensure all tests are updated when making changes to the Duration type implementation.
