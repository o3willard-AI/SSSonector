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

	srcIP, dstIP, srcPort, dstPort, tcpOff, err := parseIPv4TCP(pkt)
	if err != nil {
		e.droppedPackets.Add(1)
		return nil, err
	}

	decision, ruleIdx, comment := e.acl.Evaluate(srcIP, dstIP, dstPort)
	if decision != ACLAllow {
		e.droppedPackets.Add(1)
		e.logDenyRateLimited(srcIP, dstIP, dstPort, ruleIdx, comment)
		return nil, ErrNoTranslation
	}

	key, err := IPKey(srcIP, srcPort, dstIP, dstPort)
	if err != nil {
		e.droppedPackets.Add(1)
		return nil, err
	}

	now := time.Now()
	entry, ok := e.table.lookupOrAllocate(key, now)
	if !ok {
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

// ProcessEgressPacket handles a packet read from the egress TUN (return
// traffic of a translated flow). If it matches a conntrack entry the
// destination is reverse-translated back to the tunnel-side address;
// otherwise the packet is dropped (fail closed — an unrelated egress
// packet is not tunnel traffic and must not leak into the tunnel).
func (e *Engine) ProcessEgressPacket(pkt []byte) ([]byte, error) {
	if !e.cfg.Forward.Enabled || e.table == nil {
		return nil, ErrNoTranslation
	}

	_, dstIP, _, dstPort, tcpOff, err := parseIPv4TCP(pkt)
	if err != nil {
		e.droppedPackets.Add(1)
		return nil, err
	}

	// Return traffic: dst was rewritten to egressIP:snatPort.
	if dstIP.Equal(e.egressIP) {
		snatted, ok := e.table.Reverse(uint16(dstPort), time.Now())
		if !ok {
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

	// Not addressed to us: not return traffic of any flow. Drop.
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
