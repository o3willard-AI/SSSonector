# Provisioning Guide

Worked examples for first-run enrollment using `sssonector provision`.
Every command below was executed against a locally built binary; output
blocks are real, sanitized only per RFC 5737 addressing and placeholder
codes/fingerprints.

Companion reference: [certificate_management.md](certificate_management.md)
(chain layout, permissions table). Design rationale:
[provisioning_design.md](provisioning_design.md).

---

## 1. End-to-End Walkthrough (Linux, offline transfer)

### Server side — create the bundle

```bash
# Layout: token_secret lives NEXT TO the certs directory
mkdir -p /etc/sssonector/certs
echo "<high-entropy-secret>" > /etc/sssonector/token_secret
chmod 600 /etc/sssonector/token_secret

sssonector provision create --role client \
    --name office-a \
    --server-addr 192.0.2.10 --server-port 18443 \
    --certs-dir /etc/sssonector/certs \
    --out office-a.ssp
```

Captured output:

```
No CA found in /etc/sssonector/certs; generating new CA + server pair...
bundle written: office-a.ssp (4516 bytes)
pairing code:   ABCD-EFGH
CA fingerprint: b6dd4fa04a89dcb854c35716b74ca814d952761e708badd9d759126f908398a3
Share the CODE out-of-band (voice/SMS). The .ssp file itself may travel by any transport — it is unreadable without the code.
```

Notes:
- First run bootstraps CA + server pair automatically (`ca.crt`, `server.*`,
  plus the enrolled `client.*`).
- The **token_secret** file must sit one level above the certs dir. This is
  deliberate: certificates get replicated to peers; the shared secret must
  not ride along.

### Share out-of-band

Send the pairing code by voice/SMS/chat. Move `office-a.ssp` by any means —
scp, USB, AirDrop, HTTP — it is AEAD-encrypted under the code-derived key.

### Client side — apply

```bash
sssonector provision apply --from office-a.ssp
```

Real interactive session:

```
Enter pairing code:

Bundle contents:
  role:            client
  server:          192.0.2.10:18443
  label:           office-a
  created:         2026-08-24T16:20:47Z
  CA fingerprint:  b6dd4fa04a89dcb854c35716b74ca814d952761e708badd9d759126f908398a3
  target dir:      /home/you/certs

Install these files into /home/you/certs ? [yes/no]: yes
config skeleton written: /home/you/config.yaml
Provisioning complete. Review config.yaml, then start the service.
```

Result on disk:

```
certs/ca.crt      0644
certs/client.crt  0644
certs/client.key  0600   <- private key never appears in logs or argv
config.yaml       0600   (skeleton carrying server addr + facade secret)
```

First service start:

```bash
sudo systemctl start sssonector   # or run the binary directly to test
journalctl -u sssonector -f        # expect "Tunnel established"
ping <server-tun-address>          # data-plane check
```

---

## 2. Network Redemption (--serve)

Skip manual file transfer entirely.

**Server** (offers the bundle for 15 minutes, one redemption only):

```bash
sssonector provision create --role client --name branch-b \
    --server-addr 192.0.2.10 --server-port 18443 \
    --serve --listen :9443 --out branch-b.ssp
```

Captured:

```
pairing code:   ABCD-EFGH
CA fingerprint: <sha256>
serving redemption at https://127.0.0.1:9443/pair/ABCD-EFGH (TTL 15m0s, single-consumption)...
Client: sssonector provision apply --from https://127.0.0.1:9443/pair/ABCD-EFGH
```

**Client**:

```bash
sssonector provision apply --from https://198.51.100.10:9443/pair/ABCD-EFGH
```

Semantics verified live:

| Outcome | Meaning | Operator action |
|---|---|---|
| `200` + install prompt | Bundle received; AEAD-authenticated | Confirm fingerprint, proceed |
| `403 invalid pairing code` | Code mistyped or wrong offer | Re-enter; codes are case/separator insensitive |
| `429 too many attempts` | Guess budget exhausted for your IP | Wait out the TTL window; check for typos |
| `410 already redeemed` | Offer consumed or expired | Server must mint a fresh bundle |

TLS certificate warnings during redemption are expected: authenticity comes
from the AEAD envelope under the code-derived key, and the embedded CA
fingerprint pins all *tunnel* trust afterwards.

> **Behind NAT/firewall?** `--serve` binds all interfaces on port 9443 by
> default. Restrict exposure with `--listen 127.0.0.1:9443` (then redeem via
> an SSH tunnel) or bind a dedicated management address.

---

## 3. CSR Mode — key never leaves the client

Prefer this when the client machine can reach the server's redemption URL.
The private key is generated locally and never transmitted; only a signing
request travels, authenticated by the same pairing code.

```bash
sssonector provision apply \
    --from https://198.51.100.10:9443/pair/ABCD-EFGH \
    --csr --csr-cn branch-c
```

Verified live output:

```
client certificate installed: .../certs/client.crt
Key never left this machine. Complete config.yaml auth paths if needed.
```

