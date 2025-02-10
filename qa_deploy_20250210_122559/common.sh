#!/bin/bash

# Common functions for QA testing scripts

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Function to log messages
log() {
    local level=$1
    local message=$2
    local color="${NC}"
    
    case $level in
        "INFO") color="${GREEN}" ;;
        "WARN") color="${YELLOW}" ;;
        "ERROR") color="${RED}" ;;
    esac
    
    echo -e "${color}[$(date '+%Y-%m-%d %H:%M:%S')] [$level] $message${NC}"
}

# Function to check command status
check_status() {
    if [ $? -eq 0 ]; then
        log "INFO" "[✓] $1"
        return 0
    else
        log "ERROR" "[✗] $1"
        return 1
    fi
}

# Function to run command on remote VM
remote_cmd() {
    local vm=$1
    local cmd=$2
    log "INFO" "Executing on $vm: $cmd"
    
    # Use SSH with sudo
    ssh -o StrictHostKeyChecking=no "${QA_USER}@${vm}" "echo ${QA_SUDO_PASSWORD} | sudo -S bash -c '$cmd'" 2>/dev/null
    local status=$?
    
    if [ $status -ne 0 ]; then
        log "ERROR" "Command failed on $vm: $cmd"
        return $status
    fi
    
    return 0
}

# Function to verify installation
verify_installation() {
    local vm=$1
    log "INFO" "Verifying installation on $vm..."
    
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
    local timeout=$QA_TUNNEL_TIMEOUT
    log "INFO" "Verifying tunnel on $vm..."
    
    # Wait for interface with timeout
    local start_time=$(date +%s)
    while true; do
        if remote_cmd $vm "/sbin/ip addr show tun0" >/dev/null 2>&1; then
            break
        fi
        
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))
        
        if [ $elapsed -ge $timeout ]; then
            log "ERROR" "Timeout waiting for tun0 interface"
            return 1
        fi
        
        log "INFO" "Waiting for tun0 interface... (${elapsed}s/${timeout}s)"
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
    log "INFO" "Testing connectivity $direction..."
    
    # Initial statistics
    local rx_bytes_before=$(remote_cmd $src_vm "cat /sys/class/net/tun0/statistics/rx_bytes")
    local tx_bytes_before=$(remote_cmd $src_vm "cat /sys/class/net/tun0/statistics/tx_bytes")
    
    # Run ping test
    local ping_output=$(remote_cmd $src_vm "ping -c $QA_PING_COUNT -W 2 $dst_ip")
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
    
    log "INFO" "Bytes received: $rx_diff"
    log "INFO" "Bytes transmitted: $tx_diff"
    log "INFO" "Packet loss: $packet_loss%"
    
    # Check results
    if [ $ping_status -eq 0 ] && [ "$packet_loss" -eq 0 ]; then
        check_status "$direction connectivity test (${QA_PING_COUNT} packets)"
        return 0
    else
        check_status "$direction connectivity test failed"
        return 1
    fi
}

# Function to clean up VM state
cleanup_vm() {
    local vm=$1
    log "INFO" "Cleaning up VM state on $vm..."
    
    # Stop any running instances
    remote_cmd $vm "pkill -f sssonector" || true
    sleep 2
    
    # Force kill if still running
    remote_cmd $vm "pkill -9 -f sssonector" || true
    
    # Remove TUN interface if exists
    remote_cmd $vm "ip link delete tun0" || true
    
    # Clean up log directory
    remote_cmd $vm "rm -rf /var/log/sssonector/*" || true
    
    return 0
}

# Function to verify process cleanup
verify_cleanup() {
    local vm=$1
    local timeout=$QA_CLEANUP_TIMEOUT
    log "INFO" "Verifying cleanup on $vm..."
    
    # Wait for process termination
    local start_time=$(date +%s)
    while true; do
        if ! remote_cmd $vm "pgrep -f sssonector" >/dev/null 2>&1; then
            break
        fi
        
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))
        
        if [ $elapsed -ge $timeout ]; then
            log "ERROR" "Timeout waiting for process cleanup"
            return 1
        fi
        
        log "INFO" "Waiting for process cleanup... (${elapsed}s/${timeout}s)"
        sleep 1
    done
    
    # Verify interface is gone
    if remote_cmd $vm "/sbin/ip link show tun0" >/dev/null 2>&1; then
        log "ERROR" "TUN interface still exists"
        return 1
    fi
    
    # Check for orphaned processes
    if remote_cmd $vm "pgrep -f sssonector" >/dev/null 2>&1; then
        log "ERROR" "Orphaned processes found"
        return 1
    fi
    
    check_status "Cleanup verification on $vm"
    return 0
}

# Function to collect logs
collect_logs() {
    local scenario=$1
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local log_dir="${QA_LOG_DIR}/${scenario}_${timestamp}"
    
    log "INFO" "Collecting logs for $scenario..."
    mkdir -p $log_dir
    
    # System logs
    remote_cmd $QA_SERVER_VM "journalctl -u sssonector --no-pager -n 200" > "${log_dir}/server_journal.log"
    remote_cmd $QA_CLIENT_VM "journalctl -u sssonector --no-pager -n 200" > "${log_dir}/client_journal.log"
    
    # Application logs
    remote_cmd $QA_SERVER_VM "cat /var/log/sssonector/server.log" > "${log_dir}/server_app.log"
    remote_cmd $QA_CLIENT_VM "cat /var/log/sssonector/client.log" > "${log_dir}/client_app.log"
    
    # Network state
    remote_cmd $QA_SERVER_VM "/sbin/ip addr show; /sbin/ip route show" > "${log_dir}/server_network.log"
    remote_cmd $QA_CLIENT_VM "/sbin/ip addr show; /sbin/ip route show" > "${log_dir}/client_network.log"
    
    # Process state
    remote_cmd $QA_SERVER_VM "ps aux | grep sssonector" > "${log_dir}/server_process.log"
    remote_cmd $QA_CLIENT_VM "ps aux | grep sssonector" > "${log_dir}/client_process.log"
    
    # Interface statistics
    remote_cmd $QA_SERVER_VM "cat /sys/class/net/tun0/statistics/*" > "${log_dir}/server_stats.log" 2>/dev/null || true
    remote_cmd $QA_CLIENT_VM "cat /sys/class/net/tun0/statistics/*" > "${log_dir}/client_stats.log" 2>/dev/null || true
    
    # System resources
    remote_cmd $QA_SERVER_VM "free -m; df -h; uptime" > "${log_dir}/server_resources.log"
    remote_cmd $QA_CLIENT_VM "free -m; df -h; uptime" > "${log_dir}/client_resources.log"
    
    log "INFO" "Logs collected in ${log_dir}"
}
