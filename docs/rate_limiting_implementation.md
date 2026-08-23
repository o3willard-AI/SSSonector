# Rate Limiting Implementation Notes

This document describes how rate limiting actually works in SSSonector
today. For configuration guidance see
[Rate Limiting Configuration](config/rate_limiting.md).

## Architecture

The limiter lives in `internal/throttle` and is a custom
reservation/"debt" token bucket — not `golang.org/x/time/rate`.

- Each direction (read and write) gets its own bucket.
- Effective rate: `configured_rate * 1.1` (`TCPOverheadFactor`, see
  `constants.go`) to compensate for TCP/IP headers and retransmissions.
- Burst is always 100ms of effective rate; buckets start empty so pacing
  begins with the first byte.
- Oversized requests are funded across multiple intervals instead of being
  rejected.
- A wait that exceeds the timeout aborts with an error rather than blocking
  forever.

## Concurrency Rules

- No sleeps are ever performed while holding a lock; waiting happens via
  channel-based reservation funding so one connection cannot
  head-of-line-block others.
- `Limiter.Update` swaps rate/burst atomically under the bucket mutex,
  which is what makes SIGHUP hot reload safe against in-flight transfers.

## Data Path Integration

`Transfer` (`internal/tunnel/transfer.go`) wraps each direction's
reader/writer in a `throttle.Limiter`; `io.Copy` pumps bytes through the
limiter. When throttling is disabled (`throttle.enabled: false`) the
limiter passes through untouched.

## Metrics

- `LimiterMetrics` tracks effective rate, burst, and `LimitHits`
  (requests that had to wait).
- Byte counters on the tunnel connections feed the monitor sampler and are
  exposed via Prometheus (`sssonector_bytes_*`) and SNMP
  (`.1.3.6.1.4.1.54321.1.1/.1.2`).

## Testing Contract

Per AGENTS.md, rate/QoS claims require timing assertions backed by
deterministic math: `limiter_test.go` proves pacing (e.g., 4096 bytes at
2 KB/s takes ~1.82s) and mutation tests guard the timeout semantics.
