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
LOG_DIR="server_test_${TIMESTAMP}"
mkdir -p "$LOG_DIR"

log "INFO" "Starting server initialization sanity check"
log "INFO" "Logs will be saved to: $LOG_DIR"

# Check if service is already running properly
check_running_service() {
    local vm=$1
    log "INFO" "Checking existing service on $vm..."

    # Check if service is running
    if systemctl is-active --quiet sssonector; then
        # Verify TUN interface
        if ip link show tun0 >/dev/null 2>&1; then
            # Verify listening port
            if ss -tlnp | grep -q ':8443.*sssonector'; then
                # Verify state
                if journalctl -u sssonector -n 50 | grep -q "State: Running"; then
                    log "INFO" "Service is already running properly"
                    return 0
                fi
            fi
        fi
    fi
    return 1
}

# Pre-flight system checks
check_system_requirements() {
    local vm=$1
    log "INFO" "Performing system requirement checks on $vm..."

    # Check network capabilities by verifying TUN interface
    if ! ip link show tun0 >/dev/null 2>&1; then
        log "ERROR" "TUN interface not available"
        return 1
    fi

    # Check process limits
    remote_cmd $vm "ulimit -n" > "$LOG_DIR/ulimit.log"
    check_status "Process limits check" || return 1

    # Check system resources
    remote_cmd $vm "free -m; df -h; uptime" > "$LOG_DIR/resources.log"
    check_status "System resources check" || return 1

    # Check state directory
    remote_cmd $vm "ls -la /var/lib/sssonector/" > "$LOG_DIR/state_dir.log"
    check_status "State directory check" || return 1

    return 0
}

# Enhanced server verification
verify_server_installation() {
    local vm=$1
    log "INFO" "Verifying server installation on $vm..."

    # Check binary
    remote_cmd $vm "which sssonector" > "$LOG_DIR/binary_path.log"
    check_status "Binary location check" || return 1

    # Check binary permissions
    remote_cmd $vm "ls -l $(which sssonector)" > "$LOG_DIR/binary_perms.log"
    check_status "Binary permissions check" || return 1

    # Check configuration directory
    remote_cmd $vm "ls -la /etc/sssonector/" > "$LOG_DIR/config_dir.log"
    check_status "Configuration directory check" || return 1

    # Verify configuration syntax
    remote_cmd $vm "sssonector -validate-config -config /etc/sssonector/config.yaml" > "$LOG_DIR/config_validation.log" 2>&1
    check_status "Configuration validation" || return 1

    # Check capabilities
    remote_cmd $vm "getcap $(which sssonector)" > "$LOG_DIR/capabilities.log"
    check_status "Capabilities check" || return 1

    return 0
}

# Enhanced TUN interface verification
verify_tun_interface() {
    local vm=$1
    log "INFO" "Verifying TUN interface setup on $vm..."

    # Check TUN device permissions
    remote_cmd $vm "ls -l /dev/net/tun" > "$LOG_DIR/tun_perms.log"
    check_status "TUN device permissions" || return 1

    # Verify interface configuration
    remote_cmd $vm "ip addr show tun0" > "$LOG_DIR/tun_config.log"
    check_status "TUN interface configuration" || return 1

    # Verify interface is up
    remote_cmd $vm "ip link show tun0 | grep -q 'UP'" 
    check_status "TUN interface UP state" || return 1

    # Capture network state
    remote_cmd $vm "ip addr show; ip route show" > "$LOG_DIR/network_state.log"
    remote_cmd $vm "dmesg | tail -n 50" > "$LOG_DIR/kernel_state.log"

    # Check interface statistics
    remote_cmd $vm "ip -s link show tun0" > "$LOG_DIR/tun_stats.log"
    check_status "TUN interface statistics" || return 1

    return 0
}

