#!/bin/bash

# Setup SNMP monitoring cron job
SCRIPT_DIR="/home/sblanken/Desktop"
MONITOR_SCRIPT="$SCRIPT_DIR/monitor_snmp_health.exp"
LOG_DIR="$SCRIPT_DIR/snmp_logs"

# Create logs directory if it doesn't exist
mkdir -p "$LOG_DIR"

# Create a wrapper script that includes proper environment
cat > "$SCRIPT_DIR/run_snmp_monitor.sh" << 'EOF'
#!/bin/bash
SCRIPT_DIR="/home/sblanken/Desktop"
LOG_DIR="$SCRIPT_DIR/snmp_logs"

# Run the monitor script
$SCRIPT_DIR/monitor_snmp_health.exp

# Archive logs older than 7 days
find "$LOG_DIR" -name "snmp_health_*.log" -mtime +7 -exec gzip {} \;

# Delete logs older than 30 days
find "$LOG_DIR" -name "snmp_health_*.log.gz" -mtime +30 -delete
EOF

# Make wrapper script executable
chmod +x "$SCRIPT_DIR/run_snmp_monitor.sh"

# Add cron job to run every 5 minutes
(crontab -l 2>/dev/null; echo "*/5 * * * * $SCRIPT_DIR/run_snmp_monitor.sh") | crontab -

echo "SNMP monitoring has been set up:"
echo "- Monitor script: $MONITOR_SCRIPT"
echo "- Logs directory: $LOG_DIR"
echo "- Running every 5 minutes"
echo "- Logs are compressed after 7 days"
echo "- Logs are deleted after 30 days"
echo
echo "To view current monitoring status:"
echo "tail -f $LOG_DIR/snmp_health_*.log"
