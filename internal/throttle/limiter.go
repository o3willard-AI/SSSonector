package throttle

import (
	"context"
	"io"
	"sync"

	"golang.org/x/time/rate"
)

// Limiter implements bandwidth throttling
type Limiter struct {
	limiter *rate.Limiter
	mu      sync.RWMutex
}

// NewLimiter creates a new bandwidth limiter
func NewLimiter(bytesPerSecond int64) *Limiter {
	var limiter *rate.Limiter
	if bytesPerSecond > 0 {
		limiter = rate.NewLimiter(rate.Limit(bytesPerSecond), int(bytesPerSecond))
	}
	return &Limiter{
		limiter: limiter,
	}
}

// SetLimit updates the bandwidth limit
func (l *Limiter) SetLimit(bytesPerSecond int64) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if bytesPerSecond > 0 {
		if l.limiter == nil {
			l.limiter = rate.NewLimiter(rate.Limit(bytesPerSecond), int(bytesPerSecond))
		} else {
			l.limiter.SetLimit(rate.Limit(bytesPerSecond))
			l.limiter.SetBurst(int(bytesPerSecond))
		}
	} else {
		l.limiter = nil
	}
}

// Reader wraps an io.Reader with bandwidth throttling
type Reader struct {
	reader  io.Reader
	limiter *Limiter
}

// NewReader creates a new throttled reader
func (l *Limiter) NewReader(r io.Reader) *Reader {
	return &Reader{
		reader:  r,
		limiter: l,
	}
}

// Read implements io.Reader with bandwidth throttling
func (r *Reader) Read(p []byte) (n int, err error) {
	r.limiter.mu.RLock()
	limiter := r.limiter.limiter
	r.limiter.mu.RUnlock()

	if limiter == nil {
		return r.reader.Read(p)
	}

	n, err = r.reader.Read(p)
	if n > 0 {
		if err := limiter.WaitN(context.Background(), n); err != nil {
			return n, err
		}
	}
	return n, err
}

// Writer wraps an io.Writer with bandwidth throttling
type Writer struct {
	writer  io.Writer
	limiter *Limiter
}

// NewWriter creates a new throttled writer
func (l *Limiter) NewWriter(w io.Writer) *Writer {
	return &Writer{
		writer:  w,
		limiter: l,
	}
}

// Write implements io.Writer with bandwidth throttling
func (w *Writer) Write(p []byte) (n int, err error) {
	w.limiter.mu.RLock()
	limiter := w.limiter.limiter
	w.limiter.mu.RUnlock()

	if limiter == nil {
		return w.writer.Write(p)
	}

	n, err = w.writer.Write(p)
	if n > 0 {
		if err := limiter.WaitN(context.Background(), n); err != nil {
			return n, err
		}
	}
	return n, err
}

// ThrottledCopy copies from src to dst with bandwidth throttling
func (l *Limiter) ThrottledCopy(dst io.Writer, src io.Reader) (written int64, err error) {
	l.mu.RLock()
	limiter := l.limiter
	l.mu.RUnlock()

	if limiter == nil {
		return io.Copy(dst, src)
	}

	buf := make([]byte, 32*1024)
	for {
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = io.ErrShortWrite
				}
			}
			written += int64(nw)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
			if err := limiter.WaitN(context.Background(), nr); err != nil {
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}
