#!/bin/bash

# process_utils.sh
# Process management utilities for SSSonector testing

set -euo pipefail

# Import common functions
source "$(dirname "${BASH_SOURCE[0]}")/common.sh"

# Process tracking
declare -a PIDS_TO_CLEANUP=()

# Add process to cleanup list
track_process() {
    local pid=$1
    PIDS_TO_CLEANUP+=("${pid}")
}

# Remove process from cleanup list
untrack_process() {
    local pid=$1
    local new_pids=()
    for tracked_pid in "${PIDS_TO_CLEANUP[@]}"; do
        if [[ "${tracked_pid}" != "${pid}" ]]; then
            new_pids+=("${tracked_pid}")
        fi
    done
    PIDS_TO_CLEANUP=("${new_pids[@]}")
}

# Cleanup all tracked processes
cleanup_processes() {
    for pid in "${PIDS_TO_CLEANUP[@]}"; do
        if kill -0 "${pid}" 2>/dev/null; then
            log_info "Stopping process ${pid}"
            kill "${pid}" 2>/dev/null || true
            wait "${pid}" 2>/dev/null || true
        fi
    done
    PIDS_TO_CLEANUP=()
}

# Set up cleanup trap
trap cleanup_processes EXIT

# Start SSSonector process
start_sssonector() {
    local config=$1
    local mode=$2
    local results_dir=$3
    local log_file="${results_dir}/sssonector_${mode}.log"

    log_info "Starting SSSonector in ${mode} mode"
    local pid
    pid=$(start_process "${PROJECT_ROOT}/bin/sssonector" "${config}" "${log_file}" "${mode}")
    track_process "${pid}"

    # Wait for process to start
    sleep 2
    if ! kill -0 "${pid}" 2>/dev/null; then
        log_error "Failed to start SSSonector in ${mode} mode"
        return 1
    fi

    echo "${pid}"
}

# Check if process is running
is_process_running() {
    local pid=$1
    kill -0 "${pid}" 2>/dev/null
}

# Wait for process to exit
wait_for_exit() {
    local pid=$1
    local timeout=${2:-30}
    local count=0

    while is_process_running "${pid}"; do
        sleep 1
        ((count++))
        if [[ ${count} -ge ${timeout} ]]; then
            log_error "Process ${pid} did not exit within ${timeout} seconds"
            return 1
        fi
    done
}

# Stop process gracefully
stop_process() {
    local pid=$1
    local timeout=${2:-30}

    if is_process_running "${pid}"; then
        log_info "Stopping process ${pid}"
        kill "${pid}" 2>/dev/null || true
        wait_for_exit "${pid}" "${timeout}"
        untrack_process "${pid}"
    fi
}

# Check process status
check_process_status() {
    local pid=$1
    local name=$2
    local failed=0

    if ! is_process_running "${pid}"; then
        log_error "${name} process (PID: ${pid}) is not running"
        failed=1
    fi

    return ${failed}
}

# Monitor process resources
monitor_process() {
    local pid=$1
    local max_cpu=${2:-80}  # Default max CPU usage percentage
    local max_mem=${3:-70}  # Default max memory usage percentage
    local failed=0

    if ! is_process_running "${pid}"; then
        log_error "Process ${pid} is not running"
        return 1
    fi

    # Get CPU and memory usage
    local cpu_usage
    local mem_usage
    cpu_usage=$(ps -p "${pid}" -o %cpu= | tr -d ' ')
    mem_usage=$(ps -p "${pid}" -o %mem= | tr -d ' ')

    # Check against thresholds
    if (( $(echo "${cpu_usage} > ${max_cpu}" | bc -l) )); then
        log_error "Process ${pid} CPU usage too high: ${cpu_usage}% (max: ${max_cpu}%)"
        failed=1
    fi

    if (( $(echo "${mem_usage} > ${max_mem}" | bc -l) )); then
        log_error "Process ${pid} memory usage too high: ${mem_usage}% (max: ${max_mem}%)"
        failed=1
    fi

    return ${failed}
}
