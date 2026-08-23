# SSSonector Configuration Guide

This guide describes the **current** configuration schema (v2,
`metadata.schema_version: "2.0.0"`). Reference fixtures live in
`configs/` and `templates/`.

## Configuration File Structure

```yaml
metadata:
  schema_version: "2.0.0"
  environment: production   # development|staging|production|test|qa
  region: local

type: server                # server|client
config:
  mode: server
  logging:
    level: info             # debug|info|warn|error|fatal
    format: json            # json|console
    file: ""                # empty = stdout only; when set, output mirrors to stderr
  network:
    name: tun0
    interface: tun0
    mtu: 1500
    address: 10.0.0.1/24    # CIDR notation is required
  tunnel:
    listen_address: 0.0.0.0 # server role
    listen_port: 8443       # server role
    server_address: ""      # client role
    server_port: 8443       # client role
  security:
    tls:
      min_version: "1.2"    # production environment enforces "1.3"
      max_version: "1.3"
    auth_method: certificate
  auth:
    cert_file: /etc/sssonector/certs/server.crt
    key_file: /etc/sssonector/certs/server.key
    ca_file: /etc/sssonector/certs/ca.crt
    cert_rotation:
      enabled: false
      interval: 1h          # expiry-check cadence; hot-reloadable
  monitor:
    enabled: true
    type: prometheus        # prometheus|snmp
    interval: 10s
    prometheus:
      enabled: true
      port: 9090
      path: /metrics
  snmp:
    enabled: false
    address: "0.0.0.0"
    port: 10162
    community: "public"
  metrics:
    enabled: true           # feeds SNMP/Prometheus from the live data path
    interval: 5s
  facade:
    enabled: false          # see Facade Configuration below
throttle:
  enabled: false
  rate: 1048576             # bytes/sec; effective rate = rate x 1.1
  burst: 104858             # accepted but unused: burst is always 100ms of effective rate
```

## Basic Configuration Examples

### Server Configuration
```yaml
metadata:
  schema_version: "2.0.0"
  environment: qa
type: server
config:
  mode: server
  logging:
    level: info
    format: json
  network:
    name: tun0
    interface: tun0
    mtu: 1500
    address: 10.0.0.1/24
  tunnel:
    listen_address: "0.0.0.0"
    listen_port: 8443
  auth:
    cert_file: "/etc/sssonector/certs/server.crt"
    key_file: "/etc/sssonector/certs/server.key"
    ca_file: "/etc/sssonector/certs/ca.crt"
  monitor:
    enabled: true
    type: prometheus
    prometheus:
      enabled: true
      port: 9090
      path: /metrics
  metrics:
    enabled: true
    interval: 5s
throttle:
  enabled: true
  rate: 1000000    # ~1.1 MB/s effective
```

### Client Configuration
```yaml
metadata:
  schema_version: "2.0.0"
  environment: qa
type: client
config:
  mode: client
  logging:
    level: info
    format: json
  network:
    name: tun0
    interface: tun0
    mtu: 1500
    address: 10.0.0.2/24
  tunnel:
    server_address: "192.168.50.210"
    server_port: 8443
  auth:
    cert_file: "/etc/sssonector/certs/client.crt"
    key_file: "/etc/sssonector/certs/client.key"
    ca_file: "/etc/sssonector/certs/ca.crt"
  snmp:
    enabled: true
    address: "0.0.0.0"
    port: 10162
    community: "public"
  metrics:
    enabled: true
    interval: 5s
  monitor:
    enabled: true
    type: snmp
```

## Section Details

### Network Configuration
- `interface`: Name of the TUN interface to create
- `address`: IP address in CIDR notation (e.g. `10.0.0.2/24`); bare IPs are rejected by validation
- `mtu`: Maximum Transmission Unit (valid range 576–65535)

### Tunnel Configuration
Validation is mode-aware:
- Server: `listen_port` must be 1–65535
- Client: `server_address` must be set; `server_port` must be 1–65535

Certificate paths may be absolute or relative to the config file location.

### Tunnel Reconnect (client)
- `tunnel.reconnect.max_attempts`: dial attempts before giving up (default 10, max 100)
- `tunnel.reconnect.initial_delay`: first backoff delay (default 1s)
- `tunnel.reconnect.max_delay`: hard ceiling for any single delay (default 30s)
- `tunnel.reconnect.jitter`: fraction 0–0.9 by which each delay is randomly reduced
  (default 0.2) so a restarting fleet desynchronizes instead of stampeding

Delays double per attempt (`initial × 2^(n-1)`, capped at `max_delay`), then
jitter reduces the result. Omitting the whole block uses the defaults.
All four values are read on every failed attempt, so SIGHUP reloads apply to
subsequent retries immediately (see [Hot Reload](hot_reload.md)).

### Logging Configuration
- `level`: one of debug, info, warn, error, fatal (required)
- `format`: `json` (default) or `console`
- `file`: when set, logs are written to this file AND mirrored to stderr;
  parent directories are created automatically
