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

# Validate environment
validate_qa_env || exit 1

# Create log directory for this run
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_DIR="core_test_${TIMESTAMP}"
mkdir -p "$LOG_DIR"

log "INFO" "Starting core functionality sanity check"
log "INFO" "Logs will be saved to: $LOG_DIR"

# Clean up any existing instances
log "INFO" "Cleaning up existing instances..."
cleanup_vm "$QA_SERVER_VM"
cleanup_vm "$QA_CLIENT_VM"

# Verify installation
log "INFO" "Verifying installation on server..."
verify_installation "$QA_SERVER_VM" || exit 1

log "INFO" "Verifying installation on client..."
verify_installation "$QA_CLIENT_VM" || exit 1

# Start server in foreground
log "INFO" "Starting server in foreground mode..."
remote_cmd "$QA_SERVER_VM" "mkdir -p /var/log/sssonector"
remote_cmd "$QA_SERVER_VM" "sssonector -config /etc/sssonector/config.yaml -log-level debug > /var/log/sssonector/server.log 2>&1 &"
sleep 5

# Verify server is running
remote_cmd "$QA_SERVER_VM" "pgrep -f sssonector" || {
    log "ERROR" "Server failed to start"
    exit 1
}

# Start client in foreground
log "INFO" "Starting client in foreground mode..."
remote_cmd "$QA_CLIENT_VM" "mkdir -p /var/log/sssonector"
remote_cmd "$QA_CLIENT_VM" "sssonector -config /etc/sssonector/config.yaml -log-level debug > /var/log/sssonector/client.log 2>&1 &"
sleep 5

# Verify client is running
remote_cmd "$QA_CLIENT_VM" "pgrep -f sssonector" || {
    log "ERROR" "Client failed to start"
    exit 1
}

# Verify tunnel establishment
log "INFO" "Verifying tunnel establishment..."
verify_tunnel "$QA_SERVER_VM" || exit 1
verify_tunnel "$QA_CLIENT_VM" || exit 1

# Test connectivity: Client → Server
log "INFO" "Testing connectivity: Client → Server"
test_connectivity "Client → Server" "$QA_CLIENT_VM" "10.0.0.1" || exit 1

# Test connectivity: Server → Client
log "INFO" "Testing connectivity: Server → Client"
test_connectivity "Server → Client" "$QA_SERVER_VM" "10.0.0.2" || exit 1

# Collect tunnel statistics
log "INFO" "Collecting tunnel statistics..."
remote_cmd "$QA_SERVER_VM" "ip -s link show tun0" > "$LOG_DIR/server_tunnel_stats.log"
remote_cmd "$QA_CLIENT_VM" "ip -s link show tun0" > "$LOG_DIR/client_tunnel_stats.log"

# Clean shutdown of client
log "INFO" "Initiating client shutdown..."
remote_cmd "$QA_CLIENT_VM" "pkill -TERM -f sssonector"
sleep 5

# Verify client cleanup
log "INFO" "Verifying client cleanup..."
verify_cleanup "$QA_CLIENT_VM" || {
    log "ERROR" "Client cleanup failed"
    exit 1
}

# Verify server is still running
remote_cmd "$QA_SERVER_VM" "pgrep -f sssonector" || {
    log "ERROR" "Server stopped unexpectedly"
    exit 1
}

# Collect logs
log "INFO" "Collecting test logs..."

# System logs
remote_cmd "$QA_SERVER_VM" "journalctl -u sssonector --no-pager -n 200" > "$LOG_DIR/server_journal.log"
remote_cmd "$QA_CLIENT_VM" "journalctl -u sssonector --no-pager -n 200" > "$LOG_DIR/client_journal.log"

# Application logs
remote_cmd "$QA_SERVER_VM" "cat /var/log/sssonector/server.log" > "$LOG_DIR/server_app.log"
remote_cmd "$QA_CLIENT_VM" "cat /var/log/sssonector/client.log" > "$LOG_DIR/client_app.log"

# Network state
remote_cmd "$QA_SERVER_VM" "ip addr show; ip route show" > "$LOG_DIR/server_network.log"
remote_cmd "$QA_CLIENT_VM" "ip addr show; ip route show" > "$LOG_DIR/client_network.log"

# Clean up server
log "INFO" "Cleaning up server..."
cleanup_vm "$QA_SERVER_VM"

# Final verification
log "INFO" "Verifying final state..."
verify_cleanup "$QA_SERVER_VM" || exit 1

# Test summary
log "INFO" "Core functionality sanity check complete"
log "INFO" "All test logs available in: $LOG_DIR"

# Exit with success
exit 0
