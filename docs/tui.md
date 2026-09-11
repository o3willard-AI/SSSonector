# SSSonector TUI — Specification

Status: PROPOSED (implementation tracked in `docs/backlog/tui-and-installers.md`, item 1)
Created: 2026-09-11
Layout chosen: single-screen dashboard (Option A) extended with an instance
selector rail for multi-instance (server) hosts.

---

## 1. Purpose

A terminal UI for management, configuration, and monitoring of running
SSSonector daemons. It surfaces data that already exists — Prometheus
`/metrics`, `/healthz`, the effective YAML config, systemd unit state, and the
daemon's logs — in one screen, without re-implementing parsing or metric
collection (AGENTS.md invariant 1).

Non-goals: remote fleet management (TUI manages the host it runs on), a
second competing CLI (no new `main`; ships as `sssonector tui` subcommand),
aggregated cross-instance math beyond what the panels already show.

## 2. Binary shape

`sssonector tui` — a subcommand of the existing `cmd/daemon` binary. No new
main package, no ADR required. Rationale: the TUI is a thin view over the
config loader and monitor surface the daemon already owns; a separate binary
would duplicate module wiring for no isolation benefit. The TUI process never
holds daemon locks; it is a read-mostly client that only triggers systemd
actions and SIGHUP.

Invocations:

```
sssonector tui                      # discover instances, open dashboard
sssonector tui --instance <name>    # open with <name> pre-focused
sssonector tui --refresh 2s         # poll interval (default 1s, min 500ms)
```

## 3. Layout

```
┌─ SSSonector v2.2.0 ── SERVER mode ────────────── daemon: RUNNING (pid 8123) ─┐
│                                                                              │
│ INSTANCES (↑↓/PgUp/PgDn select · Enter focus · Tab: rail ⇄ panels)          │
│  ▸ inst-1  :9443  tun 10.77.0.1   peers 2/4   up 2h14m   ●                   │
│    inst-2  :9444  tun 10.77.0.9   peers 1/1   up 41m     ●                   │
│    inst-3  :9445  tun 10.77.1.1   peers 0/2   down 5m    ⚠                   │
│    inst-4  :9446  —                —          never      ✖                   │
│ ─────────────────────────── focused: inst-1 ───────────────────────────────  │
│ TUNNEL                        CERTIFICATE   (shared, shown once)             │
│  state      ● up              issuer    pa-fleet-11 CA                      │
│  peers      2/4               expires   2026-11-14 (64d)                    │
│  tun addr   10.77.0.1/24                                                    │
│                                                                             │
│ NAT                                          RATE LIMITER                   │
│  fwd pkts   1,204,551                        hits in    812                 │
│  ret pkts   1,190,003                        hits out   634                 │
│  dropped    12 (ACL 8, malformed 4)          rate       50.0 MB/s           │
│  flows      47 active                        burst      100 MB              │
│  accepts    63 total, 8 ACL-denied                                          │
│                                                                             │
│ LOG (tail, all instances, tagged) ────────────────────────────────────────  │
│ 14:02:11 [inst-1] INFO  tunnel rekeyed peer 192.168.100.51                  │
│ 14:01:58 [inst-3] WARN  listener :9445 conn refused x3                       │
├─────────────────────────────────────────────────────────────────────────────┤
│ [s]top [r]estart [R]eload act on FOCUSED instance · [c]onfig · [q]uit       │
└──────────────────────────────────────────────────────────────────────────────┘
```

### 3.1 Panels

