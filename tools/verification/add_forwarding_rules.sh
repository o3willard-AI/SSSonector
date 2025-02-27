#!/bin/bash

# add_forwarding_rules.sh
# Script to add forwarding rules to QA systems
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
BLUE='\033[0;34m'
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

# Run command on remote host
run_remote() {
    local host=$1
    local command=$2
    
    sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "${command}"
}

# Add iptables rules for forwarding
add_forwarding_rules() {
    local host=$1
    local host_type=$2
    
    log_step "Adding forwarding rules on ${host_type} (${host})"
    
    # Enable IP forwarding
    log_info "Enabling IP forwarding"
    run_remote "${host}" "echo 1 | sudo tee /proc/sys/net/ipv4/ip_forward"
    
    # Add iptables rules for forwarding
    log_info "Adding iptables rules for forwarding"
    
    # Flush existing rules
    run_remote "${host}" "sudo iptables -F FORWARD"
    run_remote "${host}" "sudo iptables -t nat -F POSTROUTING"
    
    # Allow forwarding between tun0 and eth0/enp0s3
    run_remote "${host}" "sudo iptables -A FORWARD -i tun0 -o enp0s3 -j ACCEPT"
    run_remote "${host}" "sudo iptables -A FORWARD -i enp0s3 -o tun0 -j ACCEPT"
    
    # Enable NAT for outgoing connections
    run_remote "${host}" "sudo iptables -t nat -A POSTROUTING -o enp0s3 -j MASQUERADE"
    
    # Save iptables rules
    log_info "Saving iptables rules"
    run_remote "${host}" "sudo sh -c 'iptables-save > /etc/iptables.rules'"
    
    # Make IP forwarding persistent
    log_info "Making IP forwarding persistent"
    run_remote "${host}" "echo 'net.ipv4.ip_forward=1' | sudo tee /etc/sysctl.d/99-ip-forward.conf"
    run_remote "${host}" "sudo sysctl -p /etc/sysctl.d/99-ip-forward.conf"
    
    log_info "Forwarding rules added successfully"
}

# Main function
main() {
    log_info "Starting to add forwarding rules"
    
    # Check if sshpass is installed
    if ! command -v sshpass &> /dev/null; then
        log_info "Installing sshpass..."
        sudo apt-get update && sudo apt-get install -y sshpass
    fi
    
    # Test SSH connection to QA servers
    log_info "Testing SSH connection to QA servers"
    
    if ! sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${QA_SERVER}" "echo 'SSH connection test successful'" &> /dev/null; then
        log_error "Cannot SSH to server ${QA_SERVER}"
        exit 1
    fi
    
    if ! sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${QA_CLIENT}" "echo 'SSH connection test successful'" &> /dev/null; then
        log_error "Cannot SSH to client ${QA_CLIENT}"
        exit 1
    fi
    
    # Add forwarding rules to server
    add_forwarding_rules "${QA_SERVER}" "server"
    
    # Add forwarding rules to client
    add_forwarding_rules "${QA_CLIENT}" "client"
    
    log_info "Forwarding rules added to both systems"
    
    # Restart SSSonector services
    log_info "Restarting SSSonector services"
    
    # Stop any running SSSonector processes
    run_remote "${QA_SERVER}" "sudo pkill -f sssonector || true"
    run_remote "${QA_CLIENT}" "sudo pkill -f sssonector || true"
    
    log_info "Forwarding rules setup completed"
}

# Run main function
main "$@"
