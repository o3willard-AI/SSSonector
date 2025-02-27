#!/bin/bash

# investigate_firewall_rules.sh
# Script to investigate and add firewall rules for ICMP traffic
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

# Start SSSonector and test ping
start_and_test() {
    log_step "Starting SSSonector and testing ping"
    
    # Start SSSonector on server
    log_info "Starting SSSonector on server"
    run_remote "${QA_SERVER}" "sudo /opt/sssonector/bin/sssonector -config /opt/sssonector/config/server_foreground.yaml > /tmp/server.log 2>&1 &"
    sleep 5
    
    # Start SSSonector on client
    log_info "Starting SSSonector on client"
    run_remote "${QA_CLIENT}" "sudo /opt/sssonector/bin/sssonector -config /opt/sssonector/config/client_foreground.yaml > /tmp/client.log 2>&1 &"
    sleep 5
    
    # Test ping from client to server
    log_info "Testing ping from client to server"
    run_remote "${QA_CLIENT}" "ping -c 5 10.0.0.1" || log_warn "Ping from client to server failed"
    
    # Test ping from server to client
    log_info "Testing ping from server to client"
    run_remote "${QA_SERVER}" "ping -c 5 10.0.0.2" || log_warn "Ping from server to client failed"
    
    # Stop SSSonector
    log_info "Stopping SSSonector"
    run_remote "${QA_CLIENT}" "sudo pkill -f sssonector || true"
    run_remote "${QA_SERVER}" "sudo pkill -f sssonector || true"
    sleep 2
}

# Check current firewall rules
check_firewall_rules() {
    local host=$1
    local host_type=$2
    
    log_step "Checking current firewall rules on ${host_type} (${host})"
    
    # Check iptables rules
    log_info "Checking iptables rules"
    run_remote "${host}" "sudo iptables -L -v -n"
    
    # Check NAT rules
    log_info "Checking NAT rules"
    run_remote "${host}" "sudo iptables -t nat -L -v -n"
}

# Add ICMP rules to FORWARD chain
add_icmp_forward_rules() {
    local host=$1
    local host_type=$2
    
    log_step "Adding ICMP rules to FORWARD chain on ${host_type} (${host})"
    
    # Add rule to allow ICMP traffic from tun0 to physical interface
    log_info "Adding rule to allow ICMP traffic from tun0 to physical interface"
    run_remote "${host}" "sudo iptables -I FORWARD -p icmp -i tun0 -o enp0s3 -j ACCEPT"
    
    # Add rule to allow ICMP traffic from physical interface to tun0
    log_info "Adding rule to allow ICMP traffic from physical interface to tun0"
    run_remote "${host}" "sudo iptables -I FORWARD -p icmp -i enp0s3 -o tun0 -j ACCEPT"
}

# Add ICMP rules to INPUT chain
add_icmp_input_rules() {
    local host=$1
    local host_type=$2
    
    log_step "Adding ICMP rules to INPUT chain on ${host_type} (${host})"
    
    # Add rule to allow ICMP traffic to tun0 interface
    log_info "Adding rule to allow ICMP traffic to tun0 interface"
    run_remote "${host}" "sudo iptables -I INPUT -p icmp -i tun0 -j ACCEPT"
}

# Main function
main() {
    log_info "Starting firewall rules investigation"
    
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
    
    # Check current firewall rules
    check_firewall_rules "${QA_SERVER}" "server"
    check_firewall_rules "${QA_CLIENT}" "client"
    
    # Test baseline connectivity
    log_step "Testing baseline connectivity"
    start_and_test
    
    # Add ICMP rules to FORWARD chain
    add_icmp_forward_rules "${QA_SERVER}" "server"
    add_icmp_forward_rules "${QA_CLIENT}" "client"
    
    # Test connectivity after adding FORWARD rules
    log_step "Testing connectivity after adding FORWARD rules"
    start_and_test
    
    # Add ICMP rules to INPUT chain
    add_icmp_input_rules "${QA_SERVER}" "server"
    add_icmp_input_rules "${QA_CLIENT}" "client"
    
    # Test connectivity after adding INPUT rules
    log_step "Testing connectivity after adding INPUT rules"
    start_and_test
    
    log_info "Firewall rules investigation completed"
}

# Run main function
main "$@"
