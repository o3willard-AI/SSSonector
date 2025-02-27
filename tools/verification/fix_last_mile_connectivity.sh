#!/bin/bash

# fix_last_mile_connectivity.sh
# Script to fix last mile connectivity issues between SSSonector client and server
set -euo pipefail

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

# QA server details
QA_SERVER="192.168.50.210"
QA_CLIENT="192.168.50.211"
QA_USER="sblanken"
QA_SUDO_PASSWORD="password"

# Test SSH connection to QA servers
log_info "Testing SSH connection to QA servers"
ssh -o ConnectTimeout=5 -o BatchMode=yes -o StrictHostKeyChecking=no ${QA_USER}@${QA_SERVER} echo "SSH connection to server successful" || log_error "Failed to connect to server"
ssh -o ConnectTimeout=5 -o BatchMode=yes -o StrictHostKeyChecking=no ${QA_USER}@${QA_CLIENT} echo "SSH connection to client successful" || log_error "Failed to connect to client"

# Function to test connectivity
test_connectivity() {
    log_step "Testing connectivity"
    
    # Start SSSonector
    log_info "Starting SSSonector on server"
    ssh ${QA_USER}@${QA_SERVER} "sudo systemctl start sssonector" || log_warn "Failed to start SSSonector on server"
    
    log_info "Starting SSSonector on client"
    ssh ${QA_USER}@${QA_CLIENT} "sudo systemctl start sssonector" || log_warn "Failed to start SSSonector on client"
    
    # Wait for SSSonector to start
    sleep 5
    
    # Test ping from client to server
    log_info "Testing ping from client to server"
    PING_RESULT=$(ssh ${QA_USER}@${QA_CLIENT} "ping -c 5 -W 2 10.0.0.1")
    echo "${PING_RESULT}"
    
    local client_success=false
    local server_success=false
    local overall_success=1  # 0 = success, 1 = failure (for return code)
    
    if echo "${PING_RESULT}" | grep -q "0 received"; then
        log_warn "Ping from client to server failed"
        client_success=false
    else
        log_info "Ping from client to server successful"
        client_success=true
    fi
    
    # Test ping from server to client
    log_info "Testing ping from server to client"
    PING_RESULT=$(ssh ${QA_USER}@${QA_SERVER} "ping -c 5 -W 2 10.0.0.2")
    echo "${PING_RESULT}"
    
    if echo "${PING_RESULT}" | grep -q "0 received"; then
        log_warn "Ping from server to client failed"
        server_success=false
    else
        log_info "Ping from server to client successful"
        server_success=true
    fi
    
    # Stop SSSonector
    log_info "Stopping SSSonector"
    ssh ${QA_USER}@${QA_SERVER} "sudo systemctl stop sssonector" || log_warn "Failed to stop SSSonector on server"
    ssh ${QA_USER}@${QA_CLIENT} "sudo systemctl stop sssonector" || log_warn "Failed to stop SSSonector on client"
    
    # Check if either direction was successful
    if $client_success || $server_success; then
        overall_success=0
    fi
    
    return $overall_success
}

# Fix 1: Add ICMP rules to INPUT chain
fix_firewall_rules() {
    log_step "Fixing firewall rules"
    
    # Add ICMP rules to INPUT chain on server
    log_info "Adding ICMP rules to INPUT chain on server"
    ssh ${QA_USER}@${QA_SERVER} "sudo iptables -A INPUT -p icmp -j ACCEPT"
    ssh ${QA_USER}@${QA_SERVER} "sudo iptables -A INPUT -i tun0 -j ACCEPT"
    
    # Add ICMP rules to INPUT chain on client
    log_info "Adding ICMP rules to INPUT chain on client"
    ssh ${QA_USER}@${QA_CLIENT} "sudo iptables -A INPUT -p icmp -j ACCEPT"
    ssh ${QA_USER}@${QA_CLIENT} "sudo iptables -A INPUT -i tun0 -j ACCEPT"
    
    # Test connectivity after fixing firewall rules
    if test_connectivity; then
        log_info "Firewall rules fix successful"
        return 0
    else
        log_warn "Firewall rules fix did not resolve the issue"
        return 1
    fi
}

