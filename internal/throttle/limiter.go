package throttle

import (
	"io"
	"time"

	"golang.org/x/time/rate"
)

// Limiter implements bandwidth throttling for network connections
type Limiter struct {
	reader     io.Reader
	writer     io.Writer
	readLimit  *rate.Limiter
	writeLimit *rate.Limiter
}

// NewLimiter creates a new bandwidth limiter
func NewLimiter(reader io.Reader, writer io.Writer, uploadKbps, downloadKbps int64) *Limiter {
	var readLimit, writeLimit *rate.Limiter

	if downloadKbps > 0 {
		bytesPerSec := downloadKbps * 1024 / 8 // Convert Kbps to bytes per second
		readLimit = rate.NewLimiter(rate.Limit(bytesPerSec), int(bytesPerSec))
	}

	if uploadKbps > 0 {
		bytesPerSec := uploadKbps * 1024 / 8 // Convert Kbps to bytes per second
		writeLimit = rate.NewLimiter(rate.Limit(bytesPerSec), int(bytesPerSec))
	}

	return &Limiter{
		reader:     reader,
		writer:     writer,
		readLimit:  readLimit,
		writeLimit: writeLimit,
	}
}

// Read implements io.Reader with bandwidth throttling
func (l *Limiter) Read(p []byte) (n int, err error) {
	if l.readLimit == nil {
		return l.reader.Read(p)
	}

	n, err = l.reader.Read(p)
	if n <= 0 {
		return n, err
	}

	now := time.Now()
	reservation := l.readLimit.ReserveN(now, n)
	if !reservation.OK() {
		reservation.Cancel()
		return n, err
	}

	delay := reservation.DelayFrom(now)
	if delay > 0 {
		time.Sleep(delay)
	}

	return n, err
}

// Write implements io.Writer with bandwidth throttling
func (l *Limiter) Write(p []byte) (n int, err error) {
	if l.writeLimit == nil {
		return l.writer.Write(p)
	}

	n, err = l.writer.Write(p)
	if n <= 0 {
		return n, err
	}

	now := time.Now()
	reservation := l.writeLimit.ReserveN(now, n)
	if !reservation.OK() {
		reservation.Cancel()
		return n, err
	}

	delay := reservation.DelayFrom(now)
	if delay > 0 {
		time.Sleep(delay)
	}

	return n, err
}

// Close implements io.Closer
func (l *Limiter) Close() error {
	if closer, ok := l.reader.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			return err
		}
	}
	if closer, ok := l.writer.(io.Closer); ok {
		if err := closer.Close(); err != nil {
			return err
		}
	}
	return nil
}

// SetUploadLimit updates the upload bandwidth limit
func (l *Limiter) SetUploadLimit(kbps int64) {
	if kbps > 0 {
		bytesPerSec := kbps * 1024 / 8
		l.writeLimit = rate.NewLimiter(rate.Limit(bytesPerSec), int(bytesPerSec))
	} else {
		l.writeLimit = nil
	}
}

// SetDownloadLimit updates the download bandwidth limit
func (l *Limiter) SetDownloadLimit(kbps int64) {
	if kbps > 0 {
		bytesPerSec := kbps * 1024 / 8
		l.readLimit = rate.NewLimiter(rate.Limit(bytesPerSec), int(bytesPerSec))
	} else {
		l.readLimit = nil
	}
}

// GetUploadLimit returns the current upload limit in Kbps
func (l *Limiter) GetUploadLimit() int64 {
	if l.writeLimit == nil {
		return 0
	}
	return int64(l.writeLimit.Limit()) * 8 / 1024
}

// GetDownloadLimit returns the current download limit in Kbps
func (l *Limiter) GetDownloadLimit() int64 {
	if l.readLimit == nil {
		return 0
	}
	return int64(l.readLimit.Limit()) * 8 / 1024
}
