package collect

import (
	"fmt"
	"net"
	"strings"

	cfg "github.com/o3willard-AI/SSSonector/internal/config"
)

// ValidateDraftFunc adapts ValidateDraft for the wizard (WI 5.2 seam).
type ValidateDraftFunc func(draft string) (*cfg.AppConfig, error)

// AppConfigAlias re-exports the loader's AppConfig type for wizard tests
// (the TUI may import internal/config for pure loading — AGENTS.md).
type AppConfigAlias = cfg.AppConfig

// PortProbeFunc reports whether a TCP port is free to listen on. The
// production implementation shells `ss -tlnp` via the injectable
// CommandRunner; tests inject canned output (no real ss).
type PortProbeFunc func(port int) bool

// NewSSPortProbe builds the production PortProbeFunc over the given
// runner: parses `ss -tlnp` output for any LISTEN line bound to the port.
func NewSSPortProbe(runner CommandRunner) PortProbeFunc {
	return func(port int) bool {
		out, err := runner.Run("ss", "-tlnp")
		if err != nil {
			return false // fail-closed: cannot prove the port is free
		}
		return !portListedInSS(out, port)
	}
}

// portListedInSS reports whether ss output contains a LISTEN socket on
// the port (Local Address:Port column, e.g. "0.0.0.0:9443" or ":9443").
func portListedInSS(out string, port int) bool {
	suffix := fmt.Sprintf(":%d", port)
	for _, line := range strings.Split(out, "\n") {
		if !strings.Contains(line, "LISTEN") {
			continue
		}
		for _, f := range strings.Fields(line) {
			if strings.HasSuffix(f, suffix) {
				return true
			}
		}
	}
	return false
}

// ValidateCIDR checks the TUN address text is a valid CIDR with both a
// network address and a prefix (e.g. 10.77.0.1/24 — the host address plus
// subnet). Returns a user-facing error, nil when valid.
func ValidateCIDR(s string) error {
	ip, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		return fmt.Errorf("%q is not a valid CIDR (e.g. 10.77.0.1/24)", s)
	}
	if ip.To4() == nil {
		return fmt.Errorf("%q: only IPv4 TUN addresses are supported", s)
	}
	// The address must be the HOST address inside the subnet (the TUN
	// interface's own IP), not the bare network address.
	if ip.Equal(ipnet.IP) {
		return fmt.Errorf("%q: use the host address inside the subnet, not the network address", s)
	}
	return nil
}

// OverlapWith returns the existing subnet that overlaps the candidate
// CIDR ("" when none). Existing entries are instance TUN subnets (e.g.
// "10.77.0.1/24" — host address or network address both accepted).
func OverlapWith(candidate string, existing []string) string {
	_, cnet, err := net.ParseCIDR(candidate)
	if err != nil {
		return "" // ValidateCIDR already reported the malformed input
	}
	for _, e := range existing {
		_, enet, err := net.ParseCIDR(e)
		if err != nil {
			continue
		}
		if cnet.Contains(enet.IP) || enet.Contains(cnet.IP) {
			return e
		}
	}
	return ""
}
