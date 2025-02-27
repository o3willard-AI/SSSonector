#!/bin/bash

# enhanced_qa_testing.sh
# Comprehensive script for SSSonector QA testing with improved reliability and diagnostics
set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    return 1
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

log_debug() {
    echo -e "${CYAN}[DEBUG]${NC} $1"
}

# QA environment details - load from config file if exists
QA_CONFIG_FILE="qa_environment.conf"
QA_SERVER="192.168.50.210"
QA_CLIENT="192.168.50.211"
QA_USER="sblanken"
QA_SUDO_PASSWORD=""

# Load configuration if exists
if [ -f "${QA_CONFIG_FILE}" ]; then
    log_info "Loading configuration from ${QA_CONFIG_FILE}"
    source "${QA_CONFIG_FILE}"
else
    log_warn "Configuration file ${QA_CONFIG_FILE} not found, using defaults"
    
    # Create a template configuration file
    cat > "${QA_CONFIG_FILE}" << EOF
# SSSonector QA Environment Configuration
# Edit this file to match your environment

# QA environment details
QA_SERVER="192.168.50.210"
QA_CLIENT="192.168.50.211"
QA_USER="sblanken"
QA_SUDO_PASSWORD=""

# Test settings
TEST_TIMEOUT=300  # Timeout in seconds
PACKET_COUNT=20   # Number of packets to send in each direction
RETRY_COUNT=3     # Number of retries for failed tests
EOF
    
    log_info "Created template configuration file ${QA_CONFIG_FILE}"
    log_info "Please edit it to match your environment and run this script again"
    exit 1
fi

# Check if QA_SUDO_PASSWORD is set
if [ -z "${QA_SUDO_PASSWORD}" ]; then
    log_error "QA_SUDO_PASSWORD is not set in ${QA_CONFIG_FILE}"
    log_info "Please edit ${QA_CONFIG_FILE} to set QA_SUDO_PASSWORD"
    exit 1
fi

# Test settings
TEST_TIMEOUT=${TEST_TIMEOUT:-300}  # Default timeout: 5 minutes
PACKET_COUNT=${PACKET_COUNT:-20}   # Default packet count: 20
RETRY_COUNT=${RETRY_COUNT:-3}      # Default retry count: 3

# Create results directory
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RESULTS_DIR="qa_results_${TIMESTAMP}"
mkdir -p "${RESULTS_DIR}"
mkdir -p "${RESULTS_DIR}/logs"
mkdir -p "${RESULTS_DIR}/pcaps"
mkdir -p "${RESULTS_DIR}/configs"

# Set up logging
LOGFILE="${RESULTS_DIR}/qa_test.log"
exec > >(tee -a "${LOGFILE}") 2>&1

# Log system information
log_step "System Information"
echo "Date: $(date)"
echo "Hostname: $(hostname)"
echo "Kernel: $(uname -r)"
echo "Architecture: $(uname -m)"
echo "QA Server: ${QA_SERVER}"
echo "QA Client: ${QA_CLIENT}"
echo "QA User: ${QA_USER}"
echo "Test Timeout: ${TEST_TIMEOUT} seconds"
echo "Packet Count: ${PACKET_COUNT}"
echo "Retry Count: ${RETRY_COUNT}"
echo "Results Directory: ${RESULTS_DIR}"

# Function to run command on remote host with timeout
run_remote() {
    local host=$1
    local command=$2
    local timeout=${3:-30}
    local description=${4:-"Remote command"}
    
    log_debug "Running on ${host}: ${command}"
    
    # Run command with timeout
    if ! timeout ${timeout} sshpass -p "${QA_SUDO_PASSWORD}" ssh -o ConnectTimeout=5 -o BatchMode=yes -o StrictHostKeyChecking=no "${QA_USER}@${host}" "${command}" > "${RESULTS_DIR}/logs/${host}_${description// /_}.log" 2>&1; then
        log_error "Failed to run command on ${host}: ${command}"
        log_info "See ${RESULTS_DIR}/logs/${host}_${description// /_}.log for details"
        return 1
    fi
    
    return 0
}

# Function to copy file from remote host
copy_from_remote() {
    local host=$1
    local src=$2
    local dst=$3
    
    log_debug "Copying from ${host}:${src} to ${dst}"
    
    # Copy file
    if ! sshpass -p "${QA_SUDO_PASSWORD}" scp -o ConnectTimeout=5 -o BatchMode=yes -o StrictHostKeyChecking=no "${QA_USER}@${host}:${src}" "${dst}" > /dev/null 2>&1; then
        log_warn "Failed to copy ${src} from ${host}"
        return 1
    fi
    
    return 0
}

# Function to copy file to remote host
copy_to_remote() {
    local host=$1
    local src=$2
    local dst=$3
    
    log_debug "Copying to ${host}:${dst} from ${src}"
    
    # Copy file
    if ! sshpass -p "${QA_SUDO_PASSWORD}" scp -o ConnectTimeout=5 -o BatchMode=yes -o StrictHostKeyChecking=no "${src}" "${QA_USER}@${host}:${dst}" > /dev/null 2>&1; then
        log_error "Failed to copy ${src} to ${host}:${dst}"
        return 1
    fi
    
    return 0
}

