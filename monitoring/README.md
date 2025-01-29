# SSSonector Monitoring

This directory contains configuration files and tools for monitoring SSSonector instances using Prometheus, Grafana, and SNMP.

## Components

- Prometheus: Time series database for storing metrics
- SNMP Exporter: Converts SNMP metrics to Prometheus format
- Grafana: Visualization and alerting platform
- MIB Files: SNMP Management Information Base definitions
- Docker Compose: Easy deployment of the monitoring stack

## Quick Start

1. Install Docker and Docker Compose:
   ```bash
   # Ubuntu/Debian
   sudo apt-get install docker.io docker-compose

   # macOS
   brew install docker docker-compose

   # Windows
   # Download Docker Desktop from https://www.docker.com/products/docker-desktop
   ```

2. Start the monitoring stack:
   ```bash
   cd monitoring
   docker-compose up -d
   ```

3. Access the monitoring interfaces:
   - Grafana: http://localhost:3000 (admin/admin)
   - Prometheus: http://localhost:9090
   - SNMP Exporter: http://localhost:9116

## Configuration

### SSSonector Configuration

Enable SNMP monitoring in your SSSonector configuration:

```yaml
monitor:
  snmp_enabled: true
  snmp_port: 161
  community: "public"
```

### SNMP MIB Installation

1. Copy MIB file to system directory:
   ```bash
   # Linux
   sudo cp ../mibs/SSL-TUNNEL-MIB.txt /usr/share/snmp/mibs/

   # macOS
   sudo cp ../mibs/SSL-TUNNEL-MIB.txt /usr/local/share/snmp/mibs/

   # Windows
   copy ..\mibs\SSL-TUNNEL-MIB.txt C:\usr\share\snmp\mibs\
   ```

2. Update SNMP configuration to load the MIB:
   ```bash
   # Linux/macOS
   echo "mibs +SSL-TUNNEL-MIB" >> ~/.snmp/snmp.conf

   # Windows
   echo mibs +SSL-TUNNEL-MIB >> %USERPROFILE%\.snmp\snmp.conf
   ```

## Available Metrics

1. Performance Metrics:
   - Bytes Received/Sent
   - Packets Lost
   - Latency
   - Connection Status

2. System Metrics:
   - CPU Usage
   - Memory Usage
   - Uptime

## Grafana Dashboards

The default dashboard includes:
- Bandwidth Usage Graph
- Latency Gauge
- Packet Loss Graph
- CPU/Memory Usage Gauges

To import additional dashboards:
1. Go to Grafana (http://localhost:3000)
2. Navigate to Dashboards -> Import
3. Upload JSON file or paste dashboard JSON

## Alerting

Default alerts are configured for:
- High Latency (>500ms)
- Packet Loss Spikes
- High CPU Usage (>90%)
- High Memory Usage (>90%)

To configure alert notifications:
1. Go to Grafana -> Alerting -> Notification channels
2. Add your preferred notification method (Email, Slack, etc.)
3. Update alert rules to use your notification channel

## Testing

Use the provided test script to verify SNMP functionality:

```bash
./scripts/test-snmp.sh

# Test remote server
./scripts/test-snmp.sh -h tunnel.example.com -p 161 -c mycommunity
```

## Troubleshooting

1. SNMP Connection Issues:
   ```bash
   # Check if SNMP agent is responding
   snmpwalk -v2c -c public localhost:161 .1.3.6.1.4.1.2021.10.1.3

   # Check SNMP port is open
   netstat -an | grep 161
   ```

2. Prometheus Issues:
   ```bash
   # Check Prometheus targets
   curl http://localhost:9090/api/v1/targets

   # Check SNMP exporter metrics
   curl http://localhost:9116/metrics
   ```

3. Grafana Issues:
   - Verify Prometheus data source is working
   - Check Grafana logs: `docker-compose logs grafana`
   - Ensure dashboard JSON is valid

## Security Considerations

1. SNMP Security:
   - Change default community string
   - Use firewall rules to restrict SNMP access
   - Consider using SNMPv3 for production

2. Grafana Security:
   - Change default admin password
   - Enable HTTPS
   - Use authentication proxy for production

3. Prometheus Security:
   - Enable authentication
   - Use HTTPS
   - Restrict network access

## Maintenance

1. Log Rotation:
   - Prometheus data is stored in a Docker volume
   - Configure retention in prometheus.yml
   - Monitor disk usage

2. Backup:
   ```bash
   # Backup Grafana data
   docker run --rm --volumes-from SSSonector-grafana \
     -v $(pwd):/backup alpine tar czf /backup/grafana-backup.tar.gz \
     /var/lib/grafana

   # Backup Prometheus data
   docker run --rm --volumes-from SSSonector-prometheus \
     -v $(pwd):/backup alpine tar czf /backup/prometheus-backup.tar.gz \
     /prometheus
   ```

3. Updates:
   ```bash
   # Update containers
   docker-compose pull
   docker-compose up -d

   # Check versions
   docker-compose exec prometheus prometheus --version
   docker-compose exec grafana grafana-server -v
