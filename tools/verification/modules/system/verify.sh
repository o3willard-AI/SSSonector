#!/bin/bash

# system/verify.sh
# System requirements verification module
set -euo pipefail

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../../lib/common.sh"

# OpenSSL verification
verify_openssl() {
    local failed=0
    local version
    local config_dir
    local fips_status

    log_info "Verifying OpenSSL configuration"

    # Version check
    version=$(openssl version | cut -d' ' -f2)
    if version_gt "1.1.1" "${version}"; then
        track_result "system_openssl_version" "FAIL" "OpenSSL version ${version} is older than required 1.1.1"
        failed=1
    else
        track_result "system_openssl_version" "PASS" "OpenSSL version ${version} meets requirements"
    fi

    # Config directory
    config_dir=$(openssl version -d | cut -d'"' -f2)
    if [[ -d "${config_dir}" ]]; then
        track_result "system_openssl_config" "PASS" "OpenSSL config directory exists: ${config_dir}"
    else
        track_result "system_openssl_config" "FAIL" "OpenSSL config directory not found: ${config_dir}"
        failed=1
    fi

    # Check providers
    if openssl list -providers | grep -q "OpenSSL Default Provider"; then
        track_result "system_openssl_providers" "PASS" "Default provider available"
    else
        track_result "system_openssl_providers" "FAIL" "Default provider not available"
        failed=1
    fi

    # Check AES-GCM support
    if openssl list -cipher-algorithms | grep -q "id-aes.*-GCM"; then
        track_result "system_openssl_aes_gcm" "PASS" "AES-GCM support available"
    else
        track_result "system_openssl_aes_gcm" "FAIL" "AES-GCM support not available"
        failed=1
    fi

    return ${failed}
}

# TUN module verification
verify_tun() {
    local failed=0

    log_info "Verifying TUN module support"

    # Check if TUN is built into kernel
    if grep -q "CONFIG_TUN=y" /boot/config-$(uname -r) 2>/dev/null; then
        track_result "system_tun_kernel" "PASS" "TUN support is built into kernel"
    else
        # Check if TUN module is available
        if lsmod | grep -q "^tun"; then
            track_result "system_tun_module" "PASS" "TUN module is loaded"
        else
            if modinfo tun &>/dev/null; then
                track_result "system_tun_module" "WARN" "TUN module available but not loaded"
            else
                track_result "system_tun_module" "FAIL" "TUN module not available"
                failed=1
            fi
        fi
    fi

    # Check /dev/net/tun
    if [[ -e /dev/net/tun ]]; then
        if [[ -r /dev/net/tun && -w /dev/net/tun ]]; then
            track_result "system_tun_device" "PASS" "TUN device is accessible"
        else
            track_result "system_tun_device" "FAIL" "TUN device permissions incorrect"
            failed=1
        fi
    else
        track_result "system_tun_device" "FAIL" "TUN device not found"
        failed=1
    fi

    return ${failed}
}

# System resources verification
verify_resources() {
    local failed=0

    log_info "Verifying system resources"

    # File descriptor limits
    local fd_limit
    fd_limit=$(ulimit -n)
    if [[ ${fd_limit} -ge 65535 ]]; then
        track_result "system_fd_limit" "PASS" "File descriptor limit sufficient: ${fd_limit}"
    else
        track_result "system_fd_limit" "FAIL" "File descriptor limit too low: ${fd_limit} (min: 65535)"
        failed=1
    fi

    # Memory
    local mem_total mem_free
    mem_total=$(free -m | awk '/^Mem:/{print $2}')
    mem_free=$(free -m | awk '/^Mem:/{print $4}')
    
    # Get environment-specific threshold
    local env_type
    env_type=$(load_state "ENVIRONMENT")
    local mem_threshold=512  # Default
    if [[ "${env_type}" =~ ^qa ]]; then
        mem_threshold=1024
    fi

    if [[ ${mem_free} -ge ${mem_threshold} ]]; then
        track_result "system_memory" "PASS" "Free memory sufficient: ${mem_free}MB"
    else
        track_result "system_memory" "FAIL" "Free memory too low: ${mem_free}MB (min: ${mem_threshold}MB)"
        failed=1
    fi

    # Disk space
    local disk_space
    disk_space=$(df -BG . | awk 'NR==2 {gsub("G","",$4); print $4}')
    if [[ ${disk_space} -ge 1 ]]; then
        track_result "system_disk_space" "PASS" "Disk space sufficient: ${disk_space}GB"
    else
        track_result "system_disk_space" "FAIL" "Disk space too low: ${disk_space}GB (min: 1GB)"
        failed=1
    fi

    # CPU cores
    local cpu_cores
    cpu_cores=$(nproc)
    if [[ ${cpu_cores} -ge 2 ]]; then
        track_result "system_cpu_cores" "PASS" "CPU cores sufficient: ${cpu_cores}"
    else
        track_result "system_cpu_cores" "WARN" "Limited CPU cores available: ${cpu_cores}"
    fi

    # System load
    local load_avg
    load_avg=$(cut -d' ' -f1 /proc/loadavg)
    if [[ $(echo "${load_avg} < ${cpu_cores}" | bc) -eq 1 ]]; then
        track_result "system_load" "PASS" "System load acceptable: ${load_avg}"
    else
        track_result "system_load" "WARN" "High system load: ${load_avg}"
    fi

    return ${failed}
}

# Main verification function
main() {
    local failed=0

    # Run verifications
    verify_openssl || failed=1
    verify_tun || failed=1
    verify_resources || failed=1

    return ${failed}
}

# Run main function
main "$@"