# Function to check if a host is reachable
check_host() {
    local host=$1
    
    log_debug "Checking if ${host} is reachable"
    
    # Check if host is reachable
    if ! ping -c 1 -W 2 "${host}" > /dev/null 2>&1; then
        log_error "Host ${host} is not reachable"
        return 1
    fi
    
    # Check if SSH is available
    if ! timeout 5 ssh -o ConnectTimeout=5 -o BatchMode=yes -o StrictHostKeyChecking=no "${QA_USER}@${host}" "echo 'SSH connection test successful'" > /dev/null 2>&1; then
        log_error "SSH connection to ${host} failed"
        return 1
    fi
    
    # Check if sudo is available
    if ! timeout 5 sshpass -p "${QA_SUDO_PASSWORD}" ssh -o ConnectTimeout=5 -o BatchMode=yes -o StrictHostKeyChecking=no "${QA_USER}@${host}" "echo '${QA_SUDO_PASSWORD}' | sudo -S echo 'Sudo test successful'" > /dev/null 2>&1; then
        log_error "Sudo access on ${host} failed"
        return 1
    fi
    
    log_info "Host ${host} is reachable and has SSH and sudo access"
    return 0
}

# Function to check if sshpass is installed
check_sshpass() {
    log_debug "Checking if sshpass is installed"
    
    # Check if sshpass is installed
    if ! command -v sshpass &> /dev/null; then
        log_warn "sshpass is not installed, attempting to install it"
        
        # Try to install sshpass
        if command -v apt-get &> /dev/null; then
            sudo apt-get update && sudo apt-get install -y sshpass
        elif command -v yum &> /dev/null; then
            sudo yum install -y sshpass
        elif command -v dnf &> /dev/null; then
            sudo dnf install -y sshpass
        elif command -v brew &> /dev/null; then
            brew install sshpass
        else
            log_error "Could not install sshpass, please install it manually"
            return 1
        fi
        
        # Check if installation was successful
        if ! command -v sshpass &> /dev/null; then
            log_error "Failed to install sshpass"
            return 1
        fi
    fi
    
    log_info "sshpass is installed"
    return 0
}

# Function to check if tcpdump is installed on remote host
check_tcpdump() {
    local host=$1
    
    log_debug "Checking if tcpdump is installed on ${host}"
    
    # Check if tcpdump is installed
    if ! run_remote "${host}" "command -v tcpdump" 5 "check_tcpdump"; then
        log_warn "tcpdump is not installed on ${host}, attempting to install it"
        
        # Try to install tcpdump
        if ! run_remote "${host}" "sudo apt-get update && sudo apt-get install -y tcpdump" 60 "install_tcpdump"; then
            log_error "Failed to install tcpdump on ${host}"
            return 1
        fi
        
        # Check if installation was successful
        if ! run_remote "${host}" "command -v tcpdump" 5 "check_tcpdump_after_install"; then
            log_error "Failed to install tcpdump on ${host}"
            return 1
        fi
    fi
    
    log_info "tcpdump is installed on ${host}"
    return 0
}

# Function to check if SSSonector binary exists
check_binary() {
    local binary_path="../../../sssonector"
    
    log_debug "Checking if SSSonector binary exists at ${binary_path}"
    
    # Check if binary exists
    if [ ! -f "${binary_path}" ]; then
        log_error "SSSonector binary not found at ${binary_path}"
        return 1
    fi
    
    # Check if binary is executable
    if [ ! -x "${binary_path}" ]; then
        log_warn "SSSonector binary is not executable, attempting to make it executable"
        chmod +x "${binary_path}"
        
        # Check if chmod was successful
        if [ ! -x "${binary_path}" ]; then
            log_error "Failed to make SSSonector binary executable"
            return 1
        fi
    fi
    
    log_info "SSSonector binary exists and is executable"
    return 0
}

# Function to verify environment
verify_environment() {
    log_step "Verifying Environment"
    
    # Check if sshpass is installed
    check_sshpass || return 1
    
    # Check if SSSonector binary exists
    check_binary || return 1
    
    # Check if hosts are reachable
    check_host "${QA_SERVER}" || return 1
    check_host "${QA_CLIENT}" || return 1
    
    # Check if tcpdump is installed
    check_tcpdump "${QA_SERVER}" || return 1
    check_tcpdump "${QA_CLIENT}" || return 1
    
    # Check if IP forwarding is enabled
    log_info "Checking IP forwarding on server"
    run_remote "${QA_SERVER}" "cat /proc/sys/net/ipv4/ip_forward" 5 "check_ip_forwarding_server"
    
    log_info "Checking IP forwarding on client"
    run_remote "${QA_CLIENT}" "cat /proc/sys/net/ipv4/ip_forward" 5 "check_ip_forwarding_client"
    
    # Check if TUN module is loaded
    log_info "Checking TUN module on server"
    run_remote "${QA_SERVER}" "lsmod | grep tun" 5 "check_tun_module_server"
    
    log_info "Checking TUN module on client"
    run_remote "${QA_CLIENT}" "lsmod | grep tun" 5 "check_tun_module_client"
    
    log_info "Environment verification completed successfully"
    return 0
}

