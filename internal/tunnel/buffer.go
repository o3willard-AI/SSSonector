package tunnel

import (
	"github.com/o3willard-AI/SSSonector/internal/buffer"
)

var (
	// defaultBufferConfig is the default configuration for tunnel buffers
	defaultBufferConfig = buffer.Config{
		MinSize: 1500, // Standard MTU size
		MaxSize: 9000, // Jumbo frame size
	}

	// globalBufferPool is a shared buffer pool for all tunnels
	globalBufferPool = buffer.NewPool(defaultBufferConfig)
)

// getBuffer gets a buffer from the pool sized according to the MTU
func getBuffer(mtu int) []byte {
	return globalBufferPool.GetWithMTU(mtu)
}

// putBuffer returns a buffer to the pool
func putBuffer(buf []byte) {
	globalBufferPool.Put(buf)
}
