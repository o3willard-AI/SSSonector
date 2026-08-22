# Known Issues

This file tracks currently-true issues only. The previous 798-line roadmap
described files and phases that no longer exist (or never did); it has been
retired. Historical context lives in git history.

## Open

1. **SNMP stack duplication** — `internal/monitor` hand-rolls ~2,100 LoC of
   ASN.1/PDU handling beside the `gosnmp` dependency. Strangler replacement
   with a gosnmp-backed exporter is planned; keep interfaces stable until then.
2. **Config package sprawl** — `internal/config` spans five sub-packages
   (`types`, `interfaces`, `manager`, `store`, `validator`). Consolidation is
   planned; public behavior must not change during the move.
3. **Artifact signing** — releases publish SHA256SUMS verified by installers;
   cosign keyless signing is the next step.
4. **Historical private keys were purged from git history** (2026-08-22,
   owner-approved `git filter-repo` rewrite; `certs/`, release binaries and
   QA snapshots removed from all branches/tags). Any clones made before this
   date still contain the old history — reclone or fetch-and-reset. The
   exposed CA must still be treated as compromised until its deployment
   disposition is confirmed by the owner.

## Recently resolved

- Rate limiting was non-functional end-to-end (wrappers bypassed the limiter).
- Facade token secret could be derived from the public CA certificate.
- Dead packages: `internal/{pool,memory,connection}`, `cert/validator`,
  duplicate validator, phantom `sssonectorctl` build targets.
