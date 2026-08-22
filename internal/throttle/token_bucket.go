package throttle

import (
	"errors"
	"sync"
	"time"
)

// errRateLimitTimeout is returned when a token acquisition cannot complete
// within defaultTimeout.
var errRateLimitTimeout = errors.New("rate limit wait timeout")

// tokenBucket implements a refilling token-bucket byte counter using a
// reservation ("debt") model:
//
//   - Tokens accrue continuously at `rate` tokens/second while idle,
//     capped at `burst` tokens.
//   - An acquisition larger than the currently available tokens reserves
//     them immediately (driving the balance negative) and returns the
//     delay after which the reservation becomes funded. Accrual pays off
//     debt linearly; the burst cap only limits idle accumulation.
//   - The bucket starts empty, so the first transfer is paced immediately
//     instead of receiving an instant full-burst loan.
//
// acquire never sleeps while holding the lock, so concurrent waiters are
// never head-of-line blocked behind another waiter's nap.
type tokenBucket struct {
	mu         sync.Mutex
	rate       float64   // tokens per second
	burst      float64   // maximum stored tokens when idle
	tokens     float64   // current balance (may be negative while funding a reservation)
	lastUpdate time.Time // last refill timestamp
}

func newTokenBucket(rate, burst float64) *tokenBucket {
	return &tokenBucket{
		rate:       rate,
		burst:      burst,
		tokens:     0,
		lastUpdate: time.Now(),
	}
}

// Update changes the rate and burst cap, clamping stored tokens.
func (b *tokenBucket) Update(rate, burst float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.refillLocked(time.Now())
	b.rate = rate
	b.burst = burst
	if b.tokens > burst {
		b.tokens = burst
	}
}

// acquire blocks until n tokens have been reserved and funded.
// It reports whether any waiting was required (i.e. the request was
// initially rate-limited). It returns errRateLimitTimeout if the request
// could not be satisfied within defaultTimeout; failed reservations are
// rolled back.
func (b *tokenBucket) acquire(n float64) (waited bool, err error) {
	if b.rate <= 0 {
		return false, nil // unlimited
	}

	b.mu.Lock()
	now := time.Now()
	b.refillLocked(now)

	if b.tokens >= n {
		b.tokens -= n
		b.mu.Unlock()
		return false, nil
	}

	// Reserve now, fund later: drive the balance negative and compute the
	// time required for accrual to bring it back to zero.
	deficit := n - b.tokens
	b.tokens -= n
	delay := time.Duration(deficit / b.rate * float64(time.Second))
	b.mu.Unlock()

	if delay > defaultTimeout {
		time.Sleep(defaultTimeout)
		b.cancel(n)
		return true, errRateLimitTimeout
	}

	if delay > 0 {
		time.Sleep(delay)
	}
	return true, nil
}

// cancel rolls back a previously made reservation of n tokens.
func (b *tokenBucket) cancel(n float64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.refillLocked(time.Now())
	b.tokens += n
	if b.tokens > b.burst {
		b.tokens = b.burst
	}
}

// refillLocked tops up tokens based on elapsed time; caller holds b.mu.
func (b *tokenBucket) refillLocked(now time.Time) {
	elapsed := now.Sub(b.lastUpdate).Seconds()
	if elapsed <= 0 {
		return
	}
	b.tokens += elapsed * b.rate
	if b.tokens > b.burst {
		b.tokens = b.burst
	}
	b.lastUpdate = now
}
