package buffer

import (
	"sync"
)

// Pool manages multiple pools of byte slices for efficient memory reuse
type Pool struct {
	pools   []*sync.Pool
	minSize int
	maxSize int
}

// Config holds configuration for the buffer pool
type Config struct {
	// MinSize is the minimum buffer size to allocate
	MinSize int
	// MaxSize is the maximum buffer size to allocate
	MaxSize int
}

// sizeClass represents a buffer size class
type sizeClass struct {
	size  int
	class int
}

// nextPowerOfTwo returns the next power of two greater than or equal to n
func nextPowerOfTwo(n int) int {
	if n <= 0 {
		return 1
	}
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++
	return n
}

// getSizeClass returns the appropriate size class for a given size
func getSizeClass(size, minSize int) sizeClass {
	if size <= minSize {
		return sizeClass{minSize, 0}
	}

	// Round up to the next power of 2
	actualSize := nextPowerOfTwo(size)
	minSizePower := nextPowerOfTwo(minSize)

	// Calculate class based on the difference in powers of 2
	class := 0
	for s := minSizePower; s < actualSize; s *= 2 {
		class++
	}

	return sizeClass{actualSize, class}
}

// NewPool creates a new buffer pool with the given configuration
func NewPool(cfg Config) *Pool {
	if cfg.MinSize <= 0 {
		cfg.MinSize = 1500 // Default MTU size
	}
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = cfg.MinSize * 2
	}

	// Round up max size to next power of 2
	maxSize := nextPowerOfTwo(cfg.MaxSize)
	minSize := nextPowerOfTwo(cfg.MinSize)

	// Calculate number of size classes
	numClasses := 1
	for size := minSize; size < maxSize; size *= 2 {
		numClasses++
	}

	pools := make([]*sync.Pool, numClasses)
	for i := range pools {
		size := minSize << uint(i)
		pools[i] = &sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, size)
			},
		}
	}

	return &Pool{
		pools:   pools,
		minSize: cfg.MinSize,
		maxSize: cfg.MaxSize,
	}
}

// Get retrieves a buffer from the pool with at least the specified size
func (p *Pool) Get(size int) []byte {
	if size <= 0 {
		size = p.minSize
	}
	if size > p.maxSize {
		// For unusually large requests, allocate directly
		return make([]byte, size)
	}

	sc := getSizeClass(size, p.minSize)
	if sc.class >= len(p.pools) {
		// Size class too large, allocate directly
		return make([]byte, size)
	}

	buf := p.pools[sc.class].Get().([]byte)
	return buf[:size]
}

// Put returns a buffer to the pool
func (p *Pool) Put(buf []byte) {
	cap := cap(buf)
	if cap > p.maxSize || cap < p.minSize {
		// Don't pool buffers that are too large or too small
		return
	}

	sc := getSizeClass(cap, p.minSize)
	if sc.class >= len(p.pools) {
		// Size class too large, don't pool
		return
	}

	// Reset length but keep capacity
	buf = buf[:0]
	p.pools[sc.class].Put(buf)
}

// GetWithMTU gets a buffer sized according to the interface MTU
func (p *Pool) GetWithMTU(mtu int) []byte {
	if mtu <= 0 {
		mtu = p.minSize
	}
	return p.Get(mtu)
}

// GetZeroCopy gets a buffer that can be used for zero-copy operations
// The returned buffer will have its capacity set to maxSize to allow for
// direct kernel reads without reallocations
func (p *Pool) GetZeroCopy() []byte {
	buf := p.Get(p.maxSize)
	return buf[:0]
}

// GetBatch gets multiple buffers of the same size for batch processing
func (p *Pool) GetBatch(size, count int) [][]byte {
	bufs := make([][]byte, count)
	for i := range bufs {
		bufs[i] = p.Get(size)
	}
	return bufs
}

// PutBatch returns multiple buffers to the pool
func (p *Pool) PutBatch(bufs [][]byte) {
	for _, buf := range bufs {
		p.Put(buf)
	}
}
