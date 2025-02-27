#!/bin/bash

# investigate_packet_filtering.sh
# Script to check packet filtering
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

# Check for DROP or REJECT rules
check_drop_reject_rules() {
    local host=$1
    local host_type=$2
    
    log_step "Checking for DROP or REJECT rules on ${host_type} (${host})"
    
    # Check iptables for DROP or REJECT rules
    log_info "Checking iptables for DROP or REJECT rules"
    run_remote "${host}" "sudo iptables -L -v -n | grep -E 'DROP|REJECT'"
    
    # Check NAT table for DROP or REJECT rules
    log_info "Checking NAT table for DROP or REJECT rules"
    run_remote "${host}" "sudo iptables -t nat -L -v -n | grep -E 'DROP|REJECT'"
    
    # Check mangle table for DROP or REJECT rules
    log_info "Checking mangle table for DROP or REJECT rules"
    run_remote "${host}" "sudo iptables -t mangle -L -v -n | grep -E 'DROP|REJECT'"
}

# Check for ebtables or arptables rules
check_other_tables() {
    local host=$1
    local host_type=$2
    
    log_step "Checking for ebtables or arptables rules on ${host_type} (${host})"
    
    # Check if ebtables is installed
    log_info "Checking if ebtables is installed"
    if run_remote "${host}" "which ebtables" &> /dev/null; then
        log_info "ebtables is installed, checking rules"
        run_remote "${host}" "sudo ebtables -L"
    else
        log_info "ebtables is not installed"
    fi
    
    # Check if arptables is installed
    log_info "Checking if arptables is installed"
    if run_remote "${host}" "which arptables" &> /dev/null; then
        log_info "arptables is installed, checking rules"
        run_remote "${host}" "sudo arptables -L"
    else
        log_info "arptables is not installed"
    fi
    
    # Check if nftables is installed
    log_info "Checking if nftables is installed"
    if run_remote "${host}" "which nft" &> /dev/null; then
        log_info "nftables is installed, checking rules"
        run_remote "${host}" "sudo nft list ruleset"
    else
        log_info "nftables is not installed"
    fi
}

# Temporarily disable filtering rules
disable_filtering() {
    local host=$1
    local host_type=$2
    
    log_step "Temporarily disabling filtering rules on ${host_type} (${host})"
    
    # Save current iptables rules
    log_info "Saving current iptables rules"
    run_remote "${host}" "sudo iptables-save > /tmp/iptables.rules"
    
    # Flush all iptables rules
    log_info "Flushing all iptables rules"
    run_remote "${host}" "sudo iptables -F"
    run_remote "${host}" "sudo iptables -t nat -F"
    run_remote "${host}" "sudo iptables -t mangle -F"
    
    # Set default policies to ACCEPT
    log_info "Setting default policies to ACCEPT"
    run_remote "${host}" "sudo iptables -P INPUT ACCEPT"
    run_remote "${host}" "sudo iptables -P FORWARD ACCEPT"
    run_remote "${host}" "sudo iptables -P OUTPUT ACCEPT"
}

# Restore filtering rules
restore_filtering() {
    local host=$1
    local host_type=$2
    
    log_step "Restoring filtering rules on ${host_type} (${host})"
    
    # Restore iptables rules
    log_info "Restoring iptables rules"
    run_remote "${host}" "sudo iptables-restore < /tmp/iptables.rules"
}

# Add exceptions for tunnel traffic
add_tunnel_exceptions() {
    local host=$1
    local host_type=$2
    
    log_step "Adding exceptions for tunnel traffic on ${host_type} (${host})"
    
    # Add rule to allow ICMP traffic from tun0 to physical interface
    log_info "Adding rule to allow ICMP traffic from tun0 to physical interface"
    run_remote "${host}" "sudo iptables -I FORWARD -p icmp -i tun0 -o enp0s3 -j ACCEPT"
    
    # Add rule to allow ICMP traffic from physical interface to tun0
    log_info "Adding rule to allow ICMP traffic from physical interface to tun0"
    run_remote "${host}" "sudo iptables -I FORWARD -p icmp -i enp0s3 -o tun0 -j ACCEPT"
    
    # Add rule to allow TCP traffic from tun0 to physical interface
    log_info "Adding rule to allow TCP traffic from tun0 to physical interface"
    run_remote "${host}" "sudo iptables -I FORWARD -p tcp -i tun0 -o enp0s3 -j ACCEPT"
    
    # Add rule to allow TCP traffic from physical interface to tun0
    log_info "Adding rule to allow TCP traffic from physical interface to tun0"
    run_remote "${host}" "sudo iptables -I FORWARD -p tcp -i enp0s3 -o tun0 -j ACCEPT"
    
    # Add rule to allow UDP traffic from tun0 to physical interface
    log_info "Adding rule to allow UDP traffic from tun0 to physical interface"
    run_remote "${host}" "sudo iptables -I FORWARD -p udp -i tun0 -o enp0s3 -j ACCEPT"
    
    # Add rule to allow UDP traffic from physical interface to tun0
    log_info "Adding rule to allow UDP traffic from physical interface to tun0"
    run_remote "${host}" "sudo iptables -I FORWARD -p udp -i enp0s3 -o tun0 -j ACCEPT"
    
    # Add rule to allow ICMP traffic to tun0 interface
    log_info "Adding rule to allow ICMP traffic to tun0 interface"
    run_remote "${host}" "sudo iptables -I INPUT -p icmp -i tun0 -j ACCEPT"
    
    # Add rule to allow TCP traffic to tun0 interface
    log_info "Adding rule to allow TCP traffic to tun0 interface"
    run_remote "${host}" "sudo iptables -I INPUT -p tcp -i tun0 -j ACCEPT"
    
    # Add rule to allow UDP traffic to tun0 interface
    log_info "Adding rule to allow UDP traffic to tun0 interface"
    run_remote "${host}" "sudo iptables -I INPUT -p udp -i tun0 -j ACCEPT"
    
    # Enable NAT for outgoing connections
    log_info "Enabling NAT for outgoing connections"
    run_remote "${host}" "sudo iptables -t nat -A POSTROUTING -o enp0s3 -j MASQUERADE"
}

# Main function
main() {
    log_info "Starting packet filtering investigation"
    
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
    
    # Check for DROP or REJECT rules
    check_drop_reject_rules "${QA_SERVER}" "server"
    check_drop_reject_rules "${QA_CLIENT}" "client"
    
    # Check for ebtables or arptables rules
    check_other_tables "${QA_SERVER}" "server"
    check_other_tables "${QA_CLIENT}" "client"
    
    # Test baseline connectivity
    log_step "Testing baseline connectivity"
    start_and_test
    
    # Temporarily disable filtering rules
    disable_filtering "${QA_SERVER}" "server"
    disable_filtering "${QA_CLIENT}" "client"
    
    # Test connectivity with filtering disabled
    log_step "Testing connectivity with filtering disabled"
    start_and_test
    
    # Restore filtering rules
    restore_filtering "${QA_SERVER}" "server"
    restore_filtering "${QA_CLIENT}" "client"
    
    # Add exceptions for tunnel traffic
    add_tunnel_exceptions "${QA_SERVER}" "server"
    add_tunnel_exceptions "${QA_CLIENT}" "client"
    
    # Test connectivity with tunnel exceptions
    log_step "Testing connectivity with tunnel exceptions"
    start_and_test
    
    log_info "Packet filtering investigation completed"
}

# Run main function
main "$@"
