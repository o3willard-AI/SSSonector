# SSSonector Backlog

Queued feature work for the implementation agent. Each item is scoped, self-contained, and carries a [DONE] gate. Created 2026-09-09.

---

## 1. Terminal UI (TUI)

Spec, architect, and build a terminal UI for management, configuration, and monitoring of a running SSSonector daemon.

### Context (what exists today)

SSSonector is a Go daemon (`cmd/daemon`) with:

- **CLI**: `-version`, `-help`, `provision create|apply|verify`, `-config`
- **Config**: YAML at `/etc/sssonector/config.yaml` (schema v2.0.0), validated on load and reload
- **Monitoring**: Prometheus `/metrics` (tunnel + NAT counters, rate limiter, cert status), SNMP agent (v2c), and a separate Flask web monitor (SNMP visualization)
- **Lifecycle**: systemd unit (linux), SIGHUP hot-reload

There is no native terminal UI today. Management is CLI + config file + web/SNMP.

### Scope (what to build)

A TUI that runs locally and reads the daemon's existing monitoring and config surfaces, covering:

1. **Management** — daemon status (running/stopped), start/stop/restart/reload, provision status
2. **Configuration** — view the *effective* config, validate a proposed edit, apply a change (SIGHUP reload), fail-closed on invalid config
3. **Monitoring** — live tunnel status (peers, TUN address, up/down), NAT counters (forwarded/return packets, flows, listener accepts), rate-limiter pacing, cert status + expiry, recent log lines

### Architecture requirements

- Ship as a subcommand of the existing daemon (`sssonector tui`) or a separate small binary in the same repo — your call, but justify it in the spec. Do NOT add a second competing `main` without an ADR (AGENTS.md).
- Go. Reuse the existing config loader + monitor/metrics packages; the TUI must NOT re-implement parsing or metric collection.
- Read metrics from the daemon's Prometheus `/metrics` endpoint (or the live monitor), not a separate source of truth.
- Fail closed: if the daemon is unreachable, show an explicit error — never stale or fabricated data (anti-mock posture).
- Write the spec down first (`docs/`): layout, keybindings, data sources, degradation when the daemon is down.

### Acceptance ([DONE] gate)

1. A written spec (`docs/tui.md` or an ADR) covering layout, keybindings, data sources, failure modes.
2. The TUI builds (`go build ./...`) and runs against a live daemon on the QA rig (`pa-fleet-11`), showing real tunnel + NAT + cert state.
3. Config view/edit shows the EFFECTIVE config (post-merge with defaults); an invalid edit is rejected without breaking the running daemon.
4. `go vet` + `go test -race ./...` stay green; new TUI code carries tests for any new parsing/formatting logic.
5. Paste the commit SHA + a demo transcript (or screenshot) of the TUI against the rig.

---

## 2. Installers for all three platforms

Create proper installers (or at minimum robust install scripts) for Linux, macOS, and Windows, replacing the current raw-binary + `curl | bash`-only distribution.

### Context (what exists today)

- `release.yml` ships 5 raw cross-compiled binaries (linux amd64/arm64, darwin amd64/arm64, windows amd64) + SHA256SUMS + SBOM.
- `install.sh` (`curl | bash`) downloads the raw binary + config templates, sets up a systemd unit (linux), and does interactive setup (mode, instance, TUN address, server, port).
- `scripts/build-deb.sh` + `scripts/build-installers.sh` exist but are NOT wired into `release.yml` — the release publishes raw binaries only.
- `scripts/install_macos.sh` + `scripts/install-from-source.sh` exist as manual helpers.

### Scope (what to build)

1. **Linux** — `.deb` + `.rpm` packages (via nfpm, matching the PairAdmin convention), installing the daemon + a default systemd unit + config template, with post-install service enable.
2. **macOS** — a `.pkg` (or a `.dmg` with an install script) installing the daemon + a launchd plist + config template. Unsigned is acceptable (entity-free posture); document the Gatekeeper/quarantine steps.
3. **Windows** — an installer (`.msi` via WiX, or `.exe` via NSIS) installing the daemon + a Windows service registration + config template.

If a full native installer is not tractable for a platform, fall back to a robust, tested, idempotent install script — but document that choice explicitly.

### Architecture requirements

- Wire the installers into `release.yml` so a tag builds AND uploads them alongside the raw binaries + SHA256SUMS + SBOM.
- SHA256SUMS must cover every installer artifact; the install flow must verify checksums before installing (matching the existing `install.sh` verify behavior).
- Reuse nfpm config conventions from PairAdmin where they fit (version from tag, homepage/license correct).
- Keep the raw binaries available (some users want them); installers are additive, not a replacement.

### Acceptance ([DONE] gate)

1. A tagged release produces linux `.deb` + `.rpm`, a macOS installer, and a Windows installer, all listed in SHA256SUMS and downloadable.
2. A fresh-install test on each platform (clean VM/box) succeeds end-to-end: install → service starts → daemon runs → uninstall cleanly removes it.
3. `go build`/`go vet`/`go test` stay green (packaging-only changes must not regress the repo).
4. Paste the release URL + the fresh-install results per platform.
