#!/bin/bash

# test_tunnel_transfer.sh
# Script to test file transfer through the tunnel
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
    log_info "Starting SSSonector in server foreground mode"
    
    # Start in foreground mode
    run_remote "${QA_SERVER}" "sudo /opt/sssonector/bin/sssonector -config /opt/sssonector/config/server_foreground.yaml > /tmp/server.log 2>&1 &"
    
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
    log_info "Starting SSSonector in client foreground mode"
    
    # Start in foreground mode
    run_remote "${QA_CLIENT}" "sudo /opt/sssonector/bin/sssonector -config /opt/sssonector/config/client_foreground.yaml > /tmp/client.log 2>&1 &"
    
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

# Test file transfer through the tunnel
test_file_transfer() {
    log_info "Testing file transfer through the tunnel"
    
    # Create a test file on the server
    log_info "Creating a test file on the server"
    run_remote "${QA_SERVER}" "echo 'This is a test file for SSSonector tunnel transfer' | sudo tee /tmp/test_file.txt"
    
    # Start a simple HTTP server on the server
    log_info "Starting a simple HTTP server on the server"
    run_remote "${QA_SERVER}" "cd /tmp && sudo python3 -m http.server 8000 &"
    
    # Wait for the server to start
    sleep 2
    
    # Download the file from the client through the tunnel
    log_info "Downloading the file from the client through the tunnel"
    run_remote "${QA_CLIENT}" "curl -s http://10.0.0.1:8000/test_file.txt -o /tmp/downloaded_file.txt"
    
    # Check if the file was downloaded successfully
    log_info "Checking if the file was downloaded successfully"
    if run_remote "${QA_CLIENT}" "cat /tmp/downloaded_file.txt"; then
        log_info "File transfer successful"
    else
        log_error "File transfer failed"
        return 1
    fi
    
    # Stop the HTTP server
    log_info "Stopping the HTTP server"
    run_remote "${QA_SERVER}" "sudo pkill -f 'python3 -m http.server'" || true
    
    return 0
}

# Main function
main() {
    log_info "Starting tunnel transfer test"
    
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
    
    # Start server
    start_server || exit 1
    
    # Start client
    start_client || {
        stop_sssonector "${QA_SERVER}"
        exit 1
    }
    
    # Check tunnel status
    check_tunnel || {
        stop_sssonector "${QA_CLIENT}"
        stop_sssonector "${QA_SERVER}"
        exit 1
    }
    
    # Test file transfer
    test_file_transfer || {
        stop_sssonector "${QA_CLIENT}"
        stop_sssonector "${QA_SERVER}"
        exit 1
    }
    
    # Stop client
    stop_sssonector "${QA_CLIENT}"
    
    # Stop server
    stop_sssonector "${QA_SERVER}"
    
    log_info "Tunnel transfer test completed successfully"
}

# Run main function
main "$@"
