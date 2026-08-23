# SNMP Monitoring Guide

SSSonector ships an SNMPv2c agent (Get/GetNext) that exposes live tunnel
metrics under the enterprise MIB root `.1.3.6.1.4.1.54321`.

The agent is enabled when both `config.monitor.enabled: true` and
`config.snmp.enabled: true` are set:

```yaml
config:
  monitor:
    enabled: true
    type: snmp
    interval: 10s
  snmp:
    enabled: true
    address: "0.0.0.0"   # Bind address for the UDP listener
    port: 10162
    community: "public"
```

## Supported Operations

- `snmpget` / `snmpwalk` / `snmpgetnext` (SNMPv2c)
- Community-string verification with AuthorizationError responses

Not supported: traps/informs, SNMP SET (entries marked read-write in the
MIB accept the definition but values are not applied), SNMPv3.

## MIB Layout

All OIDs are relative to the enterprise root `.1.3.6.1.4.1.54321`.

### Tunnel metrics (.1)

| OID | Name | Type | Description |
|---|---|---|---|
| .1.1 | bytesIn | Counter64 | Total bytes received from tunnel peers |
| .1.2 | bytesOut | Counter64 | Total bytes sent to tunnel peers |
| .1.7 | activeConnections | Gauge32 | Currently active tunnel connections |
| .1.8 | cpuUsage | Gauge32 | Process CPU usage percentage |
| .1.9 | memoryUsage | Gauge32 | Process memory usage (MB) |

### Tunnel state (.2)

| OID | Name | Type | Description |
|---|---|---|---|
| .2.1 | tunnelStatus | Integer | 0 = down, 1 = up |
| .2.2 | lastError | OCTET STRING | Last recorded error message |
| .2.3 | startTime | Counter64 | Service start time (unix seconds) |

### Limits and throttle events (.3)

| OID | Name | Type | Description |
|---|---|---|---|
| .3.1 | maxConnections | Integer | read-write (not applied at runtime) |
| .3.2 | uploadRate | Gauge32 | Effective upload pacing rate in kbps (0 when throttling disabled) |
| .3.3 | downloadRate | Gauge32 | Effective download pacing rate in kbps |
| .3.4 | rateLimitHitsIn | Counter64 | Inbound requests that had to wait for tokens |
| .3.5 | rateLimitHitsOut | Counter64 | Outbound requests that had to wait for tokens |

Byte/packet counters are cumulative since service start and are sampled
from the live data path every `config.metrics.interval`.

## Usage Examples

```bash
# Walk all core metrics
snmpwalk -v2c -c public localhost:10162 .1.3.6.1.4.1.54321.1

# Tunnel state branch
snmpwalk -v2c -c public localhost:10162 .1.3.6.1.4.1.54321.1.2

# Single value: bytes received
snmpget -v2c -c public localhost:10162 .1.3.6.1.4.1.54321.1.1.0

# Watch throughput every 5 seconds
watch -n 5 'snmpwalk -v2c -c public localhost:10162 .1.3.6.1.4.1.54321.1'
```

## Prometheus Integration

Two supported paths (see `monitoring/`):

1. **Direct scrape (recommended)**: run with
   `monitor.type: prometheus` and scrape `/metrics` on the configured
   port. Metric families use the `sssonector_*` prefix.
2. **SNMP exporter**: use `monitoring/snmp.yml` as the exporter module and
   the `SSSonector-snmp` job in `monitoring/prometheus.yml`, which maps the
   enterprise OIDs above to `sssonector_snmp_*` metrics.
