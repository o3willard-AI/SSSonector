#!/bin/bash

# Cleanup script for SSSonector
# This script removes old binaries, configuration files, and service files

set -euo pipefail

SERVER_IP=${SERVER_IP:-"192.168.50.210"}
CLIENT_IP=${CLIENT_IP:-"192.168.50.211"}

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

# Validate environment and requirements
validate_environment() {
    # Check SSH connectivity
    if ! ssh -o BatchMode=yes -o ConnectTimeout=5 "$SERVER_IP" "echo 'Connection test'" &>/dev/null; then
        log_error "Cannot connect to server $SERVER_IP. Please ensure SSH access is configured"
        exit 1
    fi
    if ! ssh -o BatchMode=yes -o ConnectTimeout=5 "$CLIENT_IP" "echo 'Connection test'" &>/dev/null; then
        log_error "Cannot connect to client $CLIENT_IP. Please ensure SSH access is configured"
        exit 1
    fi
}

# Function to cleanup a host
cleanup_host() {
    local host=$1
    local mode=$2
    local max_retries=3
    local retry_count=0

    log_info "Cleaning up $mode on $host..."

    while [ $retry_count -lt $max_retries ]; do
        if ssh "$host" "
            # Stop service if running
            sudo systemctl stop sssonector.service &>/dev/null || true
            sudo systemctl disable sssonector.service &>/dev/null || true

            # Remove service file
            sudo rm -f /etc/systemd/system/sssonector.service
            sudo systemctl daemon-reload

            # Remove binary
            sudo rm -f /usr/bin/sssonector

            # Remove configuration files
            sudo rm -rf /etc/sssonector

            # Remove runtime directories
            sudo rm -rf /var/lib/sssonector
            sudo rm -rf /var/log/sssonector
            sudo rm -rf /run/sssonector

            # Remove any temporary files
            sudo rm -rf /tmp/sssonector*
            "; then
            log_info "Successfully cleaned up $mode on $host"
            return 0
        fi

        retry_count=$((retry_count + 1))
        if [ $retry_count -lt $max_retries ]; then
            log_warn "Attempt $retry_count failed. Retrying in 5 seconds..."
            sleep 5
        fi
    done

    log_error "Failed to clean up $mode on $host after $max_retries attempts"
    return 1
}

# Main execution
main() {
    log_info "Starting cleanup..."
    
    # Validate environment first
    validate_environment
    
    # Clean up server
    cleanup_host "$SERVER_IP" "server" || {
        log_error "Failed to clean up server"
        exit 1
    }
    
    # Clean up client
    cleanup_host "$CLIENT_IP" "client" || {
        log_error "Failed to clean up client"
        exit 1
    }
    
    # Clean up local build artifacts
    log_info "Cleaning up local build artifacts..."
    rm -rf bin/release
    
    log_info "Cleanup complete"
}

# Execute main function
main
