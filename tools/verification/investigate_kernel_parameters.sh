#!/bin/bash

# investigate_kernel_parameters.sh
# Script to investigate and adjust kernel parameters
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

# Check kernel parameters
check_kernel_parameters() {
    local host=$1
    local host_type=$2
    
    log_step "Checking kernel parameters on ${host_type} (${host})"
    
    # Check IP forwarding
    log_info "Checking IP forwarding"
    run_remote "${host}" "cat /proc/sys/net/ipv4/ip_forward"
    
    # Check reverse path filtering
    log_info "Checking reverse path filtering"
    run_remote "${host}" "cat /proc/sys/net/ipv4/conf/all/rp_filter"
    run_remote "${host}" "cat /proc/sys/net/ipv4/conf/default/rp_filter"
    run_remote "${host}" "cat /proc/sys/net/ipv4/conf/tun0/rp_filter 2>/dev/null || echo 'tun0 not found'"
    
    # Check ICMP echo ignore
    log_info "Checking ICMP echo ignore"
    run_remote "${host}" "cat /proc/sys/net/ipv4/icmp_echo_ignore_all"
    run_remote "${host}" "cat /proc/sys/net/ipv4/icmp_echo_ignore_broadcasts"
    
    # Check accept source route
    log_info "Checking accept source route"
    run_remote "${host}" "cat /proc/sys/net/ipv4/conf/all/accept_source_route"
    run_remote "${host}" "cat /proc/sys/net/ipv4/conf/default/accept_source_route"
    run_remote "${host}" "cat /proc/sys/net/ipv4/conf/tun0/accept_source_route 2>/dev/null || echo 'tun0 not found'"
}

# Adjust kernel parameters
adjust_kernel_parameters() {
    local host=$1
    local host_type=$2
    
    log_step "Adjusting kernel parameters on ${host_type} (${host})"
    
    # Enable IP forwarding
    log_info "Enabling IP forwarding"
    run_remote "${host}" "echo 1 | sudo tee /proc/sys/net/ipv4/ip_forward"
    
    # Disable reverse path filtering
    log_info "Disabling reverse path filtering"
    run_remote "${host}" "echo 0 | sudo tee /proc/sys/net/ipv4/conf/all/rp_filter"
    run_remote "${host}" "echo 0 | sudo tee /proc/sys/net/ipv4/conf/default/rp_filter"
    run_remote "${host}" "sudo sysctl -w net.ipv4.conf.all.rp_filter=0"
    run_remote "${host}" "sudo sysctl -w net.ipv4.conf.default.rp_filter=0"
    
    # Disable ICMP echo ignore
    log_info "Disabling ICMP echo ignore"
    run_remote "${host}" "echo 0 | sudo tee /proc/sys/net/ipv4/icmp_echo_ignore_all"
    run_remote "${host}" "echo 0 | sudo tee /proc/sys/net/ipv4/icmp_echo_ignore_broadcasts"
    
    # Enable accept source route
    log_info "Enabling accept source route"
    run_remote "${host}" "echo 1 | sudo tee /proc/sys/net/ipv4/conf/all/accept_source_route"
    run_remote "${host}" "echo 1 | sudo tee /proc/sys/net/ipv4/conf/default/accept_source_route"
}

# Main function
main() {
    log_info "Starting kernel parameters investigation"
    
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
    
    # Check current kernel parameters
    check_kernel_parameters "${QA_SERVER}" "server"
    check_kernel_parameters "${QA_CLIENT}" "client"
    
    # Test baseline connectivity
    log_step "Testing baseline connectivity"
    start_and_test
    
    # Adjust kernel parameters
    adjust_kernel_parameters "${QA_SERVER}" "server"
    adjust_kernel_parameters "${QA_CLIENT}" "client"
    
    # Test connectivity after adjusting kernel parameters
    log_step "Testing connectivity after adjusting kernel parameters"
    start_and_test
    
    log_info "Kernel parameters investigation completed"
}

# Run main function
main "$@"
