#!/bin/bash

# Performance test scenario
set -euo pipefail

# Import utilities
source "$(dirname "${BASH_SOURCE[0]}")/../../lib/common.sh"
source "$(dirname "${BASH_SOURCE[0]}")/../../lib/process_utils.sh"

# Initialize results directory
SCENARIO_NAME="03_performance"
RESULTS_DIR=$(init_results "${SCENARIO_NAME}")

# Performance thresholds
MIN_THROUGHPUT=50  # MB/s
MAX_LATENCY=10    # ms
MAX_PACKET_LOSS=1 # percent
TEST_DURATION=60  # seconds
PAYLOAD_SIZE=1024 # bytes

# Start test processes
start_test_processes() {
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

    # Wait for tunnel to be ready
    if ! wait_for_interface "tun0" 30; then
        log_error "Tunnel interface not available"
        cleanup
        return 1
    fi

    return ${failed}
}

# Run throughput test
test_throughput() {
    local failed=0
    local throughput_log="${RESULTS_DIR}/throughput.log"

    log_info "Running throughput test (${TEST_DURATION}s)"

    # Check if iperf3 is available
    if command -v iperf3 &>/dev/null; then
        # Start iperf3 server on tunnel interface
        iperf3 -s -B 10.0.0.1 -D -1 > "${RESULTS_DIR}/iperf_server.log" 2>&1
        sleep 2

        # Run iperf3 client test
        if ! iperf3 -c 10.0.0.1 -t "${TEST_DURATION}" -l "${PAYLOAD_SIZE}" > "${throughput_log}" 2>&1; then
            log_error "Throughput test failed"
            failed=1
        fi

        # Kill iperf3 server
        pkill -f "iperf3 -s" || true
    else
        # Fallback to ping flood test
        if ! ping -f -s "${PAYLOAD_SIZE}" -c "$((TEST_DURATION * 100))" 10.0.0.1 > "${throughput_log}" 2>&1; then
            log_error "Ping flood test failed"
            failed=1
        fi
    fi

    # Analyze results
    local measured_throughput
    if command -v iperf3 &>/dev/null; then
        measured_throughput=$(awk '/receiver/ {print $7}' "${throughput_log}")
    else
        measured_throughput=$(awk -F'/' '/packets transmitted/ {print ($1 * 1024 * 8) / 1000000 / '"${TEST_DURATION}"'}' "${throughput_log}")
    fi

    if (( $(echo "${measured_throughput} < ${MIN_THROUGHPUT}" | bc -l) )); then
        log_error "Throughput below threshold: ${measured_throughput} MB/s (min: ${MIN_THROUGHPUT} MB/s)"
        failed=1
    fi

    return ${failed}
}

# Run latency test
test_latency() {
    local failed=0
    local latency_log="${RESULTS_DIR}/latency.log"

    log_info "Running latency test"

    # Run ping test with normal interval
    if ! ping -c 100 -i 0.1 10.0.0.1 > "${latency_log}" 2>&1; then
        log_error "Latency test failed"
        return 1
    fi

    # Analyze results
    local avg_latency
    local packet_loss
    avg_latency=$(awk -F'/' '/rtt/ {print $5}' "${latency_log}")
    packet_loss=$(awk -F'[,%]' '/packet loss/ {print $7}' "${latency_log}")

    if (( $(echo "${avg_latency} > ${MAX_LATENCY}" | bc -l) )); then
        log_error "Latency above threshold: ${avg_latency} ms (max: ${MAX_LATENCY} ms)"
        failed=1
    fi

    if (( $(echo "${packet_loss} > ${MAX_PACKET_LOSS}" | bc -l) )); then
        log_error "Packet loss above threshold: ${packet_loss}% (max: ${MAX_PACKET_LOSS}%)"
        failed=1
    fi

    return ${failed}
}

# Monitor resource usage during tests
monitor_resources() {
    local failed=0

    # Monitor server process
    if ! monitor_process "${SERVER_PID}" 80 70; then
        log_error "Server process exceeded resource limits"
        failed=1
    fi

    # Monitor client process
    if ! monitor_process "${CLIENT_PID}" 80 70; then
        log_error "Client process exceeded resource limits"
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

    # Kill any leftover iperf processes
    pkill -f "iperf3" || true
}

# Main test function
main() {
    local failed=0

    # Check requirements
    check_requirements || exit 1

    # Start processes
    if ! start_test_processes; then
        save_result "${SCENARIO_NAME}" "FAIL" "Failed to start test processes"
        return 1
    fi

    # Run throughput test
    if ! test_throughput; then
        save_result "${SCENARIO_NAME}" "FAIL" "Throughput test failed"
        cleanup
        return 1
    fi

    # Run latency test
    if ! test_latency; then
        save_result "${SCENARIO_NAME}" "FAIL" "Latency test failed"
        cleanup
        return 1
    fi

    # Monitor resource usage
    if ! monitor_resources; then
        save_result "${SCENARIO_NAME}" "FAIL" "Resource monitoring failed"
        cleanup
        return 1
    fi

    # Clean up
    cleanup

    save_result "${SCENARIO_NAME}" "PASS" "Performance tests completed successfully"
    return 0
}

# Set up cleanup trap
trap cleanup EXIT

# Run main function
main "$@"
