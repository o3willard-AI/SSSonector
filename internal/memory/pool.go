package memory

import (
	"sync"
)

const (
	// Common buffer sizes in bytes
	size4K   = 4 * 1024
	size8K   = 8 * 1024
	size16K  = 16 * 1024
	size32K  = 32 * 1024
	size64K  = 64 * 1024
	size128K = 128 * 1024
	size256K = 256 * 1024
	size512K = 512 * 1024
	size1M   = 1024 * 1024
)

// BufferPool manages pools of byte buffers of different sizes
type BufferPool struct {
	manager *Manager
	pools   map[int]*sync.Pool
}

// NewBufferPool creates a new buffer pool with memory limit management
func NewBufferPool(manager *Manager) *BufferPool {
	bp := &BufferPool{
		manager: manager,
		pools:   make(map[int]*sync.Pool),
	}

	// Initialize pools for common buffer sizes
	sizes := []int{
		size4K, size8K, size16K, size32K,
		size64K, size128K, size256K, size512K, size1M,
	}

	for _, size := range sizes {
		size := size // Capture for closure
		bp.pools[size] = &sync.Pool{
			New: func() interface{} {
				if bp.manager.CheckAndReserve(int64(size)) {
					return make([]byte, size)
				}
				return nil
			},
		}
	}

	// Register cleanup function
	manager.RegisterCleanupFunc(bp.cleanup)

	return bp
}

// Get returns a buffer of the specified size
// If the requested size matches a pool size, returns a pooled buffer
// Otherwise, allocates a new buffer if within memory limits
func (bp *BufferPool) Get(size int) []byte {
	// Check if we have a pool for this size
	if pool, ok := bp.pools[size]; ok {
		if buf := pool.Get(); buf != nil {
			return buf.([]byte)
		}
	}

	// For non-standard sizes or when pool is empty, allocate new buffer
	if bp.manager.CheckAndReserve(int64(size)) {
		return make([]byte, size)
	}

	return nil
}

// Put returns a buffer to its pool if it matches a standard size
// Otherwise, just releases the memory
func (bp *BufferPool) Put(buf []byte) {
	size := len(buf)
	if pool, ok := bp.pools[size]; ok {
		pool.Put(buf)
	} else {
		bp.manager.Release(int64(size))
	}
}

// cleanup releases memory from all pools
func (bp *BufferPool) cleanup() {
	for size, pool := range bp.pools {
		// Clear the pool by setting New to nil temporarily
		oldNew := pool.New
		pool.New = nil

		// Get and discard all buffers
		for {
			if buf := pool.Get(); buf == nil {
				break
			}
			bp.manager.Release(int64(size))
		}

		// Restore the New function
		pool.New = oldNew
	}
}

// GetClosestPoolSize returns the closest standard pool size that can accommodate
// the requested size. Returns -1 if the requested size is too large.
func (bp *BufferPool) GetClosestPoolSize(size int) int {
	sizes := []int{
		size4K, size8K, size16K, size32K,
		size64K, size128K, size256K, size512K, size1M,
	}

	for _, poolSize := range sizes {
		if poolSize >= size {
			return poolSize
		}
	}

	return -1
}

// Stats returns statistics about the buffer pools
type PoolStats struct {
	Size        int
	Capacity    int64
	InUse       int64
	Available   int64
	Allocations int64
}

// GetStats returns statistics for all buffer pools
func (bp *BufferPool) GetStats() map[int]PoolStats {
	stats := make(map[int]PoolStats)
	for size := range bp.pools {
		stats[size] = PoolStats{
			Size:     size,
			Capacity: bp.manager.config.MaxMemoryMB * 1024 * 1024,
		}
	}
	return stats
}

// Reset clears all pools and releases memory
func (bp *BufferPool) Reset() {
	bp.cleanup()
}

// IsManagedSize returns true if the given size matches a standard pool size
func (bp *BufferPool) IsManagedSize(size int) bool {
	_, ok := bp.pools[size]
	return ok
}
