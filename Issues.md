# Known Issues

This file tracks currently-true issues only.

## Open

1. **Packet counters report zero** — bytes are counted exactly
   (`countingConn`), but packet in/out counts remain 0 because TCP transports
   do not expose packet boundaries without per-syscall detail. SNMP/Prometheus
   packet families exist for schema compatibility. Implement via recvmsg with
   IP_PKTINFO or move counting to a packet-oriented layer if packet metrics
   become a requirement.
3. **Adapter requires privileged interface setup** — Linux TUN creation
   and configuration use the native ioctl + netlink UAPI (commit 7a2b3e5,
   zero shell-outs); the process must run as root or hold CAP_NET_ADMIN
   (shipped systemd unit sets `AmbientCapabilities=CAP_NET_ADMIN`).
   Environments without `/dev/net/tun` fail at startup by design (fail
   closed).

   pin when the stack is next exercised).
5. **Kubernetes deployment artifacts pruned** — the manifest shipped a
   legacy JSON ConfigMap the current loader cannot parse, and the k8s
   prometheus config assumed in-cluster conventions nothing established.
   Deleted until a real k8s deployment exists; revival needs a v2-schema
   config, TUN device mapping + NET_ADMIN (or the netlink privilege rework),
   and probes on `/healthz`.
6. **SNMP agent is v2c/community-auth only** — adequate for the lab/QA
   monitoring contract; SNMPv3 (authPriv) would be required for untrusted
   networks. Not planned unless a deployment demands it.
7. **Zero-downtime server restart not implemented** — restarts rely on fast
   client auto-reconnect (~1–2 s default). Decision record
   (`docs/zero_downtime_restart_decision.md`) recommends systemd socket
   activation only if a deployment demands sub-second listener failover.
8. **Coverage gaps in platform-specific code** — `internal/adapter` platform
   backends and OS-coupled monitor collectors cannot be integration-tested in
   unprivileged CI; they are exercised via QA runs on TUN-capable hosts.
9. **NAT/PAT is TCP + IPv4 only** — the forward SNAT parser rejects
   non-TCP/non-IPv4 packets (fail closed), and reverse PAT publishes TCP
   services only. UDP requires connectionless flow tracking on the
   forward path and netstack UDP endpoints on the reverse path; IPv6
   needs dual-stack ACLs and translation. Not planned unless a
   deployment demands it.
10. **NAT end-to-end QA on TUN-capable hosts pending** — unit and
    in-memory wire tests cover the engines; full jump-host and
    service-publishing flows against real kernels are exercised via the
    QA VMs (192.168.101.171/.172) before release tagging.

## Resolved (2026-08 backlog execution)

- ~~Config package sprawl~~ — `internal/config` consolidated into one package.
- ~~SNMP stack duplication~~ — wire layer replaced by gosnmp-backed
  encode + minimal decode; agent now provably interoperates with real SNMP
  clients (conformance suite). MIB mapping retained.
- ~~CLI duplication~~ — single entry point `cmd/daemon`; `cmd/tunnel` retired.
- ~~Artifact signing/SBOM~~ — keyless cosign signatures + CycloneDX SBOM
  published and verified per release.
- ~~Mutation campaign~~ — facade/token + throttle at 100% non-equivalent kill
  rate; methodology in `docs/testing/mutation-testing.md`.

## Resolved (2026-08 reliability track)

- ~~Silent plaintext downgrade~~ — missing/broken certificates now refuse to
  start unless `security.allow_plaintext` is explicitly enabled; both roles
  gate before serving or dialing.
## Resolved (2026-08 external audit remediation)

- ~~Docker image tags unpinned~~ — both compose files pinned to verified
  stables (prometheus v3.14.0, snmp-exporter v0.30.1, grafana 13.1.4);
  monitoring stack additionally runs `no-new-privileges` + read-only
  filesystems; root compose tags the locally built image `sssonector:local`.
- ~~Semgrep: compose hardening~~ — addressed by the same change.

