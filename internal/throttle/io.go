package throttle

import (
	"io"
	"sync"
)

const (
	defaultBufferSize = 64 * 1024   // 64KB default buffer size
	minBufferSize     = 4 * 1024    // 4KB minimum buffer size
	maxBufferSize     = 1024 * 1024 // 1MB maximum buffer size
)

// BufferPool manages a pool of byte slices for efficient I/O operations
type BufferPool struct {
	pool    sync.Pool
	size    int
	maxSize int
}

// NewBufferPool creates a new buffer pool with the specified buffer size
func NewBufferPool(size, maxPoolSize int) *BufferPool {
	if size < minBufferSize {
		size = minBufferSize
	}
	if size > maxBufferSize {
		size = maxBufferSize
	}

	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
		size:    size,
		maxSize: maxPoolSize,
	}
}

// Get retrieves a buffer from the pool
func (p *BufferPool) Get() []byte {
	return p.pool.Get().([]byte)
}

// Put returns a buffer to the pool
func (p *BufferPool) Put(buf []byte) {
	if cap(buf) == p.size {
		p.pool.Put(buf)
	}
}

// ThrottledReader wraps an io.Reader with rate limiting and buffer pooling
type ThrottledReader struct {
	reader  io.Reader
	limiter *Limiter
	pool    *BufferPool
}

// NewThrottledReader creates a new throttled reader
func NewThrottledReader(reader io.Reader, limiter *Limiter, bufferSize, maxPoolSize int) *ThrottledReader {
	return &ThrottledReader{
		reader:  reader,
		limiter: limiter,
		pool:    NewBufferPool(bufferSize, maxPoolSize),
	}
}

// Read implements io.Reader with rate limiting and buffer pooling
func (r *ThrottledReader) Read(p []byte) (n int, err error) {
	if err := r.limiter.Wait(true); err != nil {
		return 0, err
	}

	// Use pooled buffer if destination buffer is large
	if len(p) >= minBufferSize {
		buf := r.pool.Get()
		defer r.pool.Put(buf)

		// Read into pooled buffer
		n, err = r.reader.Read(buf[:len(p)])
		if n > 0 {
			copy(p, buf[:n])
		}
		return n, err
	}

	// For small reads, use destination buffer directly
	return r.reader.Read(p)
}

// ThrottledWriter wraps an io.Writer with rate limiting and buffer pooling
type ThrottledWriter struct {
	writer  io.Writer
	limiter *Limiter
	pool    *BufferPool
}

// NewThrottledWriter creates a new throttled writer
func NewThrottledWriter(writer io.Writer, limiter *Limiter, bufferSize, maxPoolSize int) *ThrottledWriter {
	return &ThrottledWriter{
		writer:  writer,
		limiter: limiter,
		pool:    NewBufferPool(bufferSize, maxPoolSize),
	}
}

// Write implements io.Writer with rate limiting and buffer pooling
func (w *ThrottledWriter) Write(p []byte) (n int, err error) {
	if err := w.limiter.Wait(false); err != nil {
		return 0, err
	}

	// Use pooled buffer for large writes
	if len(p) >= minBufferSize {
		buf := w.pool.Get()
		defer w.pool.Put(buf)

		// Copy to pooled buffer
		copy(buf, p)
		return w.writer.Write(buf[:len(p)])
	}

	// For small writes, use source buffer directly
	return w.writer.Write(p)
}

// ThrottledReadWriter combines ThrottledReader and ThrottledWriter
type ThrottledReadWriter struct {
	*ThrottledReader
	*ThrottledWriter
}

// NewThrottledReadWriter creates a new throttled read/writer
func NewThrottledReadWriter(rw io.ReadWriter, limiter *Limiter, bufferSize, maxPoolSize int) *ThrottledReadWriter {
	return &ThrottledReadWriter{
		ThrottledReader: NewThrottledReader(rw, limiter, bufferSize, maxPoolSize),
		ThrottledWriter: NewThrottledWriter(rw, limiter, bufferSize, maxPoolSize),
	}
}
