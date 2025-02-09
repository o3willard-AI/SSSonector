package throttle

import (
	"context"
	"io"
	"sync"
	"time"

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
	// Account for TCP overhead (~5%) and use smaller burst size
	adjustedDownloadRate := rate.Limit(float64(downloadKbps*1024) * 1.05) // 5% overhead
	adjustedUploadRate := rate.Limit(float64(uploadKbps*1024) * 1.05)     // 5% overhead
	downloadBurst := int(float64(downloadKbps*1024) * 0.1)                // 100ms worth of data
	uploadBurst := int(float64(uploadKbps*1024) * 0.1)                    // 100ms worth of data

	return &Limiter{
		reader:   reader,
		writer:   writer,
		inLimit:  rate.NewLimiter(adjustedDownloadRate, downloadBurst),
		outLimit: rate.NewLimiter(adjustedUploadRate, uploadBurst),
	}
}

// Read implements io.Reader with download throttling
func (l *Limiter) Read(p []byte) (n int, err error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if l.inLimit == nil {
		return l.reader.Read(p)
	}

	// Use context with timeout to prevent indefinite waiting
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := l.inLimit.WaitN(ctx, len(p)); err != nil {
		if err == context.DeadlineExceeded {
			return 0, io.ErrShortBuffer // Return recoverable error
		}
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

	// Use context with timeout to prevent indefinite waiting
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := l.outLimit.WaitN(ctx, len(p)); err != nil {
		if err == context.DeadlineExceeded {
			return 0, io.ErrShortBuffer // Return recoverable error
		}
		return 0, err
	}

	return l.writer.Write(p)
}

// SetUploadLimit updates the upload bandwidth limit
func (l *Limiter) SetUploadLimit(kbps int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	adjustedRate := rate.Limit(float64(kbps*1024) * 1.05) // 5% overhead
	burst := int(float64(kbps*1024) * 0.1)                // 100ms worth of data
	l.outLimit.SetLimit(adjustedRate)
	l.outLimit.SetBurst(burst)
}

// SetDownloadLimit updates the download bandwidth limit
func (l *Limiter) SetDownloadLimit(kbps int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	adjustedRate := rate.Limit(float64(kbps*1024) * 1.05) // 5% overhead
	burst := int(float64(kbps*1024) * 0.1)                // 100ms worth of data
	l.inLimit.SetLimit(adjustedRate)
	l.inLimit.SetBurst(burst)
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
