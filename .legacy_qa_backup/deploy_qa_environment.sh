#!/bin/bash

# Configuration
CLIENT_IP="192.168.50.211"
SERVER_IP="192.168.50.210"
MONITOR_IP="192.168.50.212"
SSSONECTOR_HOME="/home/test/sssonector"
BUILD_DIR="go/src/github.com/o3willard-AI/SSSonector"
TEST_FILE="/home/test/data/DryFire_v4_10.zip"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${GREEN}Setting up QA test environment...${NC}"

# Build SSSonector locally
echo -e "${YELLOW}Building SSSonector on dev system...${NC}"
cd ${BUILD_DIR}
if ! make build; then
    echo -e "${RED}Failed to build SSSonector${NC}"
    exit 1
fi
cd -

# Function to setup a remote system
setup_system() {
    local host=$1
    local name=$2
    echo -e "${YELLOW}Setting up $name ($host)...${NC}"
    
    # Create necessary directories with sudo
    ssh sblanken@${host} "sudo mkdir -p ${SSSONECTOR_HOME}/{bin,configs,certs,logs,test_results/rate_limit} && \
                         sudo chown -R sblanken:sblanken ${SSSONECTOR_HOME}"
    
    # Copy binaries
    scp ${BUILD_DIR}/build/sssonector sblanken@${host}:${SSSONECTOR_HOME}/bin/
    scp monitor sblanken@${host}:${SSSONECTOR_HOME}/bin/
    
    # Make binaries executable
    ssh sblanken@${host} "chmod 755 ${SSSONECTOR_HOME}/bin/sssonector && \
                         chmod 755 ${SSSONECTOR_HOME}/bin/monitor"
    
    echo -e "${GREEN}Setup completed on $name${NC}"
}

# Function to verify system setup
verify_system() {
    local host=$1
    local name=$2
    echo -e "${YELLOW}Verifying $name ($host)...${NC}"
    
    # Check if binaries exist and are executable
    if ! ssh sblanken@${host} "[ -x ${SSSONECTOR_HOME}/bin/sssonector ] && [ -x ${SSSONECTOR_HOME}/bin/monitor ]"; then
        echo -e "${RED}Binaries missing or not executable on $name${NC}"
        return 1
    fi
    
    # Check if directories exist and are writable
    if ! ssh sblanken@${host} "[ -d ${SSSONECTOR_HOME}/configs ] && [ -d ${SSSONECTOR_HOME}/certs ] && [ -d ${SSSONECTOR_HOME}/logs ] && \
                              [ -w ${SSSONECTOR_HOME}/configs ] && [ -w ${SSSONECTOR_HOME}/certs ] && [ -w ${SSSONECTOR_HOME}/logs ]"; then
        echo -e "${RED}Required directories missing or not writable on $name${NC}"
        return 1
    fi
    
    echo -e "${GREEN}Verification passed for $name${NC}"
    return 0
}

