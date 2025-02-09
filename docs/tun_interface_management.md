# TUN Interface Management

This document details the TUN interface lifecycle management in SSSonector, including state transitions, cleanup procedures, and error handling.

## Overview

SSSonector implements a robust TUN interface management system that ensures proper resource handling and cleanup in all scenarios. The system uses a state machine approach with timeout-based cleanup and automatic recovery mechanisms.

## Interface Lifecycle

### States

1. `StateUninitialized`:
   - Initial state when interface object is created
   - No system resources allocated
   - No cleanup required

2. `StateInitializing`:
   - Transitioning during interface creation
   - Allocating system resources
   - Creating TUN device

3. `StateReady`:
   - Interface fully configured and operational
   - Ready for data transfer
   - All resources properly allocated

4. `StateStopping`:
   - Transitioning during cleanup
   - Releasing system resources
   - Cleaning up routes and addresses

5. `StateStopped`:
   - Final state after successful cleanup
   - All resources released
   - No further operations allowed

6. `StateError`:
   - Error condition detected
   - Partial cleanup may be required
   - Recovery procedures initiated

### State Transitions

```
Uninitialized ---> Initializing ---> Ready ---> Stopping ---> Stopped
                        |              ^           |
                        |              |           |
                        +---> Error <--+           |
                                      ^           |
                                      +-----------+
```

## Implementation Details

### Configuration Options

```go
type Options struct {
    RetryAttempts  int  // Number of retry attempts for operations
    RetryDelay     int  // Milliseconds between retries
    CleanupTimeout int  // Milliseconds to wait for cleanup
    ValidateState  bool // Whether to validate interface state
}
```

Default values:
- RetryAttempts: 5
- RetryDelay: 200ms
- CleanupTimeout: 10000ms (10 seconds)
- ValidateState: true

### Thread Safety

The implementation uses a combination of `sync.Mutex` and `atomic.Value` to ensure thread-safe state transitions:

```go
type linuxInterface struct {
    state   atomic.Value // Holds InterfaceState
    stateMu sync.Mutex   // Protects state transitions
}
```

### Error Handling

1. State Transition Errors:
   - `ErrInvalidStateTransition`: Invalid state change attempted
   - `ErrInterfaceNotReady`: Operation attempted in wrong state
   - `ErrCleanupTimeout`: Cleanup operation timed out
   - `ErrConfigurationFailed`: Interface configuration failed

2. Recovery Procedures:
   - Automatic retry with exponential backoff
   - State validation before operations
   - Cleanup verification
   - Resource leak prevention

## Cleanup Process

### Normal Cleanup

1. Transition to StateStopping
2. Bring interface down
3. Remove IP address configuration
4. Release system resources
5. Verify cleanup completion
6. Transition to StateStopped

### Timeout-based Cleanup

```go
func (i *linuxInterface) Cleanup() error {
    if !i.transitionState(StateReady, StateStopping) {
        return ErrInvalidStateTransition
    }

    done := make(chan error, 1)
    go func() {
        done <- i.performCleanup()
    }()

    select {
    case err := <-done:
        if err != nil {
            i.setState(StateError)
            return err
        }
        i.setState(StateStopped)
        return nil
    case <-time.After(time.Duration(i.opts.CleanupTimeout) * time.Millisecond):
        i.setState(StateError)
        return ErrCleanupTimeout
    }
}
```

### Cleanup Verification

1. Check interface existence
2. Verify route removal
3. Confirm address removal
4. Validate system resources

## Best Practices

1. State Management:
   - Always check current state before operations
   - Use proper state transitions
   - Handle all error cases
   - Log state changes

2. Resource Handling:
   - Use defer for cleanup
   - Implement timeout protection
   - Verify resource release
   - Monitor system resources

3. Error Recovery:
   - Implement retry mechanisms
   - Log detailed error information
   - Maintain system stability
   - Prevent resource leaks

4. Monitoring:
   - Log all state transitions
   - Track cleanup success/failure
   - Monitor resource usage
   - Alert on repeated failures

## Example Usage

### Creating and Configuring Interface

```go
adapterOpts := &adapter.Options{
    RetryAttempts:  5,     // More retries for initial setup
    RetryDelay:     200,   // 200ms between retries
    CleanupTimeout: 10000, // 10 seconds for cleanup
    ValidateState:  true,  // Always validate interface state
}

iface, err := adapter.New(tunName, adapterOpts)
if err != nil {
    return fmt.Errorf("failed to create adapter: %w", err)
}

defer func() {
    if err := iface.Cleanup(); err != nil {
        log.Error("Failed to cleanup interface", err)
    }
}()
```

### Handling Cleanup

```go
func cleanup() error {
    // Ensure proper state transition
    if !transitionState(StateReady, StateStopping) {
        return ErrInvalidStateTransition
    }

    // Perform cleanup with timeout
    done := make(chan error, 1)
    go performCleanup(done)

    select {
    case err := <-done:
        return handleCleanupResult(err)
    case <-time.After(cleanupTimeout):
        return handleCleanupTimeout()
    }
}
```

## Troubleshooting

1. Interface Not Cleaning Up:
   - Check current interface state
   - Verify cleanup timeout settings
   - Look for blocked cleanup operations
   - Check system resource usage

2. State Transition Failures:
   - Verify current state
   - Check for concurrent operations
   - Review operation logs
   - Validate configuration

3. Resource Leaks:
   - Monitor system interfaces
   - Track resource allocation
   - Check cleanup completion
   - Verify error handling

4. Performance Issues:
   - Review retry settings
   - Check timeout values
   - Monitor system load
   - Analyze error patterns
