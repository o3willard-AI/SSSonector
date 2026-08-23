package tunnel

import (
	"math"
	"math/rand"
	"time"
)

const (
	defaultReconnectAttempts = 10
	defaultReconnectInitial  = time.Second
	defaultReconnectMax      = 30 * time.Second
	defaultReconnectJitter   = 0.2

	backoffMultiplier = 2.0
	maxJitterFraction = 0.9
)

// computeBackoff returns the wait before the given 1-based retry attempt.
// The delay doubles from base and is capped at max; the jitter fraction
// (0..0.9) then reduces it by up to that fraction ("decaying" jitter), so
// max stays a hard ceiling while a restarting fleet still desynchronizes.
func computeBackoff(attempt int, base, max time.Duration, jitter float64, rng *rand.Rand) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if base <= 0 {
		base = defaultReconnectInitial
	}
	if max < base {
		max = base
	}
	if jitter < 0 {
		jitter = 0
	} else if jitter > maxJitterFraction {
		jitter = maxJitterFraction
	}

	d := float64(base) * math.Pow(backoffMultiplier, float64(attempt-1))
	if d > float64(max) {
		d = float64(max)
	}
	if jitter > 0 && rng != nil {
		d -= d * jitter * rng.Float64()
	}
	return time.Duration(d)
}
