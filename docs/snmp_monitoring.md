# Net-SNMP Monitoring Guide for SSSonector QA

## Overview
This guide documents the setup and usage of Net-SNMP for monitoring SSSonector in the QA environment. The SNMP monitoring server is configured on 192.168.50.212 (sssonector-qa-monitor).

## Installation

### Prerequisites
- Ubuntu 24.04
- SSH access to QA monitor system
- Sudo privileges

### Installation Steps
1. Install Net-SNMP packages:
```bash
sudo apt-get update
sudo apt-get install -y snmpd snmp snmp-mibs-downloader
```

2. Configure SNMP daemon:
```bash
# Backup original configuration
sudo cp /etc/snmp/snmpd.conf /etc/snmp/snmpd.conf.original

# Create new configuration
sudo tee /etc/snmp/snmpd.conf > /dev/null << 'EOL'
# Listen on all interfaces
agentAddress udp:161

# Configure access control
rocommunity public 192.168.50.0/24

# System information
syslocation "QA Environment"
syscontact "QA Team"

# Enable all SNMP versions
master agentx

# Load required MIBs
view systemview included .1.3.6.1.2.1.1
view systemview included .1.3.6.1.2.1.25.1
view systemview included .1.3.6.1.4.1.54321

# Performance monitoring
extend sssonector-throughput /usr/local/bin/check_throughput.sh
extend sssonector-connections /usr/local/bin/check_connections.sh
extend sssonector-latency /usr/local/bin/check_latency.sh
EOL
```

3. Create monitoring scripts:
```bash
# Throughput monitoring script
sudo tee /usr/local/bin/check_throughput.sh > /dev/null << 'EOL'
#!/bin/bash
interface="enp0s3"
rx_bytes=$(cat /sys/class/net/$interface/statistics/rx_bytes)
tx_bytes=$(cat /sys/class/net/$interface/statistics/tx_bytes)
echo "$rx_bytes:$tx_bytes"
EOL

# Connection monitoring script
sudo tee /usr/local/bin/check_connections.sh > /dev/null << 'EOL'
#!/bin/bash
netstat -an | grep "ESTABLISHED" | wc -l
EOL

# Latency monitoring script
sudo tee /usr/local/bin/check_latency.sh > /dev/null << 'EOL'
#!/bin/bash
ping -c 1 192.168.50.210 | grep "time=" | cut -d "=" -f 4
EOL

# Make scripts executable
sudo chmod +x /usr/local/bin/check_*.sh
```

4. Restart SNMP service:
```bash
sudo systemctl restart snmpd
sudo systemctl enable snmpd
```

## Verification

### Basic SNMP Queries
```bash
# Test system information
snmpwalk -v2c -c public 192.168.50.212 system

# Test throughput monitoring
snmpwalk -v2c -c public 192.168.50.212 .1.3.6.1.4.1.8072.1.3.2.3.1.2.19.115.115.115.111.110.101.99.116.111.114.45.116.104.114.111.117.103.104.112.117.116

# Test connection monitoring
snmpwalk -v2c -c public 192.168.50.212 .1.3.6.1.4.1.8072.1.3.2.3.1.2.21.115.115.115.111.110.101.99.116.111.114.45.99.111.110.110.101.99.116.105.111.110.115
```

## Common Usage Examples

### Monitor Rate Limiting
```bash
# Get current throughput
snmpget -v2c -c public 192.168.50.212 .1.3.6.1.4.1.8072.1.3.2.3.1.2.19.115.115.115.111.110.101.99.116.111.114.45.116.104.114.111.117.103.104.112.117.116.1.0

# Monitor continuously (every 5 seconds)
watch -n 5 'snmpget -v2c -c public 192.168.50.212 .1.3.6.1.4.1.8072.1.3.2.3.1.2.19.115.115.115.111.110.101.99.116.111.114.45.116.104.114.111.117.103.104.112.117.116.1.0'
```

