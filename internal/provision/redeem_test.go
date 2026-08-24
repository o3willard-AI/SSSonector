package provision

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/cert/generator"
)

// startRedemption spins a full TLS redemption server on 127.0.0.1:0 backed
// by generated host certs, mirroring `provision create --serve`.
func startRedemption(t *testing.T, code string, ttl time.Duration) *RedemptionServer {
	t.Helper()
	dir := t.TempDir()
	if err := generator.GenerateCertificates(dir); err != nil {
		t.Fatalf("GenerateCertificates: %v", err)
	}
	payload := validPayload()
	bundle, err := Seal(payload, code)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}

	srv, err := NewRedemptionServer(bundle, code, ttl)
	if err != nil {
		t.Fatalf("NewRedemptionServer: %v", err)
	}
	if err := srv.Listen("127.0.0.1:0"); err != nil {
		t.Fatalf("Listen: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() {
		_ = srv.ServeTLS(ctx, filepath.Join(dir, "server.crt"), filepath.Join(dir, "server.key"))
	}()

	waitFor := time.Now().Add(3 * time.Second)
	for time.Now().Before(waitFor) {
		conn, derr := net.DialTimeout("tcp", srv.Addr(), 200*time.Millisecond)
		if derr == nil {
			conn.Close()
			return srv
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatal("redemption server never became reachable")
	return nil
}

func redeemClient() *http.Client {
	return &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{TLSClientConfig: &tls.Config{
			// #nosec G402 -- redemption authenticity comes from the AEAD tag,
			// matching the production apply --from URL path.
			InsecureSkipVerify: true,
		}},
	}
}

func get(t *testing.T, cl *http.Client, url string) (int, []byte) {
	t.Helper()
	resp, err := cl.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return resp.StatusCode, body
}

func TestRedemptionHappyPathSingleConsumption(t *testing.T) {
	code := "AAAA-BBBB"
	srv := startRedemption(t, code, time.Minute)
	cl := redeemClient()
	url := "https://" + srv.Addr() + "/pair/" + code

	status, body := get(t, cl, url)
	if status != http.StatusOK {
		t.Fatalf("first redemption status = %d, want 200", status)
	}
	payload, err := Open(body, code)
	if err != nil {
		t.Fatalf("redeemed bundle does not open: %v", err)
	}
	if payload.ServerAddr != validPayload().ServerAddr {
		t.Errorf("payload server addr = %q", payload.ServerAddr)
	}

	status, _ = get(t, cl, url)
	if status != http.StatusGone {
		t.Errorf("second redemption status = %d, want 410 Gone", status)
	}
}

func TestRedemptionRateLimitAndWrongCode(t *testing.T) {
	srv := startRedemption(t, "CCCC-DDDD", time.Minute)
	cl := redeemClient()

	last := 0
	for i := 0; i <= MaxAttemptsPerClient; i++ {
		status, _ := get(t, cl, "https://"+srv.Addr()+"/pair/WRNG-WRNG")
		last = status
		if status == http.StatusTooManyRequests {
			break
		}
	}
	if last != http.StatusTooManyRequests {
		t.Fatalf("expected eventual 429, got %d after %d attempts", last, MaxAttemptsPerClient+1)
	}
}

func TestRedemptionTTLClosesServer(t *testing.T) {
	srv := startRedemption(t, "EEEE-FFFF", 150*time.Millisecond)
	time.Sleep(300 * time.Millisecond)

	cl := &http.Client{Timeout: 2 * time.Second}
	_, err := cl.Get("https://" + srv.Addr() + "/pair/" + "EEEE-FFFF")
	if err == nil {
		t.Fatal("expected connection failure after TTL expiry")
	}
}
