package nat

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/waiter"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"go.uber.org/zap"
)

// ForwardProxy implements the netstack-proxy forward-NAT design (QA
// Phase B / T4 decision): TCP flows from the tunnel are terminated in a
// dedicated netstack and re-originated as normal local sockets toward
// the egress destination.
//
// Why netstack instead of packet-rewrite SNAT: rewriting the client's
// source to the host's own IP makes the LAN service's replies locally
// delivered by the kernel — they never re-enter tun0 for reverse
// translation. Terminating the flow in netstack and re-dialing via a
// local socket avoids kernel forwarding entirely (ip_forward stays 0),
// keeps ACL enforcement per-flow in-daemon, and rides the same
// battle-tested gVisor TCP implementation as reverse PAT.
//
// Frame path: tunnel-side packets are fed to DeliverTunnelPacket
// (called from the tunnel read path); netstack-emitted frames are
// pumped into the tunnel via the frame sink.
type ForwardProxy struct {
	s   *stack.Stack
	ch  *channel.Endpoint
	nic tcpip.NICID

	// acl/allowlist: tunnel-side source CIDRs permitted to use the
	// proxy (compiled from forward rules' src_cidr — union).
	allowedSrcs []*net.IPNet

	// rules drive destination selection: first rule whose src matches
	// and whose dst_cidr contains the packet's dst wins; dst port must
	// be in the rule's port set.
	rules []compiledProxyRule

	outMu     sync.Mutex
	frameSink interface{ WritePacket(p []byte) error }

	stopCh chan struct{}
	pumpWG sync.WaitGroup
	closeO sync.Once

	sessionsTotal atomic.Uint64
	aclDenies     atomic.Uint64
	relayErrors   atomic.Uint64
	dropped       atomic.Uint64 // non-TCP / unmatched (fail closed)

	logger *zap.Logger

	// Last badTCPHdrLen diagnostics (QA, inspected by tests).
	lastBadData12 int
	lastBadLen    int
	lastBadHex    string

	// Decomposition drop labels (QA): which early return fired.
	dropNoNetHdr       atomic.Uint64
	dropNoPullUp       atomic.Uint64
	dropBadTCPHdrLen   atomic.Uint64
	dropNoTransportHdr atomic.Uint64
	dropBadChecksum    atomic.Uint64
}

// compiledProxyRule mirrors config.NATForwardRule with parsed CIDRs
// and a port set (empty = matches nothing, fail closed).
type compiledProxyRule struct {
	src    *net.IPNet
	dst    *net.IPNet
	ports  map[int]bool
	ruleID int
}

// NewForwardProxy builds the proxy netstack. localIP/prefix is this
// host's tunnel-side address (the stack answers as it); frames the
// stack emits are written into the tunnel via sink.
func NewForwardProxy(localIP net.IP, tunnelCIDR string, rules []config.NATForwardRule, logger *zap.Logger) (*ForwardProxy, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}
	ip4 := localIP.To4()
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

	ch := channel.New(256, 1500, "")
	// Advertise RX checksum offload: packets were validated upstream by
	// tcpChecksumValidBytes, and gVisor's own verification of injected
	// frames was observed to fail spuriously (validSeg=0, no counters),
	// silently discarding client SYNs.
	ch.LinkEPCapabilities = stack.CapabilityRXChecksumOffload
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
		{Destination: header.IPv4EmptySubnet, NIC: 1},
	})
	// This gVisor version requires explicit NIC enablement before the
	// link endpoint delivers packets (QA: without it, InjectInbound
	// packets hit the disabledRx counter and vanish silently).
	if err := s.EnableNIC(1); err != nil {
		return nil, fmt.Errorf("netstack EnableNIC: %v", err)
	}

	// Tunnel-side allowlist: union of rule src_cidrs. Flows from other
	// sources are denied (fail closed).
	allow := make(map[string]*net.IPNet)
	compiled := make([]compiledProxyRule, 0, len(rules))
	for i, r := range rules {
		_, srcNet, err := net.ParseCIDR(r.SrcCIDR)
		if err != nil {
			return nil, fmt.Errorf("rule %d: invalid src_cidr: %w", i, err)
		}
		_, dstNet, err := net.ParseCIDR(r.DstCIDR)
		if err != nil {
			return nil, fmt.Errorf("rule %d: invalid dst_cidr: %w", i, err)
		}
		ports := make(map[int]bool, len(r.Ports))
		for _, p := range r.Ports {
			ports[p] = true
		}
		compiled = append(compiled, compiledProxyRule{
			src: srcNet, dst: dstNet, ports: ports, ruleID: i,
		})
		allow[r.SrcCIDR] = srcNet
	}
	allowlist := make([]*net.IPNet, 0, len(allow))
	for _, n := range allow {
		allowlist = append(allowlist, n)
	}

	return &ForwardProxy{
		s:           s,
		ch:          ch,
		nic:         1,
		allowedSrcs: allowlist,
		rules:       compiled,
		logger:      logger,
		stopCh:      make(chan struct{}),
	}, nil
}

