# SNMP Monitoring Guide

This guide provides comprehensive documentation for setting up and using SNMP monitoring with SSSonector v2.0.0.

## Overview

SSSonector provides SNMP monitoring capabilities for real-time metrics collection and system monitoring. The SNMP implementation supports:
- SNMPv2c protocol
- Custom MIB for SSSonector-specific metrics
- Rate limiting statistics
- Connection tracking
- System resource utilization
- Tunnel performance metrics

## MIB Structure

### Root OID
- Enterprise Root: .1.3.6.1.4.1.54321
- SSSonector Branch: .1.3.6.1.4.1.54321.1

### Metric Categories

1. System Information (.1)
```
.1.1 - Version
.1.2 - Uptime
.1.3 - Mode (Server/Client)
.1.4 - Build Info
```

2. Network Metrics (.2)
```
.2.1 - Bytes In
.2.2 - Bytes Out
.2.3 - Packets In
.2.4 - Packets Out
.2.5 - Current Connections
.2.6 - Peak Connections
```

3. Rate Limiting (.3)
```
.3.1 - Current Rate In
.3.2 - Current Rate Out
.3.3 - Rate Limit
.3.4 - Burst Limit
.3.5 - Rate Limit Hits
```

4. Error Statistics (.4)
```
.4.1 - Total Errors
.4.2 - Connection Errors
.4.3 - Protocol Errors
.4.4 - Rate Limit Errors
```

5. Resource Usage (.5)
```
.5.1 - CPU Usage
.5.2 - Memory Usage
.5.3 - Open Files
.5.4 - Goroutines
```

## Configuration Examples

### Example 1: Basic SNMP Monitoring
```yaml
monitor:
  enabled: true
  snmp_enabled: true
  snmp_address: "0.0.0.0"
  snmp_port: 10161
  snmp_community: "public"
  update_interval: 30
```

### Example 2: Detailed Monitoring with Rate Limiting
```yaml
monitor:
  enabled: true
  snmp_enabled: true
  snmp_address: "0.0.0.0"
  snmp_port: 10161
  snmp_community: "public"
  update_interval: 10
  detailed_logging: true
  metrics:
    include_system: true
    include_network: true
    include_errors: true
    include_resources: true
  rate_limiting:
    track_individual: true
    track_aggregate: true
    history_size: 3600
```

### Example 3: High-Performance Monitoring
```yaml
monitor:
  enabled: true
  snmp_enabled: true
  snmp_address: "0.0.0.0"
  snmp_port: 10161
  snmp_community: "public"
  update_interval: 5
  buffer_size: 10000
  metrics:
    include_system: true
    include_network: true
    include_errors: true
    include_resources: true
  performance:
    metric_buffer: 1000
    async_updates: true
    batch_size: 100
```

## Usage Examples

### Basic Metrics Collection

1. Get system information:
```bash
# Version information
snmpget -v2c -c public localhost:10161 .1.3.6.1.4.1.54321.1.1.1.0

# Uptime
snmpget -v2c -c public localhost:10161 .1.3.6.1.4.1.54321.1.1.2.0
```

2. Monitor network metrics:
```bash
# Current throughput
snmpwalk -v2c -c public localhost:10161 .1.3.6.1.4.1.54321.1.2

# Watch network metrics (updates every 5 seconds)
watch -n 5 'snmpwalk -v2c -c public localhost:10161 .1.3.6.1.4.1.54321.1.2'
```

### Rate Limiting Monitoring

1. Monitor current rates:
```bash
# Get current rate information
snmpwalk -v2c -c public localhost:10161 .1.3.6.1.4.1.54321.1.3

# Monitor rate limiting hits
snmpget -v2c -c public localhost:10161 .1.3.6.1.4.1.54321.1.3.5.0
```

