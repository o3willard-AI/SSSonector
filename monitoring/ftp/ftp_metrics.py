#!/usr/bin/env python3
import psutil
import socket
import time
import sys

def get_ftp_connections():
    try:
        count = 0
        for conn in psutil.net_connections():
            if conn.laddr and conn.laddr.port == 21:
                count += 1
        return str(count)
    except:
        return "0"

def get_network_throughput():
    try:
        net_io = psutil.net_io_counters()
        return f"{net_io.bytes_recv}:{net_io.bytes_sent}"
    except:
        return "0:0"

def get_latency():
    try:
        start = time.time()
        sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
        sock.settimeout(1)
        sock.connect(('localhost', 21))
        latency = (time.time() - start) * 1000
        sock.close()
        return f"{latency:.2f}"
    except:
        return "0"

if __name__ == "__main__":
    if len(sys.argv) < 2:
        sys.exit(1)

    metric = sys.argv[1]
    if metric == "connections":
        print(get_ftp_connections())
    elif metric == "throughput":
        print(get_network_throughput())
    elif metric == "latency":
        print(get_latency())
    else:
        sys.exit(1)
