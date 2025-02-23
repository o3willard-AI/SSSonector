#!/bin/bash

# Setup passwordless sudo access for SSSonector service
# This script configures sudoers to allow the service to run without password

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

# Sudoers configuration content
SUDOERS_CONTENT="# Allow SSSonector service to run without password
%sudo ALL=(ALL) NOPASSWD: /usr/bin/sssonector
%sudo ALL=(ALL) NOPASSWD: /bin/systemctl start sssonector
%sudo ALL=(ALL) NOPASSWD: /bin/systemctl stop sssonector
%sudo ALL=(ALL) NOPASSWD: /bin/systemctl restart sssonector
%sudo ALL=(ALL) NOPASSWD: /bin/systemctl status sssonector
%sudo ALL=(ALL) NOPASSWD: /usr/sbin/ip link set tun0*
%sudo ALL=(ALL) NOPASSWD: /usr/sbin/ip link delete tun0*
%sudo ALL=(ALL) NOPASSWD: /usr/bin/killall sssonector
%sudo ALL=(ALL) NOPASSWD: /bin/kill -9 *sssonector*"

# Function to setup sudo access on a host
setup_sudo() {
    local host=$1
    log_info "Setting up sudo access on $host..."
    
    # Create temporary sudoers file
    echo "$SUDOERS_CONTENT" > /tmp/sssonector_sudoers
    
    # Copy and install sudoers file
    scp /tmp/sssonector_sudoers "${host}:/tmp/sssonector_sudoers"
    ssh "$host" "
        sudo chown root:root /tmp/sssonector_sudoers
        sudo chmod 440 /tmp/sssonector_sudoers
        sudo mv /tmp/sssonector_sudoers /etc/sudoers.d/sssonector
    "
    
    # Clean up
    rm -f /tmp/sssonector_sudoers
    
    log_info "Sudo access configured on $host"
}

# Setup sudo access on both hosts
setup_sudo "$SERVER_IP" || {
    log_error "Failed to setup sudo access on server"
    exit 1
}

setup_sudo "$CLIENT_IP" || {
    log_error "Failed to setup sudo access on client"
    exit 1
}

log_info "Sudo access setup complete"
