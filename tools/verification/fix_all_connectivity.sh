#!/bin/bash

# fix_all_connectivity.sh
# Script to fix connectivity issues between SSSonector client and server by applying all fixes at once
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
    
    # Check if either direction was successful
    if $client_success || $server_success; then
        overall_success=0
    fi
    
    return $overall_success
}

# Function to stop SSSonector
stop_sssonector() {
    log_info "Stopping SSSonector"
    ssh ${QA_USER}@${QA_SERVER} "sudo systemctl stop sssonector" || log_warn "Failed to stop SSSonector on server"
    ssh ${QA_USER}@${QA_CLIENT} "sudo systemctl stop sssonector" || log_warn "Failed to stop SSSonector on client"
    sleep 2
}

# Function to restart SSSonector
restart_sssonector() {
    log_info "Restarting SSSonector"
    stop_sssonector
    
    log_info "Starting SSSonector on server"
    ssh ${QA_USER}@${QA_SERVER} "sudo systemctl start sssonector" || log_warn "Failed to start SSSonector on server"
    
    log_info "Starting SSSonector on client"
    ssh ${QA_USER}@${QA_CLIENT} "sudo systemctl start sssonector" || log_warn "Failed to start SSSonector on client"
    
    sleep 5
}

# Function to check tunnel interfaces
check_tunnel_interfaces() {
    log_step "Checking tunnel interfaces"
    
    # Check tunnel interface on server
    log_info "Checking tunnel interface on server"
    TUN_INFO=$(ssh ${QA_USER}@${QA_SERVER} "ip addr show tun0 2>/dev/null || echo 'Interface not found'")
    echo "${TUN_INFO}"
    
    if echo "${TUN_INFO}" | grep -q "Interface not found"; then
        log_warn "Tunnel interface not found on server"
    else
        log_info "Tunnel interface found on server"
    fi
    
    # Check tunnel interface on client
    log_info "Checking tunnel interface on client"
    TUN_INFO=$(ssh ${QA_USER}@${QA_CLIENT} "ip addr show tun0 2>/dev/null || echo 'Interface not found'")
    echo "${TUN_INFO}"
    
    if echo "${TUN_INFO}" | grep -q "Interface not found"; then
        log_warn "Tunnel interface not found on client"
    else
        log_info "Tunnel interface found on client"
    fi
}

