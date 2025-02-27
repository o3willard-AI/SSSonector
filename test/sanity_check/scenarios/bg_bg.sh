#!/bin/bash

# bg_bg.sh
# Tests SSSonector with both server and client in background mode
set -euo pipefail

# Import common utilities
source "$(dirname "${BASH_SOURCE[0]}")/../../lib/common.sh"

# Test settings
PACKET_COUNT=20
PACKET_INTERVAL=0.1
PACKET_TIMEOUT=2

# Start server in background
start_server() {
    local failed=0
    log_info "Starting server in background mode"

    # Start server in background mode
    ssh "${SSSONECTOR_SERVER_USER}@${SSSONECTOR_SERVER_HOST}" \
        "cd ${SSSONECTOR_DEPLOY_ROOT} && \
         sudo ${SSSONECTOR_DEPLOY_ROOT}/bin/sssonector \
         -config ${SSSONECTOR_DEPLOY_ROOT}/etc/sssonector/server.yaml -background" \
        > "${TEST_DIR}/server.log" 2>&1

    # Wait for server to initialize
    sleep 5

    # Get server PID from pidfile
    local server_pid
    server_pid=$(ssh "${SSSONECTOR_SERVER_USER}@${SSSONECTOR_SERVER_HOST}" \
        "cat ${SSSONECTOR_DEPLOY_ROOT}/run/sssonector.pid") || failed=1

    if [[ ${failed} -eq 0 ]]; then
        # Store PID
        echo "${server_pid}" > "${TEST_DIR}/server.pid"

        # Check if server is running
        if ! ssh "${SSSONECTOR_SERVER_USER}@${SSSONECTOR_SERVER_HOST}" "pgrep -f sssonector" > /dev/null; then
            log_error "Server failed to start"
            failed=1
        fi

        # Wait for tunnel interface
        for i in {1..10}; do
            if ssh "${SSSONECTOR_SERVER_USER}@${SSSONECTOR_SERVER_HOST}" "ip link show tun0" &>/dev/null; then
                log_info "Server tunnel interface is up"
                return 0
            fi
            sleep 1
        done

        log_error "Server tunnel interface failed to come up"
        failed=1
    else
        log_error "Failed to get server PID"
    fi

    return ${failed}
}

# Start client in background
start_client() {
    local failed=0
    log_info "Starting client in background mode"

    # Start client in background mode
    ssh "${SSSONECTOR_CLIENT_USER}@${SSSONECTOR_CLIENT_HOST}" \
        "cd ${SSSONECTOR_DEPLOY_ROOT} && \
         sudo ${SSSONECTOR_DEPLOY_ROOT}/bin/sssonector \
         -config ${SSSONECTOR_DEPLOY_ROOT}/etc/sssonector/client.yaml -background" \
        > "${TEST_DIR}/client.log" 2>&1

    # Wait for client to initialize
    sleep 5

    # Get client PID from pidfile
    local client_pid
    client_pid=$(ssh "${SSSONECTOR_CLIENT_USER}@${SSSONECTOR_CLIENT_HOST}" \
        "cat ${SSSONECTOR_DEPLOY_ROOT}/run/sssonector.pid") || failed=1

    if [[ ${failed} -eq 0 ]]; then
        # Store PID
        echo "${client_pid}" > "${TEST_DIR}/client.pid"

        # Check if client is running
        if ! ssh "${SSSONECTOR_CLIENT_USER}@${SSSONECTOR_CLIENT_HOST}" "pgrep -f sssonector" > /dev/null; then
            log_error "Client failed to start"
            failed=1
        fi

        # Wait for tunnel interface
        for i in {1..10}; do
            if ssh "${SSSONECTOR_CLIENT_USER}@${SSSONECTOR_CLIENT_HOST}" "ip link show tun0" &>/dev/null; then
                log_info "Client tunnel interface is up"
                return 0
            fi
            sleep 1
        done

        log_error "Client tunnel interface failed to come up"
        failed=1
    else
        log_error "Failed to get client PID"
    fi

    return ${failed}
}

# Test tunnel connectivity
test_tunnel() {
    local direction=$1
    local src_host=$2
    local src_user=$3
    local dst_ip=$4
    local failed=0

    log_info "Testing ${direction} connectivity"

    # Wait for tunnel interface to be ready
    sleep 5

    # Send test packets
    if ! ssh "${src_user}@${src_host}" \
        "ping -c ${PACKET_COUNT} -i ${PACKET_INTERVAL} -W ${PACKET_TIMEOUT} ${dst_ip}" \
        > "${TEST_DIR}/${direction}_ping.log" 2>&1; then
        
        log_error "${direction} connectivity test failed"
        failed=1
    else
        log_info "${direction} connectivity test passed"
    fi

    return ${failed}
}