// SetFrameSink installs the live tunnel frame sink (TLS-wrapped conn).
func (f *ForwardProxy) SetFrameSink(sink interface{ WritePacket(p []byte) error }) {
	f.outMu.Lock()
	f.frameSink = sink
	f.outMu.Unlock()
}

// DeliverTunnelPacket injects one raw IPv4 frame from the tunnel into
// the proxy netstack. TCP flows matching a forward rule are terminated
// here and proxied; everything else is dropped (fail closed).
func (f *ForwardProxy) DeliverTunnelPacket(pkt []byte) {
	if len(pkt) < header.IPv4MinimumSize {
		f.dropped.Add(1)
		return
	}
	if pkt[0]>>4 != 4 {
		f.dropped.Add(1)
		return
	}
	proto := pkt[9]
	if proto != 6 { // TCP
		// ICMP and other non-TCP to the host's tunnel IP pass to the
		// kernel via the normal TUN path (handled upstream); the proxy
		// only carries TCP flows.
		f.dropped.Add(1)
		return
	}

	// ACL pre-check by address: source must be in the tunnel allowlist.
	src := net.IP(pkt[12:16])
	if !f.srcAllowed(src) {
		f.aclDenies.Add(1)
		f.dropped.Add(1)
		return
	}

	// Diagnostics (QA): verify the checksum before injection so a
	// netstack silent drop is distinguishable from a bad packet.
	if !tcpChecksumValidBytes(pkt) {
		f.dropBadChecksum.Add(1)
		f.dropped.Add(1)
		return
	}

	// Inject the full raw IPv4 packet as Payload; gVisor's parse.IPv4
	// consumes the network header itself during HandlePacket (QA: the
	// earlier failures were the missing EnableNIC and a malformed
	// synthetic SYN missing the data-offset field, not framing).
	pb := stack.NewPacketBuffer(stack.PacketBufferOptions{
		Payload: buffer.MakeWithData(pkt),
	})
	defer pb.DecRef()
	f.ch.InjectInbound(header.IPv4ProtocolNumber, pb)
}

// tcpChecksumValidBytes verifies the TCP checksum of a raw IPv4 packet.
func tcpChecksumValidBytes(pkt []byte) bool {
	if len(pkt) < 40 {
		return false
	}
	ihl := int(pkt[0]&0x0F) * 4
	if ihl < 20 || len(pkt) < ihl+20 {
		return false
	}
	tcpOff := ihl
	ckAt := tcpOff + 16
	stored := uint16(pkt[ckAt])<<8 | uint16(pkt[ckAt+1])
	sum := uint32(0)
	tcpLen := len(pkt) - tcpOff
	sum += uint32(pkt[12])<<8 | uint32(pkt[13])
	sum += uint32(pkt[14])<<8 | uint32(pkt[15])
	sum += uint32(pkt[16])<<8 | uint32(pkt[17])
	sum += uint32(pkt[18])<<8 | uint32(pkt[19])
	sum += 6
	sum += uint32(tcpLen)
	for i := tcpOff; i < len(pkt); i += 2 {
		if i == ckAt {
			continue
		}
		if i+1 < len(pkt) {
			sum += uint32(pkt[i])<<8 | uint32(pkt[i+1])
		} else {
			sum += uint32(pkt[i]) << 8
		}
	}
	for sum>>16 != 0 {
		sum = (sum >> 16) + (sum & 0xFFFF)
	}
	return ^uint16(sum) == stored
}

func (f *ForwardProxy) srcAllowed(src net.IP) bool {
	for _, n := range f.allowedSrcs {
		if n.Contains(src) {
			return true
		}
	}
	return false
}

// StartPump drains netstack-emitted frames into the tunnel sink.
func (f *ForwardProxy) StartPump(logger *zap.Logger) {
	f.pumpWG.Add(1)
	go func() {
		defer f.pumpWG.Done()
		for {
			pkt := f.ch.Read()
			if pkt == nil {
				select {
				case <-f.stopCh:
					return
				default:
				}
				continue
			}
			frame := pkt.ToView().AsSlice()

			f.outMu.Lock()
			sink := f.frameSink
			f.outMu.Unlock()

			if sink != nil {
				if err := sink.WritePacket(frame); err != nil {
					logger.Warn("forward proxy frame write failed", zap.Error(err))
				}
			}
			pkt.DecRef()
		}
	}()
}

