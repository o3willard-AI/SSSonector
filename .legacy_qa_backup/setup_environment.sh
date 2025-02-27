#!/bin/bash

# Setup environment for SSSonector testing
# This script configures system settings and permissions required for testing

set -euo pipefail

SERVER_HOST="sblanken@192.168.50.210"
CLIENT_HOST="sblanken@192.168.50.211"

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

# Configure system settings on a host
configure_host() {
    local host=$1
    log_info "Configuring system settings on $host..."
    
    ssh "$host" "
        # Enable IP forwarding
        sudo sysctl -w net.ipv4.ip_forward=1
        
        # Create required directories with correct permissions
        sudo mkdir -p ~/sssonector/{bin,config,certs,log,state}
        sudo chown -R \$USER:\$USER ~/sssonector
        sudo chmod -R 755 ~/sssonector
        sudo chmod 700 ~/sssonector/certs
        sudo chmod 755 ~/sssonector/log
        sudo chmod 755 ~/sssonector/state
        
        # Create log file if it doesn't exist
        sudo touch ~/sssonector/log/sssonector.log
        sudo chown \$USER:\$USER ~/sssonector/log/sssonector.log
        sudo chmod 644 ~/sssonector/log/sssonector.log
        
        # Add user to required groups
        sudo usermod -a -G tun \$USER || true
        
        # Configure TUN device permissions
        sudo mkdir -p /dev/net
        sudo mknod /dev/net/tun c 10 200 || true
        sudo chmod 0666 /dev/net/tun
    "
}

# Main execution
main() {
    log_info "Starting environment setup..."
    
    # Configure both hosts
    configure_host "$SERVER_HOST"
    configure_host "$CLIENT_HOST"
    
    log_info "Environment setup complete"
}

# Execute main function
main
