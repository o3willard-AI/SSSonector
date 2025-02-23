#!/bin/bash

# Setup SSH key-based authentication for test environment
# This script sets up passwordless SSH access between the test machine and target hosts

set -euo pipefail

SERVER_IP=${SERVER_IP:-"192.168.50.210"}
CLIENT_IP=${CLIENT_IP:-"192.168.50.211"}

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if SSH key exists, generate if not
if [ ! -f ~/.ssh/id_rsa ]; then
    log_info "Generating SSH key pair..."
    ssh-keygen -t rsa -b 4096 -f ~/.ssh/id_rsa -N ""
fi

# Function to copy SSH key to a host
copy_key_to_host() {
    local host=$1
    log_info "Copying SSH key to $host..."
    
    # First try to copy the key
    if ! ssh-copy-id -i ~/.ssh/id_rsa.pub "$host" &>/dev/null; then
        log_error "Failed to copy SSH key to $host"
        return 1
    fi
    
    # Verify SSH access
    if ! ssh -o BatchMode=yes -o ConnectTimeout=5 "$host" "echo 'SSH access verified'" &>/dev/null; then
        log_error "Failed to verify SSH access to $host"
        return 1
    fi
    
    log_info "Successfully set up SSH access to $host"
    return 0
}

# Copy keys to both hosts
copy_key_to_host "$SERVER_IP" || exit 1
copy_key_to_host "$CLIENT_IP" || exit 1

log_info "SSH key-based authentication setup complete"
