#!/bin/bash

# Cleanup script for SSSonector testing
# This script ensures TUN interfaces and ports are free before testing

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

# Clean up resources on a host
cleanup_host() {
    local host=$1
    local role=$2
    log_info "Cleaning up resources on $host ($role)..."
    
    # Kill any existing sssonector processes
    ssh "$host" "
        # Kill sssonector processes
        sudo pkill -f sssonector || true
        
        # Remove TUN interface
        sudo ip link delete tun0 2>/dev/null || true
        
        # Kill any process using port 8080 (server only)
        if [ \"$role\" = \"server\" ]; then
            pid=\$(sudo lsof -t -i:8080 2>/dev/null || true)
            if [ -n \"\$pid\" ]; then
                sudo kill -9 \$pid
            fi
        fi
        
        # Wait for processes to terminate
        sleep 2
        
        # Verify port 8080 is free (server only)
        if [ \"$role\" = \"server\" ]; then
            if sudo ss -tuln | grep -q :8080; then
                echo 'Error: Port 8080 is still in use'
                exit 1
            fi
        fi
        
        # Verify TUN interface is gone
        if ip link show tun0 &>/dev/null; then
            echo 'Error: TUN interface still exists'
            exit 1
        fi
    "
}

# Main execution
main() {
    log_info "Starting resource cleanup..."
    
    # Clean up server
    cleanup_host "$SERVER_HOST" "server"
    
    # Clean up client
    cleanup_host "$CLIENT_HOST" "client"
    
    log_info "Resource cleanup complete"
}

# Execute main function
main
