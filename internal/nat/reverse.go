package nat

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"go.uber.org/zap"
)

// PacketWriter emits raw IPv4 frames into a conduit (the tunnel).
type PacketWriter interface {
	WritePacket(p []byte) error
}

// ReverseNAT publishes services through the tunnel: public listeners on
// this host relay TCP connections through the tunnel toward services
// behind the peer's TUN.
//
// Mechanism: a gVisor netstack owns a channel link endpoint. Frames the
// stack emits are pumped into the tunnel; tunnel frames are injected
// into the stack. Outbound relays dial via netstack (the real TCP
// handshake, windowing, retransmit and half-close are handled by
// netstack, which is why TCP is not hand-rolled). Half-close is
// preserved in both relay directions.
type ReverseNAT struct {
	s   *stack.Stack
	ch  *channel.Endpoint
	nic tcpip.NICID

	mu        sync.Mutex
	listeners map[int]*publicListener

	stopCh  chan struct{}
	closeOnce sync.Once

	acceptsTotal atomic.Uint64
	aclDenies    atomic.Uint64
	relayErrors  atomic.Uint64
}

// publicListener is one published port mapping.
type publicListener struct {
	cfg     config.NATListenerRule
	acl     *ListenerACL
	ln      *net.TCPListener
	wg      sync.WaitGroup
	done    chan struct{}
	accepts atomic.Uint64
	denies  atomic.Uint64
}

// NewReverseNAT builds a netstack whose link writes frames into
// tunnelWriter and whose inbound frames arrive via DeliverTunnelPacket.
// tunnelIP is this host's tunnel-side address (the stack answers as it);
// tunnelCIDR scopes the route through the virtual link.
func NewReverseNAT(tunnelWriter PacketWriter, tunnelIP net.IP, tunnelCIDR string, logger *zap.Logger) (*ReverseNAT, error) {
	if tunnelWriter == nil {
		return nil, fmt.Errorf("tunnel writer is required")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	ip4 := tunnelIP.To4()
	if ip4 == nil {
		return nil, fmt.Errorf("tunnel address must be IPv4")
	}
	_, ipNet, err := net.ParseCIDR(tunnelCIDR)
	if err != nil {
		return nil, fmt.Errorf("tunnel subnet: %w", err)
	}
	ones, _ := ipNet.Mask.Size()

	s := stack.New(stack.Options{
		NetworkProtocols:   []stack.NetworkProtocolFactory{ipv4.NewProtocol},
		TransportProtocols: []stack.TransportProtocolFactory{tcp.NewProtocol},
	})

	// Channel link endpoint: stack writes outbound frames into the
	// channel queue; we drain them into the tunnel. Inbound frames are
	// injected with InjectInbound.
	ch := channel.New(256, 1500, "")
	if err := s.CreateNIC(1, ch); err != nil {
		return nil, fmt.Errorf("netstack CreateNIC: %v", err)
	}
	if err := s.AddProtocolAddress(1, tcpip.ProtocolAddress{
		Protocol: ipv4.ProtocolNumber,
		AddressWithPrefix: tcpip.AddressWithPrefix{
			Address:   tcpip.AddrFrom4([4]byte(ip4)),
			PrefixLen: ones,
		},
	}, stack.AddressProperties{}); err != nil {
		return nil, fmt.Errorf("netstack address: %v", err)
	}
	s.SetRouteTable([]tcpip.Route{
		{
			Destination: header.IPv4EmptySubnet,
			NIC:         1,
		},
	})

	r := &ReverseNAT{
		s:         s,
		ch:        ch,
		nic:       1,
		listeners: make(map[int]*publicListener),
		stopCh:    make(chan struct{}),
	}

	// Outbound pump: frames the stack emits go into the tunnel.
	go r.pumpOutbound(tunnelWriter, logger)

	return r, nil
}

// pumpOutbound drains stack-emitted frames and writes them into the tunnel.
func (r *ReverseNAT) pumpOutbound(w PacketWriter, logger *zap.Logger) {
	for {
		pkt := r.ch.Read()
		if pkt == nil {
			select {
			case <-r.stopCh:
				return
			default:
			}
			continue
		}
		frame := pkt.ToView().AsSlice()
		if err := w.WritePacket(frame); err != nil {
			logger.Warn("netstack frame write failed", zap.Error(err))
		}
		pkt.DecRef()
	}
}

// DeliverTunnelPacket injects one raw IPv4 frame (read from the tunnel)
// into the netstack, toward published services.
func (r *ReverseNAT) DeliverTunnelPacket(pkt []byte) {
	if len(pkt) < header.IPv4MinimumSize {
		return
	}
	pb := stack.NewPacketBuffer(stack.PacketBufferOptions{
		Payload: buffer.MakeWithData(pkt),
	})
	defer pb.DecRef()
	r.ch.InjectInbound(header.IPv4ProtocolNumber, pb)
}

// StartListener binds a public port and relays accepted connections
// through netstack to the tunnel-side destination. ACL gates sources.
func (r *ReverseNAT) StartListener(rule config.NATListenerRule) error {
	acl, err := CompileListenerACL(rule.AllowedCIDRs)
	if err != nil {
		return err
	}
	dstHost, dstPortStr, err := net.SplitHostPort(rule.Dst)
	if err != nil {
		return fmt.Errorf("listener %d: %w", rule.ListenPort, err)
	}
	dstIP := net.ParseIP(dstHost)
	if dstIP == nil {
		return fmt.Errorf("listener %d: dst host %q is not an IP", rule.ListenPort, dstHost)
	}
	var dstPort int
	if _, err := fmt.Sscanf(dstPortStr, "%d", &dstPort); err != nil || dstPort < 1 || dstPort > 65535 {
		return fmt.Errorf("listener %d: invalid dst port %q", rule.ListenPort, dstPortStr)
	}

	ln, err := net.ListenTCP("tcp4", &net.TCPAddr{Port: rule.ListenPort})
	if err != nil {
		return fmt.Errorf("listener %d bind: %w", rule.ListenPort, err)
	}

	pl := &publicListener{
		cfg:  rule,
		acl:  acl,
		ln:   ln,
		done: make(chan struct{}),
	}

	r.mu.Lock()
	if _, exists := r.listeners[rule.ListenPort]; exists {
		r.mu.Unlock()
		ln.Close()
		return fmt.Errorf("listener %d already running", rule.ListenPort)
	}
	r.listeners[rule.ListenPort] = pl
	r.mu.Unlock()

	pl.wg.Add(1)
	go r.acceptLoop(pl, dstIP, dstPort)
	return nil
}

// acceptLoop gates public connections through the listener ACL, dials
// the tunnel destination via netstack, and relays both directions.
func (r *ReverseNAT) acceptLoop(pl *publicListener, dstIP net.IP, dstPort int) {
	defer pl.wg.Done()
	for {
		conn, err := pl.ln.Accept()
		if err != nil {
			select {
			case <-pl.done:
				return
			default:
			}
			continue
		}

		ra, ok := conn.RemoteAddr().(*net.TCPAddr)
		if !ok || !pl.acl.Evaluate(ra.IP) {
			pl.denies.Add(1)
			r.aclDenies.Add(1)
			conn.Close()
			continue
		}

		pl.accepts.Add(1)
		r.acceptsTotal.Add(1)
		go r.relay(conn, dstIP, dstPort)
	}
}

// relay dials the tunnel destination through netstack — the TCP
// handshake travels the tunnel link — and copies bytes both directions
// with half-close on each finish.
func (r *ReverseNAT) relay(publicConn net.Conn, dstIP net.IP, dstPort int) {
	gconn, err := gonet.DialTCPWithBind(context.Background(), r.s,
		tcpip.FullAddress{},
		tcpip.FullAddress{
			NIC:  r.nic,
			Addr: tcpip.AddrFrom4([4]byte(dstIP.To4())),
			Port: uint16(dstPort),
		},
		ipv4.ProtocolNumber,
	)
	if err != nil {
		r.relayErrors.Add(1)
		publicConn.Close()
		return
	}
	relayBidirectional(publicConn, gconn)
}

// relayBidirectional copies both directions with half-close on finish
// (data-path correctness invariant 4).
func relayBidirectional(a, b net.Conn) {
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(a, b)
		if cw, ok := a.(interface{ CloseWrite() error }); ok {
			_ = cw.CloseWrite()
		}
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(b, a)
		if cw, ok := b.(interface{ CloseWrite() error }); ok {
			_ = cw.CloseWrite()
		}
	}()
	wg.Wait()
}

