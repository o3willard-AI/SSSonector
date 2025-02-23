#!/bin/bash

# Setup systemd service files for SSSonector
# This script creates and installs systemd service files on both server and client

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

# Service file content
SERVICE_CONTENT="[Unit]
Description=SSSonector Tunnel Service
After=network.target

[Service]
Type=simple
ExecStart=/usr/bin/sssonector -config /etc/sssonector/config.yaml
Restart=always
RestartSec=5
User=root
Group=root

[Install]
WantedBy=multi-user.target"

# Function to setup systemd service on a host
setup_service() {
    local host=$1
    log_info "Setting up systemd service on $host..."
    
    # Create temporary service file
    echo "$SERVICE_CONTENT" > /tmp/sssonector.service
    
    # Copy and install service file
    scp /tmp/sssonector.service "${host}:/tmp/sssonector.service"
    ssh "$host" "
        sudo mv /tmp/sssonector.service /etc/systemd/system/sssonector.service
        sudo chown root:root /etc/systemd/system/sssonector.service
        sudo chmod 644 /etc/systemd/system/sssonector.service
        sudo systemctl daemon-reload
        sudo systemctl enable sssonector
        echo 'Service file installed and enabled'
        
        # Verify service file
        if ! systemctl cat sssonector &>/dev/null; then
            echo 'Error: Service file not properly installed'
            exit 1
        fi
        
        # Stop service if running
        sudo systemctl stop sssonector &>/dev/null || true
    "
    
    # Clean up
    rm -f /tmp/sssonector.service
    
    log_info "Service setup complete on $host"
}

# Setup service on both hosts
setup_service "$SERVER_IP" || {
    log_error "Failed to setup service on server"
    exit 1
}

setup_service "$CLIENT_IP" || {
    log_error "Failed to setup service on client"
    exit 1
}

log_info "Systemd service setup complete"