8. **Provisioning & key-sharing gap (T1–T9)** — first-run enrollment requires
   manual openssl + file transfer with no pairing/verification story.
   ADR accepted (`docs/provisioning_design.md`, Status: ACCEPTED):
   `provision create|apply|verify` subcommands in `cmd/daemon`, encrypted
   `.ssp` bundles (Argon2id + XChaCha20-Poly1305), CSR mode as documented
   default once shipped. Implementation in progress.

9. **Non-interactive enrollment absent** - `provision apply` requires a TTY
   by design (fail-closed); fully unattended enrollment (env/config injected
   pairing secret for CI/lab automation) is deliberately not implemented.
   Revisit behind an explicit design decision with a documented threat-model
   tradeoff.

## Resolved (2026-08 external audits — code + docs)

- ~~F1 /tmp/ TLS skip heuristic~~ · ~~D1 Dockerfile toolchain~~ ·
  ~~D2 stale Issues.md #3~~ — fixed in e0210ea (auditors ran against the
  parent commit); verified present on HEAD.
- ~~Phantom certificate flags~~ (`-keygen`, `-validate-certs`,
  `-generate-certs-only`, `-keyfile`, `-test-without-certs`) — superseded by
  the accepted provisioning design; real `provision` subcommands replace
  them; `docs/certificate_management.md` rewritten in the same change set.
- ~~H1 DR scripts unrunnable~~ — three `}`→`fi` syntax errors fixed;
  all three pass `bash -n`.
- ~~H2 phantom DR flags~~ — README Script Usage rewritten to the real
  interfaces (positional timestamp restore, --delete, no selection flags).
- ~~H3 nonexistent make install~~ — real `install`/`install-macos` targets
  added (delegate to install-from-source/install_macos helpers).
- ~~M1 imaginary interface_tests suite~~ — README deleted per
  wire-it-or-delete-it.
- ~~M2–M4 missing QA scripts~~ — references repointed at
  test/qa_scripts/{build_and_deploy,cleanup}.sh and setup_net_snmp.sh.
- ~~M5 nonexistent --socket flag~~ — snippet removed from DEVELOPMENT.md.
- ~~L1 dual install.sh~~ — scripts/install.sh renamed to
  scripts/install-from-source.sh (root installer keeps its documented name).

## Resolved (2026-08 lab E2E follow-up)

- ~~Client give-up exits cleanly~~ — exhausting `tunnel.reconnect.max_attempts`
  now surfaces as a fatal error (non-zero exit), so `Restart=on-failure`
  revives the unit and the retry cycle starts fresh when the peer returns.

## Resolved (2026-08 quick-wins batch)

- ~~Limiter hit metrics not exposed~~ — `sssonector_throttle_hits_total{direction}`
  on `/metrics`, `.3.4`/`.3.5` in the MIB, and the rate gauges now report the
  live effective pacing instead of a static placeholder.
## Resolved (2026-08 logging & observability refactor)

- ~~Orphaned observability subsystem~~ — `internal/monitor` was unreachable
  from the daemon; now instantiated from config with lifecycle wiring.
- ~~Mixed logging systems~~ — cert.Manager migrated from stdlib log to
  structured zap; custom dead logger implementation deleted.
- ~~Silent critical paths~~ — adapter create/configure/ioctl/cleanup and all
  config load/validation failures now emit contextual structured logs.
- ~~Configured-but-ignored settings~~ — `logging.*`, `monitor.*`, `snmp.*`,
  `metrics.*` are honored; startup and reload enforce full validation
  (pre-existing validator drift vs live schema corrected: CIDR addresses,
  mode-aware tunnel checks, qa environment, decorative throttle.burst).
- ~~Documented-but-fictional hot reload~~ — SIGHUP pipeline implemented and
  tested end-to-end (validation gate, live limiter updates, atomic log-level
  changes, restart-required warnings); docs reconciled across ten files.
- ~~Prometheus endpoint absent~~ — `/metrics` text exposition served on the
  configured port; monitoring/ stack configs repaired to valid syntax and
  real OIDs.
