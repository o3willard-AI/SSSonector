package nat

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config"
	"go.uber.org/zap"
)

// Engine is the NAT/PAT subsystem. It intercepts packets on the tunnel
// data path, enforces ACLs, and performs stateful translation.
//
// Wire format contract: Read/Write operate on raw IPv4 packets (as a TUN
// device in L3 mode delivers them). The engine is inserted only when NAT
// is enabled; when disabled it is never constructed and the data path is
// untouched (zero behavior change).
type Engine struct {
	cfg      *config.NATConfig
	logger   *zap.Logger
	acl      *ForwardACL
	table    *ConnTable
	egressIP net.IP // SNAT source address (the egress interface IP)

	// tunnel net: the tunnel-side subnet (client TUN IPs live here)
	tunnelSubnet *net.IPNet

	mu sync.RWMutex // guards cfg-derived state for hot reload

	// Stats
	forwardedPackets atomic.Uint64 // tunnel→egress translated
	returnPackets    atomic.Uint64 // egress→tunnel reverse-translated
	droppedPackets   atomic.Uint64 // ACL denies + malformed + no-translation
	poolExhausted    atomic.Uint64

	// Drop-reason breakdown (QA Phase A2): every drop site has a label.
	dropTunnelICMPNoSubnet atomic.Uint64 // non-TCP from src outside tunnel subnet
	dropTunnelParse        atomic.Uint64 // tunnel-side malformed/non-IPv4
	dropTunnelACL          atomic.Uint64 // tunnel-side ACL deny
	dropTunnelKey          atomic.Uint64 // tunnel-side key build failure (IPv6 etc.)
	dropTunnelPool         atomic.Uint64 // SNAT pool exhaustion
	dropEgressParse        atomic.Uint64 // egress-side malformed/non-IPv4
	dropEgressNoFlow       atomic.Uint64 // SNAT return with no conntrack entry
	dropEgressNotOurs      atomic.Uint64 // egress packet not to us, not tunnel-bound

	aclDropToken atomic.Uint32 // rate-limiter: last unix second a deny was logged
}

// Options configures the engine.
type Options struct {
	// EgressIP is the source address used for SNAT. Required when
	// forward NAT is enabled (fail closed without it).
	EgressIP net.IP
	// SNATPortBase/Max bound the ephemeral pool used for translations.
	SNATPortBase uint16
	SNATPortMax  uint16
	// FlowIdleTimeout is how long an idle conntrack entry survives.
	// Zero defaults to 5m.
	FlowIdleTimeout time.Duration
	// TunnelSubnet is the tunnel-side network (e.g. 10.77.0.0/24).
	// Egress-side packets addressed into this subnet are normal tunnel
	// traffic and pass through untranslated; without it the NAT would
	// eat the host kernel's replies to peers.
	TunnelSubnet *net.IPNet
}

// NewEngine builds an engine from validated NAT config. Returns nil when
// NAT is disabled — callers must treat that as "no engine, data path
// unchanged" rather than an error.
func NewEngine(cfg *config.NATConfig, opts Options, logger *zap.Logger) (*Engine, error) {
	if cfg == nil || !cfg.Enabled {
		return nil, nil // disabled: no engine, no error
	}
	if logger == nil {
		return nil, fmt.Errorf("logger is required")
	}

	eng := &Engine{
		cfg:    cfg,
		logger: logger.Named("nat"),
	}

	// Fail closed: forward NAT without an egress address is a config error.
	if cfg.Forward.Enabled {
		if opts.EgressIP == nil || opts.EgressIP.To4() == nil {
			return nil, ErrNoEgressAddress
		}
		base := opts.SNATPortBase
		if base == 0 {
			base = 40000
		}
		max := opts.SNATPortMax
		if max == 0 {
			max = 60000
		}
		table, err := NewConnTable(base, max)
		if err != nil {
			return nil, err
		}
		eng.table = table
		eng.egressIP = opts.EgressIP.To4()
	}

	// Compile ACLs from config.
	if cfg.Forward.Enabled {
		acl, err := CompileForwardACL(cfg.Forward.Rules)
		if err != nil {
			return nil, fmt.Errorf("forward ACL: %w", err)
		}
		eng.acl = acl

		// Tunnel subnet: egress-side packets addressed here are normal
		// tunnel traffic (kernel replies to peers) and must pass through
		// untranslated. Callers supply it via Options.
		if opts.TunnelSubnet != nil {
			eng.tunnelSubnet = opts.TunnelSubnet
		}
	}

	return eng, nil
}