# Function to clean up QA environment
cleanup_environment() {
    log_step "Cleaning up QA environment"
    
    # Stop packet captures
    log_info "Stopping packet captures"
    run_remote "${QA_SERVER}" "sudo pkill -f tcpdump" 5 "stop_tcpdump_server" || true
    run_remote "${QA_CLIENT}" "sudo pkill -f tcpdump" 5 "stop_tcpdump_client" || true
    
    # Stop SSSonector processes
    log_info "Stopping SSSonector processes on server"
    run_remote "${QA_SERVER}" "sudo pkill -f sssonector" 5 "stop_sssonector_server" || true
    
    log_info "Stopping SSSonector processes on client"
    run_remote "${QA_CLIENT}" "sudo pkill -f sssonector" 5 "stop_sssonector_client" || true
    
    # Wait for processes to stop
    sleep 2
    
    # Check if processes are still running
    log_info "Checking if SSSonector processes are still running on server"
    if run_remote "${QA_SERVER}" "pgrep -f sssonector" 5 "check_sssonector_server" > /dev/null 2>&1; then
        log_warn "SSSonector processes still running on server, forcing kill"
        run_remote "${QA_SERVER}" "sudo pkill -9 -f sssonector" 5 "force_kill_sssonector_server" || true
    fi
    
    log_info "Checking if SSSonector processes are still running on client"
    if run_remote "${QA_CLIENT}" "pgrep -f sssonector" 5 "check_sssonector_client" > /dev/null 2>&1; then
        log_warn "SSSonector processes still running on client, forcing kill"
        run_remote "${QA_CLIENT}" "sudo pkill -9 -f sssonector" 5 "force_kill_sssonector_client" || true
    fi
    
    # Remove TUN interfaces
    log_info "Removing TUN interfaces on server"
    run_remote "${QA_SERVER}" "sudo ip link show | grep tun | cut -d: -f2 | cut -d@ -f1 | xargs -I{} sudo ip link delete {}" 5 "remove_tun_server" || true
    
    log_info "Removing TUN interfaces on client"
    run_remote "${QA_CLIENT}" "sudo ip link show | grep tun | cut -d: -f2 | cut -d@ -f1 | xargs -I{} sudo ip link delete {}" 5 "remove_tun_client" || true
    
    # Clean up directories
    log_info "Cleaning up directories on server"
    run_remote "${QA_SERVER}" "sudo rm -rf /opt/sssonector" 5 "cleanup_dirs_server" || true
    
    log_info "Cleaning up directories on client"
    run_remote "${QA_CLIENT}" "sudo rm -rf /opt/sssonector" 5 "cleanup_dirs_client" || true
    
    # Create directories
    log_info "Creating directories on server"
    run_remote "${QA_SERVER}" "sudo mkdir -p /opt/sssonector/{bin,certs,config,log,state}" 5 "create_dirs_server" || true
    
    log_info "Creating directories on client"
    run_remote "${QA_CLIENT}" "sudo mkdir -p /opt/sssonector/{bin,certs,config,log,state}" 5 "create_dirs_client" || true
    
    log_info "QA environment cleanup completed successfully"
    return 0
}

# Function to generate certificates
generate_certificates() {
    log_step "Generating certificates"
    
    # Create certificate directory
    local cert_dir="${RESULTS_DIR}/certs"
    mkdir -p "${cert_dir}"
    
    # Generate CA certificate
    log_info "Generating CA certificate"
    openssl req -x509 -new -nodes -keyout "${cert_dir}/ca.key" -sha256 -days 365 -out "${cert_dir}/ca.crt" -subj "/C=US/ST=California/L=San Francisco/O=SSSonector/OU=QA/CN=SSSonector CA" || return 1
    
    # Generate server certificate
    log_info "Generating server certificate"
    openssl req -new -nodes -keyout "${cert_dir}/server.key" -out "${cert_dir}/server.csr" -subj "/C=US/ST=California/L=San Francisco/O=SSSonector/OU=QA/CN=${QA_SERVER}" || return 1
    openssl x509 -req -in "${cert_dir}/server.csr" -CA "${cert_dir}/ca.crt" -CAkey "${cert_dir}/ca.key" -CAcreateserial -out "${cert_dir}/server.crt" -days 365 -sha256 || return 1
    
    # Generate client certificate
    log_info "Generating client certificate"
    openssl req -new -nodes -keyout "${cert_dir}/client.key" -out "${cert_dir}/client.csr" -subj "/C=US/ST=California/L=San Francisco/O=SSSonector/OU=QA/CN=${QA_CLIENT}" || return 1
    openssl x509 -req -in "${cert_dir}/client.csr" -CA "${cert_dir}/ca.crt" -CAkey "${cert_dir}/ca.key" -CAcreateserial -out "${cert_dir}/client.crt" -days 365 -sha256 || return 1
    
    # Clean up CSR files
    rm -f "${cert_dir}/*.csr"
    
    log_info "Certificates generated successfully"
    return 0
}

