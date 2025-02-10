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

# Pre-flight system checks
check_system_requirements() {
    local vm=$1
    log "INFO" "Performing system requirement checks on $vm..."

# Check TUN module and setup
remote_cmd $vm "lsmod | grep -q '^tun '" || {
    log "INFO" "TUN module not loaded, attempting to load..."
    remote_cmd $vm "sudo modprobe tun"
}
remote_cmd $vm "lsmod | grep -q '^tun '"
check_status "TUN module loaded" || {
    log "ERROR" "Failed to load TUN module"
    return 1
}

# Verify TUN device exists and has correct permissions
remote_cmd $vm "test -e /dev/net/tun" || {
    log "INFO" "Creating /dev/net/tun..."
    remote_cmd $vm "sudo mkdir -p /dev/net && sudo mknod /dev/net/tun c 10 200 && sudo chmod 0666 /dev/net/tun"
}
remote_cmd $vm "test -e /dev/net/tun"
check_status "TUN device exists" || return 1

remote_cmd $vm "ls -l /dev/net/tun" > "$LOG_DIR/tun_perms_pre.log"

    # Check network capabilities
    remote_cmd $vm "getcap /usr/local/bin/sssonector | grep -q 'cap_net_admin'"
    check_status "Network admin capabilities" || {
        log "ERROR" "Missing network admin capabilities"
        return 1
    }

    # Check process limits
    remote_cmd $vm "ulimit -n" > "$LOG_DIR/ulimit.log"
    check_status "Process limits check" || return 1

    # Check system resources
    remote_cmd $vm "free -m; df -h; uptime" > "$LOG_DIR/resources.log"
    check_status "System resources check" || return 1

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

    return 0
}

# Enhanced TUN interface verification
verify_tun_interface() {
    local vm=$1
    log "INFO" "Verifying TUN interface setup on $vm..."

    # Capture pre-setup network state
    remote_cmd $vm "ip addr show; ip route show" > "$LOG_DIR/network_pre.log"

    # Check TUN device permissions
    remote_cmd $vm "ls -l /dev/net/tun" > "$LOG_DIR/tun_perms.log"
    check_status "TUN device permissions" || return 1

    # Monitor kernel messages
    remote_cmd $vm "dmesg | tail -n 50" > "$LOG_DIR/kernel_pre.log"

# Configure extended retry parameters
log "INFO" "Configuring extended retry parameters..."
cat > /tmp/server_config.yaml << EOL
mode: "server"
network:
  interface: "tun0"
  address: "10.0.0.1/24"
  mtu: 1500
adapter:
  retry_attempts: 10
  retry_delay: 500
  cleanup_timeout: 10000
  validate_state: true
monitor:
  enabled: true
  log_level: "debug"
EOL
remote_cmd $vm "sudo mkdir -p /etc/sssonector"
remote_cmd $vm "sudo mv /tmp/server_config.yaml /etc/sssonector/config.yaml"

# Start server with extra logging
log "INFO" "Starting server with debug logging..."
remote_cmd $vm "sudo mkdir -p /var/log/sssonector"
remote_cmd $vm "sudo sssonector -config /etc/sssonector/config.yaml -log-level debug > /var/log/sssonector/server.log 2>&1 &"

# Give more time for initialization with increased retries
log "INFO" "Waiting for server initialization..."
sleep 10

    # Check process status
    remote_cmd $vm "pgrep -f sssonector" > "$LOG_DIR/process_id.log" || {
        log "ERROR" "Server failed to start"
        # Collect failure logs
        remote_cmd $vm "cat /var/log/sssonector/server.log" > "$LOG_DIR/server_failure.log"
        remote_cmd $vm "dmesg | tail -n 50" > "$LOG_DIR/kernel_failure.log"
        return 1
    }

# Wait for TUN interface with increased timeout
local timeout=60
    local start_time=$(date +%s)
    while true; do
        if remote_cmd $vm "ip link show tun0" > "$LOG_DIR/tun_interface.log" 2>&1; then
            break
        fi

        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))

        if [ $elapsed -ge $timeout ]; then
            log "ERROR" "Timeout waiting for TUN interface"
            remote_cmd $vm "cat /var/log/sssonector/server.log" > "$LOG_DIR/server_timeout.log"
            return 1
        fi

        log "INFO" "Waiting for TUN interface... (${elapsed}s/${timeout}s)"
        sleep 2
    done

    # Verify interface configuration
    remote_cmd $vm "ip addr show tun0" > "$LOG_DIR/tun_config.log"
    check_status "TUN interface configuration" || return 1

    # Verify interface is up
    remote_cmd $vm "ip link show tun0 | grep -q 'UP'" 
    check_status "TUN interface UP state" || return 1

    # Capture post-setup network state
    remote_cmd $vm "ip addr show; ip route show" > "$LOG_DIR/network_post.log"
    remote_cmd $vm "dmesg | tail -n 50" > "$LOG_DIR/kernel_post.log"

    return 0
}

# Main test execution
main() {
    local vm=$QA_SERVER_VM

    # Clean up any existing state
    log "INFO" "Cleaning up existing state..."
    cleanup_vm $vm

    # Run pre-flight checks
    check_system_requirements $vm || exit 1

    # Verify installation
    verify_server_installation $vm || exit 1

    # Verify TUN interface setup
    verify_tun_interface $vm || exit 1

    # Collect final logs
    log "INFO" "Collecting final logs..."
    remote_cmd $vm "cat /var/log/sssonector/server.log" > "$LOG_DIR/server_final.log"
    remote_cmd $vm "journalctl -u sssonector --no-pager -n 200" > "$LOG_DIR/journal_final.log"

    # Clean up
    log "INFO" "Cleaning up..."
    cleanup_vm $vm

    log "INFO" "Server initialization sanity check complete"
    log "INFO" "All test logs available in: $LOG_DIR"
}

# Run main function
main
