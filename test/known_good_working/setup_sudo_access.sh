#!/bin/bash

# Setup passwordless sudo access for SSSonector service
# This script configures sudoers to allow the service to run without password

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

# Validate IP address format
validate_ip() {
    local ip=$1
    if [[ ! $ip =~ ^[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}$ ]]; then
        return 1
    fi
    for octet in $(echo "$ip" | tr '.' ' '); do
        if [[ $octet -lt 0 || $octet -gt 255 ]]; then
            return 1
        fi
    done
    return 0
}

# Validate environment and requirements
validate_environment() {
    # Check if running as root or with sudo
    if [[ $EUID -ne 0 ]]; then
        log_error "This script must be run as root or with sudo"
        exit 1
    fi

    # Validate IP addresses
    if ! validate_ip "$SERVER_IP"; then
        log_error "Invalid server IP address: $SERVER_IP"
        exit 1
    fi
    if ! validate_ip "$CLIENT_IP"; then
        log_error "Invalid client IP address: $CLIENT_IP"
        exit 1
    fi

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

# Sudoers configuration content with restricted commands
SUDOERS_CONTENT="# Allow SSSonector service to run without password
# Configuration added $(date '+%Y-%m-%d')

# Binary execution
%sudo ALL=(ALL) NOPASSWD: /usr/bin/sssonector

# Service control
%sudo ALL=(ALL) NOPASSWD: /bin/systemctl start sssonector
%sudo ALL=(ALL) NOPASSWD: /bin/systemctl stop sssonector
%sudo ALL=(ALL) NOPASSWD: /bin/systemctl restart sssonector
%sudo ALL=(ALL) NOPASSWD: /bin/systemctl status sssonector

# Network interface management
%sudo ALL=(ALL) NOPASSWD: /usr/sbin/ip link set tun0
%sudo ALL=(ALL) NOPASSWD: /usr/sbin/ip link delete tun0

# Process management
%sudo ALL=(ALL) NOPASSWD: /usr/bin/killall sssonector
%sudo ALL=(ALL) NOPASSWD: /bin/kill -9 *sssonector*

# Maintain secure file permissions
Defaults secure_path=\"/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin\"
"

# Function to validate sudoers syntax
validate_sudoers() {
    local file=$1
    if command -v visudo >/dev/null 2>&1; then
        if ! visudo -c -f "$file"; then
            log_error "Invalid sudoers file syntax"
            return 1
        fi
    else
        log_warn "visudo not found, skipping syntax validation"
    fi
    return 0
}

# Function to setup sudo access on a host with enhanced security
setup_sudo() {
    local host=$1
    local temp_dir
    local temp_file
    local max_retries=3
    local retry_count=0

    log_info "Setting up sudo access on $host..."
    
    # Create secure temporary directory
    temp_dir=$(ssh "$host" "mktemp -d")
    if [ -z "$temp_dir" ]; then
        log_error "Failed to create temporary directory on $host"
        return 1
    fi

    temp_file="${temp_dir}/sssonector_sudoers"

    # Cleanup function
    cleanup() {
        ssh "$host" "rm -rf $temp_dir" || log_warn "Failed to cleanup temporary directory on $host"
    }
    trap cleanup EXIT

    # Create and validate sudoers file locally first
    echo "$SUDOERS_CONTENT" > /tmp/sssonector_sudoers.tmp
    if ! validate_sudoers /tmp/sssonector_sudoers.tmp; then
        rm -f /tmp/sssonector_sudoers.tmp
        return 1
    fi

    # Copy and install sudoers file
    while [ $retry_count -lt $max_retries ]; do
        if scp /tmp/sssonector_sudoers.tmp "${host}:${temp_file}" &&
           ssh "$host" "
               sudo chown root:root ${temp_file} &&
               sudo chmod 440 ${temp_file} &&
               sudo mv ${temp_file} /etc/sudoers.d/sssonector &&
               sudo chmod 440 /etc/sudoers.d/sssonector
           "; then
            log_info "Sudo access configured on $host"
            rm -f /tmp/sssonector_sudoers.tmp
            return 0
        fi
        
        retry_count=$((retry_count + 1))
        if [ $retry_count -lt $max_retries ]; then
            log_warn "Attempt $retry_count failed. Retrying..."
            sleep 5
        fi
    done

    rm -f /tmp/sssonector_sudoers.tmp
    log_error "Failed to setup sudo access on $host after $max_retries attempts"
    return 1
}

# Function to verify sudo access
verify_sudo_access() {
    local host=$1
    log_info "Verifying sudo access on $host..."
    
    if ssh "$host" "sudo -n true" &>/dev/null; then
        if ssh "$host" "sudo -n systemctl status sssonector" &>/dev/null; then
            log_info "Sudo access verified on $host"
            return 0
        fi
    fi
    
    log_error "Sudo access verification failed on $host"
    return 1
}

# Main execution
main() {
    log_info "Starting sudo access setup..."
    
    # Validate environment first
    validate_environment
    
    # Setup sudo access on both hosts
    setup_sudo "$SERVER_IP" || {
        log_error "Failed to setup sudo access on server"
        exit 1
    }
    
    setup_sudo "$CLIENT_IP" || {
        log_error "Failed to setup sudo access on client"
        exit 1
    }
    
    # Verify sudo access
    verify_sudo_access "$SERVER_IP" || exit 1
    verify_sudo_access "$CLIENT_IP" || exit 1
    
    log_info "Sudo access setup complete and verified"
}

# Execute main function
main
