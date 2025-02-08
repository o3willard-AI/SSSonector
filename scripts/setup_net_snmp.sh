#!/bin/bash

# Exit on any error
set -e

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo "Please run as root"
    exit 1
fi

echo "Installing Net-SNMP packages..."
apt-get update
apt-get install -y snmpd snmp snmp-mibs-downloader

echo "Backing up original SNMP configuration..."
cp /etc/snmp/snmpd.conf /etc/snmp/snmpd.conf.original

echo "Creating new SNMP configuration..."
cat > /etc/snmp/snmpd.conf << 'EOL'
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

echo "Creating monitoring scripts..."

# Throughput monitoring script
cat > /usr/local/bin/check_throughput.sh << 'EOL'
#!/bin/bash
interface="enp0s3"
rx_bytes=$(cat /sys/class/net/$interface/statistics/rx_bytes)
tx_bytes=$(cat /sys/class/net/$interface/statistics/tx_bytes)
echo "$rx_bytes:$tx_bytes"
EOL

# Connection monitoring script
cat > /usr/local/bin/check_connections.sh << 'EOL'
#!/bin/bash
netstat -an | grep "ESTABLISHED" | wc -l
EOL

# Latency monitoring script
cat > /usr/local/bin/check_latency.sh << 'EOL'
#!/bin/bash
ping -c 1 192.168.50.210 | grep "time=" | cut -d "=" -f 4
EOL

echo "Setting script permissions..."
chmod +x /usr/local/bin/check_*.sh

echo "Creating SNMP log directory..."
mkdir -p /var/log/snmp
chown snmp:snmp /var/log/snmp

echo "Restarting SNMP service..."
systemctl restart snmpd
systemctl enable snmpd

echo "Verifying SNMP installation..."
if systemctl is-active --quiet snmpd; then
    echo "SNMP service is running"
else
    echo "Error: SNMP service failed to start"
    exit 1
fi

echo "Testing SNMP connectivity..."
if snmpwalk -v2c -c public localhost system >/dev/null 2>&1; then
    echo "SNMP connectivity test successful"
else
    echo "Warning: SNMP connectivity test failed"
    echo "Please check firewall settings and SNMP configuration"
fi

echo "Opening SNMP port in firewall..."
if command -v ufw >/dev/null 2>&1; then
    ufw allow 161/udp
    echo "Firewall rule added for SNMP"
fi

echo "Installation complete!"
echo "You can verify the installation by running: snmpwalk -v2c -c public localhost system"
echo "For more information, see: /home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/docs/snmp_monitoring.md"
