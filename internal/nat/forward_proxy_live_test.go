package nat

import (
	"net"
	"sync/atomic"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"go.uber.org/zap"
)

// TestForwardProxyLiveChannel isolates the QA finding: a real SYN fed
// through DeliverTunnelPacket must reach the TCP forwarder and produce
// a session. Uses the real channel endpoint + netstack (not mocks).
func TestForwardProxyLiveChannel(t *testing.T) {
	logger := zap.NewNop()
	proxy, err := NewForwardProxy(net.ParseIP("10.77.0.1"), "10.77.0.0/24",
		[]config.NATForwardRule{
			{
				SrcCIDR: "10.77.0.0/24",
				DstCIDR: "192.0.2.0/24",
				Ports:   []int{8080},
			},
		}, logger)
	if err != nil {
		t.Fatalf("proxy: %v", err)
	}
	defer proxy.Stop()

	// Count frames the proxy emits (SYN-ACK should appear here).
	var emitted atomic.Uint64
	proxy.SetFrameSink(countWriter{&emitted})
	proxy.StartPump(logger)

	if err := proxy.SetTransportHandler(proxyRulesForTest(), logger); err != nil {
		t.Fatalf("handler: %v", err)
	}

	// Build a real SYN: 10.77.0.2:12345 -> 192.0.2.10:8080 (an egress
	// host, NOT the tunnel IP: frames addressed to the tunnel IP belong
	// to reverse-PAT flows and are skipped by the forward proxy).
	syn := buildSynPacket(net.ParseIP("10.77.0.2"), net.ParseIP("192.0.2.10"), 12345, 8080)

	proxy.DeliverTunnelPacket(syn)

	// Wait briefly for netstack to process and emit SYN-ACK.
	for i := 0; i < 100 && emitted.Load() == 0; i++ {
		time.Sleep(10 * time.Millisecond)
	}
	if emitted.Load() == 0 {
		st := proxy.s.Stats()
		t.Logf("netstack IP stats: received=%d malformed=%d invalidDst=%d invalidSrc=%d disabledRx=%d",
			st.IP.PacketsReceived.Value(),
			st.IP.MalformedPacketsReceived.Value(),
			st.IP.InvalidDestinationAddressesReceived.Value(),
			st.IP.InvalidSourceAddressesReceived.Value(),
			st.IP.DisabledPacketsReceived.Value())
		t.Logf("netstack TCP stats: validSeg=%d invalidSeg=%d segmentsSent=%d resetsSent=%d failedConn=%d synDrop=%d",
			st.TCP.ValidSegmentsReceived.Value(),
			st.TCP.InvalidSegmentsReceived.Value(),
			st.TCP.SegmentsSent.Value(),
			st.TCP.ResetsSent.Value(),
			st.TCP.FailedConnectionAttempts.Value(),
			st.TCP.ListenOverflowSynDrop.Value())
		s, den, rerr, dr := proxy.Stats()
		t.Logf("proxy: sessions=%d denies=%d relayErr=%d dropped=%d badCK=%d noNetHdr=%d noPullUp=%d badTCPLen=%d noTransHdr=%d",
			s, den, rerr, dr,
			proxy.dropBadChecksum.Load(), proxy.dropNoNetHdr.Load(),
			proxy.dropNoPullUp.Load(), proxy.dropBadTCPHdrLen.Load(),
			proxy.dropNoTransportHdr.Load())
		t.Logf("badTCPHdrLen diagnostics: data12=%d len=%d hex=%s",
			proxy.lastBadData12, proxy.lastBadLen, proxy.lastBadHex)
		t.Fatal("netstack emitted no SYN-ACK: SYN was dropped inside the stack")
	}
	t.Logf("netstack emitted %d frames", emitted.Load())
}

type countWriter struct{ n *atomic.Uint64 }

func (w countWriter) WritePacket(p []byte) error { w.n.Add(1); return nil }

func proxyRulesForTest() []config.NATForwardRule {
	return []config.NATForwardRule{
		{
			SrcCIDR: "10.77.0.0/24",
			DstCIDR: "192.0.2.0/24",
			Ports:   []int{8080},
		},
	}
}
