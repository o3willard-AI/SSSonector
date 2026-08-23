# Rate Limiting Configuration Guide

Rate limiting in SSSonector paces tunnel traffic in both directions using a
reservation/"debt" token bucket (`internal/throttle`). Pacing is enforced on
the live data path and covered by deterministic timing tests.

## Configuration

```yaml
throttle:
  enabled: true     # master switch (hot-reloadable)
  rate: 1048576     # sustained limit in bytes/sec (hot-reloadable)
  burst: 104858     # accepted but unused - see below
```

### Effective Rate

The limiter accounts for TCP/IP overhead:

```
effective_rate = rate * 1.1   # TCPOverheadFactor
burst          = effective_rate * 0.1   # always 100ms of effective rate
```

Example: `rate: 1048576` paces at ~1.05 MB/s with a burst allowance of
~115 KB. The `throttle.burst` key is accepted for schema compatibility but
is **not used** by the limiter.

### Behavior Details

- Buckets start empty: throughput is paced from the first byte.
- Requests larger than the burst are funded over multiple intervals rather
  than rejected.
- Read and write directions have independent buckets.
- Wait-timeouts abort the transfer with an error (logged at WARN) instead of
  blocking indefinitely.
- No sleeps are ever taken while holding locks, so one slow connection
  cannot head-of-line-block others sharing a bucket.

## Hot Reload

Both `enabled` and `rate` can be changed at runtime:

```bash
# edit config, then:
kill -HUP $(pidof sssonector)
```

Live transfers pick up new rates immediately; new connections pick them up
automatically. See [Hot Reload](../hot_reload.md) for the full reload
contract.

## Monitoring

Current counters are exposed through both observability paths:

- Prometheus (`monitor.type: prometheus`): `sssonector_bytes_in_total`,
  `sssonector_bytes_out_total`, `sssonector_errors_total`
- SNMP (`.1.3.6.1.4.1.54321` MIB): bytesIn/bytesOut counters and the
  configured rate-limit gauges (upload/download kbps)

There is deliberately no per-client or dynamic-adjustment rate limiting;
every connection sharing a role shares the configured budget.

## Validation

`throttle.rate` must be a positive number. Invalid values fail startup
validation and reject SIGHUP reloads.

## Testing Throughput

```bash
iperf3 -c <server> -t 30 -J    # compare against expected effective rate
```