# Setup each system
for system in "Server:${SERVER_IP}" "Client:${CLIENT_IP}" "Monitor:${MONITOR_IP}"; do
    name=${system%:*}
    ip=${system#*:}
    
    setup_system ${ip} ${name}
    if ! verify_system ${ip} ${name}; then
        echo -e "${RED}Failed to verify $name system${NC}"
        exit 1
    fi
done

# Generate certificates
echo -e "${YELLOW}Generating certificates...${NC}"

# Create certificate directory
mkdir -p certs

# Create temporary config for certificate generation
mkdir -p /home/sblanken/.config/sssonector
cat > /home/sblanken/.config/sssonector/config.yaml << EOF
mode: server
network:
  interface: "eth0"
  address: "0.0.0.0"
  mtu: 1500
tunnel:
  cert_file: certs/server.crt
  key_file: certs/server.key
  ca_file: certs/ca.crt
  listen_address: "0.0.0.0"
  listen_port: 8443
  max_clients: 10
  upload_kbps: 1000
  download_kbps: 1000
logging:
  level: "info"
  file: "logs/server.log"
throttle:
  enabled: true
  rate_limit: 1000000
  burst_limit: 2000000
monitor:
  enabled: true
  snmp_enabled: true
  snmp_address: "0.0.0.0"
  snmp_port: 10161
  snmp_community: "public"
  snmp_version: "2c"
  log_file: "logs/metrics.log"
  update_interval: 30
EOF

# Generate certificates
${BUILD_DIR}/build/sssonector -keygen

# Set proper permissions
chmod 644 certs/server.crt
chmod 600 certs/server.key
chmod 644 certs/client.crt
chmod 600 certs/client.key
chmod 644 certs/ca.crt
chmod 600 certs/ca.key

# Copy certificates to systems
for system in "Server:${SERVER_IP}" "Client:${CLIENT_IP}"; do
    name=${system%:*}
    ip=${system#*:}
    echo -e "${YELLOW}Copying certificates to $name...${NC}"
    
    # Copy certificates with proper permissions
    scp certs/ca.crt sblanken@${ip}:${SSSONECTOR_HOME}/certs/
    scp certs/ca.key sblanken@${ip}:${SSSONECTOR_HOME}/certs/
    scp certs/server.crt sblanken@${ip}:${SSSONECTOR_HOME}/certs/
    scp certs/server.key sblanken@${ip}:${SSSONECTOR_HOME}/certs/
    scp certs/client.crt sblanken@${ip}:${SSSONECTOR_HOME}/certs/
    scp certs/client.key sblanken@${ip}:${SSSONECTOR_HOME}/certs/
    
    # Set permissions
    ssh sblanken@${ip} "chmod 644 ${SSSONECTOR_HOME}/certs/ca.crt && \
                        chmod 600 ${SSSONECTOR_HOME}/certs/ca.key && \
                        chmod 644 ${SSSONECTOR_HOME}/certs/server.crt && \
                        chmod 600 ${SSSONECTOR_HOME}/certs/server.key && \
                        chmod 644 ${SSSONECTOR_HOME}/certs/client.crt && \
                        chmod 600 ${SSSONECTOR_HOME}/certs/client.key"
done

# Create and copy configurations
echo -e "${YELLOW}Creating configurations...${NC}"

# Server config
cat > server.yaml << EOF
mode: server
server:
  listen_addr: 0.0.0.0:8443
  cert_file: ${SSSONECTOR_HOME}/certs/server.crt
  key_file: ${SSSONECTOR_HOME}/certs/server.key
  ca_file: ${SSSONECTOR_HOME}/certs/ca.crt
  rate_limit:
    enabled: true
    bytes_per_second: 100000000
  monitor:
    enabled: true
    snmp_enabled: true
    snmp_port: 10161
    snmp_community: "public"
    snmp_address: "0.0.0.0"
    log_file: "${SSSONECTOR_HOME}/logs/server.log"
    update_interval: 1
EOF

# Client config
cat > client.yaml << EOF
mode: client
client:
  server_addr: ${SERVER_IP}:8443
  cert_file: ${SSSONECTOR_HOME}/certs/client.crt
  key_file: ${SSSONECTOR_HOME}/certs/client.key
  ca_file: ${SSSONECTOR_HOME}/certs/ca.crt
  rate_limit:
    enabled: true
    bytes_per_second: 100000000
  monitor:
    enabled: true
    snmp_enabled: true
    snmp_port: 10162
    snmp_community: "public"
    snmp_address: "0.0.0.0"
    log_file: "${SSSONECTOR_HOME}/logs/client.log"
    update_interval: 1
EOF

# Copy configs
scp server.yaml sblanken@${SERVER_IP}:${SSSONECTOR_HOME}/configs/
scp client.yaml sblanken@${CLIENT_IP}:${SSSONECTOR_HOME}/configs/

# Create test data directory and generate test file
echo -e "${YELLOW}Creating test data...${NC}"
ssh sblanken@${SERVER_IP} "sudo mkdir -p /home/test/data && \
                          sudo chown sblanken:sblanken /home/test/data && \
                          dd if=/dev/urandom of=${TEST_FILE} bs=1M count=3300"
ssh sblanken@${CLIENT_IP} "sudo mkdir -p /home/test/data && \
                          sudo chown sblanken:sblanken /home/test/data"

# Set up monitoring
echo -e "${YELLOW}Setting up monitoring...${NC}"
ssh sblanken@${MONITOR_IP} "mkdir -p ${SSSONECTOR_HOME}/test_results/rate_limit"

# Start services
echo -e "${YELLOW}Starting services...${NC}"
ssh sblanken@${SERVER_IP} "cd ${SSSONECTOR_HOME} && \
                          nohup bin/sssonector -config configs/server.yaml > logs/server.out 2>&1 &"
ssh sblanken@${CLIENT_IP} "cd ${SSSONECTOR_HOME} && \
                          nohup bin/sssonector -config configs/client.yaml > logs/client.out 2>&1 &"

# Cleanup temporary files
rm -f server.yaml client.yaml
rm -rf certs
rm -rf /home/sblanken/.config/sssonector

echo -e "${GREEN}QA environment setup completed successfully!${NC}"
echo "Ready for rate limiting certification testing."