# Function to create configuration files
create_configurations() {
    log_step "Creating configuration files"
    
    # Create configuration directory
    local config_dir="${RESULTS_DIR}/configs"
    mkdir -p "${config_dir}"
    
    # Create server configuration
    log_info "Creating server configuration"
    cat > "${config_dir}/server.yaml" << EOF
type: server
config:
  mode: server
  logging:
    level: debug
    file: /opt/sssonector/log/server.log
    format: text
  auth:
    cert_file: /opt/sssonector/certs/server.crt
    key_file: /opt/sssonector/certs/server.key
    ca_file: /opt/sssonector/certs/ca.crt
  network:
    name: tun0
    interface: tun0
    address: 10.0.0.1/24
    mtu: 1400
  tunnel:
    listen_port: 8443
    protocol: tcp
    listen_address: 0.0.0.0
version: 1.0.0
EOF
    
    # Create client configuration
    log_info "Creating client configuration"
    cat > "${config_dir}/client.yaml" << EOF
type: client
config:
  mode: client
  logging:
    level: debug
    file: /opt/sssonector/log/client.log
    format: text
  auth:
    cert_file: /opt/sssonector/certs/client.crt
    key_file: /opt/sssonector/certs/client.key
    ca_file: /opt/sssonector/certs/ca.crt
  network:
    name: tun0
    interface: tun0
    address: 10.0.0.2/24
    mtu: 1400
  tunnel:
    server: ${QA_SERVER}:8443
    protocol: tcp
version: 1.0.0
EOF
    
    # Create server foreground configuration
    log_info "Creating server foreground configuration"
    cat > "${config_dir}/server_foreground.yaml" << EOF
type: server
config:
  mode: server
  logging:
    level: debug
    file: /opt/sssonector/log/server.log
    format: text
  auth:
    cert_file: /opt/sssonector/certs/server.crt
    key_file: /opt/sssonector/certs/server.key
    ca_file: /opt/sssonector/certs/ca.crt
  network:
    name: tun0
    interface: tun0
    address: 10.0.0.1/24
    mtu: 1400
  tunnel:
    listen_port: 8443
    protocol: tcp
    listen_address: 0.0.0.0
  daemon:
    enabled: false
version: 1.0.0
EOF
    
    # Create client foreground configuration
    log_info "Creating client foreground configuration"
    cat > "${config_dir}/client_foreground.yaml" << EOF
type: client
config:
  mode: client
  logging:
    level: debug
    file: /opt/sssonector/log/client.log
    format: text
  auth:
    cert_file: /opt/sssonector/certs/client.crt
    key_file: /opt/sssonector/certs/client.key
    ca_file: /opt/sssonector/certs/ca.crt
  network:
    name: tun0
    interface: tun0
    address: 10.0.0.2/24
    mtu: 1400
  tunnel:
    server: ${QA_SERVER}:8443
    protocol: tcp
  daemon:
    enabled: false
version: 1.0.0
EOF
    
    log_info "Configuration files created successfully"
    return 0
}

# Function to deploy SSSonector
deploy_sssonector() {
    log_step "Deploying SSSonector"
    
    local binary_path="../../../sssonector"
    local cert_dir="${RESULTS_DIR}/certs"
    local config_dir="${RESULTS_DIR}/configs"
    
    # Deploy to server
    log_info "Deploying SSSonector to server"
    
    # Copy binary
    log_debug "Copying binary to server"
    copy_to_remote "${QA_SERVER}" "${binary_path}" "/tmp/sssonector" || return 1
    run_remote "${QA_SERVER}" "sudo mv /tmp/sssonector /opt/sssonector/bin/sssonector && sudo chmod 755 /opt/sssonector/bin/sssonector" 5 "move_binary_server" || return 1
    
    # Copy certificates
    log_debug "Copying certificates to server"
    copy_to_remote "${QA_SERVER}" "${cert_dir}/ca.crt" "/tmp/ca.crt" || return 1
    copy_to_remote "${QA_SERVER}" "${cert_dir}/server.crt" "/tmp/server.crt" || return 1
    copy_to_remote "${QA_SERVER}" "${cert_dir}/server.key" "/tmp/server.key" || return 1
    run_remote "${QA_SERVER}" "sudo mv /tmp/ca.crt /tmp/server.crt /tmp/server.key /opt/sssonector/certs/ && sudo chmod 644 /opt/sssonector/certs/ca.crt /opt/sssonector/certs/server.crt && sudo chmod 600 /opt/sssonector/certs/server.key" 5 "move_certs_server" || return 1
    
    # Copy configuration files
    log_debug "Copying configuration files to server"
    copy_to_remote "${QA_SERVER}" "${config_dir}/server.yaml" "/tmp/server.yaml" || return 1
    copy_to_remote "${QA_SERVER}" "${config_dir}/server_foreground.yaml" "/tmp/server_foreground.yaml" || return 1
    run_remote "${QA_SERVER}" "sudo mv /tmp/server.yaml /tmp/server_foreground.yaml /opt/sssonector/config/ && sudo chmod 644 /opt/sssonector/config/server.yaml /opt/sssonector/config/server_foreground.yaml" 5 "move_configs_server" || return 1
    
    # Deploy to client
    log_info "Deploying SSSonector to client"
    
    # Copy binary
    log_debug "Copying binary to client"
    copy_to_remote "${QA_CLIENT}" "${binary_path}" "/tmp/sssonector" || return 1
    run_remote "${QA_CLIENT}" "sudo mv /tmp/sssonector /opt/sssonector/bin/sssonector && sudo chmod 755 /opt/sssonector/bin/sssonector" 5 "move_binary_client" || return 1
    
    # Copy certificates
    log_debug "Copying certificates to client"
    copy_to_remote "${QA_CLIENT}" "${cert_dir}/ca.crt" "/tmp/ca.crt" || return 1
    copy_to_remote "${QA_CLIENT}" "${cert_dir}/client.crt" "/tmp/client.crt" || return 1
    copy_to_remote "${QA_CLIENT}" "${cert_dir}/client.key" "/tmp/client.key" || return 1
    run_remote "${QA_CLIENT}" "sudo mv /tmp/ca.crt /tmp/client.crt /tmp/client.key /opt/sssonector/certs/ && sudo chmod 644 /opt/sssonector/certs/ca.crt /opt/sssonector/certs/client.crt && sudo chmod 600 /opt/sssonector/certs/client.key" 5 "move_certs_client" || return 1
    
    # Copy configuration files
    log_debug "Copying configuration files to client"
    copy_to_remote "${QA_CLIENT}" "${config_dir}/client.yaml" "/tmp/client.yaml" || return 1
    copy_to_remote "${QA_CLIENT}" "${config_dir}/client_foreground.yaml" "/tmp/client_foreground.yaml" || return 1
    run_remote "${QA_CLIENT}" "sudo mv /tmp/client.yaml /tmp/client_foreground.yaml /opt/sssonector/config/ && sudo chmod 644 /opt/sssonector/config/client.yaml /opt/sssonector/config/client_foreground.yaml" 5 "move_configs_client" || return 1
    
    log_info "SSSonector deployed successfully"
    return 0
}

