# Hot Reload Configuration

SSSonector supports reloading a subset of its configuration at runtime via
`SIGHUP`, without disrupting active tunnel connections.

## How to Trigger a Reload

```bash
# Using process ID
kill -HUP $(pidof sssonector)

# Using systemd
systemctl reload sssonector
```

There is no file-watching mechanism: changes take effect only when `SIGHUP`
is delivered.

## What Can Be Reloaded

| Setting | Effect on reload |
|---|---|
| `throttle.enabled` | Applies to live and future transfers immediately |
| `throttle.rate` | Live limiters are updated in place; new connections pick it up automatically |
| `logging.level` | Applied atomically via the zap atomic level |
| `auth.cert_rotation.interval` | Updates the certificate-expiry check interval on live certificate managers |
| `tunnel.reconnect.*` | Read on every failed dial; subsequent retries use the new schedule |

Notes:
- If the daemon was started with an explicit `-log-level` flag, that flag
  keeps precedence and `logging.level` changes from reloads are ignored.
- Throttle **burst is always derived** as 100ms of the effective rate
  (rate x 1.1 TCP overhead factor). The `throttle.burst` config value is
  accepted but not used by the limiter.

## What Requires a Restart

Any change to the following is detected during reload and logged as a
`Configuration change requires restart` warning; the running values remain
in effect until restart:

- `mode` (server/client)
- `network.*` (TUN interface name, address, MTU)
- `tunnel.listen_address`, `tunnel.listen_port`,
  `tunnel.server_address`, `tunnel.server_port`
- `facade.*` (enable/disable, ports, secrets)
- `logging.file`, `logging.format` (logger outputs are fixed at startup)
- `monitor.type`, `monitor.prometheus.port`, `snmp.*`, `metrics.interval`
  (endpoint lifecycle and sampler cadence are fixed at startup)
- `tunnel.keepalive_seconds`, `tunnel.idle_timeout_seconds`
  (per-connection settings)

## Reload Pipeline

On each `SIGHUP` the daemon:

1. Loads the configuration file fresh from disk.
2. Runs the same validation enforced at startup. On any failure the reload
   is rejected, the reason is logged at ERROR level, and the previous
   configuration stays active.
3. Applies the hot-reloadable subset listed above.
4. Logs each structural difference as a restart-required warning.
5. Confirms with a `Configuration reloaded` INFO entry.

## Example: Adjusting the Rate Limit

```yaml
# Before
throttle:
  enabled: true
  rate: 1048576    # ~1 MB/s effective after TCP overhead adjustment

# After editing and saving
throttle:
  enabled: true
  rate: 2097152    # ~2 MB/s
```

```bash
kill -HUP $(pidof sssonector)
```

Expected log output:

```
{"level":"info","msg":"Applying reloaded throttle settings","enabled":true,"rate":2097152,...}
{"level":"info","msg":"Configuration reloaded",...}
```

## Monitoring Reloads

```bash
journalctl -u sssonector | grep -E "reloaded|requires restart"
```

Current throughput and limiter state are observable either through the
Prometheus `/metrics` endpoint (when `monitor.type: prometheus`) or via
SNMP (see [SNMP Monitoring](snmp_monitoring.md)).

## Troubleshooting

- **Reload rejected**: check the ERROR entry directly above; it names the
  validation failure. Fix the file and resend `SIGHUP`.
- **Level change ignored**: an explicit `-log-level` flag was given at
  startup; flags win over config.
- **Structural changes not applying**: by design; restart the service.