| Panel | Per-instance or shared | Contents |
|---|---|---|
| INSTANCES rail | shared (one row per instance) | name, listen port, TUN address, peer count, state age, health dot |
| TUNNEL | per-instance | state, peers, TUN address/prefix, state age |
| CERTIFICATE | shared (daemon host's cert store) | issuer, expiry + days remaining, rotation policy |
| NAT | per-instance | forwarded/return/dropped (with ACL + malformed split), active flows, listener accepts |
| RATE LIMITER | per-instance | throttle hits in/out, effective rate, burst |
| LOG | shared, tagged | last N lines (default 12) across instances, `[inst]`-tagged |

### 3.2 Degenerate cases

- **Client host** (single instance): the rail collapses to one fixed row; the
  focused-instance banner is omitted. Same code path — client is just a
  server view with `len(instances) == 1`.
- **Daemon down (focused instance)**: the affected panels show
  `⚠ daemon unreachable (<errno text>)` — never stale numbers, never zeros.
  Panels are cleared, not grayed: stale data is forbidden (anti-mock posture).
- **Prometheus disabled** (`monitor.prometheus.enabled: false`): panels that
  have no source show `— (prometheus disabled in config)`; `/healthz`-sourced
  panels still render.
- **No instances found**: rail shows `no sssonector@* units on this host`;
  footer actions disabled.

## 4. Keybindings

| Key | Context | Action |
|---|---|---|
| `↑`/`↓`, `PgUp`/`PgDn` | rail focused | move instance selection |
| `Enter` | rail focused | focus highlighted instance |
| `Tab` | anywhere | toggle focus rail ⇄ panels |
| `s` | panels | stop focused instance (`systemctl stop sssonector@<name>`) |
| `r` | panels | restart focused instance |
| `R` | panels | reload focused instance (SIGHUP; see §6) |
| `c` | anywhere | open config view (§5) |
| `Enter` | config view | begin editing selected line |
| `v` | config view | validate draft against the config loader |
| `a` | config view | apply draft (validate → write → SIGHUP) |
| `Esc` | config view / confirm dialog | discard / cancel |
| `q` | anywhere | quit |

Destructive actions (`s`, `r`) require a confirm dialog. Actions on a stopped
instance adapt (`s` becomes disabled; rail start action available in a future
revision — out of scope for v1, documented here so the footer reserves no key
for it accidentally).

## 5. Config view

Full-screen overlay on the dashboard. Shows the **effective** config for the
focused instance — post-merge with defaults — with user-set lines rendered
bright and default-filled lines dim. Editable in place.

Apply pipeline (fail-closed, per backlog gate):

1. Take the draft text and run it through the existing config loader
   (`internal/config`), which validates against schema v2.0.0.
2. On error: show the loader's message with line number; the running daemon
   is untouched (nothing was written, nothing was signaled).
3. On success: write the instance's config file (atomic rename), then
   `SIGHUP` the focused instance's PID (from the systemd unit).
4. Reload result is surfaced: success banner with new config generation, or
   the daemon's own reload error from the log tail.

The TUI never constructs config from partial state — the draft always starts
from a full effective-config dump, so an edit can only change what the user
changed.

## 6. Data sources

| Panel data | Source | Poll |
|---|---|---|
| Instance list + unit state | `systemctl list-units 'sssonector@*'` + `show` (substate, pid, ActiveState) | 2s |
| Mode, tunnel state, uptime | `GET /healthz` on the instance's Prometheus port | refresh interval |
| Bytes/packets/errors/connections | `sssonector_bytes_*`, `sssonector_packets_*`, `sssonector_errors_total`, `sssonector_connections_*` | refresh interval |
| Throttle | `sssonector_throttle_hits_total{direction}`, `sssonector_throttle_effective_rate_bytes_per_second`, `sssonector_throttle_burst_bytes` | refresh interval |
| NAT | `sssonector_nat_forwarded_packets_total`, `..._return_packets_total`, `..._dropped_packets_total`, `..._flows_active`, `..._listener_accepts_total`, `..._acl_denied_total` | refresh interval |
| Cert | cert files from the instance's effective config (`tls.cert_file`, `ca_file`) parsed via the `internal/cert` package loaders — local read, no daemon endpoint. Shared panel shows the host CA + the focused instance's leaf. | 60s |
| Logs | `journalctl -u sssonector@<name> -n <N> --no-pager -o short-iso`, tailed incrementally | 1s |

Notes:

- Each instance's Prometheus port comes from its effective config
  (`monitor.prometheus.port`); there is no global registry — the instance rail
  is derived from systemd units, which is the authority on multi-instance
  hosts (see `docs/multi_instance_deployment.md`).
- `/metrics` is parsed from the plain exposition format the daemon already
  renders (`internal/monitor` `handleMetrics`); the parser is a small,
  tested text parser, NOT the prometheus client library — zero new
  dependencies.
- The TUI reuses `internal/config` for loading/validating and the cert
  loaders for expiry. It does not import `internal/monitor` for collection
  (that runs inside the daemon); it reads the same numbers over HTTP so the
  TUI works against daemons it does not own.

## 7. Failure modes

| Failure | Behavior |
|---|---|
| Daemon unreachable (connection refused / timeout) | Panels show explicit `⚠ daemon unreachable (…)`. Rail dot `✖`. Actions other than stop/start are disabled for that instance. |
| Prometheus endpoint disabled | Source-specific `— (prometheus disabled)` placeholders; healthz panels unaffected. |
| Config file unreadable / schema invalid | Config view shows the loader error verbatim; dashboard still runs (monitoring is read-only against /healthz + /metrics). |
| systemctl fails (no systemd, non-root) | Instance discovery degrades to a single implicit instance only if exactly one config exists under `/etc/sssonector/`; otherwise explicit error. No fabricated instance list. |
| SIGHUP reload rejected by daemon | Error from the daemon's log shown in the log tail + banner; config file is left as written (documented: rollback is manual, matching today's SIGHUP semantics). |
| Cert file missing/unparseable | Cert panel shows `⚠ cert unreadable: <err>`; never guessed. |

## 8. Implementation notes

- Go, terminal library to be chosen at implementation time (bubbletea is the
  default candidate; justification required in the PR if a different library
  is chosen). One new dependency, pinned, govulncheck-clean (AGENTS.md).
- New package `internal/tui` (views + keybindings) and `internal/tui/collect`
  (systemd/Prometheus/journalctl collectors with interfaces, so all parsing
  logic is unit-testable against fixtures).
- Tests: exposition-format parser (table-driven, covers comments, labels,
  NaN, missing metrics), config-draft diff/merge, systemd unit discovery
  (fixture-based), panel formatting. No sleeps-based tests (AGENTS.md).
- `go build ./...`, `go vet ./...`, `go test -race ./...` must stay green.
- The TUI must not mutate any state except: config files (via the apply
  pipeline), systemd unit actions, and SIGHUP. No new listening sockets.

## 9. Acceptance criteria (mirrors backlog item 1 [DONE] gate)

1. This spec exists and matches the shipped implementation (any drift is a
   bug in the implementation PR).
2. Builds and runs against a live daemon on the QA rig (`pa-fleet-11`),
   showing real tunnel + NAT + cert state for at least two server instances.
3. Config view shows effective config; an invalid edit is rejected without
   breaking the running daemon; a valid edit reloads via SIGHUP.
4. `go vet` + `go test -race ./...` green; parser/collector tests present.
5. Commit SHA + demo transcript (or screenshot) of the TUI against the rig.
