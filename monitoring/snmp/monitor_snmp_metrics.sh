#!/bin/bash

# Function to get SNMP values
get_snmp_metrics() {
    local throughput=$(snmpget -v2c -c public 192.168.50.212 NET-SNMP-EXTEND-MIB::nsExtendOutput1Line."sssonector-throughput" 2>/dev/null | cut -d'"' -f2)
    local connections=$(snmpget -v2c -c public 192.168.50.212 NET-SNMP-EXTEND-MIB::nsExtendOutput1Line."sssonector-connections" 2>/dev/null | cut -d'"' -f2)
    local latency=$(snmpget -v2c -c public 192.168.50.212 NET-SNMP-EXTEND-MIB::nsExtendOutput1Line."sssonector-latency" 2>/dev/null | cut -d'"' -f2)

    # Check if SNMP commands were successful
    if [ -z "$throughput" ] || [ -z "$connections" ] || [ -z "$latency" ]; then
        echo "$(date +%H:%M:%S),ERROR,ERROR,ERROR,ERROR"
        return
    fi
    
    # Parse throughput values
    local rx_bytes=$(echo $throughput | cut -d':' -f1)
    local tx_bytes=$(echo $throughput | cut -d':' -f2)
    
    # Convert to Mbps
    local rx_mbps=$(echo "scale=2; $rx_bytes * 8 / 1048576" | bc)
    local tx_mbps=$(echo "scale=2; $tx_bytes * 8 / 1048576" | bc)
    
    echo "$(date +%H:%M:%S),${rx_mbps},${tx_mbps},${connections},${latency}"
}

# Create results directory
mkdir -p /tmp/snmp_test_results

# Start logging
echo "Timestamp,RX_Mbps,TX_Mbps,Connections,Latency_ms" > /tmp/snmp_test_results/metrics.csv

# Monitor for 2 minutes (24 samples at 5-second intervals)
for i in {1..24}; do
    get_snmp_metrics >> /tmp/snmp_test_results/metrics.csv
    sleep 5
done

echo "SNMP monitoring completed. Results saved to /tmp/snmp_test_results/metrics.csv"
