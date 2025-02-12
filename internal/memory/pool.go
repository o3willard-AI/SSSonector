package memory

import (
	"fmt"
	"sync"

	"go.uber.org/zap"
)

const (
	// Default buffer sizes
	DefaultMinSize = 4 * 1024    // 4KB
	DefaultMaxSize = 1024 * 1024 // 1MB
	DefaultBufSize = 32 * 1024   // 32KB
)

// BufferPool manages a pool of byte slices
type BufferPool struct {
	manager *Manager
	logger  *zap.Logger
	pool    sync.Pool
	minSize int
	maxSize int
}

// NewBufferPool creates a new buffer pool
func NewBufferPool(manager *Manager, logger *zap.Logger) *BufferPool {
	if logger == nil {
		var err error
		logger, err = zap.NewDevelopment()
		if err != nil {
			panic(fmt.Sprintf("failed to create logger: %v", err))
		}
	}

	bp := &BufferPool{
		manager: manager,
		logger:  logger,
		minSize: DefaultMinSize,
		maxSize: DefaultMaxSize,
	}

	bp.pool.New = func() interface{} {
		return make([]byte, DefaultBufSize)
	}

	// Register cleanup function
	manager.RegisterCleanup(bp.cleanup)

	return bp
}

// Get retrieves a buffer from the pool
func (bp *BufferPool) Get(size int) []byte {
	if size <= 0 {
		size = DefaultBufSize
	}

	// Check memory limit before allocation
	if !bp.manager.CheckAndReserve(uint64(size)) {
		return nil
	}

	var buf []byte

	// Try to get from pool if size is within range
	if size <= bp.maxSize {
		if v := bp.pool.Get(); v != nil {
			buf = v.([]byte)
			if cap(buf) >= size {
				return buf[:size]
			}
			// Buffer too small, put it back
			bp.pool.Put(buf)
		}
	}

	// Allocate new buffer
	return make([]byte, size)
}

// Put returns a buffer to the pool
func (bp *BufferPool) Put(buf []byte) {
	if buf == nil {
		return
	}

	size := cap(buf)

	// Release memory reservation
	bp.manager.Release(uint64(size))

	// Only pool buffers within size range
	if size >= bp.minSize && size <= bp.maxSize {
		bp.pool.Put(buf)
	}
}

// cleanup releases all pooled buffers
func (bp *BufferPool) cleanup() {
	// The sync.Pool will automatically clear when the GC runs
	// We just need to ensure memory is released
	bp.manager.Release(uint64(bp.maxSize))
}

// SetSizeRange sets the minimum and maximum buffer sizes
func (bp *BufferPool) SetSizeRange(minSize, maxSize int) error {
	if minSize < 0 || maxSize < minSize {
		return fmt.Errorf("invalid size range: min=%d max=%d", minSize, maxSize)
	}

	bp.minSize = minSize
	bp.maxSize = maxSize
	return nil
}

// GetSizeRange returns the current size range
func (bp *BufferPool) GetSizeRange() (int, int) {
	return bp.minSize, bp.maxSize
}
