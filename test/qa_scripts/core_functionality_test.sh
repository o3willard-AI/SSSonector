#!/bin/bash

# Core Functionality Test Script for SSSonector
# This script tests basic functionality in three deployment scenarios:
# 1. Foreground Client / Foreground Server
# 2. Background Client / Foreground Server
# 3. Background Client / Background Server

set -euo pipefail

# Configuration
SERVER_IP=${SERVER_IP:-"192.168.50.210"}
CLIENT_IP=${CLIENT_IP:-"192.168.50.211"}
PACKET_COUNT=20
TIMEOUT=30
LOG_DIR="/var/log/sssonector"
TEST_LOG_DIR="test_logs/$(date +%Y%m%d_%H%M%S)"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Create test log directory
mkdir -p "${TEST_LOG_DIR}"

# Utility functions
check_installation() {
    local host=$1
    log_info "Checking installation on $host..."
    
    ssh "$host" "
        if ! command -v sssonector &> /dev/null; then
            echo 'SSSonector not found'
            exit 1
        fi
        if [ ! -f /etc/sssonector/config.yaml ]; then
            echo 'Configuration file not found'
            exit 1
        fi
        if [ ! -d /var/log/sssonector ]; then
            echo 'Log directory not found'
            exit 1
        fi
    " || {
        log_error "Installation check failed on $host"
        return 1
    }
    
    log_info "Installation check passed on $host"
    return 0
}

verify_tunnel() {
    local timeout=$1
    local start_time=$(date +%s)
    local tunnel_up=false
    
    log_info "Verifying tunnel establishment..."
    
    # Wait for tunnel interface to be ready on both sides
    log_info "Waiting for tunnel interfaces..."
    sleep 5
    
    # Check server tunnel interface
    log_info "Checking server tunnel interface..."
    ssh "$SERVER_IP" "ip addr show tun0" || {
        log_error "Server tunnel interface not found"
        return 1
    }
    
    # Check client tunnel interface
    log_info "Checking client tunnel interface..."
    ssh "$CLIENT_IP" "ip addr show tun0" || {
        log_error "Client tunnel interface not found"
        return 1
    }
    
    # Try to ping server from client
    while [ $(($(date +%s) - start_time)) -lt "$timeout" ]; do
        if ssh "$CLIENT_IP" "ping -c 1 -W 1 10.0.0.1" &>/dev/null; then
            tunnel_up=true
            break
        fi
        log_info "Waiting for tunnel to be ready..."
        sleep 2
    done
    
    if [ "$tunnel_up" = true ]; then
        log_info "Tunnel established successfully"
        return 0
    else
        log_error "Tunnel establishment failed after ${timeout}s"
        # Collect debug information
        log_info "Server tunnel status:"
        ssh "$SERVER_IP" "ip addr show tun0; ss -tulpn | grep sssonector"
        log_info "Client tunnel status:"
        ssh "$CLIENT_IP" "ip addr show tun0; ss -tulpn | grep sssonector"
        return 1
    fi
}

send_test_packets() {
    local direction=$1
    local count=$2
    local src_host=$3
    local dst_host=$4
    local logfile="${TEST_LOG_DIR}/${direction}_packets.log"
    
    log_info "Sending $count packets $direction..."
    
    local target_ip
    if [ "$src_host" = "$CLIENT_IP" ]; then
        target_ip="10.0.0.1"  # Client pinging server
    else
        target_ip="10.0.0.2"  # Server pinging client
    fi
    
    log_info "Pinging $target_ip from $src_host..."
    ssh "$src_host" "ping -c $count $target_ip 2>&1" > "$logfile"
    
    local received=$(grep -c "bytes from" "$logfile")
    if [ "$received" -eq "$count" ]; then
        log_info "All packets transmitted successfully"
        return 0
    else
        log_error "Packet transmission incomplete: $received/$count received"
        return 1
    fi
}

collect_logs() {
    local scenario=$1
    local timestamp=$(date +%Y%m%d_%H%M%S)
    
    log_info "Collecting logs for scenario: $scenario"
    
    # Create scenario log directory
    local scenario_dir="${TEST_LOG_DIR}/${scenario}_${timestamp}"
    mkdir -p "$scenario_dir"
    
    # Collect server logs
    log_info "Collecting server logs..."
    scp "${SERVER_IP}:${LOG_DIR}/*" "${scenario_dir}/server/" &>/dev/null || log_warn "Failed to collect server logs"
    
    # Collect client logs
    log_info "Collecting client logs..."
    scp "${CLIENT_IP}:${LOG_DIR}/*" "${scenario_dir}/client/" &>/dev/null || log_warn "Failed to collect client logs"
    
    # Collect system logs
    log_info "Collecting system logs..."
    ssh "$SERVER_IP" "journalctl -u sssonector --no-pager -n 1000" > "${scenario_dir}/server_journal.log" 2>/dev/null || log_warn "Failed to collect server journal"
    ssh "$CLIENT_IP" "journalctl -u sssonector --no-pager -n 1000" > "${scenario_dir}/client_journal.log" 2>/dev/null || log_warn "Failed to collect client journal"
}

