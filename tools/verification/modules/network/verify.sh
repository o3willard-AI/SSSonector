#!/bin/bash

# network/verify.sh
# Network configuration verification module
set -euo pipefail

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../../lib/common.sh"

# IP forwarding verification
verify_ip_forwarding() {
    local failed=0

    log_info "Verifying IP forwarding configuration"

    # Check current IP forwarding state
    local ip_forward
    ip_forward=$(cat /proc/sys/net/ipv4/ip_forward)
    if [[ "${ip_forward}" == "1" ]]; then
        track_result "network_ip_forward" "PASS" "IP forwarding is enabled"
    else
        track_result "network_ip_forward" "FAIL" "IP forwarding is disabled"
        failed=1
    fi

    # Check if IP forwarding is persistent
    if grep -q "^net.ipv4.ip_forward\s*=\s*1" /etc/sysctl.conf /etc/sysctl.d/*.conf 2>/dev/null; then
        track_result "network_ip_forward_persistent" "PASS" "IP forwarding is configured to persist"
    else
        track_result "network_ip_forward_persistent" "WARN" "IP forwarding may not persist after reboot"
    fi

    # Check all interfaces forwarding
    local all_forwarding
    all_forwarding=$(cat /proc/sys/net/ipv4/conf/all/forwarding)
    if [[ "${all_forwarding}" == "1" ]]; then
        track_result "network_all_forwarding" "PASS" "All interfaces forwarding is enabled"
    else
        track_result "network_all_forwarding" "WARN" "All interfaces forwarding is disabled"
    fi

    return ${failed}
}

# Interface configuration verification
verify_interfaces() {
    local failed=0

    log_info "Verifying network interface configuration"

    # Get environment type
    local env_type
    env_type=$(load_state "ENVIRONMENT")

    # Check main interface
    local main_interface
    main_interface=$(ip route | awk '/default/ {print $5}' | head -n1)
    if [[ -n "${main_interface}" ]]; then
        if ip link show "${main_interface}" | grep -q "UP"; then
            track_result "network_main_interface" "PASS" "Main interface ${main_interface} is up"
        else
            track_result "network_main_interface" "FAIL" "Main interface ${main_interface} is down"
            failed=1
        fi
    else
        track_result "network_main_interface" "FAIL" "No default interface found"
        failed=1
    fi

    # Check MTU
    local mtu
    mtu=$(ip link show "${main_interface}" | grep -oP 'mtu \K\d+')
    if [[ ${mtu} -ge 1500 ]]; then
        track_result "network_mtu" "PASS" "MTU is sufficient: ${mtu}"
    else
        track_result "network_mtu" "FAIL" "MTU is too low: ${mtu} (min: 1500)"
        failed=1
    fi

    # Check for existing TUN interfaces
    if ip link show | grep -q "tun[0-9]"; then
        track_result "network_tun_exists" "WARN" "Existing TUN interfaces found"
    else
        track_result "network_tun_exists" "PASS" "No existing TUN interfaces"
    fi

    return ${failed}
}

# Port availability verification
verify_ports() {
    local failed=0

    log_info "Verifying port availability"

    # Get environment type
    local env_type
    env_type=$(load_state "ENVIRONMENT")

    # Required ports from environment config
    local -a required_ports=(443 8443 9090 9091 9092 9093)
    
    # Check each port
    for port in "${required_ports[@]}"; do
        if ! ss -tuln | grep -q ":${port} "; then
            track_result "network_port_${port}" "PASS" "Port ${port} is available"
        else
            # For QA environments, some ports should be in use
            if [[ "${env_type}" =~ ^qa ]] && [[ ${port} =~ ^909[0-3]$ ]]; then
                track_result "network_port_${port}" "PASS" "Port ${port} is in use (expected in QA)"
            else
                track_result "network_port_${port}" "FAIL" "Port ${port} is already in use"
                failed=1
            fi
        fi
    done

    return ${failed}
}

# DNS resolution verification
verify_dns() {
    local failed=0

    log_info "Verifying DNS resolution"

    # Check resolv.conf
    if [[ -f /etc/resolv.conf ]]; then
        if grep -q "^nameserver" /etc/resolv.conf; then
            track_result "network_resolv_conf" "PASS" "DNS nameservers configured"
        else
            track_result "network_resolv_conf" "FAIL" "No DNS nameservers found"
            failed=1
        fi
    else
        track_result "network_resolv_conf" "FAIL" "resolv.conf not found"
        failed=1
    fi

    # Test DNS resolution
    if dig +short +timeout=2 +tries=1 google.com >/dev/null; then
        track_result "network_dns_resolution" "PASS" "DNS resolution working"
    else
        track_result "network_dns_resolution" "FAIL" "DNS resolution failed"
        failed=1
    fi

    return ${failed}
}

# Network connectivity verification
verify_connectivity() {
    local failed=0

    log_info "Verifying network connectivity"

    # Get environment type
    local env_type
    env_type=$(load_state "ENVIRONMENT")

    # Test internet connectivity
    if ping -c 1 -W 2 8.8.8.8 >/dev/null 2>&1; then
        track_result "network_internet" "PASS" "Internet connectivity available"
    else
        track_result "network_internet" "FAIL" "No internet connectivity"
        failed=1
    fi

    # Check network latency
    local latency
    latency=$(ping -c 3 8.8.8.8 2>/dev/null | tail -1 | awk -F '/' '{print $5}')
    if [[ -n "${latency}" ]]; then
        # Get environment-specific threshold
        local latency_threshold=100  # Default 100ms
        if [[ "${env_type}" =~ ^qa ]]; then
            latency_threshold=50  # Stricter for QA
        fi

        if [[ $(echo "${latency} < ${latency_threshold}" | bc) -eq 1 ]]; then
            track_result "network_latency" "PASS" "Network latency acceptable: ${latency}ms"
        else
            track_result "network_latency" "WARN" "High network latency: ${latency}ms"
        fi
    else
        track_result "network_latency" "FAIL" "Could not measure network latency"
        failed=1
    fi

    return ${failed}
}

# Main verification function
main() {
    local failed=0

    # Run verifications
    verify_ip_forwarding || failed=1
    verify_interfaces || failed=1
    verify_ports || failed=1
    verify_dns || failed=1
    verify_connectivity || failed=1

    return ${failed}
}

# Run main function
main "$@"
