package nat

import (
	"io"
	"net"
	"testing"
	"time"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/stack"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"go.uber.org/zap"
)

// wirePair is an in-memory packet conduit between the netstack and a
// "peer kernel" TCP implementation. Building a peer TCP stack in-test
// is exactly what netstack gives us for free: the second half of the
// pair is another netstack with a real TCP listener. This exercises
// the genuine wire path end to end: real SYN, handshake, data, FIN.
type wirePair struct {
	aToB chan []byte
	bToA chan []byte
}

func (w *wirePair) writeA(p []byte) error {
	b := make([]byte, len(p))
	copy(b, p)
	w.aToB <- b
	return nil
}

func (w *wirePair) writeB(p []byte) error {
	b := make([]byte, len(p))
	copy(b, p)
	w.bToA <- b
	return nil
}

// writerAdapter adapts a func into PacketWriter.
type writerAdapter struct {
	fn func([]byte) error
}

func (w *writerAdapter) WritePacket(p []byte) error { return w.fn(p) }

// readerPump forwards frames from a channel into a netstack.
func pumpInto(rev *ReverseNAT, ch <-chan []byte) {
	for pkt := range ch {
		rev.DeliverTunnelPacket(pkt)
	}
}

// TestReverseNATEndToEnd dials a public listener, crosses an in-memory
// wire via two netstacks, and exchanges data with a real TCP service on
// the "client" side. This validates: listener ACL, netstack dial
// through the tunnel link, real TCP handshake over raw frames, bidir
// copy with half-close, and clean teardown.
func TestReverseNATEndToEnd(t *testing.T) {
	logger := zap.NewNop()
	wire := &wirePair{
		aToB: make(chan []byte, 512),
		bToA: make(chan []byte, 512),
	}

	// "Server" side netstack: owns 10.77.0.1, dials published services.
	srvRev, err := NewReverseNAT(&writerAdapter{fn: wire.writeA},
		net.ParseIP("10.77.0.1"), "10.77.0.0/24", logger)
	if err != nil {
		t.Fatalf("server netstack: %v", err)
	}
	defer srvRev.Stop()

	// "Client" side netstack: owns 10.77.0.2, serves a real echo
	// listener via netstack's gonet TCP listener (acts as the peer
	// service behind the TUN).
	cliStack := newPeerStack(t, net.ParseIP("10.77.0.2"), "10.77.0.0/24",
		&writerAdapter{fn: wire.writeB}, logger)
	defer cliStack.s.Close()

	// Wire the halves together.
	go pumpInto(srvRev, wire.bToA)
	go pumpInto(cliStack.rev, wire.aToB)

	// Real service behind the peer: echo server on 10.77.0.2:8080.
	echoLn, err := cliStack.listen(net.ParseIP("10.77.0.2"), 8080)
	if err != nil {
		t.Fatalf("peer listen: %v", err)
	}
	go serveEcho(echoLn)

	// Public listener on this host: :18080 → 10.77.0.2:8080, ACL allows
	// loopback sources only.
	rule := config.NATListenerRule{
		Comment:      "test echo",
		ListenPort:   18080,
		Dst:          "10.77.0.2:8080",
		AllowedCIDRs: []string{"127.0.0.0/8"},
	}
	if err := srvRev.StartListener(rule); err != nil {
		t.Fatalf("StartListener: %v", err)
	}

	// Public client connects.
	pub, err := net.DialTimeout("tcp4", "127.0.0.1:18080", 10*time.Second)
	if err != nil {
		t.Fatalf("public dial: %v", err)
	}
	pub.SetDeadline(time.Now().Add(10 * time.Second))

	msg := []byte("hello through the tunnel")
	if _, err := pub.Write(msg); err != nil {
		t.Fatalf("write: %v", err)
	}
	buf := make([]byte, len(msg))
	if _, err := io.ReadFull(pub, buf); err != nil {
		t.Fatalf("read echo: %v", err)
	}
	if string(buf) != string(msg) {
		t.Fatalf("echo mismatch: got %q want %q", buf, msg)
	}

	// Half-close: client shuts write; peer EOF drains; server FIN
	// arrives back; read returns EOF.
	if err := pub.(*net.TCPConn).CloseWrite(); err != nil {
		t.Fatalf("CloseWrite: %v", err)
	}
	trailer := make([]byte, 16)
	if n, err := pub.Read(trailer); err != io.EOF {
		t.Fatalf("expected EOF after half-close, got n=%d err=%v", n, err)
	}
	pub.Close()

	accepts, denies, relayErrs := srvRev.Stats()
	if accepts != 1 || denies != 0 || relayErrs != 0 {
		t.Fatalf("stats: accepts=%d denies=%d relayErrors=%d", accepts, denies, relayErrs)
	}
}

