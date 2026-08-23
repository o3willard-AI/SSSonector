package tunnel

import (
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/cert"
	"github.com/o3willard-AI/SSSonector/internal/cert/generator"
	"github.com/o3willard-AI/SSSonector/internal/config"
	"github.com/o3willard-AI/SSSonector/internal/throttle"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func throttleCfg(enabled bool, rate float64, burst int) *config.AppConfig {
	cfg := config.DefaultConfig()
	cfg.Throttle = config.ThrottleConfig{Enabled: enabled, Rate: rate, Burst: burst}
	return cfg
}

func TestTransferUpdateConfigReachesLimiters(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	transfer, err := NewTransfer(c1, c2, throttleCfg(false, 0, 0), zap.NewNop())
	if err != nil {
		t.Fatalf("NewTransfer: %v", err)
	}

	in, out := transfer.limiters()
	if im, _ := in.GetMetrics(); im.Rate != 0 {
		t.Errorf("expected initial rate 0, got %f", im.Rate)
	}

	transfer.UpdateConfig(throttleCfg(true, 1000, 500))

	wantRate := float64(1000) * throttle.TCPOverheadFactor
	wantBurst := wantRate * 0.1 // burst is always 100ms of effective rate
	im, _ := in.GetMetrics()
	om, _ := out.GetMetrics()
	if im.Rate != wantRate {
		t.Errorf("in limiter rate = %f, want %f", im.Rate, wantRate)
	}
	if om.Burst != wantBurst {
		t.Errorf("out limiter burst = %f, want %f", om.Burst, wantBurst)
	}
	if !in.IsEnabled() || !out.IsEnabled() {
		t.Error("limiters should be enabled after UpdateConfig with enabled=true")
	}

	transfer.UpdateConfig(throttleCfg(false, 1000, 500))
	if in.IsEnabled() || out.IsEnabled() {
		t.Error("limiters should be disabled after UpdateConfig with enabled=false")
	}
}

func TestServerApplyConfigSwapsAndWarns(t *testing.T) {
	core, observed := observer.New(zapcore.DebugLevel)

	old := config.DefaultConfig()
	newCfg := config.DefaultConfig()
	newCfg.Config.Facade.Enabled = true
	newCfg.Throttle = config.ThrottleConfig{Enabled: true, Rate: 2048, Burst: 512}

	s := &Server{
		logger: zap.New(core),
		config: old,
	}

	if err := s.ApplyConfig(newCfg); err != nil {
		t.Fatalf("ApplyConfig: %v", err)
	}

	if s.activeConfig() != newCfg {
		t.Error("ApplyConfig should swap the active config pointer")
	}

	foundRestartWarning := false
	for _, e := range observed.All() {
		if e.Message == "Configuration change requires restart" {
			for _, f := range e.Context {
				if f.Key == "field" && f.String == "facade.enabled" {
					foundRestartWarning = true
				}
			}
		}
	}
	if !foundRestartWarning {
		t.Error("Expected a restart-required warning for facade.enabled")
	}

	if err := s.ApplyConfig(nil); err == nil {
		t.Error("ApplyConfig(nil) should fail")
	}
}

func TestClientApplyConfigUpdatesCertTunables(t *testing.T) {
	dir := t.TempDir()

	if err := generator.GenerateTemporaryCertificates(dir); err != nil {
		t.Fatalf("generate certificates: %v", err)
	}

	certMgr, err := cert.NewManager(
		filepath.Join(dir, "server.crt"),
		filepath.Join(dir, "server.key"),
		filepath.Join(dir, "ca.crt"),
		true,
		false,
		zap.NewNop(),
	)
	if err != nil {
		t.Fatalf("cert manager setup: %v", err)
	}
	defer certMgr.Stop()

	tm := &TLSManager{
		certManager: certMgr,
		config:      &TLSConfig{},
		logger:      zap.NewNop(),
	}

	c := &Client{
		logger:     zap.NewNop(),
		config:     config.DefaultConfig(),
		tlsManager: tm,
	}

	newInterval := 250 * time.Millisecond
	newCfg := config.DefaultConfig()
	newCfg.Config.Auth.CertRotation.Interval = newInterval

	if err := c.ApplyConfig(newCfg); err != nil {
		t.Fatalf("ApplyConfig: %v", err)
	}

	if got := certMgr.CheckInterval(); got != newInterval {
		t.Errorf("check interval = %v, want %v", got, newInterval)
	}
}
