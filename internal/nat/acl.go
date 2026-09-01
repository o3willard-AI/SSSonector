package nat

import (
	"fmt"
	"net"
	"sync/atomic"

	"github.com/o3willard-AI/SSSonector/internal/config"
)

// Direction classifies which side of the NAT a flow originates from.
type Direction string

const (
	// DirectionTunnelToEgress: forward NAT, packet from the tunnel
	// heading to an egress network (jump-host path).
	DirectionTunnelToEgress Direction = "tunnel_to_egress"
	// DirectionEgressToTunnel: return traffic of a forward NAT flow.
	DirectionEgressToTunnel Direction = "egress_to_tunnel"
	// DirectionPublicToListener: reverse PAT, public client hitting a
	// published listener.
	DirectionPublicToListener Direction = "public_to_listener"
)

// ACLDecision is the outcome of an ACL evaluation.
type ACLDecision string

const (
	ACLAllow ACLDecision = "allow"
	ACLDeny  ACLDecision = "deny"
)

// ACLStats records evaluation counters for metrics.
type ACLStats struct {
	Allowed uint64
	Denied  uint64
}

// ForwardACL evaluates forward-NAT rules against a flow.
// First match wins; no match denies (fail closed). The zero value
// denies everything.
type ForwardACL struct {
	rules    []compiledForwardRule
	stats    ACLStats
	dropDuma uint32 // rate-limiter token for deny logs
}

type compiledForwardRule struct {
	index   int
	comment string
	srcNet  *net.IPNet
	dstNet  *net.IPNet
	ports   map[int]bool // empty = matches nothing (fail closed)
}

// CompileForwardACL builds an evaluator from config rules. Malformed
// input is a programming error upstream (the validator rejects it); it
// is surfaced as an error here rather than silently skipped.
func CompileForwardACL(rules []config.NATForwardRule) (*ForwardACL, error) {
	compiled := make([]compiledForwardRule, 0, len(rules))
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
		compiled = append(compiled, compiledForwardRule{
			index:   i,
			comment: r.Comment,
			srcNet:  srcNet,
			dstNet:  dstNet,
			ports:   ports,
		})
	}
	return &ForwardACL{rules: compiled}, nil
}

// Evaluate returns allow/deny for a forward flow. A rule with no ports
// can never match; an unmatched flow is denied. A matching flow whose
// port set is empty is therefore denied by construction.
func (a *ForwardACL) Evaluate(srcIP, dstIP net.IP, dstPort int) (ACLDecision, int, string) {
	for _, r := range a.rules {
		if r.srcNet.Contains(srcIP) && r.dstNet.Contains(dstIP) && r.ports[dstPort] {
			atomic.AddUint64(&a.stats.Allowed, 1)
			return ACLAllow, r.index, r.comment
		}
	}
	atomic.AddUint64(&a.stats.Denied, 1)
	return ACLDeny, -1, ""
}

// Stats returns a snapshot of evaluation counters.
func (a *ForwardACL) Stats() ACLStats {
	return ACLStats{
		Allowed: atomic.LoadUint64(&a.stats.Allowed),
		Denied:  atomic.LoadUint64(&a.stats.Denied),
	}
}

// ListenerACL evaluates reverse-PAT listener source allowlists.
type ListenerACL struct {
	cidrs []*net.IPNet
}

// CompileListenerACL compiles an allowlist. The validator guarantees at
// least one CIDR; an empty list here compiles to deny-all (fail closed).
func CompileListenerACL(cidrs []string) (*ListenerACL, error) {
	compiled := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			return nil, fmt.Errorf("invalid allowed_cidr %q: %w", c, err)
		}
		compiled = append(compiled, n)
	}
	return &ListenerACL{cidrs: compiled}, nil
}

// Evaluate reports whether srcIP may use the listener.
func (a *ListenerACL) Evaluate(srcIP net.IP) bool {
	for _, n := range a.cidrs {
		if n.Contains(srcIP) {
			return true
		}
	}
	return false
}
