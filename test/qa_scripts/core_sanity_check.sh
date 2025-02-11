#!/bin/bash
set -e

# Source common functions and environment
if [ -f "$(dirname "$0")/config.env" ]; then
    set -a  # automatically export all variables
    source "$(dirname "$0")/config.env"
    set +a
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
chmod 777 "$LOG_DIR"

log "INFO" "Starting core functionality sanity check"
log "INFO" "Logs will be saved to: $LOG_DIR"

# Function to wait for tunnel to be ready with state verification
wait_for_tunnel() {
    local vm=$1
    local interface="tun0"
    local timeout=30
    local start_time=$(date +%s)

    log "INFO" "Waiting for tunnel interface on $vm..."
    while true; do
        # Check interface existence
        if remote_cmd $vm "ip link show $interface" > /dev/null 2>&1; then
            # Check interface is up
            if remote_cmd $vm "ip link show $interface | grep -q 'UP'"; then
                # Verify state is Running
                if remote_cmd $vm "journalctl -u sssonector -n 1 | grep -q 'State: Running'"; then
                    log "INFO" "Tunnel interface is up and running on $vm"
                    return 0
                fi
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

# Function to verify packet transmission with statistics
verify_packets() {
    local src_vm=$1
    local dest_ip=$2
    local count=20

    log "INFO" "Testing packet transmission from $src_vm to $dest_ip..."

    # Get initial statistics
    local initial_stats=$(remote_cmd $vm "ip -s link show tun0")
    echo "$initial_stats" > "$LOG_DIR/stats_before_${src_vm}.log"

    # Run ping and capture output
    local ping_output
    ping_output=$(remote_cmd $vm "ping -c $count $dest_ip")
    echo "$ping_output" > "$LOG_DIR/ping_${src_vm}_to_${dest_ip}.log"

    # Get final statistics
    local final_stats=$(remote_cmd $vm "ip -s link show tun0")
    echo "$final_stats" > "$LOG_DIR/stats_after_${src_vm}.log"

    # Check for packet loss
    if echo "$ping_output" | grep -q "0% packet loss"; then
        log "INFO" "All packets transmitted successfully"
        return 0
    else
        log "ERROR" "Packet loss detected"
        return 1
    fi
}

# Function to verify state transitions
verify_state_transitions() {
    local vm=$1
    local operation=$2
    log "INFO" "Verifying $operation state transitions on $vm..."

    # Get initial state
    local initial_state=$(remote_cmd $vm "journalctl -u sssonector -n 1 | grep 'State:'")
    echo "$initial_state" > "$LOG_DIR/${vm}_${operation}_initial_state.log"

    # Collect state transitions
    local transitions=$(remote_cmd $vm "journalctl -u sssonector -n 50 | grep 'State:'")
    echo "$transitions" > "$LOG_DIR/${vm}_${operation}_transitions.log"

    # Verify proper sequence
    case $operation in
        "startup")
            if ! echo "$transitions" | grep -q "Uninitialized.*Initializing.*Ready.*Running"; then
                log "ERROR" "Invalid startup transition sequence"
                return 1
            fi
            ;;
        "shutdown")
            if ! echo "$transitions" | grep -q "Running.*Stopping.*Stopped"; then
                log "ERROR" "Invalid shutdown transition sequence"
                return 1
            fi
            ;;
    esac

    return 0
}

# Function to verify resource cleanup
verify_cleanup() {
    local vm=$1
    log "INFO" "Verifying cleanup on $vm..."

    # Get resource state before stop
    remote_cmd $vm "lsof -p $(pidof sssonector)" > "$LOG_DIR/${vm}_resources_before.log"
    remote_cmd $vm "ls -l /proc/$(pidof sssonector)/fd" > "$LOG_DIR/${vm}_fd_before.log"
    remote_cmd $vm "ss -tlpn | grep sssonector" > "$LOG_DIR/${vm}_sockets_before.log"

    # Stop service
    remote_cmd $vm "systemctl stop sssonector"
    sleep 2

    # Verify cleanup
    local errors=0

    # Check TUN interface
    if remote_cmd $vm "ip link show tun0" >/dev/null 2>&1; then
        log "ERROR" "TUN interface not cleaned up on $vm"
        ((errors++))
    fi

    # Check sockets
    if remote_cmd $vm "ss -tlpn | grep sssonector" >/dev/null 2>&1; then
        log "ERROR" "Sockets not cleaned up on $vm"
        ((errors++))
    fi

    # Check state files
    if remote_cmd $vm "ls -la /var/lib/sssonector/state/" 2>/dev/null | grep -q "\.lock"; then
        log "ERROR" "Lock files not cleaned up on $vm"
        ((errors++))
    fi

    return $errors
}