# Function to start packet capture
start_packet_capture() {
    log_step "Starting packet capture"
    
    # Start packet capture on server
    log_info "Starting packet capture on server"
    run_remote "${QA_SERVER}" "sudo tcpdump -i any -w /tmp/server_capture.pcap 'port 8443 or icmp' &" 5 "start_tcpdump_server" || return 1
    
    # Start packet capture on client
    log_info "Starting packet capture on client"
    run_remote "${QA_CLIENT}" "sudo tcpdump -i any -w /tmp/client_capture.pcap 'port 8443 or icmp' &" 5 "start_tcpdump_client" || return 1
    
    log_info "Packet capture started successfully"
    return 0
}

# Function to stop packet capture
stop_packet_capture() {
    log_step "Stopping packet capture"
    
    # Stop packet capture on server
    log_info "Stopping packet capture on server"
    run_remote "${QA_SERVER}" "sudo pkill -f tcpdump" 5 "stop_tcpdump_server" || true
    
    # Stop packet capture on client
    log_info "Stopping packet capture on client"
    run_remote "${QA_CLIENT}" "sudo pkill -f tcpdump" 5 "stop_tcpdump_client" || true
    
    # Copy packet captures
    log_info "Copying packet captures"
    copy_from_remote "${QA_SERVER}" "/tmp/server_capture.pcap" "${RESULTS_DIR}/pcaps/server_capture.pcap" || true
    copy_from_remote "${QA_CLIENT}" "/tmp/client_capture.pcap" "${RESULTS_DIR}/pcaps/client_capture.pcap" || true
    
    log_info "Packet capture stopped successfully"
    return 0
}

