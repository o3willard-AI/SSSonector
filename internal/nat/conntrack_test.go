package nat

import (
	"net"
	"testing"
	"time"
)

func TestConnTableAllocateAndReverse(t *testing.T) {
	tbl, err := NewConnTable(40000, 40002)
	if err != nil {
		t.Fatalf("table: %v", err)
	}
	now := time.Now()

	src := net.ParseIP("10.77.0.2")
	dst := net.ParseIP("192.168.10.5")
	key, err := IPKey(src, 51234, dst, 80)
	if err != nil {
		t.Fatalf("key: %v", err)
	}

	e1, ok := tbl.lookupOrAllocate(key, now)
	if !ok {
		t.Fatal("first allocate must succeed")
	}
	if e1.snatPort != 40000 {
		t.Fatalf("first SNAT port: want 40000 got %d", e1.snatPort)
	}

	// Same flow returns the same entry (no new allocation).
	e2, _ := tbl.lookupOrAllocate(key, now)
	if e2.snatPort != e1.snatPort {
		t.Fatalf("repeat lookup allocated a new port: %d vs %d", e1.snatPort, e2.snatPort)
	}

	// Different flow gets a different port.
	key2, _ := IPKey(src, 51235, dst, 80)
	e3, _ := tbl.lookupOrAllocate(key2, now)
	if e3.snatPort != 40001 {
		t.Fatalf("second flow SNAT port: want 40001 got %d", e3.snatPort)
	}

	// Reverse lookup finds the first flow.
	rev, ok := tbl.Reverse(40000, now)
	if !ok || rev != e1 {
		t.Fatal("reverse lookup failed for allocated port")
	}
	if _, ok := tbl.Reverse(59999, now); ok {
		t.Fatal("reverse lookup for unallocated port must fail")
	}
}

func TestConnTablePoolExhaustion(t *testing.T) {
	tbl, err := NewConnTable(40000, 40001) // two ports only
	if err != nil {
		t.Fatalf("table: %v", err)
	}
	now := time.Now()
	src := net.ParseIP("10.77.0.2")
	dst := net.ParseIP("192.168.10.5")

	for i := 0; i < 2; i++ {
		key, _ := IPKey(src, 50000+i, dst, 80)
		if _, ok := tbl.lookupOrAllocate(key, now); !ok {
			t.Fatalf("allocation %d must succeed", i)
		}
	}
	key3, _ := IPKey(src, 50002, dst, 80)
	if _, ok := tbl.lookupOrAllocate(key3, now); ok {
		t.Fatal("third allocation must fail (pool exhausted)")
	}
}

func TestConnTableGCReturnsPorts(t *testing.T) {
	if _, err := NewConnTable(40000, 40000); err == nil {
		t.Fatal("single-port range must be rejected (needs base < max)")
	}
	tbl, err := NewConnTable(40000, 40001) // two ports: one live + one spare
	if err != nil {
		t.Fatalf("table: %v", err)
	}
	base := time.Now()
	src := net.ParseIP("10.77.0.2")
	dst := net.ParseIP("192.168.10.5")

	// Fill the two-port pool.
	key, _ := IPKey(src, 50000, dst, 80)
	if _, ok := tbl.lookupOrAllocate(key, base); !ok {
		t.Fatal("allocate failed")
	}
	keyB, _ := IPKey(src, 50001, dst, 80)
	if _, ok := tbl.lookupOrAllocate(keyB, base); !ok {
		t.Fatal("second allocate failed")
	}
	// Pool is exhausted: a third flow fails.
	keyC, _ := IPKey(src, 50002, dst, 80)
	if _, ok := tbl.lookupOrAllocate(keyC, base); ok {
		t.Fatal("third allocate must fail (pool exhausted)")
	}
	// GC after the idle window frees both ports.
	if n := tbl.GC(base.Add(6*time.Minute), 5*time.Minute); n != 2 {
		t.Fatalf("GC removed %d entries, want 2", n)
	}
	if _, ok := tbl.lookupOrAllocate(keyC, base.Add(6*time.Minute)); !ok {
		t.Fatal("allocation after GC must succeed")
	}
}

func TestConnTableGCKeepsActive(t *testing.T) {
	tbl, _ := NewConnTable(40000, 40010)
	base := time.Now()
	src := net.ParseIP("10.77.0.2")
	dst := net.ParseIP("192.168.10.5")
	key, _ := IPKey(src, 50000, dst, 80)
	e, _ := tbl.lookupOrAllocate(key, base)

	// Touch the entry within the idle window, then GC.
	e.lastActive = base.Add(4 * time.Minute)
	if n := tbl.GC(base.Add(6*time.Minute), 5*time.Minute); n != 0 {
		t.Fatalf("GC removed active entry: %d", n)
	}
	if tbl.Size() != 1 {
		t.Fatalf("active entry lost: size=%d", tbl.Size())
	}
}

func TestConnTableObserveAndRelease(t *testing.T) {
	tbl, _ := NewConnTable(40000, 40010)
	now := time.Now()
	src := net.ParseIP("10.77.0.2")
	dst := net.ParseIP("192.168.10.5")
	key, _ := IPKey(src, 50000, dst, 80)
	e, _ := tbl.lookupOrAllocate(key, now)

	if e.state != stateSynSent {
		t.Fatalf("initial state: want synSent got %v", e.state)
	}
	tbl.Observe(e, true, true, false, false, now) // SYN-ACK
	if e.state != stateEstablished {
		t.Fatalf("after SYN-ACK: want established got %v", e.state)
	}
	tbl.Observe(e, false, false, true, false, now) // FIN
	if e.state != stateFinWait {
		t.Fatalf("after FIN: want finWait got %v", e.state)
	}
	tbl.Observe(e, false, false, true, false, now) // second FIN
	if e.state != stateClosed {
		t.Fatalf("after second FIN: want closed got %v", e.state)
	}
	tbl.Release(e)
	if tbl.Size() != 0 {
		t.Fatalf("release failed: size=%d", tbl.Size())
	}
	if _, ok := tbl.Reverse(e.snatPort, now); ok {
		t.Fatal("released entry must not be reversible")
	}
}

func TestIPKeyRejectsIPv6(t *testing.T) {
	if _, err := IPKey(net.ParseIP("fd00::1"), 1, net.ParseIP("192.168.0.1"), 2); err == nil {
		t.Fatal("IPv6 source must be rejected")
	}
}