// StopListener removes a published port (hot reload).
func (r *ReverseNAT) StopListener(port int) error {
	r.mu.Lock()
	pl, ok := r.listeners[port]
	if ok {
		delete(r.listeners, port)
	}
	r.mu.Unlock()
	if !ok {
		return fmt.Errorf("listener %d not running", port)
	}
	close(pl.done)
	pl.ln.Close()
	pl.wg.Wait()
	return nil
}

// Stop tears down all listeners and the netstack.
func (r *ReverseNAT) Stop() {
	r.closeOnce.Do(func() {
		close(r.stopCh)
		r.mu.Lock()
		pls := make([]*publicListener, 0, len(r.listeners))
		for _, pl := range r.listeners {
			close(pl.done)
			pl.ln.Close()
			pls = append(pls, pl)
		}
		r.listeners = make(map[int]*publicListener)
		r.mu.Unlock()
		for _, pl := range pls {
			pl.wg.Wait()
		}
		r.s.Close()
	})
}

// Stats summarizes reverse-path counters.
func (r *ReverseNAT) Stats() (accepts, denies, relayErrors uint64) {
	return r.acceptsTotal.Load(), r.aclDenies.Load(), r.relayErrors.Load()
}

// ListenerPorts lists currently active published ports.
func (r *ReverseNAT) ListenerPorts() []int {
	r.mu.Lock()
	defer r.mu.Unlock()
	ports := make([]int, 0, len(r.listeners))
	for p := range r.listeners {
		ports = append(ports, p)
	}
	return ports
}