# Apply all fixes at once
apply_all_fixes() {
    log_step "Applying all fixes at once"
    
    # Stop SSSonector first
    stop_sssonector
    
    # 1. Add firewall rules
    log_info "Adding firewall rules"
    
    # Add ICMP rules to INPUT chain on server
    ssh ${QA_USER}@${QA_SERVER} "sudo iptables -F INPUT"
    ssh ${QA_USER}@${QA_SERVER} "sudo iptables -A INPUT -p icmp -j ACCEPT"
    ssh ${QA_USER}@${QA_SERVER} "sudo iptables -A INPUT -i tun0 -j ACCEPT"
    ssh ${QA_USER}@${QA_SERVER} "sudo iptables -A INPUT -i lo -j ACCEPT"
    ssh ${QA_USER}@${QA_SERVER} "sudo iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT"
    
    # Add ICMP rules to INPUT chain on client
    ssh ${QA_USER}@${QA_CLIENT} "sudo iptables -F INPUT"
    ssh ${QA_USER}@${QA_CLIENT} "sudo iptables -A INPUT -p icmp -j ACCEPT"
    ssh ${QA_USER}@${QA_CLIENT} "sudo iptables -A INPUT -i tun0 -j ACCEPT"
    ssh ${QA_USER}@${QA_CLIENT} "sudo iptables -A INPUT -i lo -j ACCEPT"
    ssh ${QA_USER}@${QA_CLIENT} "sudo iptables -A INPUT -m state --state ESTABLISHED,RELATED -j ACCEPT"
    
    # Add FORWARD rules on server
    ssh ${QA_USER}@${QA_SERVER} "sudo iptables -F FORWARD"
    ssh ${QA_USER}@${QA_SERVER} "sudo iptables -A FORWARD -i tun0 -o enp0s3 -j ACCEPT"
    ssh ${QA_USER}@${QA_SERVER} "sudo iptables -A FORWARD -i enp0s3 -o tun0 -j ACCEPT"
    
    # Add FORWARD rules on client
    ssh ${QA_USER}@${QA_CLIENT} "sudo iptables -F FORWARD"
    ssh ${QA_USER}@${QA_CLIENT} "sudo iptables -A FORWARD -i tun0 -o enp0s3 -j ACCEPT"
    ssh ${QA_USER}@${QA_CLIENT} "sudo iptables -A FORWARD -i enp0s3 -o tun0 -j ACCEPT"
    
    # 2. Adjust kernel parameters
    log_info "Adjusting kernel parameters"
    
    # Enable IP forwarding on server
    ssh ${QA_USER}@${QA_SERVER} "sudo sysctl -w net.ipv4.ip_forward=1"
    
    # Enable IP forwarding on client
    ssh ${QA_USER}@${QA_CLIENT} "sudo sysctl -w net.ipv4.ip_forward=1"
    
    # Disable reverse path filtering on server
    ssh ${QA_USER}@${QA_SERVER} "sudo sysctl -w net.ipv4.conf.all.rp_filter=0"
    ssh ${QA_USER}@${QA_SERVER} "sudo sysctl -w net.ipv4.conf.default.rp_filter=0"
    ssh ${QA_USER}@${QA_SERVER} "sudo sysctl -w net.ipv4.conf.tun0.rp_filter=0"
    ssh ${QA_USER}@${QA_SERVER} "sudo sysctl -w net.ipv4.conf.enp0s3.rp_filter=0"
    
    # Disable reverse path filtering on client
    ssh ${QA_USER}@${QA_CLIENT} "sudo sysctl -w net.ipv4.conf.all.rp_filter=0"
    ssh ${QA_USER}@${QA_CLIENT} "sudo sysctl -w net.ipv4.conf.default.rp_filter=0"
    ssh ${QA_USER}@${QA_CLIENT} "sudo sysctl -w net.ipv4.conf.tun0.rp_filter=0"
    ssh ${QA_USER}@${QA_CLIENT} "sudo sysctl -w net.ipv4.conf.enp0s3.rp_filter=0"
    
    # Disable ICMP echo ignore on server
    ssh ${QA_USER}@${QA_SERVER} "sudo sysctl -w net.ipv4.icmp_echo_ignore_broadcasts=0"
    ssh ${QA_USER}@${QA_SERVER} "sudo sysctl -w net.ipv4.icmp_echo_ignore_all=0"
    
    # Disable ICMP echo ignore on client
    ssh ${QA_USER}@${QA_CLIENT} "sudo sysctl -w net.ipv4.icmp_echo_ignore_broadcasts=0"
    ssh ${QA_USER}@${QA_CLIENT} "sudo sysctl -w net.ipv4.icmp_echo_ignore_all=0"
    
    # 3. Restart SSSonector to recreate tunnel interfaces
    restart_sssonector
    
    # 4. Adjust MTU
    log_info "Adjusting MTU"
    
    # Set MTU to 1400 on server
    ssh ${QA_USER}@${QA_SERVER} "sudo ip link set dev tun0 mtu 1400"
    
    # Set MTU to 1400 on client
    ssh ${QA_USER}@${QA_CLIENT} "sudo ip link set dev tun0 mtu 1400"
    
    # 5. Add explicit routes
    log_info "Adding explicit routes"
    
    # Add explicit routes on server
    ssh ${QA_USER}@${QA_SERVER} "sudo ip route add 10.0.0.2/32 dev tun0 || true"
    
    # Add explicit routes on client
    ssh ${QA_USER}@${QA_CLIENT} "sudo ip route add 10.0.0.1/32 dev tun0 || true"
    
    # 6. Enable NAT
    log_info "Enabling NAT"
    
    # Enable NAT on server
    ssh ${QA_USER}@${QA_SERVER} "sudo iptables -t nat -F POSTROUTING"
    ssh ${QA_USER}@${QA_SERVER} "sudo iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o enp0s3 -j MASQUERADE"
    
    # Enable NAT on client
    ssh ${QA_USER}@${QA_CLIENT} "sudo iptables -t nat -F POSTROUTING"
    ssh ${QA_USER}@${QA_CLIENT} "sudo iptables -t nat -A POSTROUTING -s 10.0.0.0/24 -o enp0s3 -j MASQUERADE"
    
    # Check tunnel interfaces
    check_tunnel_interfaces
    
    # Test connectivity after applying all fixes
    if test_connectivity; then
        log_info "All fixes applied successfully"
        return 0
    else
        log_warn "All fixes did not resolve the issue"
        return 1
    fi
}

# Main function
main() {
    log_info "Starting comprehensive connectivity fix"
    
    # Test baseline connectivity
    log_step "Testing baseline connectivity"
    if test_connectivity; then
        log_info "Baseline connectivity is already working"
        stop_sssonector
        return 0
    fi
    
    stop_sssonector
    
    # Apply all fixes at once
    if apply_all_fixes; then
        log_info "Connectivity fixed by applying all fixes"
        return 0
    fi
    
    log_error "All fixes failed to resolve the connectivity issue"
    return 1
}

# Run main function
main "$@"
