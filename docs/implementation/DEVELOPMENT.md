# SSSonector Development Guide

This document provides information for developers working on the SSSonector codebase.

## Project Structure

```
cmd/
  └── daemon/          # The single binary (server | client subcommands)
internal/
  ├── adapter/         # TUN device lifecycle (native ioctl + netlink)
  ├── cert/            # Certificate manager + rotation
  │   └── generator/   # Test/lab certificate generation
  ├── config/          # Single-package config: load, validate, store
  ├── facade/          # HTTPS facade for firewall traversal
  │   ├── server.go    # HTTPS server with WebSocket upgrade
  │   ├── client.go    # Direct-connect with facade fallback
  │   ├── token.go     # HMAC-SHA256 token generation/validation
  │   └── proxy.go     # Bidirectional TCP proxy (half-close reference)
  ├── monitor/         # Prometheus /metrics, /healthz, SNMP agent
  ├── throttle/        # Reservation/debt token-bucket rate limiting
  └── tunnel/          # Server/client roles, transfer loop, TLS
docs/                  # Operator and design documentation
scripts/               # Installers and lab tooling
```

There is deliberately no `internal/service` control-socket system, no
`sssonectorctl` companion CLI, and no separate connection/pool/memory/security
packages — earlier generations of this tree accumulated those; they were
deleted under the wire-it-or-delete-it rule.

## Daemon Interface

The one binary is `cmd/daemon`:

```bash
sssonector [server|client] -config /etc/sssonector/config.yaml
sssonector -log-level debug ...
sssonector -version
```

- Mode comes from the positional argument or the config; ambiguity is fatal.
- Invalid configuration aborts startup with the offending field named.
- Runtime operations are external by design:
  - **systemd** owns start/stop/restart (`Restart=on-failure` pairs with the
    non-zero exit on reconnect exhaustion).
  - **SIGHUP** reloads the hot subset (throttle rates, log level, certificate
    rotation interval); structural changes are logged as restart-required.
  - **Prometheus endpoint** serves `/metrics` and `/healthz` on the configured
    port (`monitor.prometheus.listen_address` controls exposure).
  - **journalctl** is the log surface when run under systemd.

## Error Handling

Errors follow standard Go conventions:

- Wrap with context at every layer (`fmt.Errorf("...: %w", err)`).
- Fail closed on anything security-relevant: missing certificates,
  ambiguous run mode, invalid configuration.
- Data-path goroutines report failures through returned errors; the
  transfer loop tears connections down instead of leaking readers.
- Client reconnect exhaustion closes `Client.GiveUpChan()`, which makes the
  daemon exit non-zero so systemd restarts it.

## Configuration

Configuration is a single YAML schema (`metadata.schema_version: "2.0.0"`):

- Loaded by `internal/config` (loader → validator). Only the current schema
  is accepted; legacy layouts are rejected explicitly.
- Validation runs at startup **and** on every SIGHUP reload; rejected
  reloads keep the previous config active.
- See `docs/configuration_guide.md` for the full key reference and
  `docs/hot_reload.md` for what may change at runtime.

## Development Guidelines

1. Read `AGENTS.md` before writing code — its rules (one implementation per
   concept, wire-it-or-delete-it, fail-closed security, deterministic timing
   tests) are enforced in review and CI.
2. Error handling: wrap with `%w`; never swallow; never fall back silently.
3. New behavior ships with tests that can fail — see
   `docs/testing/mutation-testing.md` for the critical-path bar.
4. Platform-specific TUN code lives behind `AdapterNew`, so logic tests run
   unprivileged over in-memory pipes (see `internal/tunnel/loopback_test.go`).

## Building and Testing

Build the daemon:
```bash
make build-linux-amd64
```

Run the full test suite (race detector on):
```bash
go test -race -count=1 ./...
```

Lint gate:
```bash
golangci-lint run ./...
```

## Contributing

1. Follow Go coding standards and AGENTS.md.
2. Add tests for new features; keep changed-line coverage ≥80%.
3. Update documentation in the same change set.
4. Run the full test suite and lint gate before submitting.
