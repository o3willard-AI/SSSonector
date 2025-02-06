package throttle

import (
	"context"
	"io"
	"sync"

	"golang.org/x/time/rate"
)

// Limiter implements bandwidth throttling
type Limiter struct {
	reader   io.Reader
	writer   io.Writer
	inLimit  *rate.Limiter
	outLimit *rate.Limiter
	mu       sync.RWMutex
}

// NewLimiter creates a new bandwidth limiter
func NewLimiter(reader io.Reader, writer io.Writer, uploadKbps, downloadKbps int64) *Limiter {
	return &Limiter{
		reader:   reader,
		writer:   writer,
		inLimit:  rate.NewLimiter(rate.Limit(downloadKbps*1024), int(downloadKbps*1024)), // burst = 1 second worth
		outLimit: rate.NewLimiter(rate.Limit(uploadKbps*1024), int(uploadKbps*1024)),     // burst = 1 second worth
	}
}

// Read implements io.Reader with download throttling
func (l *Limiter) Read(p []byte) (n int, err error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.inLimit == nil {
		return l.reader.Read(p)
	}

	// Wait before reading to ensure we don't exceed the rate limit
	if err := l.inLimit.WaitN(context.Background(), len(p)); err != nil {
		return 0, err
	}

	return l.reader.Read(p)
}

// Write implements io.Writer with upload throttling
func (l *Limiter) Write(p []byte) (n int, err error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.outLimit == nil {
		return l.writer.Write(p)
	}

	// Wait before writing to ensure we don't exceed the rate limit
	if err := l.outLimit.WaitN(context.Background(), len(p)); err != nil {
		return 0, err
	}

	return l.writer.Write(p)
}

// SetUploadLimit updates the upload bandwidth limit
func (l *Limiter) SetUploadLimit(kbps int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.outLimit.SetLimit(rate.Limit(kbps * 1024))
	l.outLimit.SetBurst(int(kbps * 1024))
}

// SetDownloadLimit updates the download bandwidth limit
func (l *Limiter) SetDownloadLimit(kbps int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.inLimit.SetLimit(rate.Limit(kbps * 1024))
	l.inLimit.SetBurst(int(kbps * 1024))
}

// GetUploadLimit returns the current upload bandwidth limit in Kbps
func (l *Limiter) GetUploadLimit() int64 {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return int64(l.outLimit.Limit() / 1024)
}

// GetDownloadLimit returns the current download bandwidth limit in Kbps
func (l *Limiter) GetDownloadLimit() int64 {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return int64(l.inLimit.Limit() / 1024)
}
