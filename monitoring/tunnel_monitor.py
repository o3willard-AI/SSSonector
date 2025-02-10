#!/usr/bin/env python3

import curses
import time
import subprocess
import json
import logging
from datetime import datetime

# Configure logging
logging.basicConfig(
    level=logging.DEBUG,
    format='%(asctime)s - %(levelname)s - %(message)s',
    filename='tunnel_monitor.log'
)

def get_snmp_value(config, oid):
    try:
        # Build SNMP command with proper escaping
        community = config['community'].strip()
        cmd = [
            "snmpget",
            "-v1",
            "-c", community,
            "-t", "1",
            "-r", "1",
            f"{config['host']}:{config['port']}",
            oid
        ]
        
        logging.debug(f"Executing SNMP command: {' '.join(cmd)}")
        
        result = subprocess.run(cmd, capture_output=True, text=True)
        logging.debug(f"SNMP command output: {result.stdout.strip()}")
        logging.debug(f"SNMP command error: {result.stderr.strip()}")
        
        if result.returncode == 0:
            output = result.stdout.strip()
            # Handle different SNMP types
            if "Counter64:" in output:
                value = output.split("Counter64: ")[-1]
                return int(value)
            elif "Gauge32:" in output:
                value = output.split("Gauge32: ")[-1]
                return int(value)
            elif "INTEGER:" in output:
                value = output.split("INTEGER: ")[-1]
                return int(value)
            # Add more type handlers as needed
            logging.warning(f"Unknown SNMP type in output: {output}")
        else:
            logging.error(f"SNMP command failed with return code {result.returncode}")
        return 0
    except Exception as e:
        logging.error(f"SNMP Error ({config['host']}:{config['port']}): {str(e)}")
        return 0

def format_bytes(bytes):
    for unit in ['B', 'KB', 'MB', 'GB']:
        if bytes < 1024:
            return f"{bytes:.2f} {unit}"
        bytes /= 1024
    return f"{bytes:.2f} TB"

def calculate_rate(current, previous, interval):
    if previous is None:
        return 0
    return (current - previous) / interval

