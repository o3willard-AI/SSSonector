#!/bin/bash

# verify_routing_tables.sh
# Script to verify and fix routing tables
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

# Check routing tables
check_routing_tables() {
    local host=$1
    local host_type=$2
    
    log_step "Checking routing tables on ${host_type} (${host})"
    
    # Check routing table
    log_info "Checking routing table"
    run_remote "${host}" "ip route"
    
    # Check if route for tunnel network exists
    if run_remote "${host}" "ip route | grep '10.0.0.0/24'" &> /dev/null; then
        log_info "Route for tunnel network exists"
    else
        log_warn "Route for tunnel network does not exist"
    fi
}

# Fix routing tables
fix_routing_tables() {
    local host=$1
    local host_type=$2
    local tunnel_ip=$3
    
    log_step "Fixing routing tables on ${host_type} (${host})"
    
    # Add route for tunnel network if it doesn't exist
    if ! run_remote "${host}" "ip route | grep '10.0.0.0/24'" &> /dev/null; then
        log_info "Adding route for tunnel network"
        run_remote "${host}" "sudo ip route add 10.0.0.0/24 dev tun0 src ${tunnel_ip}"
    fi
    
    # Check if default route exists
    if ! run_remote "${host}" "ip route | grep default" &> /dev/null; then
        log_warn "Default route does not exist"
        log_info "Adding default route"
        run_remote "${host}" "sudo ip route add default via 192.168.50.1 dev enp0s3"
    fi
}

# Main function
main() {
    log_info "Starting routing tables verification"
    
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
    
    # Check current routing tables
    check_routing_tables "${QA_SERVER}" "server"
    check_routing_tables "${QA_CLIENT}" "client"
    
    # Test baseline connectivity
    log_step "Testing baseline connectivity"
    start_and_test
    
    # Fix routing tables
    fix_routing_tables "${QA_SERVER}" "server" "10.0.0.1"
    fix_routing_tables "${QA_CLIENT}" "client" "10.0.0.2"
    
    # Test connectivity after fixing routing tables
    log_step "Testing connectivity after fixing routing tables"
    start_and_test
    
    log_info "Routing tables verification completed"
}

# Run main function
main "$@"