// ProcessTunnelPacket handles a packet read from the tunnel (arriving
// from the peer) on a host with forward NAT enabled. It enforces the ACL
// and, if allowed, rewrites the source to the egress address.
//
// Returns the translated packet (which may be the input slice mutated in
// place) or an error. On error the packet MUST be dropped — never passed
// through untranslated (fail closed).
func (e *Engine) ProcessTunnelPacket(pkt []byte) ([]byte, error) {
	if !e.cfg.Forward.Enabled || e.table == nil {
		return nil, ErrNoTranslation
	}

	// Non-TCP tunnel traffic destined to egress (e.g. ICMP to a LAN
	// host) is ACL-checkable only by address: evaluate src against the
	// tunnel subnet allowlist; no port constraint applies.
	if len(pkt) >= 20 && pkt[0]>>4 == 4 {
		proto := pkt[9]
		dstRaw := net.IP(pkt[16:20])
		if proto != ipv4ProtoTCP {
			if e.tunnelSubnet == nil || !e.tunnelSubnet.Contains(net.IP(pkt[12:16])) {
				e.dropTunnelICMPNoSubnet.Add(1)
				e.droppedPackets.Add(1)
				return nil, ErrNoTranslation
			}
			_ = dstRaw
			return pkt, nil
		}
	}

	srcIP, dstIP, srcPort, dstPort, tcpOff, err := parseIPv4TCP(pkt)
	if err != nil {
		e.dropTunnelParse.Add(1)
		e.droppedPackets.Add(1)
		return nil, err
	}

	decision, ruleIdx, comment := e.acl.Evaluate(srcIP, dstIP, dstPort)
	if decision != ACLAllow {
		e.dropTunnelACL.Add(1)
		e.droppedPackets.Add(1)
		e.logDenyRateLimited(srcIP, dstIP, dstPort, ruleIdx, comment)
		return nil, ErrNoTranslation
	}

	key, err := IPKey(srcIP, srcPort, dstIP, dstPort)
	if err != nil {
		e.dropTunnelKey.Add(1)
		e.droppedPackets.Add(1)
		return nil, err
	}

	now := time.Now()
	entry, ok := e.table.lookupOrAllocate(key, now)
	if !ok {
		e.dropTunnelPool.Add(1)
		e.poolExhausted.Add(1)
		e.droppedPackets.Add(1)
		return nil, ErrPoolExhausted
	}

	// Observe flags for state tracking (before rewrite mutates anything).
	if syn, ack, fin, rst := tcpFlags(pkt, tcpOff); syn || ack || fin || rst {
		e.table.Observe(entry, syn, ack, fin, rst, now)
	}

	// Rewrite src IP + src port to the SNAT identity.
	oldSrcIP := [4]byte{pkt[12], pkt[13], pkt[14], pkt[15]}
	newSrcIP := [4]byte{e.egressIP[0], e.egressIP[1], e.egressIP[2], e.egressIP[3]}
	oldSrcPort := uint16(srcPort)
	newSrcPort := entry.snatPort

	rewriteIPv4Src(pkt, oldSrcIP, newSrcIP)
	rewriteTCPPort(pkt, tcpOff, 0 /*source port offset*/, oldSrcPort, newSrcPort)
	recomputeIPv4Checksum(pkt)
	recomputeTCPChecksum(pkt, tcpOff)

	e.forwardedPackets.Add(1)
	return pkt, nil
}

