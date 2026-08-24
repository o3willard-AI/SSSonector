# Provisioning & Key Sharing Design

> Status: PROPOSED · Owner: maintainers · Scope: first-run enrollment of clients
> (and server bootstrap) across Linux, macOS, and Windows.

## Problem

Standing up a tunnel requires moving secrets between two machines:

1. X.509 material — client needs `ca.crt` + its own cert/key; server holds the
   CA keypair + server pair (`internal/cert`, `scripts/generate-certs.sh`).
2. `facade.token_secret` — shared HMAC secret that **must match exactly** on
   both roles (`internal/facade/token.go`, fail-closed via `ResolveSecret`).

Today this means bash + openssl plus manual file transfer (scp/rsync), which
assumes a Unix workstation with SSH tooling. Windows users have no documented
equivalent; macOS users get implicit instructions. There is no pairing,
enrollment, or verification story — just "copy these files."

Note: `docs/certificate_management.md` already documents flags (`-keygen`,
`-validate-certs`, `-generate-certs-only`, `-keyfile`, `-test-without-certs`)
that do not exist in the binary. The intended surface was agreed but never
built. This design replaces those phantom flags with real subcommands.

## Decision (ADR summary)

**Provisioning lives in `cmd/daemon` as subcommands — not a separate binary.**

Rationale:
- Invariant #3 (single entry point) — `main.go` already dispatches
  `server|client`; adding `provision` follows the existing pattern.
- No new release/sign/notarize matrix (cosign + SBOM + SHA256SUMS per artifact
  doubles for zero capability gain).
- Reuses `internal/cert`, config store, logging, TLS stack directly.
- A dedicated binary's only advantage (run provisioning off-host) is preserved
  via `--out <path>` writing bundles anywhere, root not required until `apply`.

## Target UX

```
Server operator:
  sssonector provision create --role client [--name <label>] [--out client-a.ssp]
    -> generates/loads CA material, wraps bundle, prints:
       pairing code (e.g. 7F3K-9QPD) + server CA fingerprint (SHA-256)

Client operator (any OS):
  sssonector provision apply --from client-a.ssp        # file was moved over by any means
  sssonector provision apply --from https://server:9443/pair/7F3K-9QPD   # Phase 2
    -> prompts for pairing code, shows fingerprint for confirmation,
       installs certs + config skeleton + token_secret, sets perms
```

Key property: **the bundle is end-to-end encrypted regardless of transport**
(AEAD under a key derived from the pairing code/passphrase). USB stick,
AirDrop, scp, HTTP redemption — all acceptable because only code-holders can
read the payload. This removes the platform-specific-transfer problem entirely.

## Bundle format v1 (`.ssp`)

```
magic "SSP1" | u8 version=1 | u16 KDF params id | salt(16B) | nonce |
AEAD ciphertext(XChaCha20-Poly1305 or AES-256-GCM):
    { role, server_addr, server_port, facade_token_secret,
      ca_cert_pem, [client_cert_pem], [client_key_pem],
      created_at, name, fingerprint_of_ca }
```

- KDF: Argon2id (memory/time params fixed at v1; upgradeable via version byte).
- Pairing code is display-grouped (XXXX-XXXX, RFC 1751-ish alphabet minus
  ambiguous glyphs), ≥ ~48 bits entropy, rate-limitable at redemption.
- Fingerprint of CA embedded so `apply` can pin it before network trust.

## Tasks

### Phase 0 — record the decision
- [ ] T0: Commit this ADR; update Issues.md (new open item: provisioning gap;
      resolved item: phantom cert flags superseded by `provision` subcommands).

### Phase 1 — MVP (manual transfer path)
- [ ] T1 `provision create`: assemble + encrypt `.ssp`; emit code +
      fingerprint; `--out` anywhere (no root); refuse argv-passed secrets;
      reuse `internal/cert` generation (extend, don't duplicate).
- [ ] T2 `provision apply`: decrypt (prompt-only, no-TTY → fail closed),
      display fingerprint + target paths, require explicit confirmation,
      write to platform-correct dirs (`/etc/sssonector/certs`,
      `%ProgramData%\SSSonector\certs`, `/Library/Application Support/sssonector`),
      keys chmod 600 / ACL-restricted, refuse overwrite without `--force`.
- [ ] T3 Tests: round-trip on linux/darwin/windows; wrong-code rejection;
      tampered-ciphertext rejection; overwrite-guard; gitleaks clean.

### Phase 2 — network redemption
- [ ] T4 `create --serve`: serve the bundle over HTTPS (existing TLS stack)
      keyed by pairing code; TTL default 15m; single-consumption marker;
      bind default loopback/management address per `prometheus.listen_address`
      precedent. Client pins embedded CA fingerprint before trusting response.
- [ ] T5 Redemption rate limiting (reuse throttle package concepts).

### Phase 3 — PKI hygiene (end state)
- [ ] T6 CSR mode: client generates its key locally, submits CSR + code,
      server signs and returns chain — private key never leaves the client.
      Make this the documented default once shipped; encrypted-bundle mode
      remains for headless/automation flows.
- [ ] T7 `provision verify` (real replacement for phantom `-validate-certs`):
      expiry, chain validation, fingerprint-vs-peer comparison; wire rotation
      hooks to existing `cert_rotation` config.

### Fix-in-passing (same PRs)
- [ ] T8 Rewrite `docs/certificate_management.md` around real subcommands;
      delete phantom flags (per AGENTS.md: docs updated in the same PR).
- [ ] T9 Add per-OS quick-start snippets (incl. PowerShell examples) to
      installation guides referencing `provision`.

## Acceptance criteria / guardrails

- Secrets never logged; prompt-only entry; no-TTY fails closed.
- AEAD-authenticated envelope; constant-time comparisons on codes/fingerprints.
- Mutation tests required on envelope + parser (AGENTS.md testing rules).
- govulncheck/gitleaks clean; all actions stay SHA-pinned.
- Cross-platform matrix green before Phase 1 merges.