def main(stdscr):
    # Initialize curses
    curses.start_color()
    curses.init_pair(1, curses.COLOR_GREEN, curses.COLOR_BLACK)
    curses.init_pair(2, curses.COLOR_RED, curses.COLOR_BLACK)
    curses.init_pair(3, curses.COLOR_YELLOW, curses.COLOR_BLACK)
    curses.curs_set(0)
    stdscr.nodelay(1)

    # SNMP Configuration for server and client
    server_config = {
        "host": "192.168.50.210",
        "port": "10161",
        "community": "public"
    }
    client_config = {
        "host": "192.168.50.211",
        "port": "10162",
        "community": "public"
    }
    oids = {
        "bytes_in": ".1.3.6.1.4.1.54321.1.1",     # bytesInOID (Counter64)
        "bytes_out": ".1.3.6.1.4.1.54321.1.2",    # bytesOutOID (Counter64)
        "active_conns": ".1.3.6.1.4.1.54321.1.7", # connectionsOID (Gauge32)
        "cpu_usage": ".1.3.6.1.4.1.54321.1.8",    # cpuUsageOID (Gauge32)
        "memory_usage": ".1.3.6.1.4.1.54321.1.9"  # memoryUsageOID (Gauge32)
    }

    logging.info("Starting tunnel monitor")
    logging.info(f"Server config: {server_config}")
    logging.info(f"Client config: {client_config}")

    # Initialize previous values for rate calculation
    prev_server_bytes_in = None
    prev_server_bytes_out = None
    prev_client_bytes_in = None
    prev_client_bytes_out = None
    start_time = time.time()
    update_interval = 1.0  # seconds

    while True:
        try:
            current_time = time.time()
            elapsed = current_time - start_time

            # Get server values
            logging.debug("Fetching server metrics")
            server_bytes_in = get_snmp_value(server_config, oids["bytes_in"])
            server_bytes_out = get_snmp_value(server_config, oids["bytes_out"])
            server_conns = get_snmp_value(server_config, oids["active_conns"])
            server_cpu = get_snmp_value(server_config, oids["cpu_usage"])
            server_mem = get_snmp_value(server_config, oids["memory_usage"])

            # Get client values
            logging.debug("Fetching client metrics")
            client_bytes_in = get_snmp_value(client_config, oids["bytes_in"])
            client_bytes_out = get_snmp_value(client_config, oids["bytes_out"])
            client_conns = get_snmp_value(client_config, oids["active_conns"])
            client_cpu = get_snmp_value(client_config, oids["cpu_usage"])
            client_mem = get_snmp_value(client_config, oids["memory_usage"])

            # Calculate rates
            server_in_rate = calculate_rate(server_bytes_in, prev_server_bytes_in, update_interval)
            server_out_rate = calculate_rate(server_bytes_out, prev_server_bytes_out, update_interval)
            client_in_rate = calculate_rate(client_bytes_in, prev_client_bytes_in, update_interval)
            client_out_rate = calculate_rate(client_bytes_out, prev_client_bytes_out, update_interval)

            # Update screen
            stdscr.clear()
            
            # Title
            stdscr.addstr(0, 0, "SSSonector Tunnel Monitor", curses.A_BOLD)
            stdscr.addstr(1, 0, "=" * 50)
            
            # Server Statistics
            stdscr.addstr(3, 0, "Server Statistics:", curses.A_BOLD)
            stdscr.addstr(4, 2, f"Bytes In:  {format_bytes(server_bytes_in)} (Rate: {format_bytes(server_in_rate)}/s)")
            stdscr.addstr(5, 2, f"Bytes Out: {format_bytes(server_bytes_out)} (Rate: {format_bytes(server_out_rate)}/s)")
            stdscr.addstr(6, 2, f"Active Connections: {server_conns}")
            
            # Server Resource Usage
            color = curses.color_pair(1)  # Default green
            if server_cpu > 80:
                color = curses.color_pair(2)  # Red
            elif server_cpu > 60:
                color = curses.color_pair(3)  # Yellow
            stdscr.addstr(7, 2, f"CPU Usage: {server_cpu}%", color)
            
            color = curses.color_pair(1)  # Default green
            if server_mem > 80:
                color = curses.color_pair(2)  # Red
            elif server_mem > 60:
                color = curses.color_pair(3)  # Yellow
            stdscr.addstr(8, 2, f"Memory Usage: {server_mem}%", color)

            # Client Statistics
            stdscr.addstr(10, 0, "Client Statistics:", curses.A_BOLD)
            stdscr.addstr(11, 2, f"Bytes In:  {format_bytes(client_bytes_in)} (Rate: {format_bytes(client_in_rate)}/s)")
            stdscr.addstr(12, 2, f"Bytes Out: {format_bytes(client_bytes_out)} (Rate: {format_bytes(client_out_rate)}/s)")
            stdscr.addstr(13, 2, f"Active Connections: {client_conns}")
            
            # Client Resource Usage
            color = curses.color_pair(1)  # Default green
            if client_cpu > 80:
                color = curses.color_pair(2)  # Red
            elif client_cpu > 60:
                color = curses.color_pair(3)  # Yellow
            stdscr.addstr(14, 2, f"CPU Usage: {client_cpu}%", color)
            
            color = curses.color_pair(1)  # Default green
            if client_mem > 80:
                color = curses.color_pair(2)  # Red
            elif client_mem > 60:
                color = curses.color_pair(3)  # Yellow
            stdscr.addstr(15, 2, f"Memory Usage: {client_mem}%", color)
            
            # Runtime
            stdscr.addstr(17, 0, f"Runtime: {int(elapsed)}s")
            stdscr.addstr(18, 0, f"Last Update: {datetime.now().strftime('%H:%M:%S')}")
            
            # Instructions
            stdscr.addstr(20, 0, "Press 'q' to quit", curses.A_DIM)

            stdscr.refresh()

            # Store previous values
            prev_server_bytes_in = server_bytes_in
            prev_server_bytes_out = server_bytes_out
            prev_client_bytes_in = client_bytes_in
            prev_client_bytes_out = client_bytes_out

            # Check for quit
            c = stdscr.getch()
            if c == ord('q'):
                break

            time.sleep(update_interval)

        except KeyboardInterrupt:
            logging.info("Received keyboard interrupt, shutting down")
            break
        except curses.error:
            pass
        except Exception as e:
            logging.error(f"Unexpected error: {str(e)}")

if __name__ == "__main__":
    try:
        curses.wrapper(main)
    except KeyboardInterrupt:
        pass
    finally:
        logging.info("Tunnel monitor stopped")