// ProcessEgressPacket handles a packet read from the egress TUN. Three
// cases:
//  1. Return traffic of a translated flow (dst = SNAT identity) →
//     reverse-translate the destination back to the tunnel side.
//  2. Traffic addressed to the tunnel subnet (e.g. this host's kernel
//     replying to a peer) → pass through untranslated; this is normal
//     tunnel traffic the NAT must not eat.
//  3. Anything else → drop (fail closed; not tunnel traffic).
func (e *Engine) ProcessEgressPacket(pkt []byte) ([]byte, error) {
	if !e.cfg.Forward.Enabled || e.table == nil {
		return nil, ErrNoTranslation
	}

	// Case 2: destined to the tunnel subnet — normal tunnel traffic
	// (e.g. this host's kernel replying to a peer's ICMP echo). Checked
	// on the IP header alone, before the TCP parse, so non-TCP tunnel
	// traffic (ICMP) passes through.
	if len(pkt) >= 20 && pkt[0]>>4 == 4 {
		dstRaw := net.IP(pkt[16:20])
		if e.tunnelSubnet != nil && e.tunnelSubnet.Contains(dstRaw) {
			return pkt, nil
		}
	}

	_, dstIP, _, dstPort, tcpOff, err := parseIPv4TCP(pkt)
	if err != nil {
		e.dropEgressParse.Add(1)
		e.droppedPackets.Add(1)
		return nil, err
	}

	// Case 1: return traffic (dst was rewritten to egressIP:snatPort).
	if dstIP.Equal(e.egressIP) {
		snatted, ok := e.table.Reverse(uint16(dstPort), time.Now())
		if !ok {
			e.dropEgressNoFlow.Add(1)
			e.droppedPackets.Add(1)
			return nil, ErrNoTranslation
		}

		newDstIP := net.IP(snatted.key.srcIP[:])
		newDstPort := snatted.key.srcPort

		oldDstIP := [4]byte{pkt[16], pkt[17], pkt[18], pkt[19]}
		rewriteIPv4Dst(pkt, oldDstIP, newDstIP)
		rewriteTCPPort(pkt, tcpOff, 2 /*destination port offset*/, uint16(dstPort), newDstPort)
		recomputeIPv4Checksum(pkt)
		recomputeTCPChecksum(pkt, tcpOff)

		e.returnPackets.Add(1)
		return pkt, nil
	}

	// Case 3: not addressed to us and not tunnel-bound. Drop.
	e.dropEgressNotOurs.Add(1)
	e.droppedPackets.Add(1)
	return nil, ErrNoTranslation
}

// Stats is a point-in-time snapshot of engine counters.
type Stats struct {
	ForwardedPackets uint64
	ReturnPackets    uint64
	DroppedPackets   uint64
	PoolExhausted    uint64
	ActiveFlows      int
	ACLAllowed       uint64
	ACLDenied        uint64

	// Drop-reason breakdown (QA Phase A2)
	DropTunnelICMPNoSubnet uint64
	DropTunnelParse        uint64
	DropTunnelACL          uint64
	DropTunnelKey          uint64
	DropTunnelPool         uint64
	DropEgressParse        uint64
	DropEgressNoFlow       uint64
	DropEgressNotOurs      uint64
}

// Stats returns current counters.
func (e *Engine) Stats() Stats {
	s := Stats{
		ForwardedPackets: e.forwardedPackets.Load(),
		ReturnPackets:    e.returnPackets.Load(),
		DroppedPackets:   e.droppedPackets.Load(),
		PoolExhausted:    e.poolExhausted.Load(),
	}
	if e.table != nil {
		s.ActiveFlows = e.table.Size()
	}
	if e.acl != nil {
		st := e.acl.Stats()
		s.ACLAllowed = st.Allowed
		s.ACLDenied = st.Denied
	}
	s.DropTunnelICMPNoSubnet = e.dropTunnelICMPNoSubnet.Load()
	s.DropTunnelParse = e.dropTunnelParse.Load()
	s.DropTunnelACL = e.dropTunnelACL.Load()
	s.DropTunnelKey = e.dropTunnelKey.Load()
	s.DropTunnelPool = e.dropTunnelPool.Load()
	s.DropEgressParse = e.dropEgressParse.Load()
	s.DropEgressNoFlow = e.dropEgressNoFlow.Load()
	s.DropEgressNotOurs = e.dropEgressNotOurs.Load()
	return s
}

// logDenyRateLimited emits at most one deny log per second (token-free
// approximation using an atomic swap of the current second).
func (e *Engine) logDenyRateLimited(srcIP, dstIP net.IP, dstPort, ruleIdx int, comment string) {
	sec := uint32(time.Now().Unix())
	prev := e.aclDropToken.Swap(sec)
	if prev == sec {
		return // already logged this second
	}
	e.logger.Warn("NAT ACL denied flow",
		zap.String("src", srcIP.String()),
		zap.String("dst", fmt.Sprintf("%s:%d", dstIP.String(), dstPort)),
		zap.Int("rule", ruleIdx),
		zap.String("rule_comment", comment),
	)
}