2. Track rate limit violations:
```bash
# Set up trap receiver
snmptrapd -f -Lo -c /etc/snmp/snmptrapd.conf

# Configure rate limit alerts
snmpset -v2c -c public localhost:10161 .1.3.6.1.4.1.54321.1.3.6.0 i 90
```

### Resource Monitoring

1. System resource usage:
```bash
# Get all resource metrics
snmpwalk -v2c -c public localhost:10161 .1.3.6.1.4.1.54321.1.5

# Monitor CPU usage
watch -n 1 'snmpget -v2c -c public localhost:10161 .1.3.6.1.4.1.54321.1.5.1.0'
```

2. Connection tracking:
```bash
# Get current connections
snmpget -v2c -c public localhost:10161 .1.3.6.1.4.1.54321.1.2.5.0

# Monitor connection changes
snmptrapd -f -Lo -n -c /etc/snmp/snmptrapd.conf
```

## Integration Examples

### Prometheus Integration

1. SNMP Exporter configuration:
```yaml
modules:
  sssonector:
    walk:
      - 1.3.6.1.4.1.54321.1
    metrics:
      - name: sssonector_bytes_in
        oid: 1.3.6.1.4.1.54321.1.2.1
        type: counter
      - name: sssonector_bytes_out
        oid: 1.3.6.1.4.1.54321.1.2.2
        type: counter
      - name: sssonector_current_rate
        oid: 1.3.6.1.4.1.54321.1.3.1
        type: gauge
```

2. Prometheus configuration:
```yaml
scrape_configs:
  - job_name: 'sssonector'
    static_configs:
      - targets: ['localhost:10161']
    metrics_path: /snmp
    params:
      module: [sssonector]
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_target
      - source_labels: [__param_target]
        target_label: instance
      - target_label: __address__
        replacement: localhost:9116
```

### Grafana Dashboard

1. Import dashboard configuration:
```bash
curl -X POST -H "Content-Type: application/json" -d @sssonector-dashboard.json http://localhost:3000/api/dashboards/db
```

2. Dashboard panels:
```json
{
  "panels": [
    {
      "title": "Network Throughput",
      "type": "graph",
      "targets": [
        {
          "expr": "rate(sssonector_bytes_in[5m])",
          "legendFormat": "Bytes In"
        },
        {
          "expr": "rate(sssonector_bytes_out[5m])",
          "legendFormat": "Bytes Out"
        }
      ]
    },
    {
      "title": "Rate Limiting",
      "type": "gauge",
      "targets": [
        {
          "expr": "sssonector_current_rate",
          "legendFormat": "Current Rate"
        }
      ]
    }
  ]
}
```

## Troubleshooting

### Common Issues

1. SNMP Connection Refused
```bash
# Check if SNMP agent is running
ps aux | grep snmp

# Verify port is open
netstat -an | grep 10161

# Test SNMP connectivity
snmpwalk -v2c -c public localhost:10161 .1.3.6.1.2.1.1
```

2. Missing Metrics
```bash
# Check metric configuration
snmpwalk -v2c -c public localhost:10161 .1.3.6.1.4.1.54321.1

# Verify update interval
snmpget -v2c -c public localhost:10161 .1.3.6.1.4.1.54321.1.1.5.0
```

3. Performance Issues
```bash
# Check SNMP response times
time snmpget -v2c -c public localhost:10161 .1.3.6.1.4.1.54321.1.1.1.0

# Monitor SNMP traffic
tcpdump -i any port 10161
```

## Best Practices

1. Security
   - Change default community strings
   - Use non-standard ports
   - Implement access controls
   - Regular security audits

2. Performance
   - Adjust update intervals based on needs
   - Use bulk queries when possible
   - Monitor agent resource usage
   - Implement rate limiting for queries

3. Monitoring
   - Set up alerting thresholds
   - Monitor agent health
   - Regular metric validation
   - Backup monitoring configuration

4. Integration
   - Use standard monitoring tools
   - Implement redundancy
   - Regular backup of dashboards
   - Document custom configurations
