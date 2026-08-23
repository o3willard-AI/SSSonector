package tunnel

import (
	"math/rand"
	"testing"
	"time"

	"github.com/o3willard-AI/SSSonector/internal/config"
)

func TestComputeBackoffGrowthAndCap(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	base := 100 * time.Millisecond
	max := 2 * time.Second

	want := []time.Duration{
		100 * time.Millisecond, // attempt 1
		200 * time.Millisecond,
		400 * time.Millisecond,
		800 * time.Millisecond,
		1600 * time.Millisecond,
		2 * time.Second, // capped from here on
		2 * time.Second,
	}

	for i, w := range want {
		got := computeBackoff(i+1, base, max, 0, rng)
		if got != w {
			t.Errorf("attempt %d: got %v, want %v", i+1, got, w)
		}
	}
}

func TestComputeBackoffJitterBounds(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	base := time.Second
	max := 10 * time.Second
	jitter := 0.25

	for attempt := 1; attempt <= 8; attempt++ {
		uncapped := computeBackoff(attempt, base, max, 0, rng)
		lower := time.Duration(float64(uncapped) * (1 - jitter))
		for i := 0; i < 200; i++ {
			got := computeBackoff(attempt, base, max, jitter, rng)
			if got > uncapped || got < lower {
				t.Fatalf("attempt %d: %v outside [%v, %v]", attempt, got, lower, uncapped)
			}
		}
	}
}

func TestComputeBackoffDefensiveInputs(t *testing.T) {
	rng := rand.New(rand.NewSource(7))

	if got := computeBackoff(0, time.Second, time.Minute, 0, rng); got != time.Second {
		t.Errorf("attempt 0 should clamp to 1: got %v", got)
	}
	// zero base defaults to 1s before growth: attempt 3 -> 4s
	if got := computeBackoff(3, 0, time.Minute, 0, rng); got != 4*time.Second {
		t.Errorf("zero base should default then grow: got %v", got)
	}
	// max below base clamps UP to base so pacing never drops below the floor
	if got := computeBackoff(5, time.Second, 500*time.Millisecond, 0, rng); got != time.Second {
		t.Errorf("max below base should clamp to base: got %v", got)
	}
	// absurd jitter clamps to the max fraction: result within [10%, 100%]
	got := computeBackoff(1, time.Second, time.Minute, 5, rng)
	if got < 100*time.Millisecond || got > time.Second {
		t.Errorf("clamped jitter should stay in [0.1s, 1s]: got %v", got)
	}
}

func TestReconnectConfigNormalized(t *testing.T) {
	zeros := config.ReconnectConfig{}.Normalized()
	if zeros.MaxAttempts != defaultReconnectAttempts ||
		zeros.InitialDelay != defaultReconnectInitial ||
		zeros.MaxDelay != defaultReconnectMax ||
		zeros.Jitter != defaultReconnectJitter {
		t.Errorf("defaults not applied: %+v", zeros)
	}

	custom := config.ReconnectConfig{MaxAttempts: 3, InitialDelay: 50 * time.Millisecond}.Normalized()
	if custom.MaxAttempts != 3 || custom.InitialDelay != 50*time.Millisecond {
		t.Errorf("explicit values must survive normalization: %+v", custom)
	}
}
