package throttle

import (
	"io"

	"go.uber.org/zap"
)

// ThrottledReader implements a rate-limited io.Reader. Reads are paced
// through the limiter's inbound bucket.
type ThrottledReader struct {
	reader io.Reader
	limiter *Limiter
	logger *zap.Logger
}

// NewThrottledReader creates a new throttled reader.
func NewThrottledReader(reader io.Reader, limiter *Limiter, logger *zap.Logger) *ThrottledReader {
	return &ThrottledReader{
		reader: reader,
		limiter: limiter,
		logger: logger,
	}
}

// Read implements io.Reader.
func (r *ThrottledReader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	if err != nil {
		return n, err
	}
	if n > 0 && r.limiter.enabled {
		if werr := r.limiter.Wait(true, n); werr != nil {
			return n, werr
		}
	}
	return n, nil
}

// ThrottledWriter implements a rate-limited io.Writer. Writes are paced
// through the limiter's outbound bucket before being handed to the
// underlying writer.
type ThrottledWriter struct {
	writer io.Writer
	limiter *Limiter
	logger *zap.Logger
}

// NewThrottledWriter creates a new throttled writer.
func NewThrottledWriter(writer io.Writer, limiter *Limiter, logger *zap.Logger) *ThrottledWriter {
	return &ThrottledWriter{
		writer: writer,
		limiter: limiter,
		logger: logger,
	}
}

// Write implements io.Writer.
func (w *ThrottledWriter) Write(p []byte) (n int, err error) {
	if w.limiter.enabled {
		if werr := w.limiter.Wait(false, len(p)); werr != nil {
			return 0, werr
		}
	}
	return w.writer.Write(p)
}

// ThrottledReadWriter implements both io.Reader and io.Writer with rate limiting.
type ThrottledReadWriter struct {
	*ThrottledReader
	*ThrottledWriter
}

// NewThrottledReadWriter creates a new throttled read/writer.
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
