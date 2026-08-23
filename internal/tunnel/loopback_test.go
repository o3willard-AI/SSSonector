package tunnel

import (
	"errors"
	"net"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/adapter"
	"github.com/o3willard-AI/SSSonector/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// fakeInterface is a user-space stand-in for a TUN device: one end of an
// in-memory pipe, so two instances form a virtual point-to-point link.
type fakeInterface struct {
	name   string
	pipe   net.Conn
	mu     sync.Mutex
	closed bool

	configureCalls int
	cleanedUp      bool
}

func (f *fakeInterface) Read(p []byte) (int, error)  { return f.pipe.Read(p) }
func (f *fakeInterface) Write(p []byte) (int, error) { return f.pipe.Write(p) }
func (f *fakeInterface) GetName() string             { return f.name }
func (f *fakeInterface) GetMTU() int                 { return 1500 }
func (f *fakeInterface) GetAddress() string          { return "10.77.0.1/24" }
func (f *fakeInterface) IsUp() bool                  { return true }

func (f *fakeInterface) Configure(cfg *adapter.Config) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.configureCalls++
	return nil
}

func (f *fakeInterface) Cleanup() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cleanedUp = true
	_ = f.pipe.Close()
	f.closed = true
	return nil
}

func (f *fakeInterface) SetReadDeadline(d time.Time) error {
	return f.pipe.SetReadDeadline(d)
}

func (f *fakeInterface) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.closed {
		_ = f.pipe.Close()
		f.closed = true
	}
	return nil
}

var errUnexpectedAdapter = errors.New("unexpected adapter name requested")

// newLinkedAdapters returns provider pairs wiring "tuns" and "tunc" to the
// two ends of one in-memory link.
func newLinkedAdapters() (serverProvider, clientProvider func(name string, opts *adapter.Options) (adapter.Interface, error), serverIf, clientIf *fakeInterface) {
	a, b := net.Pipe()
	serverIf = &fakeInterface{name: "tuns", pipe: a}
	clientIf = &fakeInterface{name: "tunc", pipe: b}
	serverProvider = func(name string, _ *adapter.Options) (adapter.Interface, error) {
		if name != "tuns" {
			return nil, errUnexpectedAdapter
		}
		return serverIf, nil
	}
	clientProvider = func(name string, _ *adapter.Options) (adapter.Interface, error) {
		if name != "tunc" {
			return nil, errUnexpectedAdapter
		}
		return clientIf, nil
	}
	return
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port
}

func loopbackConfig(t *testing.T, dir string, port int) *config.AppConfig {
	t.Helper()
	cfg := config.DefaultConfig()
	cfg.Type = config.TypeServer
	cfg.Config.Mode = "server"
	cfg.Config.Logging.Level = "debug"
	cfg.Config.Network.Name = "tuns"
	cfg.Config.Network.Interface = "tuns"
	cfg.Config.Network.Address = "10.77.0.1/24"
	cfg.Config.Tunnel.ListenAddress = "127.0.0.1"
	cfg.Config.Tunnel.ListenPort = port
	cfg.Config.Metrics.Enabled = false
	cfg.Throttle.Enabled = false
	_ = os.MkdirAll(dir, 0o750)
	return cfg
}

func TestLoopbackServerClientExchange(t *testing.T) {
	core, observed := observer.New(zapcore.DebugLevel)
	logger := zap.New(core)

	port := freePort(t)
	dir := t.TempDir()

	srvCfg := loopbackConfig(t, dir, port)
	server := NewServer(srvCfg, nil, logger.Named("server"), nil)
	srvProv, cliProv, srvIf, cliIf := newLinkedAdapters()
	server.AdapterNew = srvProv

	cliCfg := loopbackConfig(t, dir, port)
	cliCfg.Type = config.TypeClient
	cliCfg.Config.Mode = "client"
	cliCfg.Config.Network.Name = "tunc"
	cliCfg.Config.Network.Interface = "tunc"
	cliCfg.Config.Network.Address = "10.77.0.2/24"
	cliCfg.Config.Tunnel.ServerAddress = "127.0.0.1"
	cliCfg.Config.Tunnel.ServerPort = port
	cliCfg.Config.Auth.CertFile = "" // plaintext loopback path
	cliCfg.Config.Auth.KeyFile = ""
	cliCfg.Config.Auth.CAFile = ""
	client := NewClient(cliCfg, nil, logger.Named("client"), nil)
	client.AdapterNew = cliProv

	if err := server.Start(); err != nil {
		t.Fatalf("server Start: %v", err)
	}
	if err := client.Start(); err != nil {
		t.Fatalf("client Start: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Stop()
		_ = server.Stop()
	})

	waitForLog(t, observed, "Tunnel established", 10*time.Second)

	if srvIf.configureCalls != 1 {
		t.Errorf("server adapter configured %d times, want 1", srvIf.configureCalls)
	}
	if cliIf.configureCalls != 1 {
		t.Errorf("client adapter configured %d times, want 1", cliIf.configureCalls)
	}

	// Push payload from the client-side TUN into the tunnel and expect it on
	// the server-side TUN end: client transfer reads tunc, writes TCP; server
	// transfer reads TCP, writes tuns.
	payload := []byte("loopback-packet-001")
	go func() {
		_, _ = cliIf.pipe.Write(payload)
	}()

	buf := make([]byte, len(payload))
	if err := readFullDeadline(srvIf.pipe, buf, 10*time.Second); err != nil {
		t.Fatalf("payload did not traverse the tunnel: %v", err)
	}
	if string(buf) != string(payload) {
		t.Errorf("payload mismatch: %q", buf)
	}

	if got := server.bytesOut.Load(); got < int64(len(payload)) {
		t.Errorf("server bytesOut = %d, want >= %d", got, len(payload))
	}
	if got := client.bytesIn.Load(); got < int64(len(payload)) {
		t.Errorf("client bytesIn = %d, want >= %d", got, len(payload))
	}

	// Graceful stop from the client side must not hang.
	stopDone := make(chan struct{})
	go func() {
		_ = client.Stop()
		close(stopDone)
	}()
	select {
	case <-stopDone:
	case <-time.After(10 * time.Second):
		buf := make([]byte, 1<<16)
		n := runtime.Stack(buf, true)
		t.Fatalf("client.Stop hung\n%s", buf[:n])
	}
}

