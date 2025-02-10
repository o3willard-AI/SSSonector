package throttle

import "time"

const (
	// TCP overhead factor for rate limiting calculations
	tcpOverheadFactor = 1.1 // 10% overhead for TCP headers

	// Default timeout for rate limiting operations
	defaultTimeout = 10 * time.Second

	// Default buffer sizes
	defaultReadBufferSize  = 64 * 1024 // 64KB
	defaultWriteBufferSize = 64 * 1024 // 64KB
	maxBufferPoolSize      = 1000      // Maximum number of buffers to keep in pool
)