// SetTransportHandler installs the TCP forwarder: flows the netstack
// accepts (i.e. client-originated TCP matching the stack's address) are
// evaluated against the forward rules and re-dialed to the egress
// destination via a local socket.
func (f *ForwardProxy) SetTransportHandler(rules []config.NATForwardRule, logger *zap.Logger) error {
	compiled, err := compileProxyRules(rules)
	if err != nil {
		return err
	}
	f.rules = compiled
	// maxInFlight=0 would silently drop every SYN (inFlight >= 0 is
	// always true); use a generous cap for point-to-point usage.
	fwd := tcp.NewForwarder(f.s, 0, 4096, func(req *tcp.ForwarderRequest) {
		id := req.ID()
		dst4 := id.LocalAddress.As4()
		dstIP := net.IP(dst4[:])
		src4 := id.RemoteAddress.As4()
		srcIP := net.IP(src4[:])
		dstPort := int(id.LocalPort)
		matched := -1
		for _, r := range f.rules {
			if r.src.Contains(srcIP) && r.dst.Contains(dstIP) && r.ports[dstPort] {
				matched = r.ruleID
				break
			}
		}
		if matched < 0 {
			f.aclDenies.Add(1)
			f.dropped.Add(1)
			req.Complete(true) // RST: fail closed
			return
		}

		// Complete the netstack handshake, then re-originate toward the
		// ORIGINAL destination via a local socket (the kernel routes it
		// to the LAN normally).
		w, werr := waitConn(req, &waiter.Queue{})
		if werr != nil {
			f.relayErrors.Add(1)
			return
		}

		egressConn, derr := net.DialTimeout("tcp4",
			net.JoinHostPort(dstIP.String(), fmt.Sprint(dstPort)),
			10*time.Second)
		if derr != nil {
			f.relayErrors.Add(1)
			w.Close()
			return
		}

		f.sessionsTotal.Add(1)
		relayBidirectional(w, egressConn)
	})
	f.s.SetTransportProtocolHandler(tcp.ProtocolNumber, fwd.HandlePacket)
	return nil
}

// compileProxyRules parses config rules into the proxy's rule form.
// Malformed input is a programming error upstream (the validator
// rejects it); surfaced as an error rather than silently skipped.
func compileProxyRules(rules []config.NATForwardRule) ([]compiledProxyRule, error) {
	compiled := make([]compiledProxyRule, 0, len(rules))
	for i, r := range rules {
		_, srcNet, err := net.ParseCIDR(r.SrcCIDR)
		if err != nil {
			return nil, fmt.Errorf("rule %d: invalid src_cidr: %w", i, err)
		}
		_, dstNet, err := net.ParseCIDR(r.DstCIDR)
		if err != nil {
			return nil, fmt.Errorf("rule %d: invalid dst_cidr: %w", i, err)
		}
		ports := make(map[int]bool, len(r.Ports))
		for _, p := range r.Ports {
			ports[p] = true
		}
		compiled = append(compiled, compiledProxyRule{
			src: srcNet, dst: dstNet, ports: ports, ruleID: i,
		})
	}
	return compiled, nil
}

// waitConn completes the forwarder request via CreateEndpoint and wraps
// the endpoint as a net.Conn. On error the request is reset.
func waitConn(req *tcp.ForwarderRequest, wq *waiter.Queue) (net.Conn, error) {
	ep, terr := req.CreateEndpoint(wq)
	if terr != nil {
		req.Complete(true) // RST
		return nil, fmt.Errorf("forwarder endpoint: %v", terr)
	}
	req.Complete(false)
	return gonet.NewTCPConn(wq, ep), nil
}

// Stop tears down the proxy netstack.
func (f *ForwardProxy) Stop() {
	f.closeO.Do(func() {
		close(f.stopCh)
		f.pumpWG.Wait()
		f.s.Close()
	})
}

// Stats summarizes the proxy path.
func (f *ForwardProxy) Stats() (sessions, denies, relayErrors, dropped uint64) {
	return f.sessionsTotal.Load(), f.aclDenies.Load(), f.relayErrors.Load(), f.dropped.Load()
}

// DropReasons exposes the QA decomposition/drop counters for diagnostics.
func (f *ForwardProxy) DropReasons() (badChecksum, noNetHdr, noPullUp, badTCPHdrLen, noTransportHdr uint64) {
	return f.dropBadChecksum.Load(), f.dropNoNetHdr.Load(), f.dropNoPullUp.Load(),
		f.dropBadTCPHdrLen.Load(), f.dropNoTransportHdr.Load()
}
