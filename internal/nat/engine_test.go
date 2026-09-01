package nat

import (
	"encoding/binary"
	"net"
	"testing"

	"github.com/o3willard-AI/SSSonector/internal/config"
)

func mkEngine(t *testing.T, cfg config.NATConfig, egress string) *Engine {
	t.Helper()
	eng, err := NewEngine(&cfg, Options{EgressIP: net.ParseIP(egress)}, testLogger())
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	if eng == nil {
		t.Fatal("engine nil for enabled config")
	}
	return eng
}

func TestEngineForwardRoundTrip(t *testing.T) {
	eng := mkEngine(t, testNATConfig(), "192.168.101.172")

	clientIP := net.ParseIP("10.77.0.2")
	serverIP := net.ParseIP("192.168.10.5")
	egressIP := net.ParseIP("192.168.101.172")

	// 1. Client SYN → forward through engine.
	syn := buildSynPacket(clientIP, serverIP, 51234, 80)
	out, err := eng.ProcessTunnelPacket(syn)
	if err != nil {
		t.Fatalf("forward SYN: %v", err)
	}
	if got := net.IP(out[12:16]); !got.Equal(egressIP) {
		t.Fatalf("SNAT src: want %v got %v", egressIP, got)
	}
	if got := net.IP(out[16:20]); !got.Equal(serverIP) {
		t.Fatalf("dst must be untouched: got %v", got)
	}
	tcpOff := 20
	if got := int(binary.BigEndian.Uint16(out[tcpOff:])); got == 51234 {
		t.Fatal("src port must be translated to a pool port")
	}
	snatPort := int(binary.BigEndian.Uint16(out[tcpOff:]))
	tcpChecksumValid(t, out)

	// 2. Server SYN-ACK back to egress:port → reverse translate.
	synack := buildSynPacket(serverIP, egressIP, 80, snatPort)
	// Mark SYN-ACK.
	synack[tcpOff+13] = 0x12
	recomputeTCPChecksum(synack, tcpOff)
	recomputeIPv4Checksum(synack)
	back, err := eng.ProcessEgressPacket(synack)
	if err != nil {
		t.Fatalf("reverse SYN-ACK: %v", err)
	}
	if got := net.IP(back[16:20]); !got.Equal(clientIP) {
		t.Fatalf("reverse dst: want client %v got %v", clientIP, got)
	}
	if got := int(binary.BigEndian.Uint16(back[tcpOff+2:])); got != 51234 {
		t.Fatalf("reverse dst port: want 51234 got %d", got)
	}
	if got := net.IP(back[12:16]); !got.Equal(serverIP) {
		t.Fatalf("reverse src must be untouched: got %v", got)
	}
	tcpChecksumValid(t, back)

	st := eng.Stats()
	if st.ForwardedPackets != 1 || st.ReturnPackets != 1 || st.ActiveFlows != 1 {
		t.Fatalf("stats: %+v", st)
	}
}

func TestEngineACLDeniedPacketsDropped(t *testing.T) {
	cfg := testNATConfig()
	// Rule only allows 10.77.0.0/24 → 192.168.10.0/24 ports 80,443.
	eng := mkEngine(t, cfg, "192.168.101.172")

	// Denied: wrong destination port.
	bad := buildSynPacket(net.ParseIP("10.77.0.2"), net.ParseIP("192.168.10.5"), 40000, 22)
	if _, err := eng.ProcessTunnelPacket(bad); err == nil {
		t.Fatal("port-22 flow must be denied")
	}

	// Denied: source outside the rule subnet.
	bad2 := buildSynPacket(net.ParseIP("10.78.0.9"), net.ParseIP("192.168.10.5"), 40001, 80)
	if _, err := eng.ProcessTunnelPacket(bad2); err == nil {
		t.Fatal("out-of-subnet flow must be denied")
	}

	st := eng.Stats()
	if st.DroppedPackets != 2 || st.ACLDenied != 2 || st.ActiveFlows != 0 {
		t.Fatalf("stats after denies: %+v", st)
	}

	// Egress packet with no conntrack entry must never leak into tunnel.
	stray := buildSynPacket(net.ParseIP("192.168.10.5"), net.ParseIP("192.168.101.172"), 80, 40000)
	if _, err := eng.ProcessEgressPacket(stray); err == nil {
		t.Fatal("stray egress packet must be dropped")
	}
}

