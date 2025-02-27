#!/bin/bash

# Resource cleanup script for SSSonector testing
# This script ensures clean state by removing any existing tunnel processes and interfaces

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
    local desc=$2

    log_info "Cleaning up $desc ($host)..."

    if [ "$host" = "localhost" ]; then
        # Local cleanup
        log_info "Cleaning up local resources..."

        # Kill any existing sssonector processes
        sudo pkill -9 -f sssonector || true

        # Remove TUN interface if it exists
        sudo ip link del tun0 2>/dev/null || true

        # Clean up any stale lock files and logs
        sudo rm -f /var/run/sssonector.* 2>/dev/null || true
        sudo rm -f startup.log 2>/dev/null || true
        sudo rm -f sssonector.log 2>/dev/null || true
        sudo rm -f sssonector.pid 2>/dev/null || true

        # Create fresh log files
        sudo mkdir -p /var/log/sssonector
        sudo touch /var/log/sssonector/sssonector.log
        sudo touch /var/log/sssonector/startup.log
        sudo chown -R root:root /var/log/sssonector
        sudo chmod 644 /var/log/sssonector/*.log
        sudo chmod 755 /var/log/sssonector
    else
        # Remote cleanup
        # Get PID from state file if it exists
        local pid
        pid=$(ssh -i ~/.ssh/qa_key "$host" "cat ~/sssonector/state/sssonector.pid 2>/dev/null")

        if [ -n "$pid" ]; then
            # Try graceful shutdown first
            ssh -i ~/.ssh/qa_key "$host" "sudo kill $pid 2>/dev/null || true"
            sleep 2

            # Force kill if still running
            if ssh -i ~/.ssh/qa_key "$host" "ps -p $pid > /dev/null 2>&1"; then
                ssh -i ~/.ssh/qa_key "$host" "sudo kill -9 $pid 2>/dev/null || true"
            fi
        fi

        # Kill any other instances
        ssh -i ~/.ssh/qa_key "$host" "sudo pkill -f sssonector || true"
        sleep 1
        ssh -i ~/.ssh/qa_key "$host" "sudo pkill -9 -f sssonector || true"

        # Remove TUN interface if it exists
        ssh -i ~/.ssh/qa_key "$host" "
            if ip link show tun0 &>/dev/null; then
                sudo ip link set tun0 down
                sudo ip link del tun0
            fi
        "

        # Clean up any stale files and logs
        ssh -A -i ~/.ssh/qa_key "$host" "
            # Save last logs for debugging
            if [ -f ~/sssonector/log/startup.log ]; then
                sudo cp ~/sssonector/log/startup.log ~/sssonector/log/startup.log.old
            fi
            if [ -f ~/sssonector/log/sssonector.log ]; then
                sudo cp ~/sssonector/log/sssonector.log ~/sssonector/log/sssonector.log.old
            fi

            # Remove runtime files
            sudo rm -f /var/run/sssonector.* 2>/dev/null || true
            sudo rm -f ~/sssonector/log/startup.log 2>/dev/null || true
            sudo rm -f ~/sssonector/log/sssonector.log 2>/dev/null || true
            sudo rm -f ~/sssonector/state/* 2>/dev/null || true

            # Create fresh log files
            sudo touch ~/sssonector/log/startup.log ~/sssonector/log/sssonector.log
            sudo chown root:root ~/sssonector/log/startup.log ~/sssonector/log/sssonector.log
            sudo chmod 644 ~/sssonector/log/startup.log ~/sssonector/log/sssonector.log
        "
    fi

    # Reset directory permissions
    if [ "$host" != "localhost" ]; then
        ssh -A -i ~/.ssh/qa_key "$host" "
            # Set root ownership for critical directories
            sudo chown -R root:root ~/sssonector/{bin,certs,config} 2>/dev/null || true
            sudo chmod -R 755 ~/sssonector/{bin,certs,config} 2>/dev/null || true
            sudo chmod 644 ~/sssonector/certs/*.crt 2>/dev/null || true
            sudo chmod 600 ~/sssonector/certs/*.key 2>/dev/null || true
            sudo chmod 644 ~/sssonector/config/config.yaml 2>/dev/null || true

            # Set root ownership for log and state directories
            sudo chown -R root:root ~/sssonector/{log,state} 2>/dev/null || true
            sudo chmod -R 755 ~/sssonector/{log,state} 2>/dev/null || true
            sudo chmod 644 ~/sssonector/log/*.log 2>/dev/null || true

            # Ensure base directories exist
            sudo mkdir -p ~/sssonector/{bin,certs,config,log,state} 2>/dev/null || true
        "
    fi

    # Verify cleanup
    if [ "$host" = "localhost" ]; then
        if pgrep -f sssonector || ip link show tun0 &>/dev/null; then
            log_error "Failed to clean up all resources on $desc"
            return 1
        else
            log_info "$desc cleanup successful"
        fi
    else
        if ssh -A -i ~/.ssh/qa_key "$host" "pgrep -f sssonector || ip link show tun0" &>/dev/null; then
            log_error "Failed to clean up all resources on $desc"
            return 1
        else
            log_info "$desc cleanup successful"
        fi
    fi
}

# Main cleanup sequence
main() {
    # Clean up server
    cleanup_host "localhost" "local server"
    
    # Clean up client
    cleanup_host "$CLIENT_HOST" "client"
    
    # Clean up server
    cleanup_host "$SERVER_HOST" "server"

    log_info "All resources cleaned up successfully"
}

# Execute main function
main
