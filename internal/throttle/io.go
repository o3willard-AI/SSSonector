package throttle

import (
	"io"

	"go.uber.org/zap"
)

// ThrottledReader implements a rate-limited io.Reader
type ThrottledReader struct {
	reader  io.Reader
	limiter *Limiter
	pool    *BufferPool
	logger  *zap.Logger
}

// NewThrottledReader creates a new throttled reader
func NewThrottledReader(reader io.Reader, limiter *Limiter, logger *zap.Logger) *ThrottledReader {
	return &ThrottledReader{
		reader:  reader,
		limiter: limiter,
		pool:    limiter.bufferPool,
		logger:  logger,
	}
}

// Read implements io.Reader with rate limiting
func (r *ThrottledReader) Read(p []byte) (n int, err error) {
	return r.limiter.Read(p)
}

// ThrottledWriter implements a rate-limited io.Writer
type ThrottledWriter struct {
	writer  io.Writer
	limiter *Limiter
	pool    *BufferPool
	logger  *zap.Logger
}

// NewThrottledWriter creates a new throttled writer
func NewThrottledWriter(writer io.Writer, limiter *Limiter, logger *zap.Logger) *ThrottledWriter {
	return &ThrottledWriter{
		writer:  writer,
		limiter: limiter,
		pool:    limiter.bufferPool,
		logger:  logger,
	}
}

// Write implements io.Writer with rate limiting
func (w *ThrottledWriter) Write(p []byte) (n int, err error) {
	return w.limiter.Write(p)
}

// ThrottledReadWriter implements both io.Reader and io.Writer with rate limiting
type ThrottledReadWriter struct {
	*ThrottledReader
	*ThrottledWriter
}

// NewThrottledReadWriter creates a new throttled read/writer
func NewThrottledReadWriter(rw interface{}, limiter *Limiter, logger *zap.Logger) *ThrottledReadWriter {
	reader, ok1 := rw.(io.Reader)
	writer, ok2 := rw.(io.Writer)

	if !ok1 || !ok2 {
		logger.Error("Invalid ReadWriter interface")
		return nil
	}

	return &ThrottledReadWriter{
		ThrottledReader: NewThrottledReader(reader, limiter, logger),
		ThrottledWriter: NewThrottledWriter(writer, limiter, logger),
	}
}