# Function to run foreground mode test
test_foreground_mode() {
    log "INFO" "Starting foreground mode test..."

    # Ensure log directories exist with proper permissions
    remote_cmd $QA_SERVER_VM "mkdir -p /var/log/sssonector && chown -R sssonector:sssonector /var/log/sssonector"
    remote_cmd $QA_CLIENT_VM "mkdir -p /var/log/sssonector && chown -R sssonector:sssonector /var/log/sssonector"

    # Start server in background
    remote_cmd $QA_SERVER_VM "cd /opt/sssonector/scripts && sudo -u sssonector sssonector -config /etc/sssonector/config.yaml > $LOG_DIR/server_fg.log 2>&1 &"
    sleep 2
    local server_pid=$(remote_cmd $QA_SERVER_VM "pgrep -f 'sssonector.*config.yaml'")
    if [ -z "$server_pid" ]; then
        log "ERROR" "Failed to start server process"
        return 1
    fi
    log "INFO" "Server started with PID $server_pid"

    # Verify server state transitions
    verify_state_transitions $QA_SERVER_VM "startup" || return 1

    # Wait for server to initialize
    sleep 5

    # Start client in background
    remote_cmd $QA_CLIENT_VM "cd /opt/sssonector/scripts && sudo -u sssonector sssonector -config /etc/sssonector/config.yaml > $LOG_DIR/client_fg.log 2>&1 &"
    sleep 2
    local client_pid=$(remote_cmd $QA_CLIENT_VM "pgrep -f 'sssonector.*config.yaml'")
    if [ -z "$client_pid" ]; then
        log "ERROR" "Failed to start client process"
        remote_cmd $QA_SERVER_VM "kill $server_pid"
        return 1
    fi
    log "INFO" "Client started with PID $client_pid"

    # Verify client state transitions
    verify_state_transitions $QA_CLIENT_VM "startup" || return 1

    # Wait for tunnels to be established
    if ! wait_for_tunnel $QA_SERVER_VM; then
        log "ERROR" "Server tunnel setup failed"
        remote_cmd $QA_SERVER_VM "kill $server_pid"
        remote_cmd $QA_CLIENT_VM "kill $client_pid"
        return 1
    fi

    if ! wait_for_tunnel $QA_CLIENT_VM; then
        log "ERROR" "Client tunnel setup failed"
        remote_cmd $QA_SERVER_VM "kill $server_pid"
        remote_cmd $QA_CLIENT_VM "kill $client_pid"
        return 1
    fi

    # Test packet transmission
    if ! verify_packets $QA_CLIENT_VM "10.0.0.1"; then
        log "ERROR" "Client to server packet transmission failed"
        remote_cmd $QA_SERVER_VM "kill $server_pid"
        remote_cmd $QA_CLIENT_VM "kill $client_pid"
        return 1
    fi

    if ! verify_packets $QA_SERVER_VM "10.0.0.2"; then
        log "ERROR" "Server to client packet transmission failed"
        remote_cmd $QA_SERVER_VM "kill $server_pid"
        remote_cmd $QA_CLIENT_VM "kill $client_pid"
        return 1
    fi

    # Clean shutdown
    log "INFO" "Shutting down client..."
    remote_cmd $QA_CLIENT_VM "kill $client_pid"
    verify_state_transitions $QA_CLIENT_VM "shutdown" || return 1
    verify_cleanup $QA_CLIENT_VM || return 1

    log "INFO" "Shutting down server..."
    remote_cmd $QA_SERVER_VM "kill $server_pid"
    verify_state_transitions $QA_SERVER_VM "shutdown" || return 1
    verify_cleanup $QA_SERVER_VM || return 1

    log "INFO" "Foreground mode test completed successfully"
    return 0
}

