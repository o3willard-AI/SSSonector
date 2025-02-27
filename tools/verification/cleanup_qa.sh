#!/bin/bash

# cleanup_qa.sh
# Script to clean up QA environment for SSSonector testing
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

# Clean up a single host
cleanup_host() {
    local host=$1
    local type=$2
    
    log_info "Cleaning up ${type} (${host})"
    
    # Check if sshpass is installed
    if ! command -v sshpass &> /dev/null; then
        log_info "Installing sshpass..."
        sudo apt-get update && sudo apt-get install -y sshpass
    fi
    
    # Test SSH connection
    if ! sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "echo 'SSH connection test successful'" &> /dev/null; then
        log_error "Cannot SSH to ${host}"
        return 1
    fi
    
    # Check if /opt/sssonector exists
    if sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "test -d /opt/sssonector"; then
        log_info "Found /opt/sssonector directory on ${type}"
        
        # Check for SSSonector binary
        if sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "test -f /opt/sssonector/bin/sssonector"; then
            log_info "Found SSSonector binary on ${type}, removing..."
            sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "sudo rm -f /opt/sssonector/bin/sssonector"
        fi
        
        # Check for certificates
        if sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "test -d /opt/sssonector/certs"; then
            log_info "Found certificates directory on ${type}, cleaning..."
            sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "sudo rm -rf /opt/sssonector/certs/*"
        fi
        
        # Check for configuration files
        if sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "test -d /opt/sssonector/config"; then
            log_info "Found configuration directory on ${type}, cleaning..."
            sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "sudo rm -rf /opt/sssonector/config/*"
        fi
    else
        log_info "No /opt/sssonector directory found on ${type}, creating..."
        sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "sudo mkdir -p /opt/sssonector/{bin,certs,config,log,state,tools/verification}"
    fi
    
    # Check for running SSSonector processes
    if sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "pgrep -f sssonector" &> /dev/null; then
        log_info "Found running SSSonector processes on ${type}, killing..."
        sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "sudo pkill -f sssonector" || true
    fi
    
    # Check for TUN interfaces
    if sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "ip link show | grep -q tun"; then
        log_info "Found TUN interfaces on ${type}, removing..."
        sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "sudo ip link show | grep tun | cut -d: -f2 | cut -d@ -f1 | xargs -I{} sudo ip link delete {}" || true
    fi
    
    # Check for blocked ports
    for port in 443 8443 9090 9091 9092 9093; do
        if sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "sudo netstat -tuln | grep -q ':${port} '"; then
            log_info "Found process using port ${port} on ${type}, killing..."
            sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${host}" "sudo lsof -i :${port} | grep LISTEN | awk '{print \$2}' | xargs -r sudo kill -9" || true
        fi
    done
    
    log_info "${type} cleanup completed successfully"
}

# Main function
main() {
    log_info "Starting QA environment cleanup"
    
    # Clean up server
    cleanup_host "${QA_SERVER}" "server" || log_error "Failed to clean up server"
    
    # Clean up client
    cleanup_host "${QA_CLIENT}" "client" || log_error "Failed to clean up client"
    
    log_info "QA environment cleanup completed successfully"
}

# Run main function
main "$@"