cleanup() {
    local host=$1
    log_info "Cleaning up on $host..."
    
    # Stop services and processes
    ssh "$host" "
        # Stop service
        sudo systemctl stop sssonector &>/dev/null || true
        
        # Kill any running processes
        sudo killall -9 sssonector &>/dev/null || true
        for pid in \$(pgrep -f sssonector); do
            sudo kill -9 \$pid &>/dev/null || true
        done
        
        # Clean up TUN interface
        if ip link show tun0 &>/dev/null; then
            sudo ip link set tun0 down
            sudo ip link delete tun0
            log_info 'TUN interface removed'
        fi
        
        # Double check and force remove if still exists
        if ip link show tun0 &>/dev/null; then
            sudo ip tuntap del mode tun tun0
            log_info 'TUN interface force removed'
        fi
        
        # Verify cleanup
        if ip link show tun0 &>/dev/null; then
            echo 'Warning: TUN interface still exists'
        else
            echo 'TUN interface cleanup verified'
        fi
        
        # Clean up any stale state directories
        sudo rm -rf /var/lib/sssonector/* &>/dev/null || true
    "
}

thorough_cleanup() {
    log_info "Performing thorough cleanup before tests..."
    cleanup "$SERVER_IP"
    cleanup "$CLIENT_IP"
    
    # Verify no sssonector processes are running
    ssh "$SERVER_IP" "pgrep -l sssonector" && log_error "Server still has sssonector processes running"
    ssh "$CLIENT_IP" "pgrep -l sssonector" && log_error "Client still has sssonector processes running"
    
    # Verify no TUN interfaces exist
    ssh "$SERVER_IP" "ip link show tun0" &>/dev/null && log_error "Server still has TUN interface"
    ssh "$CLIENT_IP" "ip link show tun0" &>/dev/null && log_error "Client still has TUN interface"
    
    log_info "Cleanup verification complete"
}

run_test_scenario() {
    local scenario=$1
    local server_mode=$2
    local client_mode=$3
    local result=0
    
    log_info "=== Starting Test Scenario: $scenario ==="
    log_info "Server Mode: $server_mode"
    log_info "Client Mode: $client_mode"
    
    # Clean up both hosts
    cleanup "$SERVER_IP"
    cleanup "$CLIENT_IP"
    
    # Check installations
    check_installation "$SERVER_IP" || return 1
    check_installation "$CLIENT_IP" || return 1
    
    # Deploy fresh configurations
    log_info "Deploying fresh configurations..."
    ./setup_configs.sh
    
    # Start server
    if [ "$server_mode" = "foreground" ]; then
        log_info "Starting server in foreground mode..."
        ssh "$SERVER_IP" "sudo sssonector -config /etc/sssonector/config.yaml" &
        server_pid=$!
    else
        log_info "Starting server in background mode..."
        ssh "$SERVER_IP" "sudo systemctl start sssonector"
    fi
    
    sleep 5
    
    # Start client
    if [ "$client_mode" = "foreground" ]; then
        log_info "Starting client in foreground mode..."
        ssh "$CLIENT_IP" "sudo sssonector -config /etc/sssonector/config.yaml" &
        client_pid=$!
    else
        log_info "Starting client in background mode..."
        ssh "$CLIENT_IP" "sudo systemctl start sssonector"
    fi
    
    # Verify tunnel
    verify_tunnel "$TIMEOUT" || {
        result=1
        log_error "Tunnel verification failed"
    }
    
    if [ $result -eq 0 ]; then
        # Send test packets in both directions
        send_test_packets "client->server" "$PACKET_COUNT" "$CLIENT_IP" "$SERVER_IP" || result=1
        send_test_packets "server->client" "$PACKET_COUNT" "$SERVER_IP" "$CLIENT_IP" || result=1
    fi
    
    # Collect logs
    collect_logs "$scenario"
    
    # Clean shutdown
    if [ "$client_mode" = "foreground" ]; then
        log_info "Stopping client (foreground)..."
        kill "$client_pid" 2>/dev/null || true
    else
        log_info "Stopping client service..."
        ssh "$CLIENT_IP" "sudo systemctl stop sssonector"
    fi
    
    if [ "$server_mode" = "foreground" ]; then
        log_info "Stopping server (foreground)..."
        kill "$server_pid" 2>/dev/null || true
    else
        log_info "Stopping server service..."
        ssh "$SERVER_IP" "sudo systemctl stop sssonector"
    fi
    
    # Ensure cleanup
    log_info "Performing final cleanup..."
    cleanup "$SERVER_IP"
    cleanup "$CLIENT_IP"
    
    # Verify clean shutdown
    sleep 5
    if ! ssh "$CLIENT_IP" "ping -c 1 -W 1 10.0.0.1" &>/dev/null; then
        log_info "Tunnel closed successfully"
    else
        log_error "Tunnel failed to close cleanly"
        result=1
    fi
    
    return $result
}

# Main test execution
main() {
    local failed_scenarios=()
    
    # Perform thorough cleanup before starting tests
    thorough_cleanup
    
    # Scenario 1: Foreground Client / Foreground Server
    if ! run_test_scenario "fg_client_fg_server" "foreground" "foreground"; then
        failed_scenarios+=("Scenario 1: Foreground Client / Foreground Server")
    fi
    
    # Scenario 2: Background Client / Foreground Server
    if ! run_test_scenario "bg_client_fg_server" "foreground" "background"; then
        failed_scenarios+=("Scenario 2: Background Client / Foreground Server")
    fi
    
    # Scenario 3: Background Client / Background Server
    if ! run_test_scenario "bg_client_bg_server" "background" "background"; then
        failed_scenarios+=("Scenario 3: Background Client / Background Server")
    fi
    
    # Print summary
    echo
    log_info "=== Test Summary ==="
    if [ ${#failed_scenarios[@]} -eq 0 ]; then
        log_info "All scenarios passed successfully"
    else
        log_error "The following scenarios failed:"
        for scenario in "${failed_scenarios[@]}"; do
            log_error "- $scenario"
        done
        exit 1
    fi
}

# Execute main function
main
