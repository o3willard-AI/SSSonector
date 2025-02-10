package throttle

import (
	"sync"

	"go.uber.org/zap"
)

// BufferPool manages a pool of byte slices for efficient memory reuse
type BufferPool struct {
	readPool  sync.Pool
	writePool sync.Pool
	logger    *zap.Logger
}

// NewBufferPool creates a new buffer pool
func NewBufferPool(logger *zap.Logger) *BufferPool {
	bp := &BufferPool{
		logger: logger,
	}

	bp.readPool.New = func() interface{} {
		buf := make([]byte, defaultReadBufferSize)
		return &buf
	}

	bp.writePool.New = func() interface{} {
		buf := make([]byte, defaultWriteBufferSize)
		return &buf
	}

	logger.Info("Initialized buffer pool",
		zap.Int("read_size", defaultReadBufferSize),
		zap.Int("write_size", defaultWriteBufferSize),
		zap.Int("max_pool_size", maxBufferPoolSize),
		zap.Bool("preallocated", true),
	)

	return bp
}

// Get gets a buffer from the pool
func (p *BufferPool) Get(size int) []byte {
	var pool *sync.Pool

	if size <= defaultReadBufferSize {
		pool = &p.readPool
	} else {
		pool = &p.writePool
	}

	// Get buffer from pool
	bufPtr := pool.Get().(*[]byte)
	buf := *bufPtr

	// If buffer is too small, create a new one
	if len(buf) < size {
		newSize := roundUpToMultiple(size, 1024)
		buf = make([]byte, newSize)
		*bufPtr = buf
	}

	return buf[:size]
}

// Put returns a buffer to the pool
func (p *BufferPool) Put(buf []byte) {
	if buf == nil {
		return
	}

	// Determine which pool to use based on buffer size
	if cap(buf) <= defaultReadBufferSize {
		p.readPool.Put(&buf)
	} else {
		p.writePool.Put(&buf)
	}
}
