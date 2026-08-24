package provision

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// RedemptionServer serves one encrypted bundle over HTTPS, keyed by the
// normalized pairing code, exactly once, until TTL expiry.
//
// Trust model: the payload is AEAD-encrypted under the code-derived key, so
// transport TLS is defense-in-depth only — authenticity and confidentiality
// come from the envelope. Clients may skip certificate verification during
// redemption alone; the embedded CA fingerprint pins all tunnel trust after.
type RedemptionServer struct {
	bundle      []byte
	code        string // normalized
	ttl         time.Duration
	consumeOnce sync.Once
	done        chan struct{} // closed when bundle delivered OR ttl expired
	srv         *http.Server
	ln          net.Listener

	mu       sync.Mutex
	attempts map[string]int
}

// MaxAttemptsPerClient bounds guessing during the TTL window.
const MaxAttemptsPerClient = 10

// NewRedemptionServer validates inputs and prepares an unstarted server.
func NewRedemptionServer(bundle []byte, code string, ttl time.Duration) (*RedemptionServer, error) {
	norm, err := NormalizePairingCode(code)
	if err != nil {
		return nil, err
	}
	if ttl <= 0 {
		ttl = 15 * time.Minute
	}
	return &RedemptionServer{
		bundle:   bundle,
		code:     norm,
		ttl:      ttl,
		attempts: make(map[string]int),
		done:     make(chan struct{}),
	}, nil
}

// Listen binds the HTTPS listener (addr like ":9443").
func (r *RedemptionServer) Listen(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("redemption listen %s: %w", addr, err)
	}
	r.ln = ln
	return nil
}

// Addr returns the bound address after a successful Listen.
func (r *RedemptionServer) Addr() string {
	if r.ln == nil {
		return ""
	}
	return r.ln.Addr().String()
}

// ServeTLS blocks until the bundle is redeemed, the TTL expires, or ctx is
// cancelled. certFile/keyFile feed the TLS handshake.
func (r *RedemptionServer) ServeTLS(ctx context.Context, certFile, keyFile string) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	mux := http.NewServeMux()
	mux.HandleFunc("/pair/", r.handlePair)

	r.srv = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}

	go func() {
		select {
		case <-time.After(r.ttl):
		case <-r.done: // consumed
			// Small grace so the winning response flushes before shutdown.
			time.Sleep(500 * time.Millisecond)
		case <-ctx.Done():
		}
		_ = r.srv.Close()
	}()

	err := r.srv.ServeTLS(r.ln, certFile, keyFile)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	if r.consumed() && (err == nil || errors.Is(err, net.ErrClosed)) {
		return nil // normal post-redemption shutdown
	}
	return err
}

func (r *RedemptionServer) consumed() bool {
	r.consumeOnce.Do(func() {})
	select {
	case <-r.done:
		return true
	default:
		return false
	}
}

func (r *RedemptionServer) handlePair(w http.ResponseWriter, req *http.Request) {
	codeParam := strings.TrimPrefix(req.URL.Path, "/pair/")
	clientIP, _, _ := net.SplitHostPort(req.RemoteAddr)

	if !r.allowAttempt(clientIP) {
		http.Error(w, "too many attempts", http.StatusTooManyRequests)
		return
	}

	submitted, err := NormalizePairingCode(codeParam)
	if err != nil {
		http.Error(w, "invalid code", http.StatusBadRequest)
		return
	}
	if subtle.ConstantTimeCompare([]byte(submitted), []byte(r.code)) != 1 {
		http.Error(w, "invalid code", http.StatusForbidden)
		return
	}
	if r.markConsumed() {
		http.Error(w, "already redeemed", http.StatusGone)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(r.bundle)
}

// allowAttempt enforces the per-client guess budget for this TTL window.
func (r *RedemptionServer) allowAttempt(clientIP string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.attempts[clientIP]++
	return r.attempts[clientIP] <= MaxAttemptsPerClient
}

func (r *RedemptionServer) markConsumed() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	select {
	case <-r.done:
		return true
	default:
		close(r.done)
		return false
	}
}
