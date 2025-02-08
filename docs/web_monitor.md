# SSSonector Web Monitor Documentation

## Overview

The SSSonector Web Monitor is a Flask-based application that provides real-time visualization of SNMP metrics. It integrates with the Net-SNMP daemon to collect and display performance metrics, connection statistics, and system status information.

## Architecture

### 1. Components

```
web_monitor/
├── snmp_web_monitor.py    # Main application
├── static/
│   └── styles.css        # CSS styling
└── templates/
    └── index.html        # Main dashboard template
```

### 2. Implementation Details

#### Main Application (snmp_web_monitor.py)
```python
# Core components
from flask import Flask, render_template_string
import subprocess
import json
from datetime import datetime
import threading
import time

# Global metrics store
metrics = {
    'throughput': {'rx': 0, 'tx': 0},
    'connections': 0,
    'latency': 0.0,
    'last_update': None
}

# SNMP interaction
def get_snmp_value(oid):
    """Get SNMP value for given OID"""
    try:
        result = subprocess.run(
            ['snmpget', '-v2c', '-c', 'public', 'localhost', oid],
            capture_output=True, text=True, check=True
        )
        value = result.stdout.strip().split('=')[-1].strip()
        return value.strip('"')
    except subprocess.CalledProcessError as e:
        print(f"SNMP error for OID {oid}: {e}")
        return None

# Metrics collection
def update_metrics():
    """Update metrics from SNMP"""
    while True:
        try:
            # Get throughput
            throughput = get_snmp_value(
                'NET-SNMP-EXTEND-MIB::nsExtendOutput1Line."sssonector-throughput"'
            )
            if throughput:
                rx, tx = map(int, throughput.split(':'))
                metrics['throughput']['rx'] = rx * 8 / (1024 * 1024)  # Mbps
                metrics['throughput']['tx'] = tx * 8 / (1024 * 1024)  # Mbps

            # Get connections
            connections = get_snmp_value(
                'NET-SNMP-EXTEND-MIB::nsExtendOutput1Line."sssonector-connections"'
            )
            if connections:
                metrics['connections'] = int(connections)

            # Get latency
            latency = get_snmp_value(
                'NET-SNMP-EXTEND-MIB::nsExtendOutput1Line."sssonector-latency"'
            )
            if latency:
                metrics['latency'] = float(latency)

            metrics['last_update'] = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        except Exception as e:
            print(f"Error updating metrics: {e}")

        time.sleep(5)  # Update interval
```

### 3. User Interface

#### Dashboard Template
```html
<!DOCTYPE html>
<html>
<head>
    <title>SSSonector SNMP Monitor</title>
    <meta http-equiv="refresh" content="5">
    <style>
        body {
            font-family: Arial, sans-serif;
            margin: 40px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 800px;
            margin: 0 auto;
            background-color: white;
            padding: 20px;
            border-radius: 8px;
            box-shadow: 0 2px 4px rgba(0,0,0,0.1);
        }
        .metric {
            margin: 20px 0;
            padding: 15px;
            background-color: #f8f9fa;
            border-radius: 4px;
        }
        .value {
            font-size: 1.8em;
            color: #007bff;
            font-weight: bold;
        }
        .unit {
            color: #666;
            font-size: 0.9em;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>SSSonector SNMP Monitor</h1>
        
        <div class="metric">
            <h2>Throughput</h2>
            <div class="value">
                RX: {{ metrics['throughput']['rx'] | round(2) }}
                <span class="unit">Mbps</span><br>
                TX: {{ metrics['throughput']['tx'] | round(2) }}
                <span class="unit">Mbps</span>
            </div>
        </div>

        <div class="metric">
            <h2>Active Connections</h2>
            <div class="value">
                {{ metrics['connections'] }}
                <span class="unit">connections</span>
            </div>
        </div>

        <div class="metric">
            <h2>Latency</h2>
            <div class="value">
                {{ metrics['latency'] | round(2) }}
                <span class="unit">ms</span>
            </div>
        </div>

        <div class="timestamp">
            Last updated: {{ metrics['last_update'] }}
        </div>
    </div>
</body>
</html>
```

