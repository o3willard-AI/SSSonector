package throttle

import "time"

const (
	// tcpOverheadFactor accounts for TCP/IP overhead in rate calculations
	tcpOverheadFactor = 1.1

	// defaultTimeout is the default timeout for rate limiting operations
	defaultTimeout = 30 * time.Second

	// defaultBufferSize is the default size for read/write buffers
	defaultBufferSize = 32 * 1024

	// defaultReadBufferSize is the default size for read buffers
	defaultReadBufferSize = 32 * 1024

	// defaultWriteBufferSize is the default size for write buffers
	defaultWriteBufferSize = 32 * 1024

	// maxBufferPoolSize is the maximum number of buffers in the pool
	maxBufferPoolSize = 1024
)
