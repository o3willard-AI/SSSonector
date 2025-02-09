package throttle

import (
	"io"
)

// RateLimitedReader implements io.Reader with rate limiting
type RateLimitedReader struct {
	reader io.Reader
	bucket *TokenBucket
}

// NewRateLimitedReader creates a new rate-limited reader
func NewRateLimitedReader(reader io.Reader, bytesPerSecond float64) *RateLimitedReader {
	return &RateLimitedReader{
		reader: reader,
		bucket: NewTokenBucket(bytesPerSecond, bytesPerSecond),
	}
}

// Read implements io.Reader
func (r *RateLimitedReader) Read(p []byte) (n int, err error) {
	// Take tokens for the requested bytes
	if !r.bucket.Take(float64(len(p))) {
		r.bucket.WaitN(float64(len(p)))
	}

	return r.reader.Read(p)
}

// RateLimitedWriter implements io.Writer with rate limiting
type RateLimitedWriter struct {
	writer io.Writer
	bucket *TokenBucket
}

// NewRateLimitedWriter creates a new rate-limited writer
func NewRateLimitedWriter(writer io.Writer, bytesPerSecond float64) *RateLimitedWriter {
	return &RateLimitedWriter{
		writer: writer,
		bucket: NewTokenBucket(bytesPerSecond, bytesPerSecond),
	}
}

// Write implements io.Writer
func (w *RateLimitedWriter) Write(p []byte) (n int, err error) {
	// Take tokens for the bytes to write
	if !w.bucket.Take(float64(len(p))) {
		w.bucket.WaitN(float64(len(p)))
	}

	return w.writer.Write(p)
}

// RateLimitedReadWriter implements io.ReadWriter with rate limiting
type RateLimitedReadWriter struct {
	*RateLimitedReader
	*RateLimitedWriter
}

// NewRateLimitedReadWriter creates a new rate-limited read-writer
func NewRateLimitedReadWriter(rw io.ReadWriter, bytesPerSecond float64) *RateLimitedReadWriter {
	return &RateLimitedReadWriter{
		RateLimitedReader: NewRateLimitedReader(rw, bytesPerSecond),
		RateLimitedWriter: NewRateLimitedWriter(rw, bytesPerSecond),
	}
}
