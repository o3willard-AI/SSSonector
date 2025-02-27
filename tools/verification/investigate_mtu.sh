#!/bin/bash

# investigate_mtu.sh
# Script to investigate MTU issues
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

# Start SSSonector
start_sssonector() {
    log_step "Starting SSSonector"
    
    # Start SSSonector on server
    log_info "Starting SSSonector on server"
    run_remote "${QA_SERVER}" "sudo /opt/sssonector/bin/sssonector -config /opt/sssonector/config/server_foreground.yaml > /tmp/server.log 2>&1 &"
    sleep 5
    
    # Start SSSonector on client
    log_info "Starting SSSonector on client"
    run_remote "${QA_CLIENT}" "sudo /opt/sssonector/bin/sssonector -config /opt/sssonector/config/client_foreground.yaml > /tmp/client.log 2>&1 &"
    sleep 5
}

# Stop SSSonector
stop_sssonector() {
    log_info "Stopping SSSonector"
    run_remote "${QA_CLIENT}" "sudo pkill -f sssonector || true"
    run_remote "${QA_SERVER}" "sudo pkill -f sssonector || true"
    sleep 2
}

# Check MTU settings
check_mtu() {
    local host=$1
    local host_type=$2
    
    log_step "Checking MTU settings on ${host_type} (${host})"
    
    # Check MTU of all interfaces
    log_info "Checking MTU of all interfaces"
    run_remote "${host}" "ip link show | grep mtu"
    
    # Check Path MTU Discovery settings
    log_info "Checking Path MTU Discovery settings"
    run_remote "${host}" "cat /proc/sys/net/ipv4/ip_no_pmtu_disc"
}

# Test ping with different packet sizes
test_ping_sizes() {
    local host=$1
    local target=$2
    local max_size=$3
    
    log_step "Testing ping with different packet sizes from ${host} to ${target}"
    
    # Test ping with different packet sizes
    for size in 64 128 256 512 1024 1472 1473 1500 1600 2000; do
        if [ "${size}" -le "${max_size}" ]; then
            log_info "Testing ping with packet size ${size}"
            run_remote "${host}" "ping -c 3 -s ${size} ${target}" || log_warn "Ping with packet size ${size} failed"
        fi
    done
}

# Adjust MTU
adjust_mtu() {
    local host=$1
    local host_type=$2
    local interface=$3
    local mtu=$4
    
    log_step "Adjusting MTU on ${host_type} (${host})"
    
    # Adjust MTU
    log_info "Setting MTU of ${interface} to ${mtu}"
    run_remote "${host}" "sudo ip link set ${interface} mtu ${mtu}"
    
    # Verify MTU
    log_info "Verifying MTU of ${interface}"
    run_remote "${host}" "ip link show ${interface} | grep mtu"
}

# Test with different MTU values
test_mtu_values() {
    local server_interface="tun0"
    local client_interface="tun0"
    
    # Test with default MTU (1500)
    log_step "Testing with default MTU (1500)"
    start_sssonector
    check_mtu "${QA_SERVER}" "server"
    check_mtu "${QA_CLIENT}" "client"
    test_ping_sizes "${QA_CLIENT}" "10.0.0.1" "1500"
    stop_sssonector
    
    # Test with MTU 1400
    log_step "Testing with MTU 1400"
    start_sssonector
    adjust_mtu "${QA_SERVER}" "server" "${server_interface}" "1400"
    adjust_mtu "${QA_CLIENT}" "client" "${client_interface}" "1400"
    test_ping_sizes "${QA_CLIENT}" "10.0.0.1" "1400"
    stop_sssonector
    
    # Test with MTU 1200
    log_step "Testing with MTU 1200"
    start_sssonector
    adjust_mtu "${QA_SERVER}" "server" "${server_interface}" "1200"
    adjust_mtu "${QA_CLIENT}" "client" "${client_interface}" "1200"
    test_ping_sizes "${QA_CLIENT}" "10.0.0.1" "1200"
    stop_sssonector
    
    # Test with MTU 1000
    log_step "Testing with MTU 1000"
    start_sssonector
    adjust_mtu "${QA_SERVER}" "server" "${server_interface}" "1000"
    adjust_mtu "${QA_CLIENT}" "client" "${client_interface}" "1000"
    test_ping_sizes "${QA_CLIENT}" "10.0.0.1" "1000"
    stop_sssonector
    
    # Test with MTU 576
    log_step "Testing with MTU 576"
    start_sssonector
    adjust_mtu "${QA_SERVER}" "server" "${server_interface}" "576"
    adjust_mtu "${QA_CLIENT}" "client" "${client_interface}" "576"
    test_ping_sizes "${QA_CLIENT}" "10.0.0.1" "576"
    stop_sssonector
}

# Adjust Path MTU Discovery
adjust_pmtu_discovery() {
    local host=$1
    local host_type=$2
    local value=$3
    
    log_step "Adjusting Path MTU Discovery on ${host_type} (${host})"
    
    # Adjust Path MTU Discovery
    log_info "Setting ip_no_pmtu_disc to ${value}"
    run_remote "${host}" "echo ${value} | sudo tee /proc/sys/net/ipv4/ip_no_pmtu_disc"
    
    # Verify Path MTU Discovery
    log_info "Verifying ip_no_pmtu_disc"
    run_remote "${host}" "cat /proc/sys/net/ipv4/ip_no_pmtu_disc"
}

# Test with Path MTU Discovery enabled/disabled
test_pmtu_discovery() {
    # Test with Path MTU Discovery enabled
    log_step "Testing with Path MTU Discovery enabled"
    adjust_pmtu_discovery "${QA_SERVER}" "server" "0"
    adjust_pmtu_discovery "${QA_CLIENT}" "client" "0"
    start_sssonector
    test_ping_sizes "${QA_CLIENT}" "10.0.0.1" "1500"
    stop_sssonector
    
    # Test with Path MTU Discovery disabled
    log_step "Testing with Path MTU Discovery disabled"
    adjust_pmtu_discovery "${QA_SERVER}" "server" "1"
    adjust_pmtu_discovery "${QA_CLIENT}" "client" "1"
    start_sssonector
    test_ping_sizes "${QA_CLIENT}" "10.0.0.1" "1500"
    stop_sssonector
}

# Main function
main() {
    log_info "Starting MTU investigation"
    
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
    
    # Check current MTU settings
    check_mtu "${QA_SERVER}" "server"
    check_mtu "${QA_CLIENT}" "client"
    
    # Test with different MTU values
    test_mtu_values
    
    # Test with Path MTU Discovery enabled/disabled
    test_pmtu_discovery
    
    log_info "MTU investigation completed"
}

# Run main function
main "$@"