# Function to run background mode test
test_background_mode() {
    log "INFO" "Starting background mode test..."

    # Start server in background
    remote_cmd $QA_SERVER_VM "systemctl start sssonector"
    sleep 2
    local server_pid=$(remote_cmd $QA_SERVER_VM "pidof sssonector")
    if [ -z "$server_pid" ]; then
        log "ERROR" "Failed to start server process"
        return 1
    fi
    log "INFO" "Server started with PID $server_pid"

    # Verify server state transitions
    verify_state_transitions $QA_SERVER_VM "startup" || return 1

    # Wait for server to initialize
    sleep 5

    # Start client in background
    remote_cmd $QA_CLIENT_VM "systemctl start sssonector"
    sleep 2
    local client_pid=$(remote_cmd $QA_CLIENT_VM "pidof sssonector")
    if [ -z "$client_pid" ]; then
        log "ERROR" "Failed to start client process"
        remote_cmd $QA_SERVER_VM "systemctl stop sssonector"
        return 1
    fi
    log "INFO" "Client started with PID $client_pid"

    # Verify client state transitions
    verify_state_transitions $QA_CLIENT_VM "startup" || return 1

    # Wait for tunnels to be established
    if ! wait_for_tunnel $QA_SERVER_VM; then
        log "ERROR" "Server tunnel setup failed"
        remote_cmd $QA_SERVER_VM "systemctl stop sssonector"
        remote_cmd $QA_CLIENT_VM "systemctl stop sssonector"
        return 1
    fi

    if ! wait_for_tunnel $QA_CLIENT_VM; then
        log "ERROR" "Client tunnel setup failed"
        remote_cmd $QA_SERVER_VM "systemctl stop sssonector"
        remote_cmd $QA_CLIENT_VM "systemctl stop sssonector"
        return 1
    fi

    # Test packet transmission
    if ! verify_packets $QA_CLIENT_VM "10.0.0.1"; then
        log "ERROR" "Client to server packet transmission failed"
        remote_cmd $QA_SERVER_VM "systemctl stop sssonector"
        remote_cmd $QA_CLIENT_VM "systemctl stop sssonector"
        return 1
    fi

    if ! verify_packets $QA_SERVER_VM "10.0.0.2"; then
        log "ERROR" "Server to client packet transmission failed"
        remote_cmd $QA_SERVER_VM "systemctl stop sssonector"
        remote_cmd $QA_CLIENT_VM "systemctl stop sssonector"
        return 1
    fi

    # Clean shutdown
    log "INFO" "Shutting down client..."
    remote_cmd $QA_CLIENT_VM "systemctl stop sssonector"
    verify_state_transitions $QA_CLIENT_VM "shutdown" || return 1
    verify_cleanup $QA_CLIENT_VM || return 1

    log "INFO" "Shutting down server..."
    remote_cmd $QA_SERVER_VM "systemctl stop sssonector"
    verify_state_transitions $QA_SERVER_VM "shutdown" || return 1
    verify_cleanup $QA_SERVER_VM || return 1

    log "INFO" "Background mode test completed successfully"
    return 0
}

# Main test execution
main() {
    # Clean up any existing state
    log "INFO" "Cleaning up existing state..."
    cleanup_vm $QA_SERVER_VM
    cleanup_vm $QA_CLIENT_VM
    sleep 2

    # Run foreground mode test
    if ! test_foreground_mode; then
        log "ERROR" "Foreground mode test failed"
        exit 1
    fi

    # Clean up between tests
    log "INFO" "Cleaning up between tests..."
    cleanup_vm $QA_SERVER_VM
    cleanup_vm $QA_CLIENT_VM
    sleep 5

    # Run background mode test
    if ! test_background_mode; then
        log "ERROR" "Background mode test failed"
        exit 1
    fi

    # Collect all logs
    log "INFO" "Collecting logs..."
    remote_cmd $QA_SERVER_VM "journalctl -u sssonector --no-pager -n 500" > "$LOG_DIR/server_journal.log"
    remote_cmd $QA_CLIENT_VM "journalctl -u sssonector --no-pager -n 500" > "$LOG_DIR/client_journal.log"
    remote_cmd $QA_SERVER_VM "dmesg | tail -n 200" > "$LOG_DIR/server_dmesg.log"
    remote_cmd $QA_CLIENT_VM "dmesg | tail -n 200" > "$LOG_DIR/client_dmesg.log"

    log "INFO" "Core functionality sanity check completed successfully"
    log "INFO" "All test logs available in: $LOG_DIR"
}

# Run main function
main