# Clean shutdown
cleanup() {
    local failed=0
    log_info "Performing clean shutdown"

    # Stop client
    if [[ -f "${TEST_DIR}/client.pid" ]]; then
        local client_pid
        client_pid=$(cat "${TEST_DIR}/client.pid")
        log_info "Stopping client (PID: ${client_pid})"
        ssh "${SSSONECTOR_CLIENT_USER}@${SSSONECTOR_CLIENT_HOST}" \
            "sudo kill ${client_pid} && sudo rm -f ${SSSONECTOR_DEPLOY_ROOT}/run/sssonector.pid" || failed=1
    fi

    # Stop server
    if [[ -f "${TEST_DIR}/server.pid" ]]; then
        local server_pid
        server_pid=$(cat "${TEST_DIR}/server.pid")
        log_info "Stopping server (PID: ${server_pid})"
        ssh "${SSSONECTOR_SERVER_USER}@${SSSONECTOR_SERVER_HOST}" \
            "sudo kill ${server_pid} && sudo rm -f ${SSSONECTOR_DEPLOY_ROOT}/run/sssonector.pid" || failed=1
    fi

    # Wait for processes to stop
    sleep 5

    # Force kill any remaining processes
    ssh "${SSSONECTOR_SERVER_USER}@${SSSONECTOR_SERVER_HOST}" "sudo pkill -9 -f sssonector || true"
    ssh "${SSSONECTOR_CLIENT_USER}@${SSSONECTOR_CLIENT_HOST}" "sudo pkill -9 -f sssonector || true"

    # Wait for tunnel interfaces to go down
    sleep 2

    # Force remove tunnel interfaces if they still exist
    if ssh "${SSSONECTOR_SERVER_USER}@${SSSONECTOR_SERVER_HOST}" "ip link show tun0" &>/dev/null; then
        log_info "Forcing removal of server tunnel interface"
        ssh "${SSSONECTOR_SERVER_USER}@${SSSONECTOR_SERVER_HOST}" "sudo ip link delete tun0" || failed=1
    fi

    if ssh "${SSSONECTOR_CLIENT_USER}@${SSSONECTOR_CLIENT_HOST}" "ip link show tun0" &>/dev/null; then
        log_info "Forcing removal of client tunnel interface"
        ssh "${SSSONECTOR_CLIENT_USER}@${SSSONECTOR_CLIENT_HOST}" "sudo ip link delete tun0" || failed=1
    fi

    # Verify clean shutdown
    if ssh "${SSSONECTOR_SERVER_USER}@${SSSONECTOR_SERVER_HOST}" "pgrep -f sssonector" &>/dev/null; then
        log_error "Server process still running"
        failed=1
    fi

    if ssh "${SSSONECTOR_CLIENT_USER}@${SSSONECTOR_CLIENT_HOST}" "pgrep -f sssonector" &>/dev/null; then
        log_error "Client process still running"
        failed=1
    fi

    # Check pidfiles are gone
    if ssh "${SSSONECTOR_SERVER_USER}@${SSSONECTOR_SERVER_HOST}" \
        "test -f ${SSSONECTOR_DEPLOY_ROOT}/run/sssonector.pid"; then
        log_error "Server pidfile still exists"
        failed=1
    fi

    if ssh "${SSSONECTOR_CLIENT_USER}@${SSSONECTOR_CLIENT_HOST}" \
        "test -f ${SSSONECTOR_DEPLOY_ROOT}/run/sssonector.pid"; then
        log_error "Client pidfile still exists"
        failed=1
    fi

    if [[ ${failed} -eq 0 ]]; then
        log_info "Clean shutdown completed successfully"
    else
        log_error "Clean shutdown failed"
    fi

    return ${failed}
}

# Main test function
main() {
    local failed=0

    # Create test directory
    TEST_DIR="${RESULTS_DIR}/bg_bg_$(date +%Y%m%d_%H%M%S)"
    mkdir -p "${TEST_DIR}"

    # Start server
    if ! start_server; then
        log_error "Failed to start server"
        cleanup
        return 1
    fi

    # Start client
    if ! start_client; then
        log_error "Failed to start client"
        cleanup
        return 1
    fi

    # Test connectivity
    if ! test_tunnel "client_to_server" "${SSSONECTOR_CLIENT_HOST}" "${SSSONECTOR_CLIENT_USER}" "10.0.0.1"; then
        failed=1
    fi

    if ! test_tunnel "server_to_client" "${SSSONECTOR_SERVER_HOST}" "${SSSONECTOR_SERVER_USER}" "10.0.0.2"; then
        failed=1
    fi

    # Clean shutdown
    if ! cleanup; then
        log_error "Clean shutdown failed"
        failed=1
    fi

    if [[ ${failed} -eq 0 ]]; then
        log_info "Background-Background test completed successfully"
    else
        log_error "Background-Background test failed"
    fi

    return ${failed}
}

# Run main function
main "$@"
