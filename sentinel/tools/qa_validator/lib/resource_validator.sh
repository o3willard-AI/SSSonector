#!/bin/bash

# resource_validator.sh
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

# CPU validation
validate_cpu() {
    local min_cores=$1
    local min_speed=$2  # MHz
    local failed=0

    log_info "Validating CPU requirements..."

    # Check CPU cores
    local cpu_cores
    cpu_cores=$(nproc)
    if [[ ${cpu_cores} -lt ${min_cores} ]]; then
        log_error "Insufficient CPU cores. Required: ${min_cores}, Found: ${cpu_cores}"
        failed=1
    fi

    # Check CPU speed
    local cpu_speed
    cpu_speed=$(lscpu | grep "CPU MHz" | awk '{print $3}' | cut -d'.' -f1)
    if [[ ${cpu_speed} -lt ${min_speed} ]]; then
        log_error "Insufficient CPU speed. Required: ${min_speed}MHz, Found: ${cpu_speed}MHz"
        failed=1
    fi

    return ${failed}
}

# Memory validation
validate_memory() {
    local min_memory=$1  # MB
    local min_available=$2  # MB
    local failed=0

    log_info "Validating memory requirements..."

    # Check total memory
    local total_memory
    total_memory=$(free -m | awk '/^Mem:/{print $2}')
    if [[ ${total_memory} -lt ${min_memory} ]]; then
        log_error "Insufficient total memory. Required: ${min_memory}MB, Found: ${total_memory}MB"
        failed=1
    fi

    # Check available memory
    local available_memory
    available_memory=$(free -m | awk '/^Mem:/{print $7}')
    if [[ ${available_memory} -lt ${min_available} ]]; then
        log_error "Insufficient available memory. Required: ${min_available}MB, Found: ${available_memory}MB"
        failed=1
    fi

    return ${failed}
}

# Disk space validation
validate_disk_space() {
    local mount_point=$1
    local min_space=$2  # MB
    local min_inodes=$3  # percentage
    local failed=0

    log_info "Validating disk space for: ${mount_point}"

    # Check available space
    local available_space
    available_space=$(df -m "${mount_point}" | awk 'NR==2 {print $4}')
    if [[ ${available_space} -lt ${min_space} ]]; then
        log_error "Insufficient disk space. Required: ${min_space}MB, Found: ${available_space}MB"
        failed=1
    fi

    # Check inode usage
    local inode_usage
    inode_usage=$(df -i "${mount_point}" | awk 'NR==2 {print $5}' | tr -d '%')
    local available_inodes=$((100 - inode_usage))
    if [[ ${available_inodes} -lt ${min_inodes} ]]; then
        log_error "Insufficient available inodes. Required: ${min_inodes}%, Found: ${available_inodes}%"
        failed=1
    fi

    return ${failed}
}

# Network capacity validation
validate_network_capacity() {
    local interface=$1
    local min_speed=$2  # Mbps
    local failed=0

    log_info "Validating network capacity for interface: ${interface}"

    # Check interface exists
    if ! ip link show "${interface}" &>/dev/null; then
        log_error "Network interface ${interface} not found"
        return 1
    fi

    # Check interface speed
    local speed
    speed=$(ethtool "${interface}" 2>/dev/null | grep "Speed:" | awk '{print $2}' | tr -d 'Mb/s')
    if [[ -n "${speed}" && ${speed} -lt ${min_speed} ]]; then
        log_error "Insufficient network speed. Required: ${min_speed}Mbps, Found: ${speed}Mbps"
        failed=1
    fi

    # Check interface state
    local state
    state=$(ip link show "${interface}" | grep -o "state.*" | awk '{print $2}')
    if [[ "${state}" != "UP" ]]; then
        log_error "Network interface ${interface} is not up (state: ${state})"
        failed=1
    fi

    return ${failed}
}

# System load validation
validate_system_load() {
    local max_load=$1
    local failed=0

    log_info "Validating system load..."

    # Get load averages
    local load_1min
    local load_5min
    local load_15min
    read -r load_1min load_5min load_15min < /proc/loadavg

    # Check against threshold
    if (( $(echo "${load_5min} > ${max_load}" | bc -l) )); then
        log_error "System load too high. Maximum: ${max_load}, Current (5min): ${load_5min}"
        failed=1
    fi

    return ${failed}
}

# Resource trend analysis
analyze_resource_trends() {
    local base_dir=$1
    local window=$2  # minutes
    local failed=0

    log_info "Analyzing resource usage trends..."

    # Create trends directory if it doesn't exist
    local trends_dir="${base_dir}/state/trends"
    mkdir -p "${trends_dir}"

    # Get current metrics
    local timestamp
    timestamp=$(date +%s)
    local memory_usage
    memory_usage=$(free -m | awk '/^Mem:/{print $3}')
    local cpu_usage
    cpu_usage=$(top -bn1 | grep "Cpu(s)" | awk '{print $2}')
    local disk_usage
    disk_usage=$(df -m "${base_dir}" | awk 'NR==2 {print $3}')

    # Save current metrics
    echo "${timestamp} ${memory_usage} ${cpu_usage} ${disk_usage}" >> "${trends_dir}/metrics.log"

    # Analyze trends (last ${window} minutes)
    local start_time=$((timestamp - window * 60))
    if [[ -f "${trends_dir}/metrics.log" ]]; then
        # Check for concerning trends
        local increasing_trend
        increasing_trend=$(awk -v start="${start_time}" '
            $1 >= start {
                if (NR > 1 && ($2 > prev_mem * 1.2 || $3 > prev_cpu * 1.2 || $4 > prev_disk * 1.2)) {
                    print "1"
                    exit
                }
                prev_mem=$2
                prev_cpu=$3
                prev_disk=$4
            }
        ' "${trends_dir}/metrics.log")

        if [[ "${increasing_trend}" == "1" ]]; then
            log_warn "Detected concerning resource usage trends"
            failed=1
        fi
    fi

    # Cleanup old entries
    sed -i -e "/^[0-9]\\{10\\} / {/^${start_time}/,\$p;d}" "${trends_dir}/metrics.log"

    return ${failed}
}

# Main resource validation function
validate_resources() {
    local base_dir=$1
    local failed=0

    # System requirements
    local min_cores=2
    local min_cpu_speed=2000  # MHz
    local min_memory=4096     # MB
    local min_available_memory=2048  # MB
    local min_disk_space=10240  # MB
    local min_inodes=20      # percentage
    local min_network_speed=1000  # Mbps
    local max_load=4.0

    # Validate CPU
    validate_cpu "${min_cores}" "${min_cpu_speed}" || failed=1

    # Validate memory
    validate_memory "${min_memory}" "${min_available_memory}" || failed=1

    # Validate disk space
    validate_disk_space "${base_dir}" "${min_disk_space}" "${min_inodes}" || failed=1

    # Validate network capacity (assuming default interface)
    local default_interface
    default_interface=$(ip route | awk '/default/ {print $5}' | head -n1)
    validate_network_capacity "${default_interface}" "${min_network_speed}" || failed=1

    # Validate system load
    validate_system_load "${max_load}" || failed=1

    # Analyze resource trends (last 30 minutes)
    analyze_resource_trends "${base_dir}" 30 || failed=1

    return ${failed}
}

# If script is run directly, show usage
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "This script is meant to be sourced by qa_validator.sh"
    exit 1
fi
