# Hot Reload Implementation Design — SUPERSEDED

> **Status: superseded.** This document described an aspirational design
> (file watchers, `ConfigWatcher` registries, dynamic per-connection
> settings) that was never implemented as written. It has been retired to
> avoid drift.

The actual, current hot-reload behavior is:

- **Trigger**: `SIGHUP` only (no file watching).
- **Pipeline**: fresh load from disk → full schema validation → apply
  reloadable subset → warn on restart-required differences.
- **Reloadable**: throttle enable/rate (live transfers included), log
  level (atomic), certificate rotation interval.
- **Implementation sites**:
  - Signal handling and orchestration: `cmd/daemon/main.go`
  - Apply logic and restart-required warnings:
    `internal/tunnel/tunnel.go` (`ApplyConfig`, `applyRuntimeSettings`)
  - Live limiter updates: `internal/throttle/limiter.go` (`Update`)
  - Certificate tunables: `internal/cert/manager.go`
    (`SetCheckInterval`, timer re-reads the interval each cycle)

Operator-facing documentation lives in
[Hot Reload Configuration](../hot_reload.md).
