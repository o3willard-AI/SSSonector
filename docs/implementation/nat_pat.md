# NAT/PAT Design

Status: IMPLEMENTED (forward NAT + reverse PAT, TCP-only v1)

## Overview

SSSonector's tunnel is a raw L3 pipe: client TUN ↔ (TLS/facade) ↔ server
TUN, with `Transfer` moving opaque IP packets both ways. The optional
NAT/PAT subsystem (`internal/nat`) adds two capabilities on top without
changing the tunnel itself:

1. **Forward NAT (jump host)** — a tunnel client reaches networks on the
   server host's other interfaces. The daemon performs stateful SNAT44
   on the packet path: no `ip_forward`, no nftables, no shell-outs.
2. **Reverse PAT (service publishing)** — public listeners on the
   internet-facing host relay TCP through the tunnel to services behind
   the peer's TUN. A low-resource VPS can front high-compute backends.

Both are disabled by default. An absent `nat:` config section means the
data path is byte-for-byte unchanged.

## Architecture

```
internal/nat/
├── nat.go        package errors + doc
├── checksum.go   RFC 1624 incremental checksum update; TCP checksum
├── packet.go     IPv4/TCP parse (strict), address/port rewrite
├── acl.go        forward ACL (first-match, default-deny) + listener allowlist
├── conntrack.go  5-tuple flow table, SNAT ephemeral pool, TCP state, GC
├── engine.go     Engine: forward + reverse packet processing, stats
├── reverse.go    ReverseNAT: gVisor netstack, public listeners, relay
└── reload.go     atomic rule swap; listener convergence
```

### Forward NAT (packet-level SNAT)

Packets from the tunnel are parsed (IPv4 + TCP only in v1), evaluated
against the forward ACL, and — if allowed — get their source rewritten
to the server's egress address with a conntrack-allocated source port.
Return packets addressed to the SNAT identity are reverse-translated
back to the tunnel-side origin. Everything else is dropped:

- ACL deny → drop + counter + rate-limited log
- malformed/non-IPv4/non-TCP → drop + counter
- egress packet not matching any conntrack entry → drop (never leaks
  into the tunnel)

Checksums: incremental RFC 1624 updates during rewrite, followed by a
full IP+TCP recompute (belt-and-braces; correctness over micro-speed on
a path that copies whole packets anyway).

Conntrack: O(1) map lookups under one small mutex (no sleeps under
locks); TCP states SYN→established→FIN-wait→closed tracked from observed
flags; idle entries GC'd every minute (default 5 min idle window),
returning SNAT ports to the pool. Pool exhaustion drops and counts —
fail closed.

Egress address: auto-discovered as the first non-tunnel, non-loopback
IPv4 address on the host. Discovery failure disables NAT (logged) —
fail closed.

### Reverse PAT (netstack)

Reverse PAT requires originating TCP toward the peer's kernel over the
raw tunnel — hand-rolled TCP is off the table (repo quality bar). The
engine embeds gVisor's netstack (`pkg/tcpip` only, not the sandbox),
pinned to the commit Tailscale ships in production.

- A channel link endpoint: stack-emitted frames are pumped into the
  tunnel; tunnel frames are injected into the stack.
- Public listeners (`listen_port` → `dst` host:port) gate sources
  through a fail-closed allowlist at accept, then dial the destination
  via netstack. Handshake, windowing, retransmit, and half-close are
  netstack's real TCP implementation.
- Relay copies both directions with half-close on each finish
  (data-path invariant 4).
- Tunnel-down: frame writes are no-ops; netstack retransmit recovers
  when the tunnel returns.

## ACL model (both directions)

Default deny, always:

- Forward rules must list explicit `src_cidr`, `dst_cidr`, and at least
  one `ports` entry to ever match; a portless rule matches nothing.
- Reverse listeners must list explicit `allowed_cidrs`; an empty
  allowlist is rejected by the validator (ambiguous intent is a config
  error, not a default).
- The validator rejects forward rules whose `dst_cidr` overlaps the
  tunnel subnet (both directions of overlap), duplicate rules, and
  duplicate listen ports.

## Configuration

```yaml
config:
  nat:
    enabled: true          # master switch; absent/false = fully off
    forward:
      enabled: true
      rules:
        - comment: "client may reach server LAN web"
          src_cidr: 10.77.0.0/24
          dst_cidr: 192.168.10.0/24
          ports: [80, 443]
    reverse:
      enabled: true
      listeners:
        - comment: "publish home web server"
          listen_port: 8080
          dst: "10.77.0.2:80"
          allowed_cidrs: ["203.0.113.0/24"]
```

Reload semantics (SIGHUP):

- `nat.enabled`, `nat.forward.enabled`, `nat.reverse.enabled` changes →
  restart-required warning (structural, like `facade.enabled`).
- Rule and listener changes → hot-applied. Established forward flows
  keep their conntrack entries; new flows hit the new rules. Reverse
  listeners converge: removed ports stop, added ports start, existing
  listeners survive partial failures.

## Observability

Prometheus (on the existing `/metrics` endpoint):

| Metric | Type | Meaning |
|---|---|---|
| `sssonector_nat_forwarded_packets_total` | counter | tunnel→egress translated |
| `sssonector_nat_return_packets_total` | counter | egress→tunnel reverse-translated |
| `sssonector_nat_dropped_packets_total` | counter | ACL denies + malformed + no-translation |
| `sssonector_nat_flows_active` | gauge | live conntrack entries |
| `sssonector_nat_listener_accepts_total` | counter | reverse-PAT accepts |
| `sssonector_nat_acl_denied_total` | counter | reverse-PAT listener ACL denies |

SNMP: OIDs `.1.3.6.1.4.1.54321.3.6` through `.3.11` mirror the same
counters (Counter64 / Gauge32), continuing the throttle-hit precedent.

Structured logs: ACL denies are rate-limited to one per second; engine
startup, listener binds, GC sweeps, and reloads are logged contextually.

## v1 limits (deliberate)

- TCP only. UDP needs conntrack for connectionless flows on the forward
  path and netstack UDP endpoints on the reverse path — designed for,
  not implemented (see Issues.md).
- IPv4 only. The parser rejects non-IPv4 packets (fail closed); IPv6 is
  a v2 concern.
- Non-initial IP fragments are not reassembled; the parser drops
  malformed/truncated input.
- Forward path intercepts the egress-side read; tunnel-side SNAT entry
  happens via the same engine call from the connection setup.
- No hairpin NAT, no DNS built-ins, no port-range PAT.
