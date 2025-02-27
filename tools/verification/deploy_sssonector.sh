#!/bin/bash

# deploy_sssonector.sh
# Script to deploy SSSonector to QA environment for testing
set -euo pipefail

# QA environment details
QA_SERVER="192.168.50.210"
QA_CLIENT="192.168.50.211"
QA_USER="sblanken"
QA_SUDO_PASSWORD="101abn"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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

# Generate certificates
generate_certs() {
    local cert_dir="$1"
    
    log_info "Generating certificates in ${cert_dir}"
    
    # Create certificate directory if it doesn't exist
    mkdir -p "${cert_dir}"
    
    # Generate CA certificate
    log_info "Generating CA certificate"
    openssl req -x509 -new -nodes -keyout "${cert_dir}/ca.key" -sha256 -days 365 -out "${cert_dir}/ca.crt" -subj "/C=US/ST=California/L=San Francisco/O=SSSonector/OU=QA/CN=SSSonector CA"
    
    # Generate server certificate
    log_info "Generating server certificate"
    openssl req -new -nodes -keyout "${cert_dir}/server.key" -out "${cert_dir}/server.csr" -subj "/C=US/ST=California/L=San Francisco/O=SSSonector/OU=QA/CN=${QA_SERVER}"
    openssl x509 -req -in "${cert_dir}/server.csr" -CA "${cert_dir}/ca.crt" -CAkey "${cert_dir}/ca.key" -CAcreateserial -out "${cert_dir}/server.crt" -days 365 -sha256
    
    # Generate client certificate
    log_info "Generating client certificate"
    openssl req -new -nodes -keyout "${cert_dir}/client.key" -out "${cert_dir}/client.csr" -subj "/C=US/ST=California/L=San Francisco/O=SSSonector/OU=QA/CN=${QA_CLIENT}"
    openssl x509 -req -in "${cert_dir}/client.csr" -CA "${cert_dir}/ca.crt" -CAkey "${cert_dir}/ca.key" -CAcreateserial -out "${cert_dir}/client.crt" -days 365 -sha256
    
    # Clean up CSR files
    rm -f "${cert_dir}/*.csr"
    
    log_info "Certificates generated successfully"
}

# Create configuration files
create_configs() {
    local config_dir="$1"
    
    log_info "Creating configuration files in ${config_dir}"
    
    # Create configuration directory if it doesn't exist
    mkdir -p "${config_dir}"
    
    # Create server configuration
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
    mtu: 1500
  tunnel:
    listen_port: 8443
    protocol: tcp
    listen_address: 0.0.0.0
version: 1.0.0
EOF
    
    # Create client configuration
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
    mtu: 1500
  tunnel:
    server: ${QA_SERVER}:8443
    protocol: tcp
version: 1.0.0
EOF
    
    # Create server foreground configuration
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
    mtu: 1500
  tunnel:
    listen_port: 8443
    protocol: tcp
    listen_address: 0.0.0.0
  daemon:
    enabled: false
version: 1.0.0
EOF
    
    # Create client foreground configuration
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
    mtu: 1500
  tunnel:
    server: ${QA_SERVER}:8443
    protocol: tcp
  daemon:
    enabled: false
version: 1.0.0
EOF
    
    log_info "Configuration files created successfully"
}

