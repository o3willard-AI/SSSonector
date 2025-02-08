#!/usr/bin/env python3
from flask import Flask, render_template_string
import subprocess
from datetime import datetime
import threading
import time

app = Flask(__name__)

metrics = {
    'throughput': {'rx': 0, 'tx': 0},
    'connections': 0,
    'latency': 0.0,
    'last_update': None
}

HTML_TEMPLATE = '''
<!DOCTYPE html>
<html>
<head>
    <title>SSSonector SNMP Monitor</title>
    <meta http-equiv="refresh" content="5">
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; background-color: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; background-color: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        h1 { color: #333; border-bottom: 2px solid #eee; padding-bottom: 10px; }
        .metric { margin: 20px 0; padding: 15px; background-color: #f8f9fa; border-radius: 4px; }
        .metric h2 { margin: 0 0 10px 0; color: #666; font-size: 1.2em; }
        .value { font-size: 1.8em; color: #007bff; font-weight: bold; }
        .unit { color: #666; font-size: 0.9em; }
        .timestamp { color: #999; font-size: 0.8em; margin-top: 20px; text-align: right; }
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

def run_command(cmd):
    try:
        result = subprocess.run(cmd, shell=True, capture_output=True, text=True, check=True)
        return result.stdout.strip()
    except subprocess.CalledProcessError as e:
        print(f"Error running command '{cmd}': {e}")
        return None

def update_metrics():
    while True:
        try:
            # Get throughput using sssonector-snmp script
            throughput = run_command('/usr/local/bin/sssonector-snmp throughput')
            if throughput:
                rx, tx = map(int, throughput.split(':'))
                metrics['throughput']['rx'] = rx * 8 / (1024 * 1024)  # Convert to Mbps
                metrics['throughput']['tx'] = tx * 8 / (1024 * 1024)  # Convert to Mbps

            # Get connections
            connections = run_command('/usr/local/bin/sssonector-snmp connections')
            if connections:
                metrics['connections'] = int(connections)

            # Get latency
            latency = run_command('/usr/local/bin/sssonector-snmp latency')
            if latency:
                metrics['latency'] = float(latency)

            metrics['last_update'] = datetime.now().strftime('%Y-%m-%d %H:%M:%S')
        except Exception as e:
            print(f"Error updating metrics: {e}")

        time.sleep(5)

@app.route('/')
def index():
    return render_template_string(HTML_TEMPLATE, metrics=metrics)

def main():
    update_thread = threading.Thread(target=update_metrics, daemon=True)
    update_thread.start()
    app.run(host='0.0.0.0', port=8080)

if __name__ == '__main__':
    main()