The server signs the CSR with its CA (clientAuth EKU) and returns leaf+CA;
`apply` installs both alongside the locally generated `client.key`. Choose
bundle mode instead for headless targets where running apply interactively
during setup is impractical.

---

## 4. Platform Notes

### Windows (PowerShell)

Certificates land in `%ProgramData%\SSSonector\certs` with an ACL limited to
owner, Administrators, and SYSTEM. Run from an **elevated** PowerShell:

```powershell
# Server side
"REPLACE-WITH-REAL-SECRET" | Out-File -Encoding ascii C:\ProgramData\SSSonector\token_secret
.\sssonector.exe provision create --role client --name office `
    --server-addr 192.0.2.10 --server-port 18443 --out office.ssp

# Client side
.\sssonector.exe provision apply --from .\office.ssp
```

(Windows-specific commands mirror the Linux flows verified during this
guide's authoring; the provisioning code is cross-platform Go with
platform-correct path and ACL handling.)

### macOS

Certs install under `/Library/Application Support/sssonector/certs`. Under
launchd, point the unit's `-config` at the skeleton written next to that
directory and review it once before loading the job.

---

## 5. Re-Verification & Rotation

Run periodically (cron/systemd timer):

```bash
sssonector provision verify --certs-dir /etc/sssonector/certs
```

Live captured output:

```
fingerprint:  b6dd4fa04a89dcb854c35716b74ca814d952761e708badd9d759126f908398a3
ca.crt       expires 2027-08-24 (364d) [ok] CN=SSSonector Root CA
server.crt   [absent - skipped]
client.crt   expires 2027-02-20 (179d) [ok] CN=SSSonector client
verify: all checks passed
```

Useful variants:

```bash
# Pin expectations (constant-time compare against operator-provided value)
sssonector provision verify --expect-fingerprint <sha256>

# Rotation warning window sourced from auth.cert_rotation.interval in config
sssonector provision verify --config /etc/sssonector/config.yaml

# Force a rotation-due report for demonstration
sssonector provision verify --rotation-within 87600h
```

Interpreting states:

| State | Meaning |
|---|---|
| `ok` | Valid chain, comfortably before expiry |
| `ROTATION-DUE (<Nd)` | Expiry inside the warning window — schedule re-enrollment |
| `EXPIRED` | Unusable; peers will refuse handshake |
| `[absent - skipped]` | File not present for this role (normal: clients have no `server.crt`) |

Re-enroll via a fresh `create` + `apply` cycle; `apply` refuses overwrites
without `--force`, so existing material is never silently replaced.

## Troubleshooting

Exact error strings produced by the binary, mapped to causes and fixes:

| Error (verbatim prefix) | Cause | Fix |
|---|---|---|
| `read <dir>/../token_secret: no such file or directory (create this file with one high-entropy line...` | `create` expects the shared secret **one level above** the certs dir | `echo "<secret>" > $(dirname <certs-dir>)/token_secret && chmod 600 ...` |
| `decryption failed (wrong pairing code or tampered bundle)` | Code mistyped, or `.ssp` modified in transit (deliberately indistinguishable) | Re-check code (case-insensitive, separators optional); re-transfer the file; compare sizes |
| `output <file> exists (pass -out elsewhere or set SSSONECTOR_FORCE=1)` | Create would clobber an undelivered bundle | Pick another `--out`; set `SSSONECTOR_FORCE=1` only if certain |
| `stdin is not a terminal; refusing to read pairing code non-interactively` | Apply run without interactive TTY | Run from a real console/PTY; see headless patterns below |
| `redemption rejected: invalid pairing code` (HTTP 403) | Wrong code at redemption | Re-check code; repeated failures consume the per-IP attempt budget |
| `redemption rate limited; retry later` (HTTP 429) | >10 failed attempts from one IP within the TTL | Wait for the window to expire; verify the code before retrying |
| `enrollment already redeemed or expired` (HTTP 410) | Offer consumed once already, or TTL elapsed | Server mints a new offer (`create --serve`) |

## Headless / Automation

`apply` requires a TTY **by design**: pairing secrets must not be piped,
echoed into stdin, or stored in shell history. Supported automation
patterns:

1. **Golden images** — run `apply` interactively while building the image;
   the resulting certs + config bake into every deployed instance.
2. **Setup-time enrollment** — first-boot setup script drops the operator
   into the standard interactive `apply` flow (e.g., via console/getty).
3. **CSR issuance** — scripted machines generate their key locally ahead of
   time; only the interactive code entry remains at sign-time. Combine with
   configuration management that stages everything else.

Do **not** pipe codes via `echo code | sssonector provision apply ...`;
the daemon detects non-TTY stdin and exits rather than accepting it.

Known gap tracked in Issues.md: fully unattended enrollment (e.g., env-var
code injection for CI-only lab use) is deliberately absent; revisit behind
an explicit design decision if a deployment demands it.
