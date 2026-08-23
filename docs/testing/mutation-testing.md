# Mutation Testing — Methodology & Results

Status: 2026-08-22 campaign on auth-critical and rate-paths.

## Scope & method

Targets: `internal/facade/token.go` (facade auth), `internal/throttle/{token_bucket,limiter}.go` (QoS enforcement).

Each mutant is a single deterministic semantic edit applied via regex, after which the package's test suite must **fail** for the mutant to count as *killed*. Suites that pass under a mutated implementation are recording a behavior gap.

## Results

| Metric | Value |
|---|---|
| Mutants | 16 |
| Killed | 13 |
| Survived | 3 (all documented equivalents, below) |
| **Score (excl. equivalents)** | **100%** |

Gate: ≥70% on critical paths (AGENTS.md). Re-run this campaign whenever these files change materially; extend the mutant set when adding semantics.

## Documented equivalent mutants

Equivalence is argued against the public API contract; chasing them would require clock injection or flaky nanosecond timing, which would weaken the suite rather than strengthen it.

1. `ttl-cmp-invert` (`elapsed > ttl` → `>=`): distinguishable only at the exact instant `elapsed == ttl`; wall-clock validation latency makes the boundary unobservable deterministically.
2. `update-clamp-off` (`Update` skips clamping tokens to new burst): every read path refills first, and refill caps at burst — the invariant self-corrects before any observation.
3. `elapsed-guard-off` (`refillLocked` drops the negative-elapsed guard): Go's monotonic clock makes negative elapsed unreachable in-process.

## Notable kills (regression protection bought by current tests)

- Skipping the HMAC compare, widening port bounds, halving timestamps → facade conformance rejects.
- Doubling refill rate / halving funding delay / skipping debits → sequential-wait timing floors (`TestTokenBucketSequentialWaits*`, `TestFastPathDebitsTokens`, `TestRefillAccrualMatchesRate`) catch pacing fraud in both directions (too fast **and** too slow).
- Timeout threshold ×10 → `TestTimeoutThresholdBoundary` proves failure surfaces at `defaultTimeout`, not after slow-paced success.
- Disabling Read/Write pacing on the limiter itself → `TestLimiterReadPathPacesThroughReader/Writer` guard the production transfer path, not just the wrapper types.