# Function to apply network fixes
apply_network_fixes() {
    log_step "Applying network fixes"
    
    # Enable IP forwarding
    log_info "Enabling IP forwarding on server"
    run_remote "${QA_SERVER}" "echo 1 | sudo tee /proc/sys/net/ipv4/ip_forward" 5 "enable_ip_forwarding_server" || return 1
    
    log_info "Enabling IP forwarding on client"
    run_remote "${QA_CLIENT}" "echo 1 | sudo tee /proc/sys/net/ipv4/ip_forward" 5 "enable_ip_forwarding_client" || return 1
    
    # Disable reverse path filtering
    log_info "Disabling reverse path filtering on server"
    run_remote "${QA_SERVER}" "sudo sysctl -w net.ipv4.conf.all.rp_filter=0" 5 "disable_rp_filter_all_server" || return 1
    run_remote "${QA_SERVER}" "sudo sysctl -w net.ipv4.conf.default.rp_filter=0" 5 "disable_rp_filter_default_server" || return 1
    run_remote "${QA_SERVER}" "sudo sysctl -w net.ipv4.conf.tun0.rp_filter=0" 5 "disable_rp_filter_tun0_server" || true
    
    log_info "Disabling reverse path filtering on client"
    run_remote "${QA_CLIENT}" "sudo sysctl -w net.ipv4.conf.all.rp_filter=0" 5 "disable_rp_filter_all_client" || return 1
    run_remote "${QA_CLIENT}" "sudo sysctl -w net.ipv4.conf.default.rp_filter=0" 5 "disable_rp_filter_default_client" || return 1
    run_remote "${QA_CLIENT}" "sudo sysctl -w net.ipv4.conf.tun0.rp_filter=0" 5 "disable_rp_filter_tun0_client" || true
    
    # Disable ICMP echo ignore
    log_info "Disabling ICMP echo ignore on server"
    run_remote "${QA_SERVER}" "sudo sysctl -w net.ipv4.icmp_echo_ignore_broadcasts=0" 5 "disable_icmp_echo_ignore_broadcasts_server" || return 1
    run_remote "${QA_SERVER}" "sudo sysctl -w net.ipv4.icmp_echo_ignore_all=0" 5 "disable_icmp_echo_ignore_all_server" || return 1
    
    log_info "Disabling ICMP echo ignore on client"
    run_remote "${QA_CLIENT}" "sudo sysctl -w net.ipv4.icmp_echo_ignore_broadcasts=0" 5 "disable_icmp_echo_ignore_broadcasts_client" || return 1
    run_remote "${QA_CLIENT}" "sudo sysctl -w net.ipv4.icmp_echo_ignore_all=0" 5 "disable_icmp_echo_ignore_all_client" || return 1
    
    # Add firewall rules
    log_info "Adding firewall rules on server"
    run_remote "${QA_SERVER}" "sudo iptables -F INPUT" 5 "flush_input_server" || return 1
    run_remote "${QA_SERVER}" "sudo iptables -A INPUT -p icmp -j ACCEPT" 5 "add_icmp_input_server" || return 1
    run_remote "${QA_SERVER}" "sudo iptables -A INPUT -i tun0 -j ACCEPT" 5 "add_tun0_input_server" || return 1
    run_remote "${QA_SERVER}" "sudo iptables -A INPUT -i lo -j ACCEPT" 5 "add_lo_input_server" || return 1
    run_remote "${QA_SERVER}" "sudo iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT" 5 "add_established_input_server" || return 1
    
    log_info "Adding firewall rules on client"
    run_remote "${QA_CLIENT}" "sudo iptables -F INPUT" 5 "flush_input_client" || return 1
    run_remote "${QA_CLIENT}" "sudo iptables -A INPUT -p icmp -j ACCEPT" 5 "add_icmp_input_client" || return 1
    run_remote "${QA_CLIENT}" "sudo iptables -A INPUT -i tun0 -j ACCEPT" 5 "add_tun0_input_client" || return 1
    run_remote "${QA_CLIENT}" "sudo iptables -A INPUT -i lo -j ACCEPT" 5 "add_lo_input_client" || return 1
    run_remote "${QA_CLIENT}" "sudo iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT" 5 "add_established_input_client" || return 1
    
    # Add FORWARD rules
    log_info "Adding FORWARD rules on server"
    run_remote "${QA_SERVER}" "sudo iptables -F FORWARD" 5 "flush_forward_server" || return 1
    run_remote "${QA_SERVER}" "sudo iptables -A FORWARD -i tun0 -o eth0 -j ACCEPT" 5 "add_tun0_to_eth0_forward_server" || return 1
    run_remote "${QA_SERVER}" "sudo iptables -A FORWARD -i eth0 -o tun0 -j ACCEPT" 5 "add_eth0_to_tun0_forward_server" || return 1
    
    log_info "Adding FORWARD rules on client"
    run_remote "${QA_CLIENT}" "sudo iptables -F FORWARD" 5 "flush_forward_client" || return 1
    run_remote "${QA_CLIENT}" "sudo iptables -A FORWARD -i tun0 -o eth0 -j ACCEPT" 5 "add_tun0_to_eth0_forward_client" || return 1
    run_remote "${QA_CLIENT}" "sudo iptables -A FORWARD -i eth0 -o tun0 -j ACCEPT" 5 "add_eth0_to_tun0_forward_client" || return 1
    
    # Enable NAT
    log_info "Enabling NAT on server"
    run_remote "${QA_SERVER}" "sudo iptables -t nat -F POSTROUTING" 5 "flush_postrouting_server" || return 1
    run_remote "${QA_SERVER}" "sudo iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o eth0 -j MASQUERADE" 5 "add_masquerade_server" || return 1
    
    log_info "Enabling NAT on client"
    run_remote "${QA_CLIENT}" "sudo iptables -t nat -F POSTROUTING" 5 "flush_postrouting_client" || return 1
    run_remote "${QA_CLIENT}" "sudo iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o eth0 -j MASQUERADE" 5 "add_masquerade_client" || return 1
    
    log_info "Network fixes applied successfully"
    return 0
}

