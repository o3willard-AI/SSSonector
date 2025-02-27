#!/bin/bash

# check_firewall.sh
# Script to check firewall and iptables rules on QA systems
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

# Check firewall status
check_firewall() {
    local host=$1
    local host_type=$2
    
    log_step "Checking firewall on ${host_type} (${host})"
    
    # Check if ufw is installed and enabled
    log_info "Checking UFW status"
    run_remote "${host}" "sudo ufw status verbose || echo 'UFW not installed'"
    
    # Check iptables rules
    log_info "Checking iptables rules"
    run_remote "${host}" "sudo iptables -L -v -n"
    
    # Check NAT rules
    log_info "Checking NAT rules"
    run_remote "${host}" "sudo iptables -t nat -L -v -n"
    
    # Check if IP forwarding is enabled in sysctl
    log_info "Checking IP forwarding in sysctl"
    run_remote "${host}" "cat /proc/sys/net/ipv4/ip_forward"
    
    # Check if IP forwarding is enabled in sysctl.conf
    log_info "Checking IP forwarding in sysctl.conf"
    run_remote "${host}" "grep -r 'net.ipv4.ip_forward' /etc/sysctl.conf /etc/sysctl.d/ || echo 'Not found in sysctl.conf'"
    
    # Check routing
    log_info "Checking routing table"
    run_remote "${host}" "ip route"
    
    # Check tun0 interface
    log_info "Checking tun0 interface"
    run_remote "${host}" "ip addr show tun0 || echo 'tun0 interface not found'"
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
    
    # Allow forwarding between tun0 and eth0
    run_remote "${host}" "sudo iptables -A FORWARD -i tun0 -o eth0 -j ACCEPT"
    run_remote "${host}" "sudo iptables -A FORWARD -i eth0 -o tun0 -j ACCEPT"
    
    # Enable NAT for outgoing connections
    run_remote "${host}" "sudo iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE"
    
    # Save iptables rules
    log_info "Saving iptables rules"
    run_remote "${host}" "sudo sh -c 'iptables-save > /etc/iptables.rules'"
    
    log_info "Forwarding rules added successfully"
}

# Main function
main() {
    log_info "Starting firewall check"
    
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
    
    # Check firewall on server
    check_firewall "${QA_SERVER}" "server"
    
    # Check firewall on client
    check_firewall "${QA_CLIENT}" "client"
    
    # Ask if user wants to add forwarding rules
    read -p "Do you want to add forwarding rules to both systems? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        # Add forwarding rules to server
        add_forwarding_rules "${QA_SERVER}" "server"
        
        # Add forwarding rules to client
        add_forwarding_rules "${QA_CLIENT}" "client"
        
        log_info "Forwarding rules added to both systems"
    else
        log_info "Skipping adding forwarding rules"
    fi
    
    log_info "Firewall check completed"
}

# Run main function
main "$@"
