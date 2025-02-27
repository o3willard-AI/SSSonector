#!/bin/bash

# process_monitor.sh
# Part of Project SENTINEL - QA Environment Validation Tool
# Version: 1.0.0

set -euo pipefail

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    return 1
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Process state validation
validate_process_state() {
    local process_name=$1
    local expected_count=$2
    local failed=0

    log_info "Validating process state for: ${process_name}"

    # Count running processes
    local process_count
    process_count=$(pgrep -c "${process_name}" || echo "0")

    if [[ "${process_count}" -ne "${expected_count}" ]]; then
        log_error "Unexpected process count for ${process_name}. Expected: ${expected_count}, Found: ${process_count}"
        failed=1
    fi

    # Check process state
    if [[ "${process_count}" -gt 0 ]]; then
        while IFS= read -r pid; do
            local state
            state=$(ps -o state= -p "${pid}")
            if [[ "${state}" != "S" && "${state}" != "R" ]]; then
                log_error "Process ${process_name} (PID: ${pid}) in unexpected state: ${state}"
                failed=1
            fi
        done < <(pgrep "${process_name}")
    fi

    return ${failed}
}

# Resource utilization monitoring
monitor_resource_usage() {
    local process_name=$1
    local max_cpu=$2  # percentage
    local max_mem=$3  # percentage
    local failed=0

    log_info "Monitoring resource usage for: ${process_name}"

    while IFS= read -r pid; do
        # Get CPU and memory usage
        local cpu_usage
        local mem_usage
        cpu_usage=$(ps -p "${pid}" -o %cpu= | tr -d ' ')
        mem_usage=$(ps -p "${pid}" -o %mem= | tr -d ' ')

        # Check against thresholds
        if (( $(echo "${cpu_usage} > ${max_cpu}" | bc -l) )); then
            log_error "Process ${process_name} (PID: ${pid}) CPU usage too high: ${cpu_usage}% (max: ${max_cpu}%)"
            failed=1
        fi

        if (( $(echo "${mem_usage} > ${max_mem}" | bc -l) )); then
            log_error "Process ${process_name} (PID: ${pid}) memory usage too high: ${mem_usage}% (max: ${max_mem}%)"
            failed=1
        fi
    done < <(pgrep "${process_name}")

    return ${failed}
}

# Service health checking
check_service_health() {
    local service_name=$1
    local port=$2
    local failed=0

    log_info "Checking service health for: ${service_name}"

    # Check if service is running
    if ! systemctl is-active "${service_name}" &>/dev/null; then
        log_error "Service ${service_name} is not running"
        failed=1
    fi

    # Check port availability
    if ! netstat -tuln | grep -q ":${port} "; then
        log_error "Service ${service_name} port ${port} is not listening"
        failed=1
    fi

    # Check service logs for errors
    if journalctl -u "${service_name}" -n 50 --no-pager | grep -i "error\|failed\|fatal" &>/dev/null; then
        log_error "Found errors in ${service_name} logs"
        failed=1
    fi

    return ${failed}
}

# Process relationship verification
verify_process_relationships() {
    local parent_process=$1
    local child_process=$2
    local failed=0

    log_info "Verifying process relationships: ${parent_process} -> ${child_process}"

    # Get parent process PID
    local parent_pid
    parent_pid=$(pgrep "${parent_process}" || echo "")

    if [[ -z "${parent_pid}" ]]; then
        log_error "Parent process ${parent_process} not found"
        return 1
    fi

    # Check child processes
    local child_count
    child_count=$(pgrep -P "${parent_pid}" "${child_process}" | wc -l)

    if [[ "${child_count}" -eq 0 ]]; then
        log_error "No child processes ${child_process} found for parent ${parent_process}"
        failed=1
    fi

    return ${failed}
}

# Main process validation function
validate_processes() {
    local base_dir=$1
    local failed=0

    # Required processes and their expected counts
    declare -A required_processes=(
        ["sssonector"]=1
        ["sssonector-client"]=1
        ["sssonector-server"]=1
    )

    # Resource limits
    declare -A resource_limits=(
        ["max_cpu"]=80
        ["max_mem"]=70
    )

    # Service ports
    declare -A service_ports=(
        ["sssonector"]=8080
        ["sssonector-client"]=8081
        ["sssonector-server"]=8082
    )

    # Process relationships
    declare -A process_relationships=(
        ["sssonector"]="sssonector-worker"
    )

    # Validate each required process
    for process in "${!required_processes[@]}"; do
        validate_process_state "${process}" "${required_processes[${process}]}" || failed=1
        monitor_resource_usage "${process}" "${resource_limits[max_cpu]}" "${resource_limits[max_mem]}" || failed=1
    done

    # Check service health
    for service in "${!service_ports[@]}"; do
        check_service_health "${service}" "${service_ports[${service}]}" || failed=1
    done

    # Verify process relationships
    for parent in "${!process_relationships[@]}"; do
        verify_process_relationships "${parent}" "${process_relationships[${parent}]}" || failed=1
    done

    return ${failed}
}

# If script is run directly, show usage
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "This script is meant to be sourced by qa_validator.sh"
    exit 1
fi