### Monitor Connections
```bash
# Get current connection count
snmpget -v2c -c public 192.168.50.212 .1.3.6.1.4.1.8072.1.3.2.3.1.2.21.115.115.115.111.110.101.99.116.111.114.45.99.111.110.110.101.99.116.105.111.110.115.1.0

# Monitor continuously
watch -n 5 'snmpget -v2c -c public 192.168.50.212 .1.3.6.1.4.1.8072.1.3.2.3.1.2.21.115.115.115.111.110.101.99.116.111.114.45.99.111.110.110.101.99.116.105.111.110.115.1.0'
```

### Monitor Latency
```bash
# Get current latency
snmpget -v2c -c public 192.168.50.212 .1.3.6.1.4.1.8072.1.3.2.3.1.2.17.115.115.115.111.110.101.99.116.111.114.45.108.97.116.101.110.99.121.1.0

# Monitor continuously
watch -n 5 'snmpget -v2c -c public 192.168.50.212 .1.3.6.1.4.1.8072.1.3.2.3.1.2.17.115.115.115.111.110.101.99.116.111.114.45.108.97.116.101.110.99.121.1.0'
```

## Maintenance

### Log Files
- SNMP daemon logs: `/var/log/syslog`
- Monitor script logs: `/var/log/snmp/`

### Configuration Files
- Main config: `/etc/snmp/snmpd.conf`
- MIB config: `/etc/snmp/snmp.conf`

### Common Tasks
1. Restart SNMP service:
```bash
sudo systemctl restart snmpd
```

2. Check SNMP service status:
```bash
sudo systemctl status snmpd
```

3. View SNMP logs:
```bash
tail -f /var/log/syslog | grep snmpd
```

## Troubleshooting

### Common Issues

1. Connection Refused
```bash
# Check if SNMP daemon is running
sudo systemctl status snmpd

# Verify firewall rules
sudo ufw status
```

2. No Response
```bash
# Test SNMP connectivity
snmpwalk -v2c -c public 192.168.50.212 system

# Check configuration
sudo cat /etc/snmp/snmpd.conf
```

3. Invalid Community String
```bash
# Verify community string in config
grep "rocommunity" /etc/snmp/snmpd.conf

# Test with explicit version
snmpwalk -v2c -c public 192.168.50.212 system
```

## Integration with Test Scripts

### Example Expect Script
```expect
#!/usr/bin/expect

set timeout 30

spawn snmpwalk -v2c -c public 192.168.50.212 system
expect {
    "system.sysDescr.0" {
        puts "SNMP query successful"
        exit 0
    }
    timeout {
        puts "SNMP query timed out"
        exit 1
    }
}
```

### Example Python Script
```python
from pysnmp.hlapi import *

def get_throughput():
    iterator = getCmd(
        SnmpEngine(),
        CommunityData('public', mpModel=1),
        UdpTransportTarget(('192.168.50.212', 161)),
        ContextData(),
        ObjectType(ObjectIdentity('1.3.6.1.4.1.8072.1.3.2.3.1.2.19.115.115.115.111.110.101.99.116.111.114.45.116.104.114.111.117.103.104.112.117.116.1.0'))
    )
    errorIndication, errorStatus, errorIndex, varBinds = next(iterator)
    
    if errorIndication:
        print(f"Error: {errorIndication}")
    elif errorStatus:
        print(f"Error: {errorStatus}")
    else:
        for varBind in varBinds:
            print(f"Throughput: {varBind[1]}")
```

## Additional Resources

1. Net-SNMP Documentation
   - [Official Net-SNMP Documentation](http://www.net-snmp.org/docs/)
   - [MIB Reference](http://www.net-snmp.org/docs/mibs/)

2. Monitoring Tools
   - snmpwalk: Walk through SNMP tree
   - snmpget: Get specific SNMP values
   - snmpset: Set SNMP values (if writable)
   - snmptrapd: Receive SNMP traps