# Deploy to a single host
deploy_to_host() {
    local host=$1
    local type=$2
    local binary=$3
    local cert_dir=$4
    local config_dir=$5
    
    log_info "Deploying SSSonector to ${type} (${host})"
    
    # Check if sshpass is installed
    if ! command -v sshpass &> /dev/null; then
        log_info "Installing sshpass..."
        sudo apt-get update && sudo apt-get install -y sshpass
    fi
    
    # Test SSH connection
    if ! sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "echo 'SSH connection test successful'" &> /dev/null; then
        log_error "Cannot SSH to ${host}"
        return 1
    fi
    
    # Create directories
    sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "sudo mkdir -p /opt/sssonector/{bin,certs,config,log,state}"
    
    # Copy binary
    sshpass -p "${QA_SUDO_PASSWORD}" scp -o StrictHostKeyChecking=no "${binary}" "${QA_USER}@${host}:/tmp/sssonector"
    sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "sudo mv /tmp/sssonector /opt/sssonector/bin/sssonector && sudo chmod 755 /opt/sssonector/bin/sssonector"
    
    # Copy certificates
    if [[ "${type}" == "server" ]]; then
        sshpass -p "${QA_SUDO_PASSWORD}" scp -o StrictHostKeyChecking=no "${cert_dir}/ca.crt" "${cert_dir}/server.crt" "${cert_dir}/server.key" "${QA_USER}@${host}:/tmp/"
        sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "sudo mv /tmp/ca.crt /tmp/server.crt /tmp/server.key /opt/sssonector/certs/ && sudo chmod 644 /opt/sssonector/certs/ca.crt /opt/sssonector/certs/server.crt && sudo chmod 600 /opt/sssonector/certs/server.key"
    else
        sshpass -p "${QA_SUDO_PASSWORD}" scp -o StrictHostKeyChecking=no "${cert_dir}/ca.crt" "${cert_dir}/client.crt" "${cert_dir}/client.key" "${QA_USER}@${host}:/tmp/"
        sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "sudo mv /tmp/ca.crt /tmp/client.crt /tmp/client.key /opt/sssonector/certs/ && sudo chmod 644 /opt/sssonector/certs/ca.crt /opt/sssonector/certs/client.crt && sudo chmod 600 /opt/sssonector/certs/client.key"
    fi
    
    # Copy configuration files
    if [[ "${type}" == "server" ]]; then
        sshpass -p "${QA_SUDO_PASSWORD}" scp -o StrictHostKeyChecking=no "${config_dir}/server.yaml" "${config_dir}/server_foreground.yaml" "${QA_USER}@${host}:/tmp/"
        sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "sudo mv /tmp/server.yaml /tmp/server_foreground.yaml /opt/sssonector/config/ && sudo chmod 644 /opt/sssonector/config/server.yaml /opt/sssonector/config/server_foreground.yaml"
    else
        sshpass -p "${QA_SUDO_PASSWORD}" scp -o StrictHostKeyChecking=no "${config_dir}/client.yaml" "${config_dir}/client_foreground.yaml" "${QA_USER}@${host}:/tmp/"
        sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "sudo mv /tmp/client.yaml /tmp/client_foreground.yaml /opt/sssonector/config/ && sudo chmod 644 /opt/sssonector/config/client.yaml /opt/sssonector/config/client_foreground.yaml"
    fi
    
    log_info "SSSonector deployed successfully to ${type}"

    # Add iptables rules to log dropped packets
    log_info "Adding iptables rules to log dropped packets"
    sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "sudo iptables -A INPUT -j LOG --log-prefix 'IPTABLES-DROPPED-INPUT: ' --log-level 4"
    sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "sudo iptables -A OUTPUT -j LOG --log-prefix 'IPTABLES-DROPPED-OUTPUT: ' --log-level 4"
    sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "sudo iptables -A FORWARD -j LOG --log-prefix 'IPTABLES-DROPPED-FORWARD: ' --log-level 4"

    # Start SSSonector service
    log_info "Starting SSSonector service"
    sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "sudo systemctl start sssonector"
    sleep 5

    # Wait for tunnel interface to be created
    log_info "Waiting for tunnel interface to be created"
    for i in $(seq 1 10); do
        if sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "ip link show tun0" &> /dev/null; then
            log_info "Tunnel interface created"
            break
        else
            log_warn "Tunnel interface not yet created, attempt $i/10"
            sleep 2
        fi
    done
}

