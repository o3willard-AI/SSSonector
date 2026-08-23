# Zero-Downtime Restart — Decision Record

Status: recommendation, no implementation committed.

## Context

`systemctl restart` drops every tunnel. Three candidate mechanisms were
considered for making restarts invisible to traffic.

## Options Compared

| Option | How it works | Pros | Cons |
|---|---|---|---|
| A. systemd socket activation | systemd owns the listening socket; daemon inherits it via `LISTEN_FDS`, so the accept queue survives restarts | Standard, no protocol change; server side becomes seamless for *new* connections | Only helps the **server's** listener; established tunnels still die; client TUN state can't be handed over |
| B. SIGUSR2 fork-exec handoff | Daemon re-execs itself, passing listeners + live conns + TUN fd to the child | Truly zero-downtime both roles | Large complexity: fd passing, state serialization, TLS session resumption, rollback on failed start; high blast radius in a security-sensitive data path |
| C. Fast-reconnect status quo | Restart is brief; clients auto-reconnect with jittered backoff | Already shipped (reconnect policy + dead-peer detection); zero new moving parts | Seconds of outage per restart; in-flight packets lost |

## Recommendation

**Adopt Option C as the supported behavior; implement A only if a real
deployment demands faster listener recovery; reject B.**

Rationale:

1. With jittered reconnects and idle detection now built in, client
   recovery after a server restart is measured in ~1–2 s by default
   (`initial_delay=1s`) — comparable to what socket activation would save,
   since established tunnels die under option A anyway.
2. Option B's cost/risk profile is wrong for this codebase: it touches the
   most security-critical path (TLS + TUN lifecycle) for a benefit that
   option C already makes marginal.
3. Revisit trigger: a documented operational requirement of sub-second
   server failover with long-lived connections. At that point, implement A
   first (small, standard) before ever considering B.

## Actionable Now

- Document `systemctl restart` semantics in the operations guide (done:
  configuration_guide reconnect section).
- Keep `Issues.md` entry open for socket activation behind a real demand.
