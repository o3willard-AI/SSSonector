#!/bin/bash

# Load VM credentials
SNMP_SERVER_IP=$(jq -r '.qa_vms[] | select(.name=="SNMP Server") | .ip' qa_environment.json)
USERNAME=$(jq -r '.qa_vms[] | select(.name=="SNMP Server") | .username' qa_environment.json)
PASSWORD=$(jq -r '.qa_vms[] | select(.name=="SNMP Server") | .sudo_password' qa_environment.json)

# Install SNMP packages
sshpass -p "$PASSWORD" ssh -o StrictHostKeyChecking=no $USERNAME@$SNMP_SERVER_IP "echo '$PASSWORD' | sudo -S apt-get update"
sshpass -p "$PASSWORD" ssh -o StrictHostKeyChecking=no $USERNAME@$SNMP_SERVER_IP "echo '$PASSWORD' | sudo -S apt-get install -y snmp snmpd ntop"

# Configure SNMP
cat > snmpd.conf << EOF
# Listen on all interfaces
agentAddress udp:161,udp6:[::1]:161

# Configure access control
rocommunity public default
rwcommunity private localhost

# System information
sysLocation    "QA Lab"
sysContact     "QA Team <qa@example.com>"
sysServices    72

# Load MIB modules
master agentx
view   systemonly  included   .1.3.6.1.2.1.1
view   systemonly  included   .1.3.6.1.2.1.25.1
EOF

# Copy SNMP config and restart service
sshpass -p "$PASSWORD" scp -o StrictHostKeyChecking=no snmpd.conf $USERNAME@$SNMP_SERVER_IP:/tmp/
sshpass -p "$PASSWORD" ssh -o StrictHostKeyChecking=no $USERNAME@$SNMP_SERVER_IP "echo '$PASSWORD' | sudo -S mv /tmp/snmpd.conf /etc/snmp/ && echo '$PASSWORD' | sudo -S systemctl restart snmpd"

# Verify SNMP is running
sshpass -p "$PASSWORD" ssh -o StrictHostKeyChecking=no $USERNAME@$SNMP_SERVER_IP "echo '$PASSWORD' | sudo -S systemctl status snmpd"

# Install and configure ntopng
sshpass -p "$PASSWORD" ssh -o StrictHostKeyChecking=no $USERNAME@$SNMP_SERVER_IP "echo '$PASSWORD' | sudo -S apt-get install -y ntopng"

# Configure ntopng
cat > ntopng.conf << EOF
-i=any
--community=public
--http-port=3000
--dns-mode=1
EOF

sshpass -p "$PASSWORD" scp -o StrictHostKeyChecking=no ntopng.conf $USERNAME@$SNMP_SERVER_IP:/tmp/
sshpass -p "$PASSWORD" ssh -o StrictHostKeyChecking=no $USERNAME@$SNMP_SERVER_IP "echo '$PASSWORD' | sudo -S mv /tmp/ntopng.conf /etc/ntopng.conf && echo '$PASSWORD' | sudo -S systemctl restart ntopng"

# Verify ntopng is running
sshpass -p "$PASSWORD" ssh -o StrictHostKeyChecking=no $USERNAME@$SNMP_SERVER_IP "echo '$PASSWORD' | sudo -S systemctl status ntopng"

# Clean up temporary files
rm -f snmpd.conf ntopng.conf

echo "SNMP and ntopng setup complete"
