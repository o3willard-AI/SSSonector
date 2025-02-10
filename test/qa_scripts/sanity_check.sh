#!/bin/bash

# Configuration
SERVER_VM="192.168.50.210"
CLIENT_VM="192.168.50.211"
SUDO_PASS="101abn"
LOG_DIR="sanity_test_logs"
PING_COUNT=20

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
    else
        echo -e "${RED}[✗] $1${NC}"
        exit 1
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
    
    # Check package installation
    remote_cmd $vm "dpkg -l | grep sssonector"
    check_status "Package installation check on $vm"
    
    # Check certificate permissions
    remote_cmd $vm "ls -l /etc/sssonector/certs/"
    check_status "Certificate permissions check on $vm"
    
    # Check config file
    remote_cmd $vm "cat /etc/sssonector/config.yaml"
    check_status "Configuration check on $vm"
}

# Function to verify tunnel
verify_tunnel() {
    local vm=$1
    log "Verifying tunnel on $vm..."
    
    # Wait for interface to come up
    for i in {1..10}; do
        if remote_cmd $vm "/sbin/ip addr show tun0" >/dev/null 2>&1; then
            break
        fi
        log "Waiting for tun0 interface... ($i/10)"
        sleep 2
    done
    
    # Check TUN interface
    remote_cmd $vm "/sbin/ip addr show tun0"
    check_status "TUN interface check on $vm"
    
    # Check routing
    remote_cmd $vm "/sbin/ip route show | grep tun0"
    check_status "Routing check on $vm"
}

# Function to test connectivity
test_connectivity() {
    log "Testing connectivity..."
    
    # Client to Server
    log "Testing Client → Server (20 packets)"
    remote_cmd $CLIENT_VM "ping -c $PING_COUNT 10.0.0.1"
    check_status "Client to Server ping test"
    
    # Server to Client
    log "Testing Server → Client (20 packets)"
    remote_cmd $SERVER_VM "ping -c $PING_COUNT 10.0.0.2"
    check_status "Server to Client ping test"
}

# Function to collect logs
collect_logs() {
    local scenario=$1
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local log_dir="${LOG_DIR}/${scenario}_${timestamp}"
    
    log "Collecting logs for $scenario..."
    mkdir -p $log_dir
    
    # Collect journalctl logs
    remote_cmd $SERVER_VM "journalctl -u sssonector --no-pager -n 100" > "${log_dir}/server.log"
    remote_cmd $CLIENT_VM "journalctl -u sssonector --no-pager -n 100" > "${log_dir}/client.log"
    
    # Collect interface status
    remote_cmd $SERVER_VM "/sbin/ip addr show; /sbin/ip route show" > "${log_dir}/server_network.log"
    remote_cmd $CLIENT_VM "/sbin/ip addr show; /sbin/ip route show" > "${log_dir}/client_network.log"
    
    # Collect process logs
    remote_cmd $SERVER_VM "ps aux | grep sssonector" > "${log_dir}/server_process.log"
    remote_cmd $CLIENT_VM "ps aux | grep sssonector" > "${log_dir}/client_process.log"
    
    # Collect application logs
    remote_cmd $SERVER_VM "cat /var/log/sssonector/server.log" > "${log_dir}/server_app.log"
    remote_cmd $CLIENT_VM "cat /var/log/sssonector/client.log" > "${log_dir}/client_app.log"
    
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
    sleep 5
    
    # Verify cleanup
    remote_cmd $CLIENT_VM "! /sbin/ip link show tun0 2>/dev/null"
    check_status "Client cleanup"
    remote_cmd $SERVER_VM "! /sbin/ip link show tun0 2>/dev/null"
    check_status "Server cleanup"
}

# Function to start process in foreground
start_foreground() {
    local vm=$1
    local role=$2
    log "Starting $role in foreground on $vm..."
    
    # Create log directory
    remote_cmd $vm "mkdir -p /var/log/sssonector"
    
    # Set PATH to include /sbin for ip command
    remote_cmd $vm "export PATH=/sbin:/usr/sbin:$PATH && nohup sssonector -config /etc/sssonector/config.yaml > /var/log/sssonector/${role}.log 2>&1 &"
    sleep 2
    
    # Check if process is running
    remote_cmd $vm "pgrep -f sssonector"
    check_status "$role startup"
    
    # Show initial logs
    log "Initial $role logs:"
    remote_cmd $vm "tail -n 5 /var/log/sssonector/${role}.log"
}

# Create log directory
mkdir -p $LOG_DIR

# Verify installation on both VMs
verify_installation $SERVER_VM
verify_installation $CLIENT_VM

# Scenario 1: Foreground Client/Server
log "=== Running Scenario 1: Foreground Client/Server ==="
cleanup

# Start server in foreground
start_foreground $SERVER_VM "server"
sleep 5

# Start client in foreground
start_foreground $CLIENT_VM "client"
sleep 5

verify_tunnel $SERVER_VM
verify_tunnel $CLIENT_VM
test_connectivity
collect_logs "scenario1_foreground"
cleanup

# Scenario 2: Background Client, Foreground Server
log "=== Running Scenario 2: Background Client, Foreground Server ==="
cleanup

# Start server in foreground
start_foreground $SERVER_VM "server"
sleep 5

# Start client as service
log "Starting client service..."
remote_cmd $CLIENT_VM "systemctl start sssonector"
sleep 5
check_status "Client service startup"

verify_tunnel $SERVER_VM
verify_tunnel $CLIENT_VM
test_connectivity
collect_logs "scenario2_mixed"
cleanup

# Scenario 3: Background Client/Server
log "=== Running Scenario 3: Background Client/Server ==="
cleanup

# Start server service
log "Starting server service..."
remote_cmd $SERVER_VM "systemctl start sssonector"
sleep 5
check_status "Server service startup"

# Start client service
log "Starting client service..."
remote_cmd $CLIENT_VM "systemctl start sssonector"
sleep 5
check_status "Client service startup"

verify_tunnel $SERVER_VM
verify_tunnel $CLIENT_VM
test_connectivity
collect_logs "scenario3_background"
cleanup

log "=== Sanity Check Complete ==="
log "All test logs are available in $LOG_DIR"
