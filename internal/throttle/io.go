package throttle

import (
	"io"
	"time"
)

// RateLimitedReader wraps an io.Reader with rate limiting
type RateLimitedReader struct {
	reader io.Reader
	bucket *TokenBucket
}

// NewRateLimitedReader creates a new rate-limited reader
func NewRateLimitedReader(reader io.Reader, bytesPerSecond float64) *RateLimitedReader {
	return &RateLimitedReader{
		reader: reader,
		bucket: NewTokenBucket(bytesPerSecond, bytesPerSecond), // capacity = 1 second worth
	}
}

// Read implements io.Reader with rate limiting
func (r *RateLimitedReader) Read(p []byte) (n int, err error) {
	if !r.bucket.Take(float64(len(p))) {
		time.Sleep(time.Second / 10) // Sleep briefly and try again
		return 0, nil
	}
	return r.reader.Read(p)
}

// RateLimitedWriter wraps an io.Writer with rate limiting
type RateLimitedWriter struct {
	writer io.Writer
	bucket *TokenBucket
}

// NewRateLimitedWriter creates a new rate-limited writer
func NewRateLimitedWriter(writer io.Writer, bytesPerSecond float64) *RateLimitedWriter {
	return &RateLimitedWriter{
		writer: writer,
		bucket: NewTokenBucket(bytesPerSecond, bytesPerSecond), // capacity = 1 second worth
	}
}

// Write implements io.Writer with rate limiting
func (w *RateLimitedWriter) Write(p []byte) (n int, err error) {
	if !w.bucket.Take(float64(len(p))) {
		time.Sleep(time.Second / 10) // Sleep briefly and try again
		return 0, nil
	}
	return w.writer.Write(p)
}
