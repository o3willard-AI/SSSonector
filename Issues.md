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
   shells out to `sudo ip tuntap/ip link`; deployments need root or
   passwordless sudo for those commands, and environments without
   `/dev/net/tun` fail at startup by design (fail closed).
4. **Docker image tags unpinned** — `monitoring/docker-compose.yml` uses
   `:latest` for prometheus/snmp-exporter/grafana (pre-existing P3 hygiene;
   pin when the stack is next exercised).
5. **K8s prometheus config assumes in-cluster service name** —
   `monitoring/prometheus/prometheus.yml` scrapes `sssonector:9090` and uses
   pod-annotation relabeling; align it if/when the Helm/k8s path is revived.
6. **SNMP agent is v2c/community-auth only** — adequate for the lab/QA
   monitoring contract; SNMPv3 (authPriv) would be required for untrusted
   networks. Not planned unless a deployment demands it.
7. **Coverage gaps in platform-specific code** — `internal/adapter` platform
   backends and OS-coupled monitor collectors cannot be integration-tested in
   unprivileged CI; they are exercised via QA runs on TUN-capable hosts.

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
