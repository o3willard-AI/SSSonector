package nat

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// tcpState tracks a forward-NAT connection through its lifetime.
type tcpState int

const (
	stateSynSent tcpState = iota
	stateEstablished
	stateFinWait // one direction half-closed
	stateClosed  // fully done, awaiting GC
)

// flowKey identifies a tunnel-side flow by its 5-tuple components
// (protocol is implicit: TCP-only in v1).
type flowKey struct {
	srcIP   [4]byte
	srcPort uint16
	dstIP   [4]byte
	dstPort uint16
}

// entry is one forward-NAT connection table record.
type entry struct {
	key        flowKey
	snatPort   uint16 // allocated egress source port
	state      tcpState
	createdAt  time.Time
	lastActive time.Time
}

// ConnTable tracks translated flows and their reverse mappings.
// A single mutex guards the table; operations are O(1) map lookups and
// never block on I/O, so lock hold times are tiny (no sleeps under lock).
type ConnTable struct {
	mu       sync.Mutex
	byFlow   map[flowKey]*entry
	bySNAT   map[uint16]*entry // snatPort -> entry (reverse translation)
	used     map[uint16]bool   // allocated SNAT ports
	nextSNAT uint16
	poolBase uint16
	poolMax  uint16
}

// NewConnTable creates an empty table. snatPortBase is the first
// ephemeral port used for translations (e.g. 40000) and snatPortMax the
// last inclusive (e.g. 60000).
func NewConnTable(snatPortBase, snatPortMax uint16) (*ConnTable, error) {
	if snatPortBase == 0 || snatPortMax <= snatPortBase {
		return nil, fmt.Errorf("invalid SNAT port range %d-%d", snatPortBase, snatPortMax)
	}
	return &ConnTable{
		byFlow:   make(map[flowKey]*entry),
		bySNAT:   make(map[uint16]*entry),
		used:     make(map[uint16]bool),
		nextSNAT: snatPortBase,
		poolBase: snatPortBase,
		poolMax:  snatPortMax,
	}, nil
}

// lookupOrAllocate returns the existing entry for the flow, or creates
// one with a fresh SNAT port. ok=false means pool exhaustion.
func (t *ConnTable) lookupOrAllocate(key flowKey, now time.Time) (*entry, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if e, ok := t.byFlow[key]; ok {
		e.lastActive = now
		return e, true
	}

	port, ok := t.allocatePortLocked()
	if !ok {
		return nil, false
	}
	e := &entry{
		key:        key,
		snatPort:   port,
		state:      stateSynSent,
		createdAt:  now,
		lastActive: now,
	}
	t.byFlow[key] = e
	t.bySNAT[port] = e
	return e, true
}

// allocatePortLocked scans from the cursor for a free port, wrapping at
// poolMax back to poolBase, visiting each port at most once.
func (t *ConnTable) allocatePortLocked() (uint16, bool) {
	total := int(t.poolMax-t.poolBase) + 1
	for i := 0; i < total; i++ {
		if !t.used[t.nextSNAT] {
			p := t.nextSNAT
			t.used[p] = true
			if t.nextSNAT == t.poolMax {
				t.nextSNAT = t.poolBase
			} else {
				t.nextSNAT++
			}
			return p, true
		}
		if t.nextSNAT == t.poolMax {
			t.nextSNAT = t.poolBase
		} else {
			t.nextSNAT++
		}
	}
	return 0, false
}

// Reverse returns the entry whose SNAT port matches, for return traffic.
func (t *ConnTable) Reverse(snatPort uint16, now time.Time) (*entry, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	e, ok := t.bySNAT[snatPort]
	if ok {
		e.lastActive = now
	}
	return e, ok
}

// Observe updates the TCP state of an entry from observed flags.
func (t *ConnTable) Observe(e *entry, syn, ack, fin, rst bool, now time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()
	e.lastActive = now
	switch {
	case rst:
		e.state = stateClosed
	case syn && !ack:
		e.state = stateSynSent
	case syn && ack:
		e.state = stateEstablished
	case fin:
		switch e.state {
		case stateEstablished:
			e.state = stateFinWait
		case stateFinWait:
			e.state = stateClosed
		}
	}
}

// Release frees an entry (e.g. RST or explicit close).
func (t *ConnTable) Release(e *entry) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.releaseLocked(e)
}

func (t *ConnTable) releaseLocked(e *entry) {
	delete(t.byFlow, e.key)
	delete(t.bySNAT, e.snatPort)
	delete(t.used, e.snatPort)
}

// GC sweeps entries idle longer than maxIdle. Returns the number removed.
func (t *ConnTable) GC(now time.Time, maxIdle time.Duration) int {
	t.mu.Lock()
	defer t.mu.Unlock()
	removed := 0
	for k, e := range t.byFlow {
		if now.Sub(e.lastActive) > maxIdle {
			t.releaseLocked(e)
			removed++
			delete(t.byFlow, k) // safe during range in Go
		}
	}
	return removed
}

// Size returns the current number of tracked flows.
func (t *ConnTable) Size() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.byFlow)
}

// IPKey builds a flowKey from addresses and ports. Returns error for
// non-IPv4 input (v1 is IPv4-only; the packet parser rejects earlier).
func IPKey(srcIP net.IP, srcPort int, dstIP net.IP, dstPort int) (flowKey, error) {
	src4 := srcIP.To4()
	dst4 := dstIP.To4()
	if src4 == nil || dst4 == nil {
		return flowKey{}, fmt.Errorf("IPv6 flows are not supported in NAT v1")
	}
	var k flowKey
	copy(k.srcIP[:], src4)
	copy(k.dstIP[:], dst4)
	k.srcPort = uint16(srcPort)
	k.dstPort = uint16(dstPort)
	return k, nil
}
