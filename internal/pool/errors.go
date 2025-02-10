package pool

import "errors"

var (
	// ErrPoolClosed is returned when attempting to use a closed pool
	ErrPoolClosed = errors.New("connection pool is closed")

	// ErrPoolExhausted is returned when the pool has reached its maximum active connections
	ErrPoolExhausted = errors.New("connection pool exhausted")

	// ErrMaxRetriesExceeded is returned when all retry attempts have failed
	ErrMaxRetriesExceeded = errors.New("maximum retry attempts exceeded")

	// ErrInvalidConnection is returned when attempting to use an invalid connection
	ErrInvalidConnection = errors.New("invalid connection")

	// ErrConnectionClosed is returned when attempting to use a closed connection
	ErrConnectionClosed = errors.New("connection is closed")

	// ErrHealthCheckFailed is returned when a connection fails its health check
	ErrHealthCheckFailed = errors.New("connection health check failed")

	// ErrConnectionStale is returned when a connection has exceeded its idle timeout
	ErrConnectionStale = errors.New("connection is stale")
)