# Function to run a test scenario
run_test_scenario() {
    local scenario=$1
    local server_mode=$2
    local client_mode=$3
    
    log_step "Running test scenario: ${scenario}"
    
    # Start server
    log_info "Starting SSSonector server in ${server_mode} mode"
    if [[ "${server_mode}" == "foreground" ]]; then
        run_remote "${QA_SERVER}" "sudo /opt/sssonector/bin/sssonector -config /opt/sssonector/config/server_foreground.yaml > /opt/sssonector/log/server.log 2>&1 &" 10 "start_server_foreground" || return 1
    else
        run_remote "${QA_SERVER}" "sudo /opt/sssonector/bin/sssonector -config /opt/sssonector/config/server.yaml" 10 "start_server_background" || return 1
    fi
    
    # Wait for server to start
    log_info "Waiting for server to start"
    sleep 5
    
    # Check if server is running
    log_info "Checking if server is running"
    if ! run_remote "${QA_SERVER}" "pgrep -f sssonector" 5 "check_server_running"; then
        log_error "Server failed to start"
        return 1
    fi
    
    # Start client
    log_info "Starting SSSonector client in ${client_mode} mode"
    if [[ "${client_mode}" == "foreground" ]]; then
        run_remote "${QA_CLIENT}" "sudo /opt/sssonector/bin/sssonector -config /opt/sssonector/config/client_foreground.yaml > /opt/sssonector/log/client.log 2>&1 &" 10 "start_client_foreground" || {
            run_remote "${QA_SERVER}" "sudo pkill -f sssonector" 5 "stop_server" || true
            return 1
        }
    else
        run_remote "${QA_CLIENT}" "sudo /opt/sssonector/bin/sssonector -config /opt/sssonector/config/client.yaml" 10 "start_client_background" || {
            run_remote "${QA_SERVER}" "sudo pkill -f sssonector" 5 "stop_server" || true
            return 1
        }
    fi
    
    # Wait for client to start
    log_info "Waiting for client to start"
    sleep 5
    
    # Check if client is running
    log_info "Checking if client is running"
    if ! run_remote "${QA_CLIENT}" "pgrep -f sssonector" 5 "check_client_running"; then
        log_error "Client failed to start"
        run_remote "${QA_SERVER}" "sudo pkill -f sssonector" 5 "stop_server" || true
        return 1
    fi
    
    # Check tunnel interfaces
    log_info "Checking tunnel interfaces"
    
    # Check if tun0 interface exists on server
    log_info "Checking tun0 interface on server"
    if ! run_remote "${QA_SERVER}" "ip link show tun0" 5 "check_tun0_server"; then
        log_error "tun0 interface not found on server"
        run_remote "${QA_CLIENT}" "sudo pkill -f sssonector" 5 "stop_client" || true
        run_remote "${QA_SERVER}" "sudo pkill -f sssonector" 5 "stop_server" || true
        return 1
    fi
    
    # Check if tun0 interface exists on client
    log_info "Checking tun0 interface on client"
    if ! run_remote "${QA_CLIENT}" "ip link show tun0" 5 "check_tun0_client"; then
        log_error "tun0 interface not found on client"
        run_remote "${QA_CLIENT}" "sudo pkill -f sssonector" 5 "stop_client" || true
        run_remote "${QA_SERVER}" "sudo pkill -f sssonector" 5 "stop_server" || true
        return 1
    fi
    
    # Test connectivity
    log_info "Testing connectivity"
    
    # Test ping from client to server
    log_info "Testing ping from client to server"
    if ! run_remote "${QA_CLIENT}" "ping -c ${PACKET_COUNT} -W 2 10.0.0.1" 30 "ping_client_to_server"; then
        log_warn "Ping from client to server failed"
        client_to_server_success=false
    else
        log_info "Ping from client to server successful"
        client_to_server_success=true
    fi
    
    # Test ping from server to client
    log_info "Testing ping from server to client"
    if ! run_remote "${QA_SERVER}" "ping -c ${PACKET_COUNT} -W 2 10.0.0.2" 30 "ping_server_to_client"; then
        log_warn "Ping from server to client failed"
        server_to_client_success=false
    else
        log_info "Ping from server to client successful"
        server_to_client_success=true
    fi
    
    # Stop client
    log_info "Stopping SSSonector client"
    run_remote "${QA_CLIENT}" "sudo pkill -f sssonector" 5 "stop_client" || true
    
    # Wait for client to stop
    log_info "Waiting for client to stop"
    sleep 2
    
    # Check if client is still running
    log_info "Checking if client is still running"
    if run_remote "${QA_CLIENT}" "pgrep -f sssonector" 5 "check_client_stopped" > /dev/null 2>&1; then
        log_warn "Client is still running, forcing kill"
        run_remote "${QA_CLIENT}" "sudo pkill -9 -f sssonector" 5 "force_kill_client" || true
    fi
    
    # Check if tunnel is closed
    log_info "Checking if tunnel is closed"
    if run_remote "${QA_SERVER}" "ip link show tun0" 5 "check_tun0_closed" > /dev/null 2>&1; then
        log_warn "tun0 interface still exists on server after client shutdown"
    else
        log_info "Tunnel closed successfully after client shutdown"
    fi
    
    # Stop server
    log_info "Stopping SSSonector server"
    run_remote "${QA_SERVER}" "sudo pkill -f sssonector" 5 "stop_server" || true
    
    # Wait for server to stop
    log_info "Waiting for server to stop"
    sleep 2
    
    # Check if server is still running
    log_info "Checking if server is still running"
    if run_remote "${QA_SERVER}" "pgrep -f sssonector" 5 "check_server_stopped" > /dev/null 2>&1; then
        log_warn "Server is still running, forcing kill"
        run_remote "${QA_SERVER}" "sudo pkill -9 -f sssonector" 5 "force_kill_server" || true
    fi
    
    # Collect logs
    log_info "Collecting logs"
    mkdir -p "${RESULTS_DIR}/logs/${scenario}"
    copy_from_remote "${QA_SERVER}" "/opt/sssonector/log/server.log" "${RESULTS_DIR}/logs/${scenario}/server.log" || true
    copy_from_remote "${QA_CLIENT}" "/opt/sssonector/log/client.log" "${RESULTS_DIR}/logs/${scenario}/client.log" || true
    
    # Check if test was successful
    if $client_to_server_success && $server_to_client_success; then
        log_info "Test scenario ${scenario} completed successfully"
        return 0
    else
        log_warn "Test scenario ${scenario} completed with issues"
        if $client_to_server_success; then
            log_info "Client to server connectivity works"
        else
            log_warn "Client to server connectivity fails"
        fi
        if $server_to_client_success; then
            log_info "Server to client connectivity works"
        else
            log_warn "Server to client connectivity fails"
        fi
        return 1
    fi
}

