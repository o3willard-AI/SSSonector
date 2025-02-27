# System Monitoring Service

## Overview
The SSSonector System Monitoring Service is a systemd-based continuous monitoring solution that collects and tracks system metrics across both server and client deployments. This service provides real-time insights into system resource utilization and performance metrics.

## Features
- Systemd service integration for reliable operation
- Automatic startup on system boot
- Continuous metrics collection
- Automatic log rotation
- Multi-system deployment support
- Standardized metrics format

## Metrics Collected
The monitoring service tracks the following system metrics:

### CPU Metrics
- User CPU usage (%)
- System CPU usage (%)
- CPU idle time (%)

### Memory Metrics
- Total memory (MB)
- Used memory (MB)
- Free memory (MB)

### Disk Usage
- Total disk space (MB)
- Used disk space (MB)
- Free disk space (MB)

### Network Traffic
- Received bytes (rx_bytes)
- Transmitted bytes (tx_bytes)

## Directory Structure
```bash
/opt/sssonector/
└── tools/
    └── metrics/           # Metrics storage directory
        └── *.metrics     # Metric files with timestamp
```

## Service Configuration
The monitoring service is configured through a systemd service unit:

```ini
[Unit]
Description=SSSonector System Monitoring Service
After=network.target

[Service]
Type=simple
User=sblanken
ExecStart=/opt/sssonector/tools/monitor.sh -c /opt/sssonector/config/monitor.ini -d
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

## Metrics File Format
Each metrics file follows a standardized format:
```
cpu_user=<value>
cpu_system=<value>
cpu_idle=<value>
memory_total=<value>
memory_used=<value>
memory_free=<value>
disk_total=<value>
disk_used=<value>
disk_free=<value>
network_rx_bytes=<value>
network_tx_bytes=<value>
```

## Service Management

### Starting the Service
```bash
sudo systemctl start sssonector-monitor
```

### Stopping the Service
```bash
sudo systemctl stop sssonector-monitor
```

### Checking Service Status
```bash
sudo systemctl status sssonector-monitor
```

### Enabling Auto-start
```bash
sudo systemctl enable sssonector-monitor
```

## Log Management
- Service logs are available through journalctl
- Metrics files are automatically rotated to prevent disk space issues
- Old metrics files are cleaned up periodically

## Monitoring Multiple Systems
The monitoring service can be deployed across multiple systems in your SSSonector infrastructure:
- Server systems
- Client systems
- Relay nodes
- Gateway systems

## Integration Points
- Metrics can be consumed by external monitoring systems
- Data format is compatible with common analysis tools
- Easy integration with visualization platforms

## Best Practices
1. Regular log rotation to manage disk space
2. Periodic validation of metrics accuracy
3. Monitor service health through systemd status
4. Review metrics collection frequency based on system load
5. Maintain consistent timezone settings across systems

## Troubleshooting
Common issues and solutions:

1. Service fails to start:
   - Check service logs: `journalctl -u sssonector-monitor`
   - Verify permissions on metrics directory
   - Ensure monitor.sh is executable

2. Missing metrics:
   - Verify service is running
   - Check disk space
   - Validate monitor.ini configuration

3. Incorrect metrics:
   - Verify system time synchronization
   - Check for system resource constraints
   - Validate monitoring script permissions

## Security Considerations
- Service runs with minimal required permissions
- Metrics files have restricted access (644)
- No sensitive data is collected
- Systemd service runs under dedicated user account
