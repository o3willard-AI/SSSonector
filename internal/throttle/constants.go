package throttle

import "time"

const (
	// tcpOverheadFactor accounts for TCP/IP overhead in rate calculations
	tcpOverheadFactor = 1.1 // 10% overhead

	// defaultTimeout is the default timeout for rate limiting operations
	defaultTimeout = 5 * time.Second

	// defaultBufferSize is the default size for read/write buffers
	defaultBufferSize = 32 * 1024 // 32KB

	// defaultReadBufferSize is the default size for read buffers
	defaultReadBufferSize = 64 * 1024 // 64KB

	// defaultWriteBufferSize is the default size for write buffers
	defaultWriteBufferSize = 64 * 1024 // 64KB

	// maxBufferPoolSize is the maximum number of buffers in the pool
	maxBufferPoolSize = 1024
)
