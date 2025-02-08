#!/usr/bin/env python3
from flask import Flask, render_template_string, render_template
import subprocess
import json
from datetime import datetime
import threading
import time

app = Flask(__name__)

# Global metrics store
metrics = {
    'throughput': {'rx': 0, 'tx': 0},
    'connections': 0,
    'latency': 0.0,
    'last_update': None
}

# HTML template with auto-refresh and basic styling
HTML_TEMPLATE = '''
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
        h1 {
            color: #333;
            border-bottom: 2px solid #eee;
            padding-bottom: 10px;
        }
        .metric {
            margin: 20px 0;
            padding: 15px;
            background-color: #f8f9fa;
            border-radius: 4px;
        }
        .metric h2 {
            margin: 0 0 10px 0;
            color: #666;
            font-size: 1.2em;
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
        .timestamp {
            color: #999;
            font-size: 0.8em;
            margin-top: 20px;
            text-align: right;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>SSSonector SNMP Monitor</h1>
        
        <div class="metric">
            <h2>Throughput</h2>
            <div class="value">
                RX: {{ metrics['throughput']['rx'] | round(2) }} <span class="unit">Mbps</span><br>
                TX: {{ metrics['throughput']['tx'] | round(2) }} <span class="unit">Mbps</span>
            </div>
        </div>

        <div class="metric">
            <h2>Active Connections</h2>
            <div class="value">
                {{ metrics['connections'] }} <span class="unit">connections</span>
            </div>
        </div>

        <div class="metric">
            <h2>Latency</h2>
            <div class="value">
                {{ metrics['latency'] | round(2) }} <span class="unit">ms</span>
            </div>
        </div>

        <div class="timestamp">
            Last updated: {{ metrics['last_update'] }}
        </div>
    </div>
</body>
</html>
'''

def get_snmp_value(oid):
    """Get SNMP value for given OID"""
    try:
        result = subprocess.run(
            ['snmpget', '-v2c', '-c', 'public', 'localhost', oid],
            capture_output=True, text=True, check=True
        )
        value = result.stdout.strip().split('=')[-1].strip()
        # Remove quotes if present
        if value.startswith('"') and value.endswith('"'):
            value = value[1:-1]
        return value
    except subprocess.CalledProcessError as e:
        print(f"SNMP error for OID {oid}: {e}")
        return None
    except Exception as e:
        print(f"Unexpected error for OID {oid}: {e}")
        return None

def update_metrics():
    """Update metrics from SNMP"""
    while True:
        try:
            # Get throughput
            throughput = get_snmp_value('NET-SNMP-EXTEND-MIB::nsExtendOutput1Line."sssonector-throughput"')
            if throughput:
                try:
                    rx, tx = map(int, throughput.split(':'))
                    # Convert to Mbps
                    metrics['throughput']['rx'] = rx * 8 / (1024 * 1024)
                    metrics['throughput']['tx'] = tx * 8 / (1024 * 1024)
                except (ValueError, AttributeError) as e:
                    print(f"Error parsing throughput value '{throughput}': {e}")

            # Get connections
            connections = get_snmp_value('NET-SNMP-EXTEND-MIB::nsExtendOutput1Line."sssonector-connections"')
            if connections:
                try:
                    metrics['connections'] = int(connections)
                except ValueError as e:
                    print(f"Error parsing connections value '{connections}': {e}")

            # Get latency
            latency = get_snmp_value('NET-SNMP-EXTEND-MIB::nsExtendOutput1Line."sssonector-latency"')
            if latency:
                try:
                    metrics['latency'] = float(latency)
                except ValueError as e:
                    print(f"Error parsing latency value '{latency}': {e}")

            metrics['last_update'] = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        except Exception as e:
            print(f"Error updating metrics: {e}")

        time.sleep(5)  # Update every 5 seconds

@app.route('/')
def index():
    """Render metrics page"""
    return render_template_string(HTML_TEMPLATE, metrics=metrics)

def main():
    # Start metrics update thread
    update_thread = threading.Thread(target=update_metrics, daemon=True)
    update_thread.start()
    
    # Start Flask server
    app.run(host='0.0.0.0', port=8080)

if __name__ == '__main__':
    main()
