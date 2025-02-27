#!/bin/bash

# Basic connectivity test scenario
set -euo pipefail

# Import utilities
source "$(dirname "${BASH_SOURCE[0]}")/../../lib/common.sh"
source "$(dirname "${BASH_SOURCE[0]}")/../../lib/process_utils.sh"

# Initialize results directory
SCENARIO_NAME="02_basic_connectivity"
RESULTS_DIR=$(init_results "${SCENARIO_NAME}")

# Start server and client
start_processes() {
    local failed=0

    # Start server
    log_info "Starting server process"
    local server_pid
    server_pid=$(start_sssonector "${PROJECT_ROOT}/configs/server.yaml" "server" "${RESULTS_DIR}")
    if [[ $? -ne 0 ]]; then
        log_error "Failed to start server process"
        return 1
    fi

    # Wait for server to initialize
    sleep 5

    # Start client
    log_info "Starting client process"
    local client_pid
    client_pid=$(start_sssonector "${PROJECT_ROOT}/configs/client.yaml" "client" "${RESULTS_DIR}")
    if [[ $? -ne 0 ]]; then
        log_error "Failed to start client process"
        stop_process "${server_pid}"
        return 1
    fi

    # Export PIDs for other functions
    export SERVER_PID="${server_pid}"
    export CLIENT_PID="${client_pid}"

    return ${failed}
}

# Test tunnel connectivity
test_tunnel() {
    local failed=0

    # Wait for tunnel interface
    log_info "Waiting for tunnel interface"
    if ! wait_for_interface "tun0" 30; then
        log_error "Tunnel interface not available"
        return 1
    fi

    # Test server to client connectivity
    log_info "Testing server to client connectivity"
    if ! check_connectivity "10.0.0.2" 5 2; then
        log_error "Failed to ping client through tunnel"
        failed=1
    fi

    # Test client to server connectivity
    log_info "Testing client to server connectivity"
    if ! check_connectivity "10.0.0.1" 5 2; then
        log_error "Failed to ping server through tunnel"
        failed=1
    fi

    return ${failed}
}

# Monitor process health
monitor_health() {
    local failed=0

    # Check server process
    if ! check_process_status "${SERVER_PID}" "server"; then
        failed=1
    fi

    # Check client process
    if ! check_process_status "${CLIENT_PID}" "client"; then
        failed=1
    fi

    # Monitor resource usage
    if ! monitor_process "${SERVER_PID}" 80 70; then
        log_error "Server process resource usage exceeded limits"
        failed=1
    fi

    if ! monitor_process "${CLIENT_PID}" 80 70; then
        log_error "Client process resource usage exceeded limits"
        failed=1
    fi

    return ${failed}
}

# Cleanup function
cleanup() {
    log_info "Cleaning up processes"

    # Stop client first
    if [[ -n "${CLIENT_PID:-}" ]]; then
        stop_process "${CLIENT_PID}"
    fi

    # Then stop server
    if [[ -n "${SERVER_PID:-}" ]]; then
        stop_process "${SERVER_PID}"
    fi

    # Wait for tunnel interface to be removed
    local count=0
    while ip link show tun0 &>/dev/null; do
        sleep 1
        ((count++))
        if [[ ${count} -ge 10 ]]; then
            log_warn "Tunnel interface still exists after cleanup"
            break
        fi
    done
}

# Main test function
main() {
    local failed=0

    # Check requirements
    check_requirements || exit 1

    # Start processes
    if ! start_processes; then
        save_result "${SCENARIO_NAME}" "FAIL" "Failed to start processes"
        return 1
    fi

    # Test tunnel connectivity
    if ! test_tunnel; then
        save_result "${SCENARIO_NAME}" "FAIL" "Tunnel connectivity test failed"
        cleanup
        return 1
    fi

    # Monitor process health
    if ! monitor_health; then
        save_result "${SCENARIO_NAME}" "FAIL" "Process health check failed"
        cleanup
        return 1
    fi

    # Clean up
    cleanup

    save_result "${SCENARIO_NAME}" "PASS" "Basic connectivity test completed successfully"
    return 0
}

# Set up cleanup trap
trap cleanup EXIT

# Run main function
main "$@"
