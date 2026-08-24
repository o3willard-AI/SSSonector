# Certificate Management Guide

## Overview

SSSonector uses mutual X.509 certificates (client + server, anchored by a
local CA) for authentication and encryption. All certificate lifecycle
operations are handled through the `provision` subcommands of the daemon:

```
sssonector provision create|apply|verify [flags]
```

There are no certificate-related flags on the main daemon. Older guides
referencing `-keygen`, `-validate-certs`, `-generate-certs-only`, `-keyfile`
or `-test-without-certs` describe a surface that never shipped; this page is
the authoritative replacement.

## Quick Start (server -> client)

On the **server** host:

```bash
# First run creates CA + server pair under /etc/sssonector/certs,
# then mints a unique client certificate and wraps an encrypted bundle:
echo "<high-entropy-secret>" > /etc/sssonector/token_secret
sssonector provision create --role client \
    --name office-a \
    --server-addr vpn.example.com --server-port 18443 \
    --out office-a.ssp
# prints:
#   pairing code:   XXXX-XXXX
#   CA fingerprint: <sha256-hex>
```

Share the **pairing code out-of-band** (voice/SMS). The `.ssp` file itself
may travel by any transport — scp, USB, HTTP — because it is sealed with
XChaCha20-Poly1305 under a key derived from the pairing code.

On the **client** host:

```bash
sssonector provision apply --from office-a.ssp
# prompts for the pairing code (hidden entry),
# shows role/server/fingerprint, asks for confirmation,
# installs ca.crt + client.crt + client.key (0600) and a config skeleton.
```

## Network Redemption

Skip the manual file move entirely:

```bash
# Server: serve instead of writing a file (Ctrl+C or redemption ends it)
sssonector provision create --role client --name office-a \
    --server-addr vpn.example.com --server-port 18443 --serve

# Client: redeem over the network
sssonector provision apply --from https://vpn.example.com:9443/pair/XXXX-XXXX
```

- The endpoint is **single-consumption**: one successful redemption per
  pairing code; afterwards it answers 410 Gone.
- Default TTL is 15 minutes (`--serve-ttl`).
- Redemption attempts are rate limited per client IP (429 on excess).
- TLS certificate warnings during redemption are expected — authenticity
  comes from the AEAD envelope, not the transport.

## CSR Mode (recommended default)

For deployments where the private key should never leave the client machine:

```bash
# Client: generate key locally, submit CSR
sssonector provision apply --from https://vpn.example.com:9443/pair/XXXX-XXXX --csr

# Server side must have been started with CSR signing enabled; see design doc.
```

The key is generated locally with restrictive permissions and never
transmitted. The server signs the submitted CSR using the CA and returns the
leaf certificate plus the CA certificate. Once shipped, this is the documented
default; encrypted-bundle mode remains available for headless/automation
flows where generating keys locally is impractical.

## Verification

```bash
# Full check: parse, chain-vs-CA, expiry with rotation window
sssonector provision verify

# Pin expectations (e.g., compare against fingerprint from the operator)
sssonector provision verify --expect-fingerprint <sha256-hex>

# Source the rotation warning window from config
sssonector provision verify --config /etc/sssonector/config.yaml
```

Exit code is non-zero when any certificate is expired, unparseable, fails
chain validation, or mismatches `--expect-fingerprint`.

## Rotation

Certificates are issued with a 180-day validity; the CA lasts one year.
`cert_rotation.enabled` plus `auth.cert_rotation.interval` in the config
control the background check cadence, and `provision verify` reports
`ROTATION-DUE` when expiry falls inside the configured interval. Re-run
`provision create --role client` to issue fresh material for a device, or
`apply --csr` flows once CSR signing is enabled server-side.

## File Locations & Permissions

| Platform | Directory | Key perms |
|---|---|---|
| Linux | `/etc/sssonector/certs` | 0600 via chmod |
| macOS | `/Library/Application Support/sssonector/certs` | 0600 |
| Windows | `%ProgramData%\SSSonector\certs` | Owner/Administrators/SYSTEM ACL |

`provision apply` refuses to overwrite existing files without `--force`.
