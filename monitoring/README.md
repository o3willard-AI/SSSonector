# SSSonector Monitoring

This directory contains configuration files and tools for monitoring
SSSonector instances using Prometheus, Grafana, and SNMP.

## Components

- **Prometheus**: time series database. Scrapes the daemon's `/metrics`
  endpoint directly (recommended) and/or the SNMP exporter.
- **SNMP Exporter**: converts the daemon's SNMPv2c MIB (enterprise root
  `1.3.6.1.4.1.54321`, see `internal/monitor/mib.go` and
  `docs/snmp_monitoring.md`) into Prometheus metrics via `./snmp.yml`.
- **Grafana**: visualization; a starter dashboard is provisioned from
  `grafana-dashboard.json`.
- **Docker Compose**: one-command deployment of the stack.

## Quick Start

1. Start the monitoring stack:

   ```bash
   cd monitoring
   docker-compose up -d
   ```

2. Run SSSonector on the host with either:

   ```yaml
   # Direct Prometheus scraping (recommended)
   config:
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
   ```

   or SNMP polling:

   ```yaml
   config:
     monitor:
       enabled: true
       type: snmp
     snmp:
       enabled: true
       address: "0.0.0.0"
       port: 10162
       community: "public"
   ```

3. Access the interfaces:
   - Grafana: http://localhost:3000 (admin/admin)
   - Prometheus: http://localhost:9090
   - SNMP Exporter: http://localhost:9116

Prometheus reaches the host daemon via `host.docker.internal`
(mapped with `extra_hosts: host-gateway` in docker-compose.yml).

## Available Metrics

Direct scrape (`sssonector_*`) — see the `/metrics` endpoint output for the
full list; key families include bytes/packets counters, error totals,
throttle hit/rate/burst gauges, connection gauges, CPU/memory/goroutines,
and uptime.

SNMP exporter path (`sssonector_snmp_*`): bytes in/out, active connections,
CPU, memory, tunnel status, start time, rate limits.

## Grafana Dashboard

The provisioned dashboard includes: bandwidth usage (bytes in/out rates),
cumulative errors, error rate over 5 minutes, and CPU usage.

To import additional dashboards: Grafana -> Dashboards -> Import.

## Health Probes

When the metrics endpoint is enabled, `/healthz` (same port as `/metrics`)
answers with `{"status":"ok","mode":...,"tunnel_state":...,"uptime_seconds":N}`.
Point systemd/k8s probes there instead of process-name checks.

## Testing

```bash
# Verify the agent answers locally (adjust port/community from your config)
snmpwalk -v2c -c public localhost:10162 .1.3.6.1.4.1.54321.1

# Verify direct exposition
curl -s http://localhost:9090/metrics | head
```