// TestReverseNATListenerACLDeny verifies a denied source never reaches
// the tunnel: the connection is closed at accept.
func TestReverseNATListenerACLDeny(t *testing.T) {
	logger := zap.NewNop()
	wire := &wirePair{
		aToB: make(chan []byte, 64),
		bToA: make(chan []byte, 64),
	}
	srvRev, err := NewReverseNAT(&writerAdapter{fn: wire.writeA},
		net.ParseIP("10.77.0.1"), "10.77.0.0/24", logger)
	if err != nil {
		t.Fatalf("netstack: %v", err)
	}
	defer srvRev.Stop()

	rule := config.NATListenerRule{
		ListenPort:   18081,
		Dst:          "10.77.0.2:9999",
		AllowedCIDRs: []string{"203.0.113.0/24"}, // loopback NOT allowed
	}
	if err := srvRev.StartListener(rule); err != nil {
		t.Fatalf("StartListener: %v", err)
	}

	conn, err := net.DialTimeout("tcp4", "127.0.0.1:18081", 5*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	// Server should close on us quickly (denied).
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	buf := make([]byte, 16)
	if _, err := conn.Read(buf); err == nil {
		t.Fatal("denied connection should be closed, not served")
	}
	conn.Close()

	_, denies, _ := srvRev.Stats()
	if denies != 1 {
		t.Fatalf("expected 1 ACL deny, got %d", denies)
	}
}

// TestReverseNATListenerReload verifies hot add/remove converges.
func TestReverseNATListenerReload(t *testing.T) {
	logger := zap.NewNop()
	wire := &wirePair{aToB: make(chan []byte, 8), bToA: make(chan []byte, 8)}
	srvRev, err := NewReverseNAT(&writerAdapter{fn: wire.writeA},
		net.ParseIP("10.77.0.1"), "10.77.0.0/24", logger)
	if err != nil {
		t.Fatalf("netstack: %v", err)
	}
	defer srvRev.Stop()

	ruleA := config.NATListenerRule{ListenPort: 18082, Dst: "10.77.0.2:80", AllowedCIDRs: []string{"0.0.0.0/0"}}
	if err := srvRev.StartListener(ruleA); err != nil {
		t.Fatalf("start A: %v", err)
	}

	// Reload: A removed, B added.
	ruleB := config.NATListenerRule{ListenPort: 18083, Dst: "10.77.0.2:81", AllowedCIDRs: []string{"0.0.0.0/0"}}
	if err := srvRev.ReloadListeners([]config.NATListenerRule{ruleB}); err != nil {
		t.Fatalf("reload: %v", err)
	}

	ports := srvRev.ListenerPorts()
	if len(ports) != 1 || ports[0] != 18083 {
		t.Fatalf("post-reload ports: %v", ports)
	}

	// A must refuse connections now.
	if c, err := net.DialTimeout("tcp4", "127.0.0.1:18082", time.Second); err == nil {
		c.Close()
		t.Fatal("removed listener A still accepting")
	}
}

// ————— helpers —————

// peerStack is a second netstack playing the "client kernel" role: it
// accepts TCP on the tunnel address, exactly as the client's kernel
// would for a service bound behind its TUN.
type peerStack struct {
	s   *stack.Stack
	rev *ReverseNAT
}

func newPeerStack(t *testing.T, ip net.IP, cidr string, w PacketWriter, logger *zap.Logger) *peerStack {
	t.Helper()
	rev, err := NewReverseNAT(w, ip, cidr, logger)
	if err != nil {
		t.Fatalf("peer netstack: %v", err)
	}
	return &peerStack{s: rev.s, rev: rev}
}

// listen binds a real TCP listener on the peer stack (as the service
// behind the TUN would through the client's kernel).
func (p *peerStack) listen(ip net.IP, port int) (net.Listener, error) {
	return gonet.ListenTCP(p.s, tcpip.FullAddress{
		NIC:  1,
		Addr: tcpip.AddrFrom4([4]byte(ip.To4())),
		Port: uint16(port),
	}, ipv4.ProtocolNumber)
}

func serveEcho(ln net.Listener) {
	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}
		go func(c net.Conn) {
			defer c.Close()
			_, _ = io.Copy(c, c)
		}(conn)
	}
}
