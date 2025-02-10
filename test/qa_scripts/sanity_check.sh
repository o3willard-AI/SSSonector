#!/bin/bash

# Configuration
SERVER_VM="192.168.50.210"
CLIENT_VM="192.168.50.211"
SUDO_PASS="101abn"
LOG_DIR="sanity_test_logs"
PING_COUNT=20
TUNNEL_TIMEOUT=30  # Seconds to wait for tunnel establishment
CLEANUP_TIMEOUT=10 # Seconds to wait for cleanup

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Function to log messages
log() {
    echo -e "${YELLOW}[$(date '+%Y-%m-%d %H:%M:%S')] $1${NC}"
}

# Function to check command status
check_status() {
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}[✓] $1${NC}"
        return 0
    else
        echo -e "${RED}[✗] $1${NC}"
        return 1
    fi
}

# Function to run command on remote VM
remote_cmd() {
    local vm=$1
    local cmd=$2
    log "Executing on $vm: $cmd"
    ssh $vm "echo $SUDO_PASS | sudo -S bash -c '$cmd'"
}

# Function to verify installation
verify_installation() {
    local vm=$1
    log "Verifying installation on $vm..."
    
    # Check binary installation
    remote_cmd $vm "which sssonector"
    check_status "Binary installation check on $vm" || return 1
    
    # Check capabilities
    remote_cmd $vm "getcap /usr/local/bin/sssonector | grep cap_net_admin"
    check_status "Capabilities check on $vm" || return 1
    
    # Check certificate permissions
    remote_cmd $vm "ls -l /etc/sssonector/certs/"
    check_status "Certificate permissions check on $vm" || return 1
    
    # Check config file
    remote_cmd $vm "test -f /etc/sssonector/config.yaml"
    check_status "Configuration file check on $vm" || return 1
    
    # Verify config file contents
    remote_cmd $vm "cat /etc/sssonector/config.yaml"
    check_status "Configuration content check on $vm" || return 1
    
    return 0
}

# Function to verify tunnel establishment
verify_tunnel() {
    local vm=$1
    local timeout=$TUNNEL_TIMEOUT
    log "Verifying tunnel on $vm..."
    
    # Wait for interface with timeout
    local start_time=$(date +%s)
    while true; do
        if remote_cmd $vm "/sbin/ip addr show tun0" >/dev/null 2>&1; then
            break
        fi
        
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))
        
        if [ $elapsed -ge $timeout ]; then
            log "Timeout waiting for tun0 interface"
            return 1
        fi
        
        log "Waiting for tun0 interface... (${elapsed}s/${timeout}s)"
        sleep 2
    done
    
    # Check TUN interface configuration
    remote_cmd $vm "/sbin/ip addr show tun0"
    check_status "TUN interface configuration on $vm" || return 1
    
    # Check interface is up
    remote_cmd $vm "/sbin/ip link show tun0 | grep -q 'UP'"
    check_status "TUN interface UP state on $vm" || return 1
    
    # Check routing
    remote_cmd $vm "/sbin/ip route show | grep tun0"
    check_status "Routing configuration on $vm" || return 1
    
    # Check interface statistics
    remote_cmd $vm "cat /sys/class/net/tun0/statistics/rx_bytes"
    check_status "Interface statistics on $vm" || return 1
    
    return 0
}

# Function to test connectivity
test_connectivity() {
    local direction=$1
    local src_vm=$2
    local dst_ip=$3
    log "Testing connectivity $direction..."
    
    # Initial statistics
    local rx_bytes_before=$(remote_cmd $src_vm "cat /sys/class/net/tun0/statistics/rx_bytes")
    local tx_bytes_before=$(remote_cmd $src_vm "cat /sys/class/net/tun0/statistics/tx_bytes")
    
    # Run ping test
    local ping_output=$(remote_cmd $src_vm "ping -c $PING_COUNT -W 2 $dst_ip")
    local ping_status=$?
    echo "$ping_output"
    
    # Get packet loss percentage
    local packet_loss=$(echo "$ping_output" | grep -oP '\d+(?=% packet loss)')
    
    # Final statistics
    local rx_bytes_after=$(remote_cmd $src_vm "cat /sys/class/net/tun0/statistics/rx_bytes")
    local tx_bytes_after=$(remote_cmd $src_vm "cat /sys/class/net/tun0/statistics/tx_bytes")
    
    # Calculate bytes transferred
    local rx_diff=$((rx_bytes_after - rx_bytes_before))
    local tx_diff=$((tx_bytes_after - tx_bytes_before))
    
    log "Bytes received: $rx_diff"
    log "Bytes transmitted: $tx_diff"
    log "Packet loss: $packet_loss%"
    
    # Check results
    if [ $ping_status -eq 0 ] && [ "$packet_loss" -eq 0 ]; then
        check_status "$direction connectivity test (${PING_COUNT} packets)"
        return 0
    else
        check_status "$direction connectivity test failed"
        return 1
    fi
}

# Function to verify process cleanup
verify_cleanup() {
    local vm=$1
    local timeout=$CLEANUP_TIMEOUT
    log "Verifying cleanup on $vm..."
    
    # Wait for process termination
    local start_time=$(date +%s)
    while true; do
        if ! remote_cmd $vm "pgrep -f sssonector" >/dev/null 2>&1; then
            break
        fi
        
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))
        
        if [ $elapsed -ge $timeout ]; then
            log "Timeout waiting for process cleanup"
            return 1
        fi
        
        log "Waiting for process cleanup... (${elapsed}s/${timeout}s)"
        sleep 1
    done
    
    # Verify interface is gone
    if remote_cmd $vm "/sbin/ip link show tun0" >/dev/null 2>&1; then
        log "TUN interface still exists"
        return 1
    fi
    
    # Check for orphaned processes
    if remote_cmd $vm "pgrep -f sssonector" >/dev/null 2>&1; then
        log "Orphaned processes found"
        return 1
    fi
    
    check_status "Cleanup verification on $vm"
    return 0
}

