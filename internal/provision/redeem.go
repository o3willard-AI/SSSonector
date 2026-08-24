package provision

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
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
	caDir       string // enables CSR signing when non-empty
	consumeOnce sync.Once
	done        chan struct{} // closed when bundle delivered OR ttl expired
	srv         *http.Server
	ln          net.Listener

	mu       sync.Mutex
	attempts map[string]int

	delivered  atomic.Bool // set when the offer has been delivered once
	responseMu sync.Mutex  // serializes terminal response writes
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
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		fmt.Printf("[HTTPDBG] %s %s from %s caDir=%q\n", req.Method, req.URL.RequestURI(), req.RemoteAddr, r.caDir)
		switch {
		case strings.HasPrefix(req.URL.Path, "/pair-csr/") && r.caDir != "":
			r.handlePairCSR(w, req)
		case strings.HasPrefix(req.URL.Path, "/pair/"):
			r.handlePair(w, req)
		default:
			http.NotFound(w, req)
		}
	})

	r.srv = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		BaseContext:       func(net.Listener) context.Context { return ctx },
	}

	go func() {
		select {
		case <-time.After(r.ttl):
		case <-r.done: // consumed; give the flushed response a moment to drain
			time.Sleep(250 * time.Millisecond)
		case <-ctx.Done():
		}
		_ = r.srv.Close()
	}()

	err := r.srv.ServeTLS(r.ln, certFile, keyFile)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	if r.deliveredYet() && (err == nil || errors.Is(err, net.ErrClosed)) {
		return nil // normal post-redemption shutdown
	}
	return err
}

func (r *RedemptionServer) deliveredYet() bool {
	return r.delivered.Load()
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
	r.finishAndShutdown(w, r.bundle)
}

// allowAttempt enforces the per-client guess budget for this TTL window.
func (r *RedemptionServer) allowAttempt(clientIP string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.attempts[clientIP]++
	return r.attempts[clientIP] <= MaxAttemptsPerClient
}

// markConsumed returns true when this caller LOST the race (already taken).
func (r *RedemptionServer) markConsumed() bool {
	return !r.delivered.CompareAndSwap(false, true)
}

// finishAndShutdown flushes the terminal response, then triggers listener
// shutdown. Safe against double-close because done closes exactly once here.
func (r *RedemptionServer) finishAndShutdown(w http.ResponseWriter, body []byte) {
	r.responseMu.Lock()
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write(body)
	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
	r.responseMu.Unlock()

	r.consumeOnce.Do(func() { close(r.done) })
}

// handlePairCSR signs a client-submitted CSR after code authentication.
// Request: POST /pair-csr/<code>, body = PEM CERTIFICATE REQUEST.
// Response: 200 body = leaf PEM followed by CA PEM. Single consumption and
// rate limiting match the bundle endpoint.
func (r *RedemptionServer) handlePairCSR(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	codeParam := strings.TrimPrefix(req.URL.Path, "/pair-csr/")
	clientIP, _, _ := net.SplitHostPort(req.RemoteAddr)

	if !r.allowAttempt(clientIP) {
		http.Error(w, "too many attempts", http.StatusTooManyRequests)
		return
	}
	submitted, nerr := NormalizePairingCode(codeParam)
	fmt.Printf("[CSRDBG] path=%q submitted=%q stored=%q nerr=%v\n",
		req.URL.Path, submitted, r.code, nerr)
	if nerr != nil || subtle.ConstantTimeCompare([]byte(submitted), []byte(r.code)) != 1 {
		http.Error(w, "invalid code", http.StatusForbidden)
		return
	}
	if r.markConsumed() {
		http.Error(w, "already redeemed", http.StatusGone)
		return
	}

	body, err := io.ReadAll(io.LimitReader(req.Body, 16<<10))
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}
	leaf, err := SignCSR(string(body), r.caDir, 0)
	if err != nil {
		// Signing failures are almost always malformed submissions; the
		// underlying reason carries no secret material.
		http.Error(w, fmt.Sprintf("sign failed: %v", err), http.StatusBadRequest)
		return
	}
	caPEM, rerr := os.ReadFile(filepath.Join(r.caDir, "ca.crt"))
	if rerr != nil {
		http.Error(w, "CA unavailable", http.StatusInternalServerError)
		return
	}
	r.finishAndShutdown(w, []byte(leaf+"\n"+string(caPEM)))
}

// EnableCSRSigning activates the /pair-csr endpoint backed by caDir.
func (r *RedemptionServer) EnableCSRSigning(caDir string) { r.caDir = caDir }
