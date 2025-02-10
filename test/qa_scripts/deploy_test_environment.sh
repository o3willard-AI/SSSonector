#!/bin/bash

# Configuration
CLIENT_IP="192.168.50.211"
SERVER_IP="192.168.50.210"
MONITOR_IP="192.168.50.212"
TUNNEL_PORT="8443"
DATA_PORT="9000"
SERVER_SNMP_PORT="10161"
CLIENT_SNMP_PORT="10162"
REPO_PATH="/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}Deploying test environment...${NC}"

# Function to setup SSSonector on a remote system
setup_sssonector() {
    local host=$1
    echo "Setting up SSSonector on $host..."
    
    # Create directory structure
    ssh sblanken@${host} "mkdir -p ~/Desktop/go/src/github.com/o3willard-AI"
    
    # Copy repository
    rsync -av --exclude '.git' ${REPO_PATH}/ sblanken@${host}:~/Desktop/go/src/github.com/o3willard-AI/SSSonector/
    
    # Build sssonector
    ssh sblanken@${host} "cd ~/Desktop/go/src/github.com/o3willard-AI/SSSonector && make build"
    
    # Setup TUN interface permissions
    ssh sblanken@${host} "sudo groupadd -f tun && \
                         sudo usermod -aG tun \$USER && \
                         sudo modprobe tun && \
                         sudo mkdir -p /dev/net && \
                         if [ ! -e /dev/net/tun ]; then sudo mknod /dev/net/tun c 10 200; fi && \
                         sudo chmod 666 /dev/net/tun && \
                         sudo apt-get update && \
                         sudo apt-get install -y iproute2"
}

# Copy monitoring scripts to monitor system
echo "Copying monitoring scripts to monitor system..."
scp tunnel_monitor.py monitor sblanken@${MONITOR_IP}:~/
ssh sblanken@${MONITOR_IP} "chmod +x ~/tunnel_monitor.py ~/monitor"

# Copy test data generator to client system
echo "Copying test data generator to client system..."
scp test_data_generator.py sblanken@${CLIENT_IP}:~/
ssh sblanken@${CLIENT_IP} "chmod +x ~/test_data_generator.py"

# Setup SSSonector on server and client
setup_sssonector ${SERVER_IP}
setup_sssonector ${CLIENT_IP}

# Create server config
echo "Creating server configuration..."
ssh sblanken@${SERVER_IP} "cat > ~/Desktop/go/src/github.com/o3willard-AI/SSSonector/server-config.yaml << EOL
mode: \"server\"
network:
  interface: \"tun0\"
  address: \"10.0.0.1/24\"
  mtu: 1500
tunnel:
  cert_file: \"certs/server.crt\"
  key_file: \"certs/server.key\"
  ca_file: \"certs/ca.crt\"
  listen_address: \"0.0.0.0\"
  listen_port: ${TUNNEL_PORT}
  max_clients: 10
  upload_kbps: 10240
  download_kbps: 10240
monitor:
  enabled: true
  snmp_enabled: true
  snmp_port: ${SERVER_SNMP_PORT}
  snmp_community: \"public\"
  snmp_address: \"0.0.0.0\"
  log_file: \"server.log\"
  update_interval: 30
EOL"

# Create client config
echo "Creating client configuration..."
ssh sblanken@${CLIENT_IP} "cat > ~/Desktop/go/src/github.com/o3willard-AI/SSSonector/client-config.yaml << EOL
mode: \"client\"
network:
  interface: \"tun0\"
  address: \"10.0.0.2/24\"
  mtu: 1500
tunnel:
  cert_file: \"certs/client.crt\"
  key_file: \"certs/client.key\"
  ca_file: \"certs/ca.crt\"
  server_address: \"${SERVER_IP}\"
  server_port: ${TUNNEL_PORT}
  upload_kbps: 10240
  download_kbps: 10240
monitor:
  enabled: true
  snmp_enabled: true
  snmp_port: ${CLIENT_SNMP_PORT}
  snmp_community: \"public\"
  snmp_address: \"0.0.0.0\"
  log_file: \"client.log\"
  update_interval: 30
EOL"

# Generate certificates on server
echo "Generating certificates on server..."
ssh sblanken@${SERVER_IP} "cd ~/Desktop/go/src/github.com/o3willard-AI/SSSonector && \
    mkdir -p certs && \
    sudo chown \$(whoami):\$(whoami) certs && \
    ./build/sssonector -test-without-certs -config server-config.yaml -keyfile certs -generate-certs-only"

# Copy certificates to client
echo "Copying certificates to client..."
ssh sblanken@${SERVER_IP} "cd ~/Desktop/go/src/github.com/o3willard-AI/SSSonector && tar czf /tmp/certs.tar.gz certs/"
scp sblanken@${SERVER_IP}:/tmp/certs.tar.gz /tmp/
scp /tmp/certs.tar.gz sblanken@${CLIENT_IP}:~/Desktop/go/src/github.com/o3willard-AI/SSSonector/
ssh sblanken@${CLIENT_IP} "cd ~/Desktop/go/src/github.com/o3willard-AI/SSSonector && \
    tar xzf certs.tar.gz && \
    chmod 644 certs/*.crt && \
    chmod 600 certs/*.key"

# Setup server
echo "Setting up server..."
ssh sblanken@${SERVER_IP} "cd ~/Desktop/go/src/github.com/o3willard-AI/SSSonector && \
    sudo ./build/sssonector -test-without-certs -config server-config.yaml -keyfile certs & \
    nc -l ${DATA_PORT} > /dev/null &"

# Setup client
echo "Setting up client..."
ssh sblanken@${CLIENT_IP} "cd ~/Desktop/go/src/github.com/o3willard-AI/SSSonector && \
    sudo ./build/sssonector -test-without-certs -config client-config.yaml -keyfile certs &"

echo -e "${GREEN}Environment deployed successfully!${NC}"
echo
echo "To start the test:"
echo "1. On the monitor system (${MONITOR_IP}):"
echo "   ./monitor"
echo
echo "2. On the client system (${CLIENT_IP}):"
echo "   ./test_data_generator.py ${DATA_PORT} 120"
echo
echo "The test will run for 2 minutes, sending random-sized packets (10MB-100MB)"
echo "Monitor the traffic in real-time using the monitor tool"
echo
echo "To clean up after testing:"
echo "1. Press Ctrl+C on the monitor"
echo "2. Run: pkill -f sssonector on both client and server"
echo "3. Run: pkill -f 'nc -l' on the server"