# Function to collect logs
collect_logs() {
    local scenario=$1
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local log_dir="${LOG_DIR}/${scenario}_${timestamp}"
    
    log "Collecting logs for $scenario..."
    mkdir -p $log_dir
    
    # System logs
    remote_cmd $SERVER_VM "journalctl -u sssonector --no-pager -n 200" > "${log_dir}/server_journal.log"
    remote_cmd $CLIENT_VM "journalctl -u sssonector --no-pager -n 200" > "${log_dir}/client_journal.log"
    
    # Application logs
    remote_cmd $SERVER_VM "cat /var/log/sssonector/server.log" > "${log_dir}/server_app.log"
    remote_cmd $CLIENT_VM "cat /var/log/sssonector/client.log" > "${log_dir}/client_app.log"
    
    # Network state
    remote_cmd $SERVER_VM "/sbin/ip addr show; /sbin/ip route show" > "${log_dir}/server_network.log"
    remote_cmd $CLIENT_VM "/sbin/ip addr show; /sbin/ip route show" > "${log_dir}/client_network.log"
    
    # Process state
    remote_cmd $SERVER_VM "ps aux | grep sssonector" > "${log_dir}/server_process.log"
    remote_cmd $CLIENT_VM "ps aux | grep sssonector" > "${log_dir}/client_process.log"
    
    # Interface statistics
    remote_cmd $SERVER_VM "cat /sys/class/net/tun0/statistics/*" > "${log_dir}/server_stats.log" 2>/dev/null || true
    remote_cmd $CLIENT_VM "cat /sys/class/net/tun0/statistics/*" > "${log_dir}/client_stats.log" 2>/dev/null || true
    
    # System resources
    remote_cmd $SERVER_VM "free -m; df -h; uptime" > "${log_dir}/server_resources.log"
    remote_cmd $CLIENT_VM "free -m; df -h; uptime" > "${log_dir}/client_resources.log"
    
    log "Logs collected in ${log_dir}"
}

# Function to clean up
cleanup() {
    log "Cleaning up..."
    
    # Stop services
    remote_cmd $CLIENT_VM "systemctl stop sssonector || true"
    remote_cmd $SERVER_VM "systemctl stop sssonector || true"
    
    # Kill any foreground processes
    remote_cmd $CLIENT_VM "pkill -f sssonector || true"
    remote_cmd $SERVER_VM "pkill -f sssonector || true"
    
    # Wait for cleanup
    sleep 2
    
    # Verify cleanup
    verify_cleanup $CLIENT_VM
    verify_cleanup $SERVER_VM
}

# Function to start process in foreground
start_foreground() {
    local vm=$1
    local role=$2
    log "Starting $role in foreground on $vm..."
    
    # Create log directory
    remote_cmd $vm "mkdir -p /var/log/sssonector"
    
    # Start process with full logging
    remote_cmd $vm "export PATH=/sbin:/usr/sbin:$PATH && nohup sssonector -config /etc/sssonector/config.yaml -log-level debug > /var/log/sssonector/${role}.log 2>&1 &"
    sleep 2
    
    # Verify process started
    remote_cmd $vm "pgrep -f sssonector"
    if ! check_status "$role startup"; then
        return 1
    fi
    
    # Show initial logs
    log "Initial $role logs:"
    remote_cmd $vm "tail -n 10 /var/log/sssonector/${role}.log"
    
    return 0
}

# Create log directory
mkdir -p $LOG_DIR

# Main test scenarios
run_test_scenario() {
    local scenario=$1
    local server_mode=$2
    local client_mode=$3
    
    log "=== Running Scenario: $scenario ==="
    log "Server Mode: $server_mode"
    log "Client Mode: $client_mode"
    
    # Clean state
    cleanup
    
    # Verify installation
    verify_installation $SERVER_VM || return 1
    verify_installation $CLIENT_VM || return 1
    
    # Start server
    if [ "$server_mode" = "foreground" ]; then
        start_foreground $SERVER_VM "server" || return 1
    else
        remote_cmd $SERVER_VM "systemctl start sssonector"
        check_status "Server service startup" || return 1
    fi
    sleep 5
    
    # Start client
    if [ "$client_mode" = "foreground" ]; then
        start_foreground $CLIENT_VM "client" || return 1
    else
        remote_cmd $CLIENT_VM "systemctl start sssonector"
        check_status "Client service startup" || return 1
    fi
    sleep 5
    
    # Verify tunnel establishment
    verify_tunnel $SERVER_VM || return 1
    verify_tunnel $CLIENT_VM || return 1
    
    # Test connectivity
    test_connectivity "Client → Server" $CLIENT_VM "10.0.0.1" || return 1
    test_connectivity "Server → Client" $SERVER_VM "10.0.0.2" || return 1
    
    # Collect logs
    collect_logs "${scenario}"
    
    # Clean up
    cleanup
    
    log "=== Scenario $scenario Completed Successfully ==="
    return 0
}

# Run all scenarios
log "Starting SSSonector Sanity Check"
log "==============================="

run_test_scenario "scenario1_foreground_both" "foreground" "foreground"
run_test_scenario "scenario2_mixed" "foreground" "background"
run_test_scenario "scenario3_background_both" "background" "background"

log "=== Sanity Check Complete ==="
log "All test logs are available in $LOG_DIR"
