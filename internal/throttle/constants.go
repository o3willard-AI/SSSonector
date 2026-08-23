package throttle

import "time"

const (
	// TCPOverheadFactor accounts for TCP/IP overhead in rate calculations
	TCPOverheadFactor = 1.1 // 10% overhead

	// defaultTimeout is the default timeout for rate limiting operations
	defaultTimeout = 5 * time.Second
)