func TestEngineDisabledNeverConstructs(t *testing.T) {
	eng, err := NewEngine(nil, Options{}, testLogger())
	if err != nil || eng != nil {
		t.Fatalf("nil config: want nil engine, got %v, %v", eng, err)
	}
	off := testNATConfig()
	off.Enabled = false
	eng, err = NewEngine(&off, Options{}, testLogger())
	if err != nil || eng != nil {
		t.Fatalf("disabled config: want nil engine, got %v, %v", eng, err)
	}
}

func TestEngineForwardRequiresEgressIP(t *testing.T) {
	cfg := testNATConfig()
	if _, err := NewEngine(&cfg, Options{}, testLogger()); err == nil {
		t.Fatal("forward NAT without egress IP must fail closed")
	}
}

func TestEngineRejectsMalformed(t *testing.T) {
	eng := mkEngine(t, testNATConfig(), "192.168.101.172")
	if _, err := eng.ProcessTunnelPacket([]byte{0x45, 0x00}); err == nil {
		t.Fatal("truncated packet must be rejected")
	}
	udp := make([]byte, 40)
	udp[0] = 0x45
	udp[9] = 17
	if _, err := eng.ProcessTunnelPacket(udp); err == nil {
		t.Fatal("non-TCP packet must be rejected")
	}
}

func TestEngineReloadRules(t *testing.T) {
	eng := mkEngine(t, testNATConfig(), "192.168.101.172")

	// Establish a flow under the original rule.
	clientIP := net.ParseIP("10.77.0.2")
	dst := net.ParseIP("192.168.10.5")
	syn := buildSynPacket(clientIP, dst, 50000, 80)
	if _, err := eng.ProcessTunnelPacket(syn); err != nil {
		t.Fatalf("initial flow: %v", err)
	}

	// Reload with tighter rules: only port 443.
	newCfg := testNATConfig()
	newCfg.Forward.Rules = []config.NATForwardRule{
		{SrcCIDR: "10.77.0.0/24", DstCIDR: "192.168.10.0/24", Ports: []int{443}},
	}
	if err := eng.ReloadRules(&newCfg); err != nil {
		t.Fatalf("reload: %v", err)
	}

	// New flow on port 80 now denied.
	syn2 := buildSynPacket(clientIP, dst, 50001, 80)
	if _, err := eng.ProcessTunnelPacket(syn2); err == nil {
		t.Fatal("port 80 must be denied after reload")
	}
	// New flow on port 443 allowed.
	syn3 := buildSynPacket(clientIP, dst, 50002, 443)
	if _, err := eng.ProcessTunnelPacket(syn3); err != nil {
		t.Fatalf("port 443 must be allowed after reload: %v", err)
	}
	// Established flow's conntrack entry survives (active flows unchanged
	// in count semantics: old flow + new flow).
	st := eng.Stats()
	if st.ActiveFlows != 2 {
		t.Fatalf("established flows must survive reload: %+v", st)
	}
}

func TestEngineReloadRejectsStructural(t *testing.T) {
	eng := mkEngine(t, testNATConfig(), "192.168.101.172")
	off := testNATConfig()
	off.Enabled = false
	if err := eng.ReloadRules(&off); err == nil {
		t.Fatal("reload disabling NAT must be rejected (structural)")
	}
	if err := eng.ReloadRules(nil); err == nil {
		t.Fatal("reload with nil must be rejected")
	}
}
