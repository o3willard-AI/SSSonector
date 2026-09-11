# SSSonector TUI — Implementation Plan

Status: PROPOSED — work plan for implementing `docs/tui.md` (spec, PR #12).
Audience: implementation agent, code reviewer, and agentic release engineer
working together to land this capability.
Created: 2026-09-11
Branch convention: one branch per phase (`feat/tui-p<N>-<slug>`); each work
item (WI) is one or more commits, each independently reviewable and testable.

---

## 0. Orchestration rules

- **Phase order is a dependency chain.** A phase starts only when every WI in
  the prior phase is merged to `main` and its verification gate passed.
- **Reviewer gate per WI.** Every WI lands as its own PR (or stacked PRs) with
  the tests named in the WI. The reviewer verifies: tests fail if the
  implementation is reverted (mutation check for critical paths), no
  convention drift (AGENTS.md), no new deps beyond the declared allow-list
  (§9), `go build ./...`, `go vet ./...`, `go test -race -count=1 ./...` green.
- **Release engineer gate per phase.** After each phase merge, the release
  engineer runs the phase's release-gate command block (§9) on the QA rig and
  records the result (build SHA, test log, rig host) in the phase's PR.
- **Definition of done for the whole plan**: backlog item 1's [DONE] gate
  (`docs/backlog/tui-and-installers.md`) is satisfied and the backlog file is
  updated in the final PR of Phase 5.
- **Any spec/plan drift found during implementation** is fixed by updating
  `docs/tui.md` / this file in the same PR as the code that drifted — docs and
  behavior change together (AGENTS.md).

## 1. Phases and work items

### Phase 0 — Foundations (no behavior change; mergeable alone)

| WI | Deliverable | Tests (implementer + reviewer verify) | Gate |
|---|---|---|---|
| 0.1 | `internal/tui` package skeleton: `go.mod` unchanged, empty views/collect subpackages compile; CI untouched | `go build ./...` green; `go vet` silent; package imported nowhere yet | pkg compiles |
| 0.2 | Exposition-format parser `internal/tui/collect/promparse.go`: parses the exact format `internal/monitor.handleMetrics` renders — comments, `# HELP/# TYPE`, labeled and unlabeled series, integers and floats, missing metrics absent (not zero) | Table-driven `promparse_test.go` with fixtures copied from a live `/metrics` capture; cases: label parse, NaN, malformed line (error, not panic), unknown metric ignored | all fixtures parse; malformed input returns error |
| 0.3 | Metrics snapshot model `internal/tui/collect/snapshot.go`: typed struct mapping parsed series → dashboard fields (NAT six, throttle three, connections, errors, bytes/packets) | Round-trip test: render fixture → parse → assert typed fields; missing-series cases yield explicit `absent` state, never 0 | typed snapshot correct for every fixture |
| 0.4 | `/healthz` client: GET + JSON decode of `{status,mode,tunnel_state,uptime_seconds}` with timeout | Fixture test (httptest server): valid, malformed JSON, timeout → explicit error | decode + error paths covered |

**Phase 0 release gate:** `go test -race ./internal/tui/...` green in CI; no
user-visible change; reviewer confirms zero new deps and zero new imports in
production code outside `internal/tui`.

### Phase 1 — Collectors (all data the dashboard shows, headless)

| WI | Deliverable | Tests | Gate |
|---|---|---|---|
| 1.1 | `systemd` collector: discover `sssonector@*` units via `systemctl list-units --plain`; per-unit `show` → ActiveState, SubState, MainPID; degradation rule (§7 of spec: no systemd + exactly one config file → implicit single instance; else error) | Fixture-driven: fake `systemctl` output via interface injection; no-systemd degradation case; ambiguous multi-config-no-systemd error case | discovery matches fixtures; error cases fail closed |
| 1.2 | Per-instance config resolution: locate `/etc/sssonector/instances/<n>/config.yaml` (fallback `/etc/sssonector/config.yaml`), load via `internal/config` (reuse — no re-implementation), extract monitor.prometheus.port/path, tls cert paths, mode | Table test with fixture configs: valid, missing file, invalid schema (loader error surfaced verbatim) | effective values correct; invalid config is a surfaced error |
| 1.3 | Poller: 1s tick assembling `Snapshot{rail, per-instance healthz+metrics, cert}` with per-source error propagation (each panel gets ok/error/absent, never stale reuse) | Fake-clock test (no sleeps): tick N returns error → snapshot marks panels errored; recovery on tick N+1 clears them | error state transitions verified |
| 1.4 | Cert expiry reader: load `tls.cert_file`/`ca_file` via `internal/cert` loaders, report issuer/notAfter/days | Fixture certs: valid (future), expiring (<30d flag), unparseable (error) | expiry math correct; errors surfaced |
| 1.5 | Log tailer: incremental `journalctl -u sssonector@<n> --no-pager -o short-iso -n N` + follow via cursor; per-line instance tag | Fake-journalctl fixture; cursor resume test; rotation (journalctl restart) handled by re-seek | tail is incremental, not full re-read |
| 1.6 | Headless CLI probe `sssonector tui --probe` (diagnostic submode of the tui command): prints assembled snapshot as text, exits 0/1 | Integration test against a fixture daemon (httptest serving captured /metrics + /healthz); rig test in Phase gate | snapshot text matches rig truth |

**Phase 1 release gate:** on QA rig `pa-fleet-11`, `sssonector tui --probe`
against a live server instance shows real NAT/throttle/healthz numbers matching
`curl /metrics` output; against a stopped daemon it prints explicit
unreachable errors and exits 1. Recorded in the phase PR.

### Phase 2 — Dashboard rendering (read-only TUI)

| WI | Deliverable | Tests | Gate |
|---|---|---|---|
| 2.1 | Terminal lib decision recorded (bubbletea candidate) in this file via PR; dep pinned, govulncheck clean, justified in PR body | dependency review | dep allow-list updated (§9) |
| 2.2 | Panel renderers (pure functions over `Snapshot`): rail, tunnel, cert, NAT, rate, log tail; fixed-width golden-output tests incl. 80-col narrow and 200-col wide | Golden tests per panel per state: ok / errored / absent / empty | rendering deterministic across states |
| 2.3 | Server-mode screen assembly (rail + panels + footer) wired to Poller; client mode = degenerate 1-instance view | Integration test: fake daemon fixture → assembled screen matches golden | full-screen golden test |
| 2.4 | Navigation + read-only keys: rail ↑↓/Enter focus, Tab, `c`, `q` (no destructive actions yet — footer hides them) | Key-event unit tests via the terminal lib's test harness | focus transitions correct |
| 2.5 | `sssonector tui` subcommand wired into `cmd/daemon` main flag parsing (no second main; `tui` mode runs views+collect only) | `go build` + smoke: `sssonector tui --help`; binary behaves as before for all pre-existing flags (regression: existing flag tests unchanged) | subcommand lives, no flag regressions |

**Phase 2 release gate:** TUI run on rig (ssh + tmux) against `pa-fleet-11`
shows real tunnel/NAT/cert/log state; screenshot/transcript attached to PR;
`go test -race ./...` green.

### Phase 3 — Config view (effective config, validate, apply)

| WI | Deliverable | Tests | Gate |
|---|---|---|---|
| 3.1 | Effective-config dump: render merged config (schema defaults + instance file) with user-set vs default-origin marking | Merge fixture: defaults file + instance file → dump marks each line's origin; drift check vs `internal/config` defaults table | origin marking correct |
| 3.2 | Draft edit + validate: full-dump → user edits → `internal/config` Load (same loader as daemon) → error verbatim w/ line, or ok | Table: valid edit, syntax error, semantic error (subnet overlap), no-op draft | invalid draft never leaves the view |
| 3.3 | Apply pipeline: atomic write (tmp+rename, perms 0600) → SIGHUP to unit PID → reload outcome from log/healthz generation | Fake-daemon integration: reload ok; reload rejected (daemon logs rejection, config file left written — documented semantics); write failure → no signal sent | signal only after successful write; failure never kills daemon |
| 3.4 | Config overlay screen + keys (`Enter/v/a/Esc`) wired into dashboard | Key-test harness: open, edit, validate-fail banner, validate-ok, apply, esc | full config-flow golden tests |

**Phase 3 release gate:** on rig: view effective config (matches on-disk file
post-merge); invalid edit rejected, daemon untouched (`systemctl status`
unchanged, healthz continuous); valid edit reloads (log shows reload ok).
Transcript in PR.

### Phase 4 — Lifecycle actions + multi-instance server surface

| WI | Deliverable | Tests | Gate |
|---|---|---|---|
| 4.1 | Confirm dialog + `s`/`r` actions (`systemctl stop/restart sssonector@<n>`); `R` = SIGHUP via §5 pipeline reuse | Unit tests with injected runner (no real systemctl in tests); confirm-cancel path | destructive only after confirm; correct unit name always |
| 4.2 | Multi-instance rail: N units sorted, per-row health dot from snapshot; focus switch re-renders per-instance panels; shared cert panel invariant | Golden tests: 1, 2, 4 instances; mixed up/down/never states | rail reflects unit truth |
| 4.3 | Log tailing for all instances with `[inst]` tagging + interleaved ordering | Fixture: two unit streams interleaved → stable time-ordered merge | ordering stable, tags correct |
| 4.4 | Degradation matrix implemented per spec §7 (daemon down, prometheus disabled, no systemd, cert unreadable) — one golden test per cell | Golden tests × states | every §7 row has a passing test |

**Phase 4 release gate:** rig run with ≥2 server instances (e.g. `client-a`,
`client-b`): focus switching shows per-instance truth, stop/restart via TUI
reflected by systemctl, one instance down renders its panels as unreachable
while the other stays live. Transcript in PR.

### Phase 5 — First-run wizard + bundle hand-off

| WI | Deliverable | Tests | Gate |
|---|---|---|---|
| 5.1 | First-run detection (no units AND no valid config → wizard) + `--from-bundle` flag parsing | Detection matrix tests: fresh host, existing config, units-only, bundle-flag | correct mode chosen per matrix row |
| 5.2 | Server wizard form (mode radio, instance, port, TUN, NAT checkbox, cert choice) with live validation (schema via loader, port via `ss -tlnp` check, TUN overlap) — mode never defaulted | Field-validation table tests incl. the never-default-to-server rule and certs fail-closed rule | wizard blocks invalid/incomplete |
| 5.3 | Wizard write path: instance config → enable+start unit → land on dashboard (reuses Phase 3 pipeline; only enable/start added) | Integration test with fake runner: file written before unit enabled; failure rollback documented | atomic-ish ordering verified |
| 5.4 | Client bundle generation (`g`): CA cert + pre-filled client config skeleton → tarball + SHA256 printed | Golden: bundle contents (certs match instance CA, skeleton fields match server config); SHA256 verified | bundle reproducible & correct |
| 5.5 | Client wizard + bundle prefill + CA-verifies-chain pre-flight (chain check against loaded CA blocks Enter on mismatch) | Table: chain ok, chain mismatch blocked, bundle missing files → error | pre-flight catches mismatch before daemon start |
| 5.6 | Establishment walkthrough end-to-end on rig (spec §3.3): server wizard → bundle → client wizard (second rig host or second instance) → both dashboards converge | Rig transcript + screenshots per §9 acceptance item 5 | [DONE] gate criteria 2–3 demonstrated |

**Phase 5 release gate = backlog [DONE] gate:** live demo against
`pa-fleet-11` with ≥2 instances, commit SHAs, transcripts; then the final PR
updates `docs/backlog/tui-and-installers.md` item 1 to DONE and removes the
item per the backlog's own convention.

## 2. Dependency graph (WI level)

```
P0: 0.1 → 0.2 → 0.3 → 0.4
P1: 1.1,1.2 → 1.3 → 1.4,1.5 → 1.6        (uses P0 parser/snapshot)
P2: 2.1 → 2.2 → 2.3 → 2.4,2.5            (uses P1 Poller/Snapshot)
P3: 3.1 → 3.2 → 3.3 → 3.4                (uses P2 view framework)
P4: 4.1,4.2 → 4.3,4.4                    (uses P2/P3)
P5: 5.1 → 5.2 → 5.3 → 5.4,5.5 → 5.6      (uses P3 pipeline + P4 rail)
```

Parallelization: within a phase, WIs on the same line of the graph can run in
parallel on separate branches; cross-phase parallelism is allowed only where
the graph shows no edge (e.g. 3.1 can start once 2.2 merges, before 2.5).

## 3. Review checklist (reviewer runs per WI)

1. Diff scope = the WI only; no drive-by changes.
2. Named tests exist, are fixture/golden based (no sleeps, no live-network
   flakiness), and fail when the WI's implementation is reverted.
3. `internal/tui` imports only: stdlib, the Phase-2-approved terminal lib,
   `internal/config`, `internal/cert`. No imports of `internal/monitor`
   collection internals, no prometheus client library.
4. AGENTS.md invariants: single entry point intact; no secrets/keys in the
   repo (wizard/bundle tests use `_test.go`-generated certs only — Gitleaks
   clean); fail-closed behavior preserved in every error path.
5. Spec/plan docs updated in the same PR if behavior drifted.

## 4. Release-engineer runbook (per phase + final)

- Per phase: after merge, `git checkout main && go build ./... && go vet ./... && go test -race -count=1 ./...`; deploy fresh binary to rig; run the phase gate commands above; record results in the phase PR before marking it merged-complete.
- Rig bootstrap: build `sssonector-linux-amd64` per AGENTS.md build block; scp to `pa-fleet-11` (192.168.100.51) and client rig .171; run under tmux for transcript capture.
- Final: tag/PR containing the backlog DONE update; verify CI green on `main` post-merge; nothing in the TUI adds listening sockets (verify with `ss -tlnp` diff on rig pre/post TUI run).

## 5. Dependency allow-list (reviewer-enforced)

| Package | Purpose | Added in |
|---|---|---|
| stdlib | everything else | — |
| one TUI lib (bubbletea or equivalent, decided in 2.1) | rendering + key input | Phase 2 |
| `golang.org/x/term` (if the TUI lib does not already provide size/raw-mode) | terminal setup | Phase 2, only if needed |

No other new modules. Anything else requires an ADR-level justification.

## 6. Risk register

| Risk | Mitigation |
|---|---|
| Exposition parser drift vs future `handleMetrics` changes | parser fixtures generated from live endpoint in CI integration test; `internal/monitor` tests assert field names the parser depends on (contract test, Phase 0 WI 0.2) |
| systemd-less environments (macOS/Windows builds) | collectors behind interfaces; platform build tags; non-Linux builds exclude systemd collector (compile-time), wizard reports explicit unsupported error — fail closed |
| TUI freezes the daemon via shared imports | TUI is a separate process; only shares `internal/config` (pure loading) — verified by import audit in review checklist |
| Wizard loosening NAT security | NAT checkbox writes default-deny config only; ACL edits post-setup only (spec §3.3); reviewer verifies no ACL-widening code path exists in wizard |