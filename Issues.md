# Known Issues

This file tracks currently-true issues only.

## Open

1. **Diff coverage on legacy code** — new PRs are gated at ≥80% changed-line
   coverage (`scripts/diff_coverage.py` in CI), but pre-existing modules
   (notably `cmd/daemon`, `internal/adapter`, `internal/monitor` system
   samplers) remain lightly covered overall. Characterization tests should
   grow opportunistically as those areas change.
2. **SNMP agent is v2c/community-auth only** — adequate for the lab/QA
   monitoring contract; SNMPv3 (authPriv) would be required for untrusted
   networks. Not planned unless a deployment demands it.

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
