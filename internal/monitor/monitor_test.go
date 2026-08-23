package monitor

import (
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestMonitorLifecycleWithPrometheus(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	core, observed := observer.New(zapcore.InfoLevel)
	m, err := New(zap.New(core), &Config{
		PromEnabled: true,
		PromPort:    port,
		PromPath:    "/metrics",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if err := m.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	endpoint := m.PromEndpoint()
	if endpoint == "" {
		t.Fatal("Expected Prometheus endpoint to be listening")
	}
	found := false
	for _, e := range observed.All() {
		if strings.Contains(e.Message, "Prometheus metrics endpoint started") {
			found = true
		}
	}
	if !found {
		t.Error("Expected a startup log for the Prometheus endpoint")
	}

	url := "http://" + endpoint + "/metrics"
	// #nosec G107 -- url is our own loopback metrics listener
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "sssonector_uptime_seconds") {
		t.Error("Metrics output missing sssonector_uptime_seconds")
	}

	m.UpdateMetrics(42, 7, 0, 0, 1, 2)
	// #nosec G107 -- url is our own loopback metrics listener
	resp, err = http.Get(url)
	if err != nil {
		t.Fatalf("Second GET /metrics: %v", err)
	}
	body, _ = io.ReadAll(resp.Body)
	resp.Body.Close()
	if !strings.Contains(string(body), "sssonector_bytes_in_total 42") ||
		!strings.Contains(string(body), "sssonector_bytes_out_total 7") ||
		!strings.Contains(string(body), "sssonector_errors_total 1") ||
		!strings.Contains(string(body), "sssonector_connections_active 2") {
		t.Errorf("Updated metrics not reflected in exposition:\n%s", body)
	}

	m.Stop()

	if m.PromEndpoint() != "" {
		t.Error("Expected Prometheus endpoint cleared after Stop")
	}
	// #nosec G107 -- url is our own loopback metrics listener
	if _, err := http.Get(url); err == nil {
		t.Error("Expected connection failure after Stop")
	}
}

func TestMonitorStartStopWithoutEndpoints(t *testing.T) {
	core, _ := observer.New(zapcore.InfoLevel)
	m, err := New(zap.New(core), &Config{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	done := make(chan struct{})
	go func() {
		m.Start()
		m.Stop()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Start/Stop without endpoints did not complete in time")
	}

	if m.PromEndpoint() != "" {
		t.Error("Expected no Prometheus endpoint when disabled")
	}
}

func TestNewRequiresLoggerAndConfig(t *testing.T) {
	if _, err := New(nil, &Config{}); err == nil {
		t.Error("Expected error for nil logger")
	}
	logger := zap.NewNop()
	if _, err := New(logger, nil); err == nil {
		t.Error("Expected error for nil config")
	}
}

func TestThrottleMetricsExposed(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()

	m, err := New(zap.NewNop(), &Config{
		PromEnabled: true,
		PromPort:    port,
		PromPath:    "/metrics",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	m.Start()
	defer m.Stop()

	m.UpdateThrottleMetrics(3, 9, 2262.8, 226.28)

	resp, err := http.Get("http://127.0.0.1:" + strconv.Itoa(port) + "/metrics") // #nosec G107 -- own loopback listener
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	text := string(body)

	for _, want := range []string{
		`sssonector_throttle_hits_total{direction="in"} 3`,
		`sssonector_throttle_hits_total{direction="out"} 9`,
		"sssonector_throttle_effective_rate_bytes_per_second 2262.8",
		"sssonector_throttle_burst_bytes 226.28",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("metrics output missing %q", want)
		}
	}

	snap := m.GetMetrics()
	if snap.ThrottleHitsIn != 3 || snap.ThrottleHitsOut != 9 {
		t.Errorf("GetMetrics hits = %d/%d, want 3/9", snap.ThrottleHitsIn, snap.ThrottleHitsOut)
	}
}
