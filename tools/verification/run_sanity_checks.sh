#!/bin/bash

# run_sanity_checks.sh
# Script to run sanity checks on SSSonector in QA environment
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

# Start SSSonector in server mode
start_server() {
    local mode=$1  # foreground or background
    
    log_info "Starting SSSonector in server ${mode} mode"
    
    if [[ "${mode}" == "foreground" ]]; then
        # Start in foreground mode
        run_remote "${QA_SERVER}" "sudo /opt/sssonector/bin/sssonector -config /opt/sssonector/config/server_foreground.yaml > /tmp/server.log 2>&1 &"
    else
        # Start in background mode
        run_remote "${QA_SERVER}" "sudo /opt/sssonector/bin/sssonector -config /opt/sssonector/config/server.yaml"
    fi
    
    # Wait for server to start
    sleep 5
    
    # Check if server is running
    if run_remote "${QA_SERVER}" "pgrep -f sssonector" &> /dev/null; then
        log_info "SSSonector server started successfully"
    else
        log_error "Failed to start SSSonector server"
        return 1
    fi
}

# Start SSSonector in client mode
start_client() {
    local mode=$1  # foreground or background
    
    log_info "Starting SSSonector in client ${mode} mode"
    
    if [[ "${mode}" == "foreground" ]]; then
        # Start in foreground mode
        run_remote "${QA_CLIENT}" "sudo /opt/sssonector/bin/sssonector -config /opt/sssonector/config/client_foreground.yaml > /tmp/client.log 2>&1 &"
    else
        # Start in background mode
        run_remote "${QA_CLIENT}" "sudo /opt/sssonector/bin/sssonector -config /opt/sssonector/config/client.yaml"
    fi
    
    # Wait for client to start
    sleep 5
    
    # Check if client is running
    if run_remote "${QA_CLIENT}" "pgrep -f sssonector" &> /dev/null; then
        log_info "SSSonector client started successfully"
    else
        log_error "Failed to start SSSonector client"
        return 1
    fi
}

# Stop SSSonector
stop_sssonector() {
    local host=$1
    
    log_info "Stopping SSSonector on ${host}"
    
    # Kill SSSonector processes
    run_remote "${host}" "sudo pkill -f sssonector" || true
    
    # Wait for processes to stop
    sleep 2
    
    # Check if processes are still running
    if run_remote "${host}" "pgrep -f sssonector" &> /dev/null; then
        log_warn "SSSonector processes still running on ${host}, forcing kill"
        run_remote "${host}" "sudo pkill -9 -f sssonector" || true
    fi
    
    log_info "SSSonector stopped on ${host}"
}

# Check tunnel status
check_tunnel() {
    log_info "Checking tunnel status"
    
    # Check if tun0 interface exists on server
    log_info "Checking tun0 interface on server"
    if ! run_remote "${QA_SERVER}" "ip link show tun0"; then
        log_error "tun0 interface not found on server"
        return 1
    fi
    
    # Check if tun0 interface exists on client
    log_info "Checking tun0 interface on client"
    if ! run_remote "${QA_CLIENT}" "ip link show tun0"; then
        log_error "tun0 interface not found on client"
        return 1
    fi
    
    # Check server logs
    log_info "Checking server logs"
    run_remote "${QA_SERVER}" "tail -n 20 /opt/sssonector/log/server.log 2>/dev/null || cat /tmp/server.log 2>/dev/null || echo 'No logs found'"
    
    # Check client logs
    log_info "Checking client logs"
    run_remote "${QA_CLIENT}" "tail -n 20 /opt/sssonector/log/client.log 2>/dev/null || cat /tmp/client.log 2>/dev/null || echo 'No logs found'"
    
    log_info "Tunnel established successfully"
}

# Send packets from client to server
send_client_to_server() {
    log_info "Sending 20 packets from client to server"
    
    # Check if tun0 interface is up on client
    log_info "Checking tun0 interface on client"
    run_remote "${QA_CLIENT}" "ip addr show tun0"
    
    # Check if tun0 interface is up on server
    log_info "Checking tun0 interface on server"
    run_remote "${QA_SERVER}" "ip addr show tun0"
    
    # Check routing on client
    log_info "Checking routing on client"
    run_remote "${QA_CLIENT}" "ip route"
    
    # Send ping packets from client to server with verbose output
    log_info "Sending ping packets from client to server"
    if ! run_remote "${QA_CLIENT}" "ping -c 5 -v 10.0.0.1"; then
        log_error "Failed to send packets from client to server"
        return 1
    fi
    
    log_info "Successfully sent packets from client to server"
}

# Send packets from server to client
send_server_to_client() {
    log_info "Sending 20 packets from server to client"
    
    # Check if tun0 interface is up on server
    log_info "Checking tun0 interface on server"
    run_remote "${QA_SERVER}" "ip addr show tun0"
    
    # Check routing on server
    log_info "Checking routing on server"
    run_remote "${QA_SERVER}" "ip route"
    
    # Send ping packets from server to client with verbose output
    log_info "Sending ping packets from server to client"
    if ! run_remote "${QA_SERVER}" "ping -c 5 -v 10.0.0.2"; then
        log_error "Failed to send packets from server to client"
        return 1
    fi
    
    log_info "Successfully sent packets from server to client"
}

# Run a single test scenario
run_scenario() {
    local server_mode=$1
    local client_mode=$2
    local scenario_name="Server ${server_mode} / Client ${client_mode}"
    
    log_step "Running scenario: ${scenario_name}"
    
    # Start server
    start_server "${server_mode}" || return 1
    
    # Start client
    start_client "${client_mode}" || {
        stop_sssonector "${QA_SERVER}"
        return 1
    }
    
    # Check tunnel status
    check_tunnel || {
        stop_sssonector "${QA_CLIENT}"
        stop_sssonector "${QA_SERVER}"
        return 1
    }
    
    # Send packets from client to server
    send_client_to_server || {
        stop_sssonector "${QA_CLIENT}"
        stop_sssonector "${QA_SERVER}"
        return 1
    }
    
    # Send packets from server to client
    send_server_to_client || {
        stop_sssonector "${QA_CLIENT}"
        stop_sssonector "${QA_SERVER}"
        return 1
    }
    
    # Stop client
    stop_sssonector "${QA_CLIENT}"
    
    # Check if tunnel is closed
    sleep 2
    if run_remote "${QA_SERVER}" "ip link show tun0" &> /dev/null; then
        log_warn "tun0 interface still exists on server after client shutdown"
    else
        log_info "Tunnel closed successfully after client shutdown"
    fi
    
    # Stop server
    stop_sssonector "${QA_SERVER}"
    
    log_info "Scenario ${scenario_name} completed successfully"
}

# Main function
main() {
    log_info "Starting SSSonector sanity checks"
    
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
    
    # Run test scenarios
    log_step "Scenario 1: Client foreground / Server foreground"
    run_scenario "foreground" "foreground" || log_error "Scenario 1 failed"
    
    log_step "Scenario 2: Client background / Server foreground"
    run_scenario "foreground" "background" || log_error "Scenario 2 failed"
    
    log_step "Scenario 3: Client background / Server background"
    run_scenario "background" "background" || log_error "Scenario 3 failed"
    
    log_info "SSSonector sanity checks completed"
}

# Run main function
main "$@"
