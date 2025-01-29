#!/bin/bash
set -e

# Default values
HOST="localhost"
PORT=161
COMMUNITY="public"

# Help message
show_help() {
    echo "Usage: $0 [options]"
    echo
    echo "Test SSSonector SNMP monitoring"
    echo
    echo "Options:"
    echo "  -h, --host      Host to query (default: localhost)"
    echo "  -p, --port      SNMP port (default: 161)"
    echo "  -c, --community SNMP community string (default: public)"
    echo "  --help          Show this help message"
    echo
    echo "Example:"
    echo "  $0 -h tunnel.example.com -p 161 -c mycommunity"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--host)
            HOST="$2"
            shift 2
            ;;
        -p|--port)
            PORT="$2"
            shift 2
            ;;
        -c|--community)
            COMMUNITY="$2"
            shift 2
            ;;
        --help)
            show_help
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Check if snmpwalk is installed
if ! command -v snmpwalk &> /dev/null; then
    echo "Error: snmpwalk is not installed. Please install net-snmp tools."
    echo "Ubuntu/Debian: sudo apt-get install snmp"
    echo "RHEL/CentOS: sudo yum install net-snmp-utils"
    echo "macOS: brew install net-snmp"
    exit 1
fi

# OIDs to query
OIDS=(
    ".1.3.6.1.4.1.2021.10.1.3.1" # Bytes Received
    ".1.3.6.1.4.1.2021.10.1.3.2" # Bytes Sent
    ".1.3.6.1.4.1.2021.10.1.3.3" # Packets Lost
    ".1.3.6.1.4.1.2021.10.1.3.4" # Latency
    ".1.3.6.1.4.1.2021.10.1.3.5" # Uptime
    ".1.3.6.1.4.1.2021.10.1.3.6" # CPU Usage
    ".1.3.6.1.4.1.2021.10.1.3.7" # Memory Usage
)

# Names for each OID
NAMES=(
    "Bytes Received"
    "Bytes Sent"
    "Packets Lost"
    "Latency (us)"
    "Uptime (s)"
    "CPU Usage"
    "Memory Usage"
)

echo "Testing SNMP monitoring for SSSonector"
echo "Host: $HOST"
echo "Port: $PORT"
echo "Community: $COMMUNITY"
echo

# Query each OID
for i in "${!OIDS[@]}"; do
    echo -n "${NAMES[$i]}: "
    snmpget -v2c -c "$COMMUNITY" "$HOST:$PORT" "${OIDS[$i]}" 2>/dev/null | awk '{print $NF}' || echo "Error"
done

echo
echo "Monitoring tunnel statistics (press Ctrl+C to stop)..."
echo

# Continuous monitoring
while true; do
    clear
    echo "SSSonector Statistics - $(date)"
    echo "----------------------------------------"
    for i in "${!OIDS[@]}"; do
        echo -n "${NAMES[$i]}: "
        snmpget -v2c -c "$COMMUNITY" "$HOST:$PORT" "${OIDS[$i]}" 2>/dev/null | awk '{print $NF}' || echo "Error"
    done
    sleep 5
done