# Main function
main() {
    local binary="/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/sssonector"
    local temp_dir="/tmp/sssonector_deploy"
    
    log_info "Starting SSSonector deployment"
    
    # Check if binary exists
    if [[ ! -f "${binary}" ]]; then
        log_error "SSSonector binary not found at ${binary}"
        exit 1
    fi
    
    # Create temporary directory
    mkdir -p "${temp_dir}/certs" "${temp_dir}/config"
    
    # Generate certificates
    generate_certs "${temp_dir}/certs"
    
    # Create configuration files
    create_configs "${temp_dir}/config"
    
    # Deploy to server
    deploy_to_host "${QA_SERVER}" "server" "${binary}" "${temp_dir}/certs" "${temp_dir}/config" || log_error "Failed to deploy to server"
    
    # Deploy to client
    deploy_to_host "${QA_CLIENT}" "client" "${binary}" "${temp_dir}/certs" "${temp_dir}/config" || log_error "Failed to deploy to client"

    # Copy logs and packet captures
    log_info "Copying logs and packet captures"
    local interface
    if [[ "${1}" == "${QA_SERVER}" ]]; then
        interface="enp0s3"
    else
        interface="enp0s3"
    fi
    sshpass -p "${QA_SUDO_PASSWORD}" scp -o StrictHostKeyChecking=no "${QA_USER}@${QA_SERVER}:/opt/sssonector/log/routing_table.log" "${temp_dir}/server_routing_table.log"
    sshpass -p "${QA_SUDO_PASSWORD}" scp -o StrictHostKeyChecking=no "${QA_USER}@${QA_SERVER}:/opt/sssonector/log/kernel_parameters.log" "${temp_dir}/server_kernel_parameters.log"
    sshpass -p "${QA_SUDO_PASSWORD}" scp -o StrictHostKeyChecking=no "${QA_USER}@${QA_SERVER}:/opt/sssonector/log/tcpdump_tun0.pcap" "${temp_dir}/server_tcpdump_tun0.pcap"
    sshpass -p "${QA_SUDO_PASSWORD}" scp -o StrictHostKeyChecking=no "${QA_USER}@${QA_SERVER}:/opt/sssonector/log/tcpdump_${interface}.pcap" "${temp_dir}/server_tcpdump_${interface}.pcap"
    sshpass -p "${QA_SUDO_PASSWORD}" scp -o StrictHostKeyChecking=no "${QA_USER}@${QA_SERVER}:/opt/sssonector/log/server.log" "${temp_dir}/server.log"
    
    sshpass -p "${QA_SUDO_PASSWORD}" scp -o StrictHostKeyChecking=no "${QA_USER}@${QA_CLIENT}:/opt/sssonector/log/routing_table.log" "${temp_dir}/client_routing_table.log"
    sshpass -p "${QA_SUDO_PASSWORD}" scp -o StrictHostKeyChecking=no "${QA_USER}@${QA_CLIENT}:/opt/sssonector/log/kernel_parameters.log" "${temp_dir}/client_kernel_parameters.log"
    sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${QA_CLIENT}:/opt/sssonector/log/tcpdump_tun0.pcap" "${temp_dir}/client_tcpdump_tun0.pcap"
    sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${QA_CLIENT}:/opt/sssonector/log/tcpdump_enp0s3.pcap" "${temp_dir}/client_tcpdump_enp0s3.pcap"
    sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${QA_CLIENT}:/opt/sssonector/log/client.log" "${temp_dir}/client.log"
    
    # Clean up temporary directory
    rm -rf "${temp_dir}"
    
    log_info "SSSonector deployment completed successfully"
    log_info "Server configuration files: /opt/sssonector/config/server.yaml, /opt/sssonector/config/server_foreground.yaml"
    log_info "Client configuration files: /opt/sssonector/config/client.yaml, /opt/sssonector/config/client_foreground.yaml"
    log_info "To run SSSonector in server mode: sudo /opt/sssonector/bin/sssonector -config /opt/sssonector/config/server.yaml"
    log_info "To run SSSonector in client mode: sudo /opt/sssonector/bin/sssonector -config /opt/sssonector/config/client.yaml"
    log_info "To run SSSonector in server foreground mode: sudo /opt/sssonector/bin/sssonector -config /opt/sssonector/config/server_foreground.yaml"
    log_info "To run SSSonector in client foreground mode: sudo /opt/sssonector/bin/sssonector -config /opt/sssonector/config/client_foreground.yaml"
}

# Run main function
main "$@"
