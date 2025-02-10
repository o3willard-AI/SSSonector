#!/bin/bash

# Source environment configuration
if [ -f "$(dirname "$0")/config.env" ]; then
    source "$(dirname "$0")/config.env"
else
    echo "Error: config.env not found"
    exit 1
fi

# Source common functions
if [ -f "$(dirname "$0")/common.sh" ]; then
    source "$(dirname "$0")/common.sh"
else
    echo "Error: common.sh not found"
    exit 1
fi

# Validate environment
validate_qa_env || exit 1

# Function to start process in foreground
start_foreground() {
    local vm=$1
    local role=$2
    log "INFO" "Starting $role in foreground on $vm..."
    
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
    log "INFO" "Initial $role logs:"
    remote_cmd $vm "tail -n 10 /var/log/sssonector/${role}.log"
    
    return 0
}

# Function to clean up
cleanup() {
    log "INFO" "Cleaning up..."
    
    # Stop services
    remote_cmd $QA_CLIENT_VM "systemctl stop sssonector || true"
    remote_cmd $QA_SERVER_VM "systemctl stop sssonector || true"
    
    # Kill any foreground processes
    remote_cmd $QA_CLIENT_VM "pkill -f sssonector || true"
    remote_cmd $QA_SERVER_VM "pkill -f sssonector || true"
    
    # Wait for cleanup
    sleep 2
    
    # Verify cleanup
    verify_cleanup $QA_CLIENT_VM
    verify_cleanup $QA_SERVER_VM
}

# Create log directory
mkdir -p $QA_LOG_DIR

# Main test scenarios
run_test_scenario() {
    local scenario=$1
    local server_mode=$2
    local client_mode=$3
    
    log "INFO" "=== Running Scenario: $scenario ==="
    log "INFO" "Server Mode: $server_mode"
    log "INFO" "Client Mode: $client_mode"
    
    # Clean state
    cleanup
    
    # Verify installation
    verify_installation $QA_SERVER_VM || return 1
    verify_installation $QA_CLIENT_VM || return 1
    
    # Start server
    if [ "$server_mode" = "foreground" ]; then
        start_foreground $QA_SERVER_VM "server" || return 1
    else
        remote_cmd $QA_SERVER_VM "systemctl start sssonector"
        check_status "Server service startup" || return 1
    fi
    sleep 5
    
    # Start client
    if [ "$client_mode" = "foreground" ]; then
        start_foreground $QA_CLIENT_VM "client" || return 1
    else
        remote_cmd $QA_CLIENT_VM "systemctl start sssonector"
        check_status "Client service startup" || return 1
    fi
    sleep 5
    
    # Verify tunnel establishment
    verify_tunnel $QA_SERVER_VM || return 1
    verify_tunnel $QA_CLIENT_VM || return 1
    
    # Test connectivity
    test_connectivity "Client → Server" $QA_CLIENT_VM "10.0.0.1" || return 1
    test_connectivity "Server → Client" $QA_SERVER_VM "10.0.0.2" || return 1
    
    # Collect logs
    collect_logs "${scenario}"
    
    # Clean up
    cleanup
    
    log "INFO" "=== Scenario $scenario Completed Successfully ==="
    return 0
}

# Run all scenarios
log "INFO" "Starting SSSonector Sanity Check"
log "INFO" "==============================="

run_test_scenario "scenario1_foreground_both" "foreground" "foreground"
run_test_scenario "scenario2_mixed" "foreground" "background"
run_test_scenario "scenario3_background_both" "background" "background"

log "INFO" "=== Sanity Check Complete ==="
log "INFO" "All test logs are available in $QA_LOG_DIR"
