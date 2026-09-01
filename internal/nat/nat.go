// Package nat implements the optional NAT/PAT subsystem for SSSonector.
//
// The engine is fail closed by construction: when NAT is disabled the
// daemon performs no translation of any kind; when enabled, every flow
// must match an explicit rule and the ACL evaluator's default is deny.
//
// Forward NAT (jump host): packets flowing tunnel→egress have their
// source rewritten to this host's egress address; return packets are
// reverse-translated via the connection table. Reverse PAT (service
// publishing): public listeners relay TCP through the tunnel to a
// service behind the peer's TUN.
package nat

import "fmt"

// Engine errors
var (
	// ErrPacketTooShort indicates a packet smaller than its headers.
	ErrPacketTooShort = fmt.Errorf("packet too short")
	// ErrNotIPv4 indicates a non-IPv4 packet where IPv4 is required.
	ErrNotIPv4 = fmt.Errorf("not an IPv4 packet")
	// ErrNotTCP indicates a non-TCP packet where TCP is required.
	ErrNotTCP = fmt.Errorf("not a TCP packet")
	// ErrNoTranslation indicates no conntrack entry and no rule
	// matched: the packet must be dropped (fail closed).
	ErrNoTranslation = fmt.Errorf("no translation available")
	// ErrPoolExhausted indicates the SNAT ephemeral port pool ran out.
	ErrPoolExhausted = fmt.Errorf("SNAT port pool exhausted")
	// ErrNoEgressAddress indicates forward NAT is enabled but no
	// egress source address was provided (fail closed).
	ErrNoEgressAddress = fmt.Errorf("egress address required for forward NAT")
)
