#!/bin/bash
set -e

# Source common functions and environment
if [ -f "$(dirname "$0")/config.env" ]; then
    source "$(dirname "$0")/config.env"
else
    echo "Error: config.env not found"
    exit 1
fi

if [ -f "$(dirname "$0")/common.sh" ]; then
    source "$(dirname "$0")/common.sh"
else
    echo "Error: common.sh not found"
    exit 1
fi

# Create log directory for this run
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_DIR="core_sanity_${TIMESTAMP}"
mkdir -p "$LOG_DIR"

log "INFO" "Starting core functionality sanity check"
log "INFO" "Logs will be saved to: $LOG_DIR"

# Function to wait for tunnel to be ready
wait_for_tunnel() {
    local vm=$1
    local interface="tun0"
    local timeout=30
    local start_time=$(date +%s)
    
    log "INFO" "Waiting for tunnel interface on $vm..."
    while true; do
        if remote_cmd $vm "ip link show $interface" > /dev/null 2>&1; then
            if remote_cmd $vm "ip link show $interface | grep -q 'UP'"; then
                log "INFO" "Tunnel interface is up on $vm"
                return 0
            fi
        fi
        
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))
        
        if [ $elapsed -ge $timeout ]; then
            log "ERROR" "Timeout waiting for tunnel interface on $vm"
            return 1
        fi
        
        sleep 1
    done
}

# Function to verify packet transmission
verify_packets() {
    local src_vm=$1
    local dest_ip=$2
    local count=20
    
    log "INFO" "Testing packet transmission from $src_vm to $dest_ip..."
    
    # Run ping and capture output
    local ping_output
    ping_output=$(remote_cmd $src_vm "ping -c $count $dest_ip")
    echo "$ping_output" > "$LOG_DIR/ping_${src_vm}_to_${dest_ip}.log"
    
    # Check for packet loss
    if echo "$ping_output" | grep -q "0% packet loss"; then
        log "INFO" "All packets transmitted successfully"
        return 0
    else
        log "ERROR" "Packet loss detected"
        return 1
    fi
}

# Function to run foreground mode test
test_foreground_mode() {
    log "INFO" "Starting foreground mode test..."
    
    # Start server in background (but with output visible)
    remote_cmd $SERVER_VM "sssonector -config /etc/sssonector/config.yaml -log-level debug > $LOG_DIR/server_fg.log 2>&1 &"
    SERVER_PID=$!
    
    # Wait for server to initialize
    sleep 5
    
    # Start client in foreground
    remote_cmd $CLIENT_VM "sssonector -config /etc/sssonector/config.yaml -log-level debug > $LOG_DIR/client_fg.log 2>&1 &"
    CLIENT_PID=$!
    
    # Wait for tunnel to be established
    wait_for_tunnel $SERVER_VM || return 1
    wait_for_tunnel $CLIENT_VM || return 1
    
    # Test packet transmission
    verify_packets $CLIENT_VM "10.0.0.1" || return 1
    verify_packets $SERVER_VM "10.0.0.2" || return 1
    
    # Clean shutdown
    log "INFO" "Shutting down client..."
    remote_cmd $CLIENT_VM "kill $CLIENT_PID"
    sleep 2
    
    log "INFO" "Shutting down server..."
    remote_cmd $SERVER_VM "kill $SERVER_PID"
    sleep 2
    
    # Verify clean shutdown
    if remote_cmd $CLIENT_VM "ip link show tun0" > /dev/null 2>&1; then
        log "ERROR" "Client tunnel interface still exists after shutdown"
        return 1
    fi
    
    if remote_cmd $SERVER_VM "ip link show tun0" > /dev/null 2>&1; then
        log "ERROR" "Server tunnel interface still exists after shutdown"
        return 1
    fi
    
    log "INFO" "Foreground mode test completed successfully"
    return 0
}

# Function to run background mode test
test_background_mode() {
    log "INFO" "Starting background mode test..."
    
    # Start server in foreground mode
    remote_cmd $SERVER_VM "sssonector -config /etc/sssonector/config.yaml -log-level debug > $LOG_DIR/server_bg.log 2>&1 &"
    SERVER_PID=$!
    
    # Wait for server to initialize
    sleep 5
    
    # Start client in background
    remote_cmd $CLIENT_VM "sssonector -config /etc/sssonector/config.yaml -log-level debug > $LOG_DIR/client_bg.log 2>&1 &"
    CLIENT_PID=$!
    
    # Wait for tunnel to be established
    wait_for_tunnel $SERVER_VM || return 1
    wait_for_tunnel $CLIENT_VM || return 1
    
    # Verify client is running in background
    if ! remote_cmd $CLIENT_VM "ps -p $CLIENT_PID > /dev/null"; then
        log "ERROR" "Client process not running in background"
        return 1
    fi
    
    # Test packet transmission
    verify_packets $CLIENT_VM "10.0.0.1" || return 1
    verify_packets $SERVER_VM "10.0.0.2" || return 1
    
    # Clean shutdown
    log "INFO" "Shutting down client..."
    remote_cmd $CLIENT_VM "kill $CLIENT_PID"
    sleep 2
    
    log "INFO" "Shutting down server..."
    remote_cmd $SERVER_VM "kill $SERVER_PID"
    sleep 2
    
    # Verify clean shutdown
    if remote_cmd $CLIENT_VM "ip link show tun0" > /dev/null 2>&1; then
        log "ERROR" "Client tunnel interface still exists after shutdown"
        return 1
    fi
    
    if remote_cmd $SERVER_VM "ip link show tun0" > /dev/null 2>&1; then
        log "ERROR" "Server tunnel interface still exists after shutdown"
        return 1
    fi
    
    log "INFO" "Background mode test completed successfully"
    return 0
}

# Main test execution
main() {
    # Clean up any existing state
    log "INFO" "Cleaning up existing state..."
    remote_cmd $SERVER_VM "sudo pkill -f sssonector || true"
    remote_cmd $CLIENT_VM "sudo pkill -f sssonector || true"
    remote_cmd $SERVER_VM "sudo ip link del tun0 2>/dev/null || true"
    remote_cmd $CLIENT_VM "sudo ip link del tun0 2>/dev/null || true"
    sleep 2
    
    # Run foreground mode test
    if ! test_foreground_mode; then
        log "ERROR" "Foreground mode test failed"
        exit 1
    fi
    
    # Clean up between tests
    log "INFO" "Cleaning up between tests..."
    remote_cmd $SERVER_VM "sudo pkill -f sssonector || true"
    remote_cmd $CLIENT_VM "sudo pkill -f sssonector || true"
    remote_cmd $SERVER_VM "sudo ip link del tun0 2>/dev/null || true"
    remote_cmd $CLIENT_VM "sudo ip link del tun0 2>/dev/null || true"
    sleep 5
    
    # Run background mode test
    if ! test_background_mode; then
        log "ERROR" "Background mode test failed"
        exit 1
    fi
    
    # Collect all logs
    log "INFO" "Collecting logs..."
    remote_cmd $SERVER_VM "journalctl -u sssonector --no-pager -n 500" > "$LOG_DIR/server_journal.log"
    remote_cmd $CLIENT_VM "journalctl -u sssonector --no-pager -n 500" > "$LOG_DIR/client_journal.log"
    
    log "INFO" "Core functionality sanity check completed successfully"
    log "INFO" "All test logs available in: $LOG_DIR"
}

# Run main function
main
