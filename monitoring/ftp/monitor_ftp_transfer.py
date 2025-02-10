#!/usr/bin/env python3
import time
import subprocess
import json
from http.server import SimpleHTTPRequestHandler, HTTPServer
from threading import Thread

class TransferMonitor:
    def __init__(self):
        self.metrics = {
            'server_rx': 0,
            'server_tx': 0,
            'client_rx': 0,
            'client_tx': 0,
            'server_connections': 0,
            'client_connections': 0,
            'server_latency': 0,
            'client_latency': 0,
            'timestamp': 0
        }
        self.history = []

    def get_snmp_value(self, host, oid):
        try:
            result = subprocess.run(['snmpget', '-v2c', '-c', 'public', host, oid], 
                                 capture_output=True, text=True)
            if result.returncode == 0:
                return result.stdout.strip().split('STRING: ')[-1]
            return "0"
        except:
            return "0"

    def update_metrics(self):
        while True:
            # Get server metrics
            server_throughput = self.get_snmp_value('192.168.50.210', 
                'NET-SNMP-EXTEND-MIB::nsExtendOutputFull."sssonector-throughput"')
            server_latency = self.get_snmp_value('192.168.50.210',
                'NET-SNMP-EXTEND-MIB::nsExtendOutputFull."sssonector-latency"')
            server_conns = self.get_snmp_value('192.168.50.210',
                'NET-SNMP-EXTEND-MIB::nsExtendOutputFull."sssonector-connections"')

            # Get client metrics
            client_throughput = self.get_snmp_value('192.168.50.212',
                'NET-SNMP-EXTEND-MIB::nsExtendOutputFull."sssonector-throughput"')
            client_latency = self.get_snmp_value('192.168.50.212',
                'NET-SNMP-EXTEND-MIB::nsExtendOutputFull."sssonector-latency"')
            client_conns = self.get_snmp_value('192.168.50.212',
                'NET-SNMP-EXTEND-MIB::nsExtendOutputFull."sssonector-connections"')

            # Parse throughput values (format: rx_bytes:tx_bytes)
            try:
                server_rx, server_tx = map(int, server_throughput.split(':'))
                client_rx, client_tx = map(int, client_throughput.split(':'))
            except:
                server_rx = server_tx = client_rx = client_tx = 0

            # Update metrics
            self.metrics.update({
                'server_rx': server_rx,
                'server_tx': server_tx,
                'client_rx': client_rx,
                'client_tx': client_tx,
                'server_connections': int(server_conns.split()[0]) if server_conns else 0,
                'client_connections': int(client_conns.split()[0]) if client_conns else 0,
                'server_latency': float(server_latency.split()[0]) if 'ms' in server_latency else 0,
                'client_latency': float(client_latency.split()[0]) if 'ms' in client_latency else 0,
                'timestamp': int(time.time())
            })

            # Keep history for graphs
            self.history.append(dict(self.metrics))
            if len(self.history) > 300:  # Keep 5 minutes of history
                self.history.pop(0)

            time.sleep(1)

class WebHandler(SimpleHTTPRequestHandler):
    def do_GET(self):
        if self.path == '/metrics':
            self.send_response(200)
            self.send_header('Content-type', 'application/json')
            self.send_header('Access-Control-Allow-Origin', '*')
            self.end_headers()
            self.wfile.write(json.dumps({
                'current': monitor.metrics,
                'history': monitor.history
            }).encode())
        else:
            self.send_response(200)
            self.send_header('Content-type', 'text/html')
            self.end_headers()
            with open('monitor_template.html', 'r') as f:
                self.wfile.write(f.read().encode())

if __name__ == '__main__':
    monitor = TransferMonitor()
    
    # Start metrics collection thread
    Thread(target=monitor.update_metrics, daemon=True).start()
    
    # Start web server
    server = HTTPServer(('0.0.0.0', 8080), WebHandler)
    print("Server started at http://localhost:8080")
    server.serve_forever()