func readFullDeadline(c net.Conn, buf []byte, d time.Duration) error {
	_ = c.SetReadDeadline(time.Now().Add(d))
	total := 0
	for total < len(buf) {
		n, err := c.Read(buf[total:])
		total += n
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() && total == len(buf) {
				return nil
			}
			return err
		}
	}
	return nil
}

func waitForLog(t *testing.T, obs *observer.ObservedLogs, msg string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		for _, e := range obs.All() {
			if e.Message == msg {
				return
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("log %q not observed within %v", msg, timeout)
}

// TestLoopbackReconnectReusesAdapter drives the production reconnect path:
// the server drops the connection, and the client reconnects on the SAME
// TUN instance without leaking readers or losing the link.
func TestLoopbackReconnectReusesAdapter(t *testing.T) {
	core, observed := observer.New(zapcore.DebugLevel)
	logger := zap.New(core)

	port := freePort(t)
	dir := t.TempDir()

	srvCfg := loopbackConfig(t, dir, port)
	server := NewServer(srvCfg, nil, logger.Named("server"), nil)
	srvProv, cliProv, srvIf, cliIf := newLinkedAdapters()
	server.AdapterNew = srvProv

	cliCfg := loopbackConfig(t, dir, port)
	cliCfg.Type = config.TypeClient
	cliCfg.Config.Mode = "client"
	cliCfg.Config.Network.Name = "tunc"
	cliCfg.Config.Network.Interface = "tunc"
	cliCfg.Config.Network.Address = "10.77.0.2/24"
	cliCfg.Config.Tunnel.ServerAddress = "127.0.0.1"
	cliCfg.Config.Tunnel.ServerPort = port
	client := NewClient(cliCfg, nil, logger.Named("client"), nil)
	client.AdapterNew = cliProv

	if err := server.Start(); err != nil {
		t.Fatalf("server Start: %v", err)
	}
	if err := client.Start(); err != nil {
		t.Fatalf("client Start: %v", err)
	}
	t.Cleanup(func() {
		_ = client.Stop()
		_ = server.Stop()
	})

	waitForLog(t, observed, "Tunnel established", 10*time.Second)

	exchange := func(tag string) {
		payload := []byte("pkt-" + tag)
		go func() { _, _ = cliIf.pipe.Write(payload) }()
		buf := make([]byte, len(payload))
		if err := readFullDeadline(srvIf.pipe, buf, 10*time.Second); err != nil {
			t.Fatalf("[%s] payload lost: %v", tag, err)
		}
	}

	exchange("r1")

	// Server drops the connection; client must reconnect automatically.
	server.mu.Lock()
	dropped := server.activeConn != nil
	if server.activeConn != nil {
		server.activeConn.Close()
	}
	server.mu.Unlock()
	if !dropped {
		t.Fatal("expected an active server-side connection to drop")
	}
	waitForLogCount(t, observed, "Tunnel established", 2, 15*time.Second)
	time.Sleep(100 * time.Millisecond)

	exchange("r2")
}

func waitForLogCount(t *testing.T, obs *observer.ObservedLogs, msg string, want int, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if n := len(obs.FilterMessage(msg).All()); n >= want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("log %q seen %d times, want >= %d within %v", msg, len(obs.FilterMessage(msg).All()), want, timeout)
}
