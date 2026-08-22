package throttle

import (
	"testing"
	"time"
)

// TestFastPathDebitsTokens proves an immediately-admitted acquisition
// consumes tokens; otherwise buckets become free-after-first-fill.
func TestFastPathDebitsTokens(t *testing.T) {
	b := newTokenBucket(10000, 100000)
	b.tokens = 500 // seeded balance

	waited, err := b.acquire(300)
	if err != nil || waited {
		t.Fatalf("first acquire should be immediate: waited=%v err=%v", waited, err)
	}
	// A few tokens may accrue between seeding and the debit; the invariant
	// under test is that the debit actually happened (500-300=200 base).
	if got := b.tokens; got < 199 || got > 210 {
		t.Fatalf("tokens after debit = %v, want ~200", got)
	}

	// Second acquisition must now take the funded-debt path (200 < 400).
	start := time.Now()
	waited, err = b.acquire(400)
	if err != nil {
		t.Fatalf("second acquire errored: %v", err)
	}
	if !waited {
		t.Fatal("second acquire should have required waiting (tokens were debited)")
	}
	if el := time.Since(start); el > time.Second {
		t.Fatalf("funding wait took too long for tiny deficit: %v", el)
	}
}

// TestRefillAccrualMatchesRate pins refill math: one second of elapsed time
// at rate R must credit exactly R tokens (capped at burst).
func TestRefillAccrualMatchesRate(t *testing.T) {
	b := newTokenBucket(1000, 100000)
	b.lastUpdate = time.Now().Add(-time.Second)

	b.Update(1000, 100000) // Update performs a refill

	if b.tokens < 900 || b.tokens > 1100 {
		t.Fatalf("tokens after 1s at rate 1000 = %v, want ~1000", b.tokens)
	}
}

// TestRefillNeverExceedsBurst verifies idle accrual is capped.
func TestRefillNeverExceedsBurst(t *testing.T) {
	b := newTokenBucket(10_000_000, 250)
	b.lastUpdate = time.Now().Add(-time.Hour)

	b.Update(10_000_000, 250)

	if b.tokens != 250 {
		t.Fatalf("tokens = %v, want capped at burst 250", b.tokens)
	}
}

// TestAcquireImmediateAtExactBalance pins the inclusive fast-path boundary:
// tokens == request size admits instantly with no waiting.
func TestAcquireImmediateAtExactBalance(t *testing.T) {
	b := newTokenBucket(1, 1000)
	b.tokens = 750

	start := time.Now()
	waited, err := b.acquire(750)
	if err != nil || waited {
		t.Fatalf("exact-balance acquire should be immediate: waited=%v err=%v", waited, err)
	}
	if el := time.Since(start); el > 50*time.Millisecond {
		t.Fatalf("exact-balance acquire slept: %v", el)
	}
}

// TestFailedReservationRolledBack proves a timed-out acquisition does not
// leave phantom debt behind that would starve later small requests.
func TestFailedReservationRolledBack(t *testing.T) {
	b := newTokenBucket(1, 10) // 1 tok/s: big asks always time out

	if _, err := b.acquire(1_000_000); err == nil {
		t.Fatal("expected timeout error")
	}

	// Balance must be restored to <= burst, not left deeply negative.
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.tokens < 0 {
		t.Fatalf("balance left negative after rollback: %v", b.tokens)
	}
	if b.tokens > b.burst {
		t.Fatalf("balance exceeds burst after rollback: %v", b.tokens)
	}
}
