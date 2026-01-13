package pool

import (
	"sync"
	"time"
)

// BufferPool provides efficient buffer allocation and reuse with size bounds
type BufferPool struct {
	pool    sync.Pool
	minSize int
	maxSize int
	stats   BufferStats
	statsMu sync.RWMutex
}

// BufferStats represents buffer pool statistics
type BufferStats struct {
	Requests   int64
	Hits       int64
	Misses     int64
	BytesAlloc uint64
	HitRate    float64
}

// NewBufferPool creates a new buffer pool with specified size bounds
func NewBufferPool(minSize, maxSize int) *BufferPool {
	if minSize <= 0 || maxSize <= 0 || minSize > maxSize {
		panic("invalid buffer pool size bounds")
	}

	return &BufferPool{
		pool:    sync.Pool{},
		minSize: minSize,
		maxSize: maxSize,
		stats:   BufferStats{},
	}
}

// Get retrieves a buffer from the pool, clamped to size bounds
func (p *BufferPool) Get(size int) []byte {
	p.incRequests()

	// Clamp size to valid range
	if size < p.minSize {
		size = p.minSize
	}
	if size > p.maxSize {
		size = p.maxSize
	}

	// Try to get from pool
	if buf := p.pool.Get(); buf != nil {
		b := buf.([]byte)
		if cap(b) >= size {
			p.incHits()
			return b[:size]
		}
		// Buffer too small, will be GC'd
	}

	// Allocate new buffer and track allocation
	p.incMisses()
	p.incBytesAlloc(uint64(size))
	return make([]byte, size, size)
}

// Put returns a buffer to the pool if it meets criteria
func (p *BufferPool) Put(buf []byte) {
	if buf == nil || cap(buf) < p.minSize {
		return
	}

	// Clear buffer before pooling for security
	for i := range buf {
		buf[i] = 0
	}

	p.pool.Put(buf)
}

// GetStats returns current pool statistics
func (p *BufferPool) GetStats() BufferStats {
	p.statsMu.RLock()
	defer p.statsMu.RUnlock()

	stats := p.stats
	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRate = float64(stats.Hits) / float64(total) * 100
	}
	return stats
}

// incRequests increment request counter thread-safely
func (p *BufferPool) incRequests() {
	p.statsMu.Lock()
	p.stats.Requests++
	p.statsMu.Unlock()
}

// incHits increment hit counter thread-safely
func (p *BufferPool) incHits() {
	p.statsMu.Lock()
	p.stats.Hits++
	p.statsMu.Unlock()
}

// incMisses increment miss counter thread-safely
func (p *BufferPool) incMisses() {
	p.statsMu.Lock()
	p.stats.Misses++
	p.statsMu.Unlock()
}

// incBytesAlloc increment bytes allocated counter thread-safely
func (p *BufferPool) incBytesAlloc(bytes uint64) {
	p.statsMu.Lock()
	p.stats.BytesAlloc += bytes
	p.statsMu.Unlock()
}

// MinSize returns minimum buffer size
func (p *BufferPool) MinSize() int {
	return p.minSize
}

// MaxSize returns maximum buffer size
func (p *BufferPool) MaxSize() int {
	return p.maxSize
}

// RequestPool manages request object pooling for performance
type RequestPool struct {
	pool sync.Pool
}

// NewRequestPool creates a new request pool
func NewRequestPool() *RequestPool {
	return &RequestPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &Request{}
			},
		},
	}
}

// Get retrieves a request from the pool
func (p *RequestPool) Get() *Request {
	return p.pool.Get().(*Request)
}

// Put returns a request to the pool
func (p *RequestPool) Put(req *Request) {
	if req == nil {
		return
	}
	req.Reset()
	p.pool.Put(req)
}

// Request represents a pooled request object with timestamp and retries
type Request struct {
	ID        string
	Type      string
	Payload   []byte
	Response  chan *Response
	Timestamp time.Time
	Retries   int
	Deadline  time.Time
	Context   interface{}
}

// Reset resets the request state for safe reuse
func (r *Request) Reset() {
	r.ID = ""
	r.Type = ""
	r.Payload = nil
	r.Response = nil
	r.Timestamp = time.Time{}
	r.Retries = 0
	r.Deadline = time.Time{}
	r.Context = nil
}

// ResponsePool manages response object pooling
type ResponsePool struct {
	pool sync.Pool
}

