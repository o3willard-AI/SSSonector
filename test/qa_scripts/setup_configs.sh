#!/bin/bash

# Setup configuration files for SSSonector test environment
# This script copies the necessary configuration files to the server and client machines

set -euo pipefail

SERVER_IP=${SERVER_IP:-"192.168.50.210"}
CLIENT_IP=${CLIENT_IP:-"192.168.50.211"}
PROJECT_ROOT="/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector"

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

# Function to setup configuration on a host
setup_host_config() {
    local host=$1
    local mode=$2
    local config_file="$PROJECT_ROOT/configs/${mode}.yaml"
    
    log_info "Setting up $mode configuration on $host..."
    
    # Create necessary directories
    ssh "$host" "sudo mkdir -p /etc/sssonector/certs /var/log/sssonector"
    
    # Copy configuration file
    scp "$config_file" "${host}:/tmp/${mode}.yaml"
    ssh "$host" "sudo mv /tmp/${mode}.yaml /etc/sssonector/config.yaml"
    
    # Set proper permissions and verify configuration
    ssh "$host" "
        sudo chown -R root:root /etc/sssonector
        sudo chmod 644 /etc/sssonector/config.yaml
        sudo chown -R root:root /var/log/sssonector
        sudo chmod 755 /var/log/sssonector
        
        echo 'Verifying configuration:'
        cat /etc/sssonector/config.yaml
        
        if ! grep -q '^type:' /etc/sssonector/config.yaml; then
            echo 'Error: Configuration missing type field'
            exit 1
        fi
        
        if ! grep -q '^config:' /etc/sssonector/config.yaml; then
            echo 'Error: Configuration missing config section'
            exit 1
        fi
    "
    
    log_info "Configuration setup and verified for $host"
}

# Setup server configuration
setup_host_config "$SERVER_IP" "server" || {
    log_error "Failed to setup server configuration"
    exit 1
}

# Setup client configuration
setup_host_config "$CLIENT_IP" "client" || {
    log_error "Failed to setup client configuration"
    exit 1
}

log_info "Configuration setup complete for both hosts"
