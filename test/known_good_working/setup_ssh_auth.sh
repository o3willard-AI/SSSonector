#!/bin/bash

# Setup SSH key-based authentication for test environment
# This script sets up passwordless SSH access between the test machine and target hosts

set -euo pipefail

SERVER_IP=${SERVER_IP:-"192.168.50.210"}
CLIENT_IP=${CLIENT_IP:-"192.168.50.211"}

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Validate IP address format
validate_ip() {
    local ip=$1
    if [[ ! $ip =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
        return 1
    fi
    for octet in $(echo "$ip" | tr '.' ' '); do
        if [[ $octet -lt 0 || $octet -gt 255 ]]; then
            return 1
        fi
    done
    return 0
}

# Validate environment
validate_environment() {
    # Check if SSH client is installed
    if ! command -v ssh >/dev/null 2>&1; then
        log_error "SSH client is not installed"
        exit 1
    fi

    # Check if ssh-keygen is available
    if ! command -v ssh-keygen >/dev/null 2>&1; then
        log_error "ssh-keygen is not installed"
        exit 1
    fi

    # Validate IP addresses
    if ! validate_ip "$SERVER_IP"; then
        log_error "Invalid server IP address: $SERVER_IP"
        exit 1
    fi
    if ! validate_ip "$CLIENT_IP"; then
        log_error "Invalid client IP address: $CLIENT_IP"
        exit 1
    fi

    # Check if IPs are reachable
    if ! ping -c 1 -W 2 "$SERVER_IP" >/dev/null 2>&1; then
        log_error "Server IP $SERVER_IP is not reachable"
        exit 1
    fi
    if ! ping -c 1 -W 2 "$CLIENT_IP" >/dev/null 2>&1; then
        log_error "Client IP $CLIENT_IP is not reachable"
        exit 1
    fi
}

# Setup SSH directory with secure permissions
setup_ssh_directory() {
    if [ ! -d ~/.ssh ]; then
        log_info "Creating ~/.ssh directory..."
        mkdir -m 700 ~/.ssh
    else
        # Ensure correct permissions on existing directory
        chmod 700 ~/.ssh
    fi
}

# Generate SSH key with enhanced security
generate_ssh_key() {
    if [ -f ~/.ssh/id_rsa ]; then
        log_warn "SSH key already exists. Backing up..."
        cp ~/.ssh/id_rsa{,.bak}
        cp ~/.ssh/id_rsa.pub{,.bak}
    fi
    
    log_info "Generating new SSH key pair..."
    ssh-keygen -t rsa -b 4096 -f ~/.ssh/id_rsa -N "" -C "sssonector_$(date +%Y%m%d)"
    chmod 600 ~/.ssh/id_rsa
    chmod 644 ~/.ssh/id_rsa.pub
}

# Function to copy SSH key to a host with enhanced error handling
copy_key_to_host() {
    local host=$1
    local max_retries=3
    local retry_count=0
    local wait_time=5

    log_info "Copying SSH key to $host..."
    
    while [ $retry_count -lt $max_retries ]; do
        if ssh-copy-id -i ~/.ssh/id_rsa.pub "$host" &>/dev/null; then
            # Verify SSH access
            if ssh -o BatchMode=yes -o ConnectTimeout=5 "$host" "echo 'SSH access verified'" &>/dev/null; then
                log_info "Successfully set up SSH access to $host"
                return 0
            fi
        fi
        
        retry_count=$((retry_count + 1))
        if [ $retry_count -lt $max_retries ]; then
            log_warn "Attempt $retry_count failed. Retrying in $wait_time seconds..."
            sleep $wait_time
        fi
    done
    
    log_error "Failed to set up SSH access to $host after $max_retries attempts"
    return 1
}

# Main execution
main() {
    log_info "Starting SSH authentication setup..."
    
    # Validate environment first
    validate_environment
    
    # Setup SSH directory
    setup_ssh_directory
    
    # Generate or verify SSH key
    if [ ! -f ~/.ssh/id_rsa ]; then
        generate_ssh_key
    else
        log_info "Using existing SSH key"
    fi
    
    # Copy keys to both hosts
    copy_key_to_host "$SERVER_IP" || exit 1
    copy_key_to_host "$CLIENT_IP" || exit 1
    
    # Final verification
    log_info "Verifying SSH access to both hosts..."
    if ssh -o BatchMode=yes -o ConnectTimeout=5 "$SERVER_IP" "echo 'Server connection verified'" &>/dev/null &&
       ssh -o BatchMode=yes -o ConnectTimeout=5 "$CLIENT_IP" "echo 'Client connection verified'" &>/dev/null; then
        log_info "SSH key-based authentication setup complete and verified"
    else
        log_error "Final verification failed. Please check the connections manually"
        exit 1
    fi
}

# Execute main function
main