// NewResponsePool creates a new response pool
func NewResponsePool() *ResponsePool {
	return &ResponsePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &Response{}
			},
		},
	}
}

// Get retrieves a response from the pool
func (p *ResponsePool) Get() *Response {
	return p.pool.Get().(*Response)
}

// Put returns a response to the pool
func (p *ResponsePool) Put(resp *Response) {
	if resp == nil {
		return
	}
	resp.Reset()
	p.pool.Put(resp)
}

// Response represents a pooled response object
type Response struct {
	ID      string
	Data    []byte
	Error   error
	Status  int
	Headers map[string]string
}

// Reset resets the response state for safe reuse
func (r *Response) Reset() {
	r.ID = ""
	r.Data = nil
	r.Error = nil
	r.Status = 0
	r.Headers = nil
}

// ChunkPool manages fixed-size chunk buffer pooling for streaming
type ChunkPool struct {
	pool       sync.Pool
	size       int
	capacity   int64
	capacityMu sync.RWMutex
}

// NewChunkPool creates a new chunk pool with specified chunk size
func NewChunkPool(size int) *ChunkPool {
	if size <= 0 {
		panic("chunk size must be positive")
	}

	return &ChunkPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
		size:     size,
		capacity: 0,
	}
}

// Get retrieves a chunk from the pool
func (p *ChunkPool) Get() []byte {
	buf := p.pool.Get().([]byte)
	if cap(buf) == p.size {
		p.capacityMu.Lock()
		p.capacity++
		p.capacityMu.Unlock()
	}
	return buf
}

// Put returns a chunk to the pool
func (p *ChunkPool) Put(chunk []byte) {
	if chunk == nil || cap(chunk) != p.size {
		return
	}
	p.pool.Put(chunk)
	p.capacityMu.Lock()
	p.capacity--
	p.capacityMu.Unlock()
}

// Size returns the chunk size
func (p *ChunkPool) Size() int {
	return p.size
}

// Capacity returns current pool capacity
func (p *ChunkPool) Capacity() int64 {
	p.capacityMu.RLock()
	defer p.capacityMu.RUnlock()
	return p.capacity
}

// PoolMetrics tracks comprehensive pool performance metrics
type PoolMetrics struct {
	GetTotal    int64
	GetOk       int64
	GetFail     int64
	PutTotal    int64
	PutFail     int64
	CurrentSize int64
	HitRate     float64
	LastUpdate  time.Time
}

// AtomicPool wraps a BufferPool with atomic metrics tracking
type AtomicPool struct {
	pool    *BufferPool
	metrics PoolMetrics
	mu      sync.RWMutex
}

// NewAtomicPool creates a new atomic pool with metrics
func NewAtomicPool(minSize, maxSize int) *AtomicPool {
	return &AtomicPool{
		pool: NewBufferPool(minSize, maxSize),
		metrics: PoolMetrics{
			LastUpdate: time.Now(),
		},
	}
}

// Get retrieves a buffer with metrics tracking
func (p *AtomicPool) Get(size int) []byte {
	p.mu.Lock()
	p.metrics.GetTotal++
	p.metrics.LastUpdate = time.Now()
	p.mu.Unlock()

	buf := p.pool.Get(size)

	p.mu.Lock()
	p.metrics.GetOk++
	p.metrics.CurrentSize++
	p.mu.Unlock()

	return buf
}

// Put returns a buffer with metrics tracking
func (p *AtomicPool) Put(buf []byte) {
	success := true
	// Handle put operation (would need to track failures)
	p.pool.Put(buf)
	if !success {
		p.mu.Lock()
		p.metrics.PutFail++
		p.mu.Unlock()
	} else {
		p.mu.Lock()
		p.metrics.PutTotal++
		p.metrics.CurrentSize--
		p.mu.Unlock()
	}
}

// GetBufferPool returns the underlying buffer pool
func (p *AtomicPool) GetBufferPool() *BufferPool {
	return p.pool
}

// GetMetrics returns current pool metrics
func (p *AtomicPool) GetMetrics() PoolMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats := p.pool.GetStats()
	metrics := p.metrics
	metrics.HitRate = stats.HitRate

	return metrics
}

// ResetMetrics resets all metrics to zero
func (p *AtomicPool) ResetMetrics() {
	p.mu.Lock()
	p.metrics = PoolMetrics{LastUpdate: time.Now()}
	p.mu.Unlock()
}