# Fix 2: Adjust kernel parameters
fix_kernel_parameters() {
    log_step "Fixing kernel parameters"
    
    # Disable reverse path filtering on server
    log_info "Disabling reverse path filtering on server"
    ssh ${QA_USER}@${QA_SERVER} "sudo sysctl -w net.ipv4.conf.all.rp_filter=0"
    ssh ${QA_USER}@${QA_SERVER} "sudo sysctl -w net.ipv4.conf.default.rp_filter=0"
    ssh ${QA_USER}@${QA_SERVER} "sudo sysctl -w net.ipv4.conf.tun0.rp_filter=0"
    
    # Disable reverse path filtering on client
    log_info "Disabling reverse path filtering on client"
    ssh ${QA_USER}@${QA_CLIENT} "sudo sysctl -w net.ipv4.conf.all.rp_filter=0"
    ssh ${QA_USER}@${QA_CLIENT} "sudo sysctl -w net.ipv4.conf.default.rp_filter=0"
    ssh ${QA_USER}@${QA_CLIENT} "sudo sysctl -w net.ipv4.conf.tun0.rp_filter=0"
    
    # Disable ICMP echo ignore on server
    log_info "Disabling ICMP echo ignore on server"
    ssh ${QA_USER}@${QA_SERVER} "sudo sysctl -w net.ipv4.icmp_echo_ignore_broadcasts=0"
    
    # Disable ICMP echo ignore on client
    log_info "Disabling ICMP echo ignore on client"
    ssh ${QA_USER}@${QA_CLIENT} "sudo sysctl -w net.ipv4.icmp_echo_ignore_broadcasts=0"
    
    # Test connectivity after fixing kernel parameters
    if test_connectivity; then
        log_info "Kernel parameters fix successful"
        return 0
    else
        log_warn "Kernel parameters fix did not resolve the issue"
        return 1
    fi
}

# Fix 3: Adjust MTU
fix_mtu() {
    log_step "Fixing MTU"
    
    # Set MTU to 1400 on server
    log_info "Setting MTU to 1400 on server"
    ssh ${QA_USER}@${QA_SERVER} "sudo ip link set dev tun0 mtu 1400"
    
    # Set MTU to 1400 on client
    log_info "Setting MTU to 1400 on client"
    ssh ${QA_USER}@${QA_CLIENT} "sudo ip link set dev tun0 mtu 1400"
    
    # Test connectivity after fixing MTU
    if test_connectivity; then
        log_info "MTU fix successful"
        return 0
    else
        log_warn "MTU fix did not resolve the issue"
        return 1
    fi
}

# Fix 4: Add explicit routes
fix_routes() {
    log_step "Fixing routes"
    
    # Add explicit routes on server
    log_info "Adding explicit routes on server"
    ssh ${QA_USER}@${QA_SERVER} "sudo ip route add 10.0.0.2/32 dev tun0"
    
    # Add explicit routes on client
    log_info "Adding explicit routes on client"
    ssh ${QA_USER}@${QA_CLIENT} "sudo ip route add 10.0.0.1/32 dev tun0"
    
    # Test connectivity after fixing routes
    if test_connectivity; then
        log_info "Routes fix successful"
        return 0
    else
        log_warn "Routes fix did not resolve the issue"
        return 1
    fi
}

# Main function
main() {
    log_info "Starting last mile connectivity fix"
    
    # Test baseline connectivity
    log_step "Testing baseline connectivity"
    if test_connectivity; then
        log_info "Baseline connectivity is already working"
        return 0
    fi
    
    # Try each fix in sequence
    if fix_firewall_rules; then
        log_info "Last mile connectivity fixed by adding firewall rules"
        return 0
    fi
    
    if fix_kernel_parameters; then
        log_info "Last mile connectivity fixed by adjusting kernel parameters"
        return 0
    fi
    
    if fix_mtu; then
        log_info "Last mile connectivity fixed by adjusting MTU"
        return 0
    fi
    
    if fix_routes; then
        log_info "Last mile connectivity fixed by adding explicit routes"
        return 0
    fi
    
    log_error "All fixes failed to resolve the last mile connectivity issue"
    return 1
}

# Run main function
main "$@"
