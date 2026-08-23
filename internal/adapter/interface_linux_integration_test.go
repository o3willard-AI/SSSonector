//go:build linux

package adapter

import (
	"os"
	"testing"
)

// TestLinuxInterfaceRealTUN exercises the native TUN lifecycle against the
// actual kernel. It is skipped unless SSSONECTOR_TUN_TEST=1 AND the process
// can access /dev/net/tun (root or CAP_NET_ADMIN), so unprivileged CI runs
// skip it by design.
func TestLinuxInterfaceRealTUN(t *testing.T) {
	if os.Getenv("SSSONECTOR_TUN_TEST") != "1" {
		t.Skip("set SSSONECTOR_TUN_TEST=1 (as root/CAP_NET_ADMIN) to run")
	}
	if _, err := os.Stat("/dev/net/tun"); err != nil {
		t.Skipf("/dev/net/tun unavailable: %v", err)
	}

	iface, err := New("sssnector-itest", DefaultOptions())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	cfg := &Config{Name: "sssnector-itest", Address: "10.231.0.1/30", MTU: 1400}
	if err := iface.Configure(cfg); err != nil {
		t.Fatalf("Configure: %v", err)
	}
	if got := iface.GetName(); got != "sssnector-itest" {
		t.Errorf("name = %q", got)
	}
	if !iface.IsUp() {
		t.Error("interface should be up after Configure")
	}
	if err := iface.Cleanup(); err != nil {
		t.Errorf("Cleanup: %v", err)
	}
}