## Configuration

### 1. SNMP Integration

#### Required SNMP Extensions
```bash
# /etc/snmp/snmpd.conf
extend sssonector-throughput /usr/local/bin/sssonector-snmp throughput
extend sssonector-connections /usr/local/bin/sssonector-snmp connections
extend sssonector-latency /usr/local/bin/sssonector-snmp latency
```

#### Metric Collection Scripts
```bash
#!/bin/bash
# /usr/local/bin/sssonector-snmp

case "$1" in
    "throughput")
        interface="enp0s3"
        rx_bytes=$(cat /sys/class/net/$interface/statistics/rx_bytes)
        tx_bytes=$(cat /sys/class/net/$interface/statistics/tx_bytes)
        echo "$rx_bytes:$tx_bytes"
        ;;
    "connections")
        netstat -an | grep "ESTABLISHED" | wc -l
        ;;
    "latency")
        ping -c 1 192.168.50.210 | grep "time=" | cut -d "=" -f 4
        ;;
    *)
        echo "Unknown metric: $1"
        exit 1
        ;;
esac
```

### 2. Web Server Configuration

#### Development Server
```python
if __name__ == '__main__':
    # Start metrics update thread
    update_thread = threading.Thread(target=update_metrics, daemon=True)
    update_thread.start()
    
    # Start Flask server
    app.run(host='0.0.0.0', port=8080)
```

#### Production Deployment
```ini
# /etc/systemd/system/sssonector-web-monitor.service
[Unit]
Description=SSSonector Web Monitor
After=network.target

[Service]
User=sssonector
WorkingDirectory=/opt/sssonector/web_monitor
ExecStart=/usr/bin/python3 snmp_web_monitor.py
Restart=always

[Install]
WantedBy=multi-user.target
```

## Usage

### 1. Starting the Monitor
```bash
# Development
python3 snmp_web_monitor.py

# Production
sudo systemctl start sssonector-web-monitor
```

### 2. Accessing the Dashboard
- URL: http://localhost:8080
- Auto-refresh interval: 5 seconds
- Metrics displayed:
  * Throughput (RX/TX in Mbps)
  * Active connections
  * Latency (ms)

## Troubleshooting

### 1. SNMP Issues

#### Connection Problems
```bash
# Verify SNMP daemon
systemctl status snmpd

# Test SNMP queries
snmpwalk -v2c -c public localhost NET-SNMP-EXTEND-MIB::nsExtendOutput1Line

# Check SNMP logs
tail -f /var/log/snmp/snmpd.log
```

#### Metric Collection Issues
```bash
# Test extend scripts directly
/usr/local/bin/sssonector-snmp throughput
/usr/local/bin/sssonector-snmp connections
/usr/local/bin/sssonector-snmp latency

# Verify script permissions
ls -l /usr/local/bin/sssonector-snmp
```

### 2. Web Monitor Issues

#### Service Problems
```bash
# Check service status
systemctl status sssonector-web-monitor

# View logs
journalctl -u sssonector-web-monitor -n 100

# Test Flask server
curl http://localhost:8080
```

#### Metric Update Issues
```bash
# Check Python process
ps aux | grep snmp_web_monitor

# Monitor metrics updates
tail -f /var/log/sssonector/web_monitor.log
```

## Future Improvements

1. Enhanced Visualization
   - Historical data graphs
   - Real-time charts
   - Custom time ranges

2. Additional Metrics
   - System resource usage
   - Error rates
   - Tunnel status

3. Authentication
   - User login
   - Role-based access
   - API keys for automation

4. Alert Integration
   - Email notifications
   - Slack integration
   - Custom thresholds