# Function to run all test scenarios
run_all_tests() {
    log_step "Running all test scenarios"
    
    local test_results=()
    local test_success=true
    
    # Scenario 1: Client foreground / Server foreground
    if run_test_scenario "scenario1_client_fg_server_fg" "foreground" "foreground"; then
        test_results+=("Scenario 1 (Client foreground / Server foreground): SUCCESS")
    else
        test_results+=("Scenario 1 (Client foreground / Server foreground): FAILED")
        test_success=false
    fi
    
    # Scenario 2: Client background / Server foreground
    if run_test_scenario "scenario2_client_bg_server_fg" "foreground" "background"; then
        test_results+=("Scenario 2 (Client background / Server foreground): SUCCESS")
    else
        test_results+=("Scenario 2 (Client background / Server foreground): FAILED")
        test_success=false
    fi
    
    # Scenario 3: Client background / Server background
    if run_test_scenario "scenario3_client_bg_server_bg" "background" "background"; then
        test_results+=("Scenario 3 (Client background / Server background): SUCCESS")
    else
        test_results+=("Scenario 3 (Client background / Server background): FAILED")
        test_success=false
    fi
    
    # Print test results
    log_step "Test Results"
    for result in "${test_results[@]}"; do
        echo "${result}"
    done
    
    # Generate test report
    generate_test_report "${test_results[@]}"
    
    if $test_success; then
        log_info "All test scenarios completed successfully"
        return 0
    else
        log_warn "Some test scenarios failed"
        return 1
    fi
}

# Function to generate test report
generate_test_report() {
    log_step "Generating test report"
    
    local test_results=("$@")
    local report_file="${RESULTS_DIR}/test_report.md"
    
    # Create report file
    cat > "${report_file}" << EOF
# SSSonector QA Test Report

## Test Information

- **Date**: $(date +"%Y-%m-%d")
- **Time**: $(date +"%H:%M:%S")
- **Server**: ${QA_SERVER}
- **Client**: ${QA_CLIENT}
- **User**: ${QA_USER}
- **Test Timeout**: ${TEST_TIMEOUT} seconds
- **Packet Count**: ${PACKET_COUNT}
- **Retry Count**: ${RETRY_COUNT}

## Test Results

EOF
    
    # Add test results
    for result in "${test_results[@]}"; do
        echo "- ${result}" >> "${report_file}"
    done
    
    # Add logs and packet captures
    cat >> "${report_file}" << EOF

## Logs and Packet Captures

- **Logs**: ${RESULTS_DIR}/logs/
- **Packet Captures**: ${RESULTS_DIR}/pcaps/

## Next Steps

1. Review the logs for any errors or warnings
2. Analyze the packet captures to identify any issues
3. Address any issues identified during testing
4. Re-run the tests if necessary

EOF
    
    log_info "Test report generated: ${report_file}"
    return 0
}

# Main function
main() {
    log_step "Starting SSSonector QA Testing"
    
    # Verify environment
    verify_environment || {
        log_error "Environment verification failed"
        exit 1
    }
    
    # Clean up QA environment
    cleanup_environment || {
        log_error "QA environment cleanup failed"
        exit 1
    }
    
    # Generate certificates
    generate_certificates || {
        log_error "Certificate generation failed"
        exit 1
    }
    
    # Create configuration files
    create_configurations || {
        log_error "Configuration creation failed"
        exit 1
    }
    
    # Deploy SSSonector
    deploy_sssonector || {
        log_error "SSSonector deployment failed"
        exit 1
    }
    
    # Start packet capture
    start_packet_capture || {
        log_error "Packet capture start failed"
        exit 1
    }
    
    # Apply network fixes
    apply_network_fixes || {
        log_error "Network fixes failed"
        stop_packet_capture
        exit 1
    }
    
    # Run all tests
    run_all_tests
    local test_result=$?
    
    # Stop packet capture
    stop_packet_capture
    
    # Clean up QA environment
    cleanup_environment
    
    log_step "SSSonector QA Testing Completed"
    
    if [ ${test_result} -eq 0 ]; then
        log_info "All tests passed"
        exit 0
    else
        log_warn "Some tests failed"
        exit 1
    fi
}

# Run main function
main "$@"
