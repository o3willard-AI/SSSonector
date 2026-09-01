package nat

import (
	"encoding/binary"
	"net"
	"testing"
)

// buildSynPacket constructs a minimal valid IPv4 TCP SYN packet for tests.
func buildSynPacket(srcIP, dstIP net.IP, srcPort, dstPort int) []byte {
	pkt := make([]byte, 40) // 20 IP + 20 TCP
	pkt[0] = 0x45           // IPv4, IHL=5
	pkt[9] = ipv4ProtoTCP
	copy(pkt[12:16], srcIP.To4())
	copy(pkt[16:20], dstIP.To4())
	binary.BigEndian.PutUint16(pkt[2:4], uint16(len(pkt)))
	tcpOff := 20
	binary.BigEndian.PutUint16(pkt[tcpOff:tcpOff+2], uint16(srcPort))
	binary.BigEndian.PutUint16(pkt[tcpOff+2:tcpOff+4], uint16(dstPort))
	pkt[tcpOff+13] = 0x02 // SYN
	recomputeTCPChecksum(pkt, tcpOff)
	recomputeIPv4Checksum(pkt)
	return pkt
}

// checksumMustVerify asserts a packet's TCP checksum is valid as computed.
func tcpChecksumValid(t *testing.T, pkt []byte) {
	t.Helper()
	tcpOff := int(pkt[0]&0x0F) * 4
	ckAt := tcpOff + 16
	stored := binary.BigEndian.Uint16(pkt[ckAt : ckAt+2])
	computed := tcpChecksum(
		[4]byte{pkt[12], pkt[13], pkt[14], pkt[15]},
		[4]byte{pkt[16], pkt[17], pkt[18], pkt[19]},
		pkt[tcpOff:],
		16,
	)
	if stored != computed {
		t.Fatalf("TCP checksum mismatch: stored %04x computed %04x", stored, computed)
	}
}

func TestChecksumUpdateKnownVector(t *testing.T) {
	// RFC 1624 eqn 3: HC' = ~(~HC + ~M + M')
	// HC=0xFFFF, M=0x0000, M'=0xFFF0:
	// ~HC=0x0000, ~M=0xFFFF, +0xFFF0 = 0x1FFEF → fold 0xFFF0 → ~ = 0x000F
	got := checksumUpdate(0xFFFF, 0x0000, 0xFFF0)
	if got != 0x000F {
		t.Fatalf("RFC1624 vector: expected 0x000F, got %04x", got)
	}
	// Identity: rewriting to the same value must not change the checksum.
	if got := checksumUpdate(0xA1B2, 0x1234, 0x1234); got != 0xA1B2 {
		t.Fatalf("identity rewrite: expected 0xA1B2, got %04x", got)
	}
}

func TestRewriteThenVerifyChecksums(t *testing.T) {
	src := net.ParseIP("10.77.0.2")
	dst := net.ParseIP("192.168.10.5")
	pkt := buildSynPacket(src, dst, 51234, 80)
	tcpChecksumValid(t, pkt)

	// Rewrite through the engine path and verify checksums still hold.
	cfg := testNATConfig()
	eng, err := NewEngine(
		&cfg,
		Options{EgressIP: net.ParseIP("192.168.101.172")},
		testLogger(),
	)
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	out, err := eng.ProcessTunnelPacket(pkt)
	if err != nil {
		t.Fatalf("forward: %v", err)
	}
	if got := net.IP(out[12:16]); !got.Equal(net.ParseIP("192.168.101.172")) {
		t.Fatalf("src not SNATed: %v", got)
	}
	tcpChecksumValid(t, out)
}

func TestParseRejectsGarbage(t *testing.T) {
	tests := []struct {
		name string
		pkt  []byte
		want error
	}{
		{"empty", nil, ErrPacketTooShort},
		{"tiny", []byte{0x45, 0, 0}, ErrPacketTooShort},
		{
			"ipv6-version-nibble",
			func() []byte {
				p := make([]byte, 40)
				p[0] = 0x65
				return p
			}(),
			ErrNotIPv4,
		},
		{
			"udp-not-tcp",
			func() []byte {
				p := make([]byte, 40)
				p[0] = 0x45
				p[9] = 17 // UDP
				return p
			}(),
			ErrNotTCP,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, _, _, err := parseIPv4TCP(tc.pkt)
			if err == nil {
				t.Fatalf("expected error %v, got nil", tc.want)
			}
		})
	}
}
