package main

import (
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

type fakeReloadable struct {
	mu      sync.Mutex
	applied []*config.AppConfig
}

func newFakeReloadable() *fakeReloadable {
	return &fakeReloadable{}
}

func (f *fakeReloadable) ApplyConfig(cfg *config.AppConfig) error {
	f.mu.Lock()
	f.applied = append(f.applied, cfg)
	f.mu.Unlock()
	return nil
}

func (f *fakeReloadable) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.applied)
}

func writeValidClientConfig(t *testing.T, dir, level string, rate float64) string {
	t.Helper()
	path := filepath.Join(dir, "config.yaml")
	content := `metadata:
  schema_version: "2.0.0"
  environment: qa
type: client
config:
  mode: client
  logging:
    level: ` + level + `
    format: json
  network:
    name: tun9
    interface: tun9
    mtu: 1500
    address: "10.77.0.2/24"
  tunnel:
    server_address: "127.0.0.1"
    server_port: 18443
  security:
    tls:
      min_version: "1.2"
      max_version: "1.3"
throttle:
  enabled: true
  rate: ` + strconv.FormatFloat(rate, 'f', -1, 64) + `
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func TestApplyReloadUpdatesLevelAndConfig(t *testing.T) {
	dir := t.TempDir()
	path := writeValidClientConfig(t, dir, "debug", 123456)

	core, observed := observer.New(zapcore.DebugLevel)
	logger := zap.New(core)
	atom := zap.NewAtomicLevelAt(zapcore.InfoLevel)
	fake := newFakeReloadable()

	got := applyReload(logger, path, "", false, zapcore.InfoLevel, &atom, fake)

	if got != zapcore.DebugLevel {
		t.Errorf("returned level = %v, want debug", got)
	}
	if atom.Level() != zapcore.DebugLevel {
		t.Errorf("atomic level = %v, want debug", atom.Level())
	}
	if fake.count() != 1 {
		t.Fatalf("ApplyConfig called %d times, want 1", fake.count())
	}
	if gotCfg := fake.applied[0]; gotCfg.Throttle.Rate != 123456 {
		t.Errorf("applied throttle rate = %v, want 123456", gotCfg.Throttle.Rate)
	}

	foundReloaded := false
	for _, e := range observed.All() {
		if e.Message == "Configuration reloaded" {
			foundReloaded = true
		}
	}
	if !foundReloaded {
		t.Error("Expected 'Configuration reloaded' confirmation log")
	}
}

func TestApplyReloadFlagPrecedence(t *testing.T) {
	dir := t.TempDir()
	path := writeValidClientConfig(t, dir, "debug", 1024)

	atom := zap.NewAtomicLevelAt(zapcore.InfoLevel)
	fake := newFakeReloadable()

	got := applyReload(zap.NewNop(), path, "error", true, zapcore.InfoLevel, &atom, fake)

	if got != zapcore.ErrorLevel || atom.Level() != zapcore.ErrorLevel {
		t.Errorf("explicit -log-level flag must win over config level; got %v / %v", got, atom.Level())
	}
	if fake.count() != 1 {
		t.Errorf("reloadable should still be applied, got %d calls", fake.count())
	}
}

func TestApplyReloadRejectsInvalidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("metadata:\n  schema_version: \"2.0.0\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	core, observed := observer.New(zapcore.DebugLevel)
	atom := zap.NewAtomicLevelAt(zapcore.InfoLevel)
	fake := newFakeReloadable()

	got := applyReload(zap.New(core), path, "", false, zapcore.InfoLevel, &atom, fake)

	if got != zapcore.InfoLevel || atom.Level() != zapcore.InfoLevel {
		t.Error("rejected reload must not change level")
	}
	if fake.count() != 0 {
		t.Error("rejected reload must not reach the reloadable")
	}
	found := false
	for _, e := range observed.All() {
		if e.Level == zapcore.ErrorLevel && e.Message == "SIGHUP reload rejected: config validation failed" {
			found = true
		}
	}
	if !found {
		t.Error("Expected rejection error log")
	}
}

func TestSignalDrivenReloadEndToEnd(t *testing.T) {
	dir := t.TempDir()
	path := writeValidClientConfig(t, dir, "warn", 4096)

	atom := zap.NewAtomicLevelAt(zapcore.InfoLevel)
	fake := newFakeReloadable()

	hupChan := make(chan os.Signal, 1)
	signal.Notify(hupChan, syscall.SIGHUP)
	defer signal.Stop(hupChan)

	done := make(chan struct{})
	go func() {
		defer close(done)
		prev := zapcore.InfoLevel
		for range hupChan {
			prev = applyReload(zap.NewNop(), path, "", false, prev, &atom, fake)
			return
		}
	}()

	if err := syscall.Kill(os.Getpid(), syscall.SIGHUP); err != nil {
		t.Fatalf("self-signal failed: %v", err)
	}

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("signal-driven reload did not complete")
	}

	if atom.Level() != zapcore.WarnLevel {
		t.Errorf("level after real SIGHUP = %v, want warn", atom.Level())
	}
	if fake.count() != 1 {
		t.Errorf("reloadable applied %d times after real SIGHUP, want 1", fake.count())
	}
}
