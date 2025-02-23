#!/bin/bash

# Control script for SSSonector tunnel testing
# This script helps manage the tunnel server and client processes

set -euo pipefail

SERVER_HOST="sblanken@192.168.50.210"
CLIENT_HOST="sblanken@192.168.50.211"

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Clean up any existing tunnel processes and interfaces
cleanup() {
    log_info "Running resource cleanup..."
    ./cleanup_resources.sh
}

# Start the tunnel process
start_tunnel() {
    local host=$1
    local mode=$2
    log_info "Starting $mode on $host..."
    
    # Start the process in the background
    ssh -t "$host" "sudo ~/sssonector/bin/sssonector -config ~/sssonector/config/config.yaml -debug" &
    
    # Wait for interface to come up
    sleep 5
    
    # Verify interface is up
    if ssh "$host" "ip addr show tun0" &>/dev/null; then
        log_info "$mode started successfully"
    else
        log_error "Failed to start $mode"
        return 1
    fi
}

# Check tunnel connectivity
check_connectivity() {
    log_info "Checking tunnel connectivity..."
    
    # Try to ping server from client
    if ssh "$CLIENT_HOST" "ping -c 3 -W 2 10.0.0.1" &>/dev/null; then
        log_info "Tunnel connectivity test passed"
        return 0
    else
        log_error "Tunnel connectivity test failed"
        return 1
    fi
}

# Show tunnel status
show_status() {
    log_info "=== Server Status ==="
    ssh "$SERVER_HOST" "
        echo 'Process:'
        ps aux | grep [s]ssonector
        echo
        echo 'Interface:'
        ip addr show tun0
        echo
        echo 'Connections:'
        sudo netstat -anp | grep sssonector
    "
    
    log_info "=== Client Status ==="
    ssh "$CLIENT_HOST" "
        echo 'Process:'
        ps aux | grep [s]ssonector
        echo
        echo 'Interface:'
        ip addr show tun0
        echo
        echo 'Connections:'
        sudo netstat -anp | grep sssonector
    "
}

# Command line interface
case "${1:-help}" in
    start)
        # Clean up any existing instances
        cleanup
        
        # Start server and client
        start_tunnel "$SERVER_HOST" "server"
        sleep 5  # Wait for server to be ready
        start_tunnel "$CLIENT_HOST" "client"
        
        # Check connectivity
        sleep 5
        check_connectivity
        ;;
        
    stop)
        cleanup
        log_info "Tunnel stopped"
        ;;
        
    status)
        show_status
        ;;
        
    restart)
        $0 stop
        sleep 2
        $0 start
        ;;
        
    *)
        echo "Usage: $0 {start|stop|status|restart}"
        echo
        echo "Commands:"
        echo "  start   - Start both server and client"
        echo "  stop    - Stop both server and client"
        echo "  status  - Show tunnel status"
        echo "  restart - Restart both server and client"
        exit 1
        ;;
esac
