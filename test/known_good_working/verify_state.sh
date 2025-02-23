#!/bin/bash

# Script to verify current environment against known good state
# This script compares the current environment with the documented known good state

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

check_permissions() {
    local host=$1
    log_info "Checking permissions on $host..."
    
    ssh "$host" "
        # Check binary permissions
        if [[ \$(stat -c %a ~/sssonector/bin/sssonector) != '755' ]]; then
            echo 'Binary has incorrect permissions'
            exit 1
        fi
        
        # Check config permissions
        if [[ \$(stat -c %a ~/sssonector/config) != '755' ]]; then
            echo 'Config directory has incorrect permissions'
            exit 1
        fi
        if [[ \$(stat -c %U:%G ~/sssonector/config) != 'root:root' ]]; then
            echo 'Config directory has incorrect ownership'
            exit 1
        fi
        
        # Check certificate permissions
        if [[ \$(stat -c %a ~/sssonector/certs/*.key 2>/dev/null || echo '000') != '600' ]]; then
            echo 'Certificate keys have incorrect permissions'
            exit 1
        fi
        if [[ \$(stat -c %a ~/sssonector/certs/*.crt 2>/dev/null || echo '000') != '644' ]]; then
            echo 'Certificate files have incorrect permissions'
            exit 1
        fi
        
        # Check log directory
        if [[ \$(stat -c %a ~/sssonector/log) != '755' ]]; then
            echo 'Log directory has incorrect permissions'
            exit 1
        fi
        
        # Check state directory
        if [[ \$(stat -c %a ~/sssonector/state) != '755' ]]; then
            echo 'State directory has incorrect permissions'
            exit 1
        fi
    " && log_info "Permissions check passed on $host" || log_error "Permissions check failed on $host"
}

check_network() {
    local host=$1
    local ip=$2
    log_info "Checking network configuration on $host..."
    
    ssh "$host" "
        # Check TUN interface
        if ! ip link show tun0 | grep -q UP; then
            echo 'TUN interface not up'
            exit 1
        fi
        
        # Check TUN IP
        if ! ip addr show tun0 | grep -q 'inet $ip'; then
            echo 'TUN interface has incorrect IP'
            exit 1
        fi
        
        # Check IP forwarding
        if ! sysctl net.ipv4.ip_forward | grep -q '= 1'; then
            echo 'IP forwarding not enabled'
            exit 1
        fi
        
        # Check TUN module
        if ! lsmod | grep -q '^tun'; then
            echo 'TUN module not loaded'
            exit 1
        fi
    " && log_info "Network check passed on $host" || log_error "Network check failed on $host"
}

check_script_versions() {
    log_info "Checking script versions..."
    
    # Get directory of this script
    local base_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    local qa_dir="$(cd "$base_dir/../qa_scripts" && pwd)"
    
    # Compare script versions
    if ! diff -r "$base_dir" "$qa_dir" > /dev/null; then
        log_error "Script versions differ from known good state"
        log_info "Run the following to see differences:"
        echo "diff -r $base_dir $qa_dir"
        return 1
    fi
    
    log_info "Script versions match known good state"
    return 0
}

check_connectivity() {
    log_info "Checking tunnel connectivity..."
    
    # Check client to server ping
    ssh "$CLIENT_HOST" "ping -c 1 -W 2 10.0.0.1" &>/dev/null && \
    # Check server to client ping
    ssh "$SERVER_HOST" "ping -c 1 -W 2 10.0.0.2" &>/dev/null && \
    log_info "Connectivity check passed" || log_error "Connectivity check failed"
}

main() {
    local errors=0
    
    # Check script versions
    check_script_versions || ((errors++))
    
    # Check server configuration
    check_permissions "$SERVER_HOST" || ((errors++))
    check_network "$SERVER_HOST" "10.0.0.1" || ((errors++))
    
    # Check client configuration
    check_permissions "$CLIENT_HOST" || ((errors++))
    check_network "$CLIENT_HOST" "10.0.0.2" || ((errors++))
    
    # Check connectivity
    check_connectivity || ((errors++))
    
    # Run full test suite
    log_info "Running full test suite..."
    if ./core_functionality_test.sh; then
        log_info "Test suite passed"
    else
        log_error "Test suite failed"
        ((errors++))
    fi
    
    # Print summary
    echo
    if [ $errors -eq 0 ]; then
        log_info "All checks passed - environment matches known good state"
        exit 0
    else
        log_error "Found $errors error(s) - environment differs from known good state"
        log_info "Review WORKING_STATE.md for troubleshooting steps"
        exit 1
    fi
}

# Execute main function
main