- The `-log-level` command-line flag overrides `logging.level`; it also
  keeps precedence across SIGHUP reloads (see [Hot Reload](hot_reload.md))

### Monitor / Metrics / SNMP Configuration
- `monitor.enabled` gates monitoring overall
- `monitor.type: prometheus` + `monitor.prometheus.enabled` serves a
  `/metrics` text-exposition endpoint on `monitor.prometheus.port`
- `monitor.type: snmp` + `snmp.enabled` starts the SNMPv2c agent on
  `snmp.address:snmp.port` (see [SNMP Monitoring](snmp_monitoring.md))
- `metrics.enabled` + `metrics.interval` control how often live data-path
  counters are sampled into those endpoints
- `/healthz` is served next to `/metrics` for liveness/readiness probes;
  it reports run mode and current tunnel state

### Throttle Configuration
- `enabled`: enable/disable rate limiting (hot-reloadable)
- `rate`: sustained limit in bytes/sec before TCP overhead adjustment;
  effective rate = `rate x 1.1` (hot-reloadable)
- `burst`: accepted for compatibility but **not used** — the limiter always
  derives burst as 100ms of the effective rate

### Facade Configuration (Firewall Traversal)

The HTTPS facade allows tunnel traffic to traverse firewalls that only permit
standard HTTPS (port 443). When enabled, tunnel connections are disguised as
WebSocket upgrades over a legitimate HTTPS web server.

**Server-side options:**
- `facade.enabled`: Enable the HTTPS facade server (default: false)
- `facade.listen_address`: Address to bind the facade (default: 0.0.0.0)
- `facade.listen_port`: Port for the facade (default/typical: 443)
- `facade.hostname`: Server hostname for TLS SNI
- `facade.web_root`: HTML content for GET / (empty = default "Hello, World" page)
- `facade.token_secret`: Shared HMAC secret (**required when the facade is enabled**; must be high-entropy and match the client)
- `facade.token_ttl`: Token validity duration (default: 30s, min: 5s, max: 120s)
- `facade.tls.cert_file`, `facade.tls.key_file`, `facade.tls.ca_file`: Optional separate TLS config (empty = inherits from auth section)
- `facade.tunnel_ports`: List of tunnel ports this facade routes to (required when enabled)

**Client-side options:**
- `facade.enabled`: Enable facade fallback (default: false)
- `facade.server_address`: Facade server address (default: same as tunnel.server_address)
- `facade.server_port`: Facade server port (default: 443)
- `facade.direct_timeout`: How long to wait for direct connection before fallback (default: 3s)
- `facade.token_secret`: **Required when the facade is enabled; must match server exactly**
- `facade.tls.cert_file`, `facade.tls.key_file`, `facade.tls.ca_file`: Optional separate TLS config

## Path Resolution Rules

1. Certificate paths:
   - Absolute paths are used as-is
   - Relative paths are resolved from config file location
   - Instance layouts under an `instances/` directory resolve relative
     cert paths against the instance directory

2. Log files (`logging.file`):
   - Absolute paths are used as-is
   - Relative paths are resolved from the current working directory
   - Parent directories are created automatically at startup

## Validation Rules

Configuration is validated at startup AND on every SIGHUP reload:

- `logging.level` must be one of debug/info/warn/error/fatal;
  `logging.format` must be json or console
- `network.address` must be valid CIDR
- Tunnel ports/address checked per mode (see above)
- Certificate path hygiene: no `..`, no duplicate slashes, correct extensions
  (.crt/.pem for certs, .key/.pem for keys, .crt/.pem for CA)
- `metadata.environment` must be one of development/staging/production/test/qa;
  production additionally requires TLS 1.3 minimum
- `monitor.type` must be prometheus or snmp when monitoring is enabled
- Invalid configurations abort startup or reject the reload — never fall
  back to defaults silently

## Common Issues and Solutions

1. **Service exits immediately with "Config validation failed"**
   The error names the offending field and the log line includes the config
   path. Fix the value; nothing is guessed.

2. **Service refuses to start: "refusing to start without TLS"**
   Certificates are missing or failed to load and
   `security.allow_plaintext` is not enabled. Provision valid certs, or set
   `security.allow_plaintext: true` deliberately for lab use.

3. **SNMP walk returns nothing**
   Check `monitor.type` is `snmp`, `snmp.enabled: true`, firewall allows the
   UDP port, and community matches.

4. **Rate changes not taking effect**
   Send SIGHUP after editing; check for "requires restart" warnings if you
   touched structural fields.

## Best Practices

- `metadata.schema_version: "2.0.0"` is REQUIRED. The loader accepts only
  the current schema and rejects anything else explicitly — configs written
  for v1.x must be migrated by hand (the old lossy auto-upgraders were
  removed because they silently discarded settings)
- Prefer explicit values over defaults for anything security-relevant
- Test config changes with `-log-level debug` first
- Restrict config file permissions; the facade token secret lives there
