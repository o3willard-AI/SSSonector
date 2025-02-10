#!/bin/bash
SCRIPT_DIR="/home/sblanken/Desktop"
LOG_DIR="$SCRIPT_DIR/snmp_logs"

# Run the monitor script
$SCRIPT_DIR/monitor_snmp_health.exp

# Archive logs older than 7 days
find "$LOG_DIR" -name "snmp_health_*.log" -mtime +7 -exec gzip {} \;

# Delete logs older than 30 days
find "$LOG_DIR" -name "snmp_health_*.log.gz" -mtime +30 -delete
