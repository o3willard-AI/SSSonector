package throttle

import (
	"io"

	"go.uber.org/zap"
)

// ThrottledReader implements a rate-limited io.Reader
type ThrottledReader struct {
	reader io.Reader
	pool   *BufferPool
	logger *zap.Logger
}

// NewThrottledReader creates a new throttled reader
func NewThrottledReader(reader io.Reader, limiter *Limiter, logger *zap.Logger) *ThrottledReader {
	return &ThrottledReader{
		reader: reader,
		pool:   limiter.bufferPool,
		logger: logger,
	}
}

// Read implements io.Reader
func (r *ThrottledReader) Read(p []byte) (n int, err error) {
	// Get buffer from pool
	buf := r.pool.Get(len(p))
	defer r.pool.Put(buf)

	// Read into buffer
	n, err = r.reader.Read(buf)
	if err != nil {
		return 0, err
	}

	// Copy to output buffer
	copy(p, buf[:n])
	return n, nil
}

// ThrottledWriter implements a rate-limited io.Writer
type ThrottledWriter struct {
	writer io.Writer
	pool   *BufferPool
	logger *zap.Logger
}

// NewThrottledWriter creates a new throttled writer
func NewThrottledWriter(writer io.Writer, limiter *Limiter, logger *zap.Logger) *ThrottledWriter {
	return &ThrottledWriter{
		writer: writer,
		pool:   limiter.bufferPool,
		logger: logger,
	}
}

// Write implements io.Writer
func (w *ThrottledWriter) Write(p []byte) (n int, err error) {
	// Get buffer from pool
	buf := w.pool.Get(len(p))
	defer w.pool.Put(buf)

	// Copy input to buffer
	copy(buf, p)

	// Write from buffer
	return w.writer.Write(buf)
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