# Verify state transitions
verify_state_transitions() {
    local vm=$1
    log "INFO" "Verifying state transitions on $vm..."

    # Get initial state
    local initial_state=$(remote_cmd $vm "journalctl -u sssonector -n 1 | grep 'State:'")
    echo "$initial_state" > "$LOG_DIR/initial_state.log"

    # Stop service
    remote_cmd $vm "systemctl stop sssonector"
    sleep 2

    # Check stopping transition
    local stopping_state=$(remote_cmd $vm "journalctl -u sssonector -n 10 | grep 'State: Stopping'")
    if [ -z "$stopping_state" ]; then
        log "ERROR" "Missing stopping state transition"
        return 1
    fi

    # Start service
    remote_cmd $vm "systemctl start sssonector"
    sleep 2

    # Check startup transitions
    local transitions=$(remote_cmd $vm "journalctl -u sssonector -n 50 | grep 'State:'")
    echo "$transitions" > "$LOG_DIR/state_transitions.log"

    # Verify transition sequence
    if ! echo "$transitions" | grep -q "Uninitialized"; then
        log "ERROR" "Missing uninitialized state"
        return 1
    fi
    if ! echo "$transitions" | grep -q "Initializing"; then
        log "ERROR" "Missing initializing state"
        return 1
    fi
    if ! echo "$transitions" | grep -q "Ready"; then
        log "ERROR" "Missing ready state"
        return 1
    fi
    if ! echo "$transitions" | grep -q "Running"; then
        log "ERROR" "Missing running state"
        return 1
    fi

    return 0
}

# Verify resource cleanup
verify_resource_cleanup() {
    local vm=$1
    log "INFO" "Verifying resource cleanup on $vm..."

    # Get initial resource state
    remote_cmd $vm "lsof -p $(pidof sssonector)" > "$LOG_DIR/initial_resources.log"
    remote_cmd $vm "ls -l /proc/$(pidof sssonector)/fd" > "$LOG_DIR/initial_fd.log"

    # Stop service
    remote_cmd $vm "systemctl stop sssonector"
    sleep 2

    # Check for lingering resources
    if ip link show tun0 >/dev/null 2>&1; then
        log "ERROR" "TUN interface not cleaned up"
        return 1
    fi

    if ss -tlnp | grep -q ':8443.*sssonector'; then
        log "ERROR" "Listening socket not cleaned up"
        return 1
    fi

    if [ -f "/var/lib/sssonector/state/server.lock" ]; then
        log "ERROR" "State lock file not cleaned up"
        return 1
    fi

    # Start service again
    remote_cmd $vm "systemctl start sssonector"
    sleep 2

    # Verify clean startup
    remote_cmd $vm "lsof -p $(pidof sssonector)" > "$LOG_DIR/final_resources.log"
    remote_cmd $vm "ls -l /proc/$(pidof sssonector)/fd" > "$LOG_DIR/final_fd.log"

    return 0
}

# Main test execution
main() {
    local vm=$QA_SERVER_VM

    # Check if service is already running properly
    if check_running_service $vm; then
        log "INFO" "Service is already running properly, skipping setup"
    else
        # Clean up any existing state
        log "INFO" "Cleaning up existing state..."
        remote_cmd $vm "sudo systemctl stop sssonector || true"
        remote_cmd $vm "sudo pkill -f sssonector || true"
        remote_cmd $vm "sudo ip link del tun0 2>/dev/null || true"
        remote_cmd $vm "sudo lsof -ti :8443 | xargs -r sudo kill -9 || true"
        sleep 2

        # Start service
        log "INFO" "Starting service..."
        remote_cmd $vm "sudo systemctl start sssonector"
        sleep 5
    fi

    # Run verification checks
    check_system_requirements $vm || exit 1
    verify_server_installation $vm || exit 1
    verify_tun_interface $vm || exit 1
    verify_state_transitions $vm || exit 1
    verify_resource_cleanup $vm || exit 1

    # Collect final logs
    log "INFO" "Collecting final logs..."
    remote_cmd $vm "journalctl -u sssonector --no-pager -n 200" > "$LOG_DIR/journal_final.log"
    remote_cmd $vm "ps -p $(pidof sssonector) -o pid,ppid,%cpu,%mem,cmd --forest" > "$LOG_DIR/process_tree.log"
    remote_cmd $vm "netstat -anp | grep sssonector" > "$LOG_DIR/network_connections.log"

    log "INFO" "Server initialization sanity check complete"
    log "INFO" "All test logs available in: $LOG_DIR"
}

# Run main function
main
