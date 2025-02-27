#!/bin/bash
set -euo pipefail

# Source common functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(cd "${SCRIPT_DIR}/../../.." && pwd)"
source "${BASE_DIR}/test/lib/common.sh"
source "${BASE_DIR}/test/lib/process_utils.sh"

# Configuration
SERVER_CONFIG="${BASE_DIR}/test/sanity_check/configs/server.yaml"
CLIENT_CONFIG="${BASE_DIR}/test/sanity_check/configs/client.yaml"
SSSONECTOR_BIN="${BASE_DIR}/sssonector"
TEST_DURATION=30
PING_COUNT=20

log_info "Starting foreground-foreground local test"
log_info "Using configurations:"
log_info "  Server: ${SERVER_CONFIG}"
log_info "  Client: ${CLIENT_CONFIG}"

# Ensure clean environment
cleanup() {
    log_info "Cleaning up processes and interfaces"
    pkill -f sssonector || true
    sudo ip link delete tun0 2>/dev/null || true
    sudo ip link delete tun1 2>/dev/null || true
    sudo ip link delete tun2 2>/dev/null || true
}
trap cleanup EXIT

# Start server
log_info "Starting server in foreground"
sudo "${SSSONECTOR_BIN}" -config "${SERVER_CONFIG}" -debug &
SERVER_PID=$!
sleep 2  # Wait for server to initialize

# Start client
log_info "Starting client in foreground"
sudo "${SSSONECTOR_BIN}" -config "${CLIENT_CONFIG}" -debug &
CLIENT_PID=$!
sleep 2  # Wait for client to initialize

# Test connectivity
log_info "Testing connectivity"
log_info "Pinging from client (10.0.0.2) to server (10.0.0.1)"
if ! ping -c ${PING_COUNT} -W 1 10.0.0.1; then
    log_error "Ping test failed"
    exit 1
fi

log_info "Pinging from server (10.0.0.1) to client (10.0.0.2)"
if ! ping -c ${PING_COUNT} -W 1 10.0.0.2; then
    log_error "Ping test failed"
    exit 1
fi

# Let the connection run for a while to test stability
log_info "Testing connection stability for ${TEST_DURATION} seconds"
sleep ${TEST_DURATION}

# Check process health
if ! kill -0 ${SERVER_PID} 2>/dev/null; then
    log_error "Server process died"
    exit 1
fi

if ! kill -0 ${CLIENT_PID} 2>/dev/null; then
    log_error "Client process died"
    exit 1
fi

log_info "Test completed successfully"
exit 0
