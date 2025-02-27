#!/bin/bash

# performance/verify.sh
# Performance baseline verification module
set -euo pipefail

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../../lib/common.sh"

# System performance verification
verify_system_performance() {
    local failed=0

    log_info "Verifying system performance metrics"

    # Get environment type for thresholds
    local env_type
    env_type=$(load_state "ENVIRONMENT")
    
    # Set thresholds based on environment
    local cpu_idle_min=50
    local memory_free_min=512
    if [[ "${env_type}" =~ ^qa ]]; then
        cpu_idle_min=70
        memory_free_min=1024
    fi

    # CPU utilization
    local cpu_idle
    cpu_idle=$(top -bn1 | grep "Cpu(s)" | awk '{print $8}' | cut -d. -f1)
    if [[ ${cpu_idle} -ge ${cpu_idle_min} ]]; then
        track_result "performance_cpu_idle" "PASS" "CPU idle percentage sufficient: ${cpu_idle}%"
    else
        track_result "performance_cpu_idle" "FAIL" "CPU idle percentage too low: ${cpu_idle}% (min: ${cpu_idle_min}%)"
        failed=1
    fi

    # Memory usage
    local mem_free
    mem_free=$(free -m | awk '/^Mem:/ {print $4}')
    if [[ ${mem_free} -ge ${memory_free_min} ]]; then
        track_result "performance_memory_free" "PASS" "Free memory sufficient: ${mem_free}MB"
    else
        track_result "performance_memory_free" "FAIL" "Free memory too low: ${mem_free}MB (min: ${memory_free_min}MB)"
        failed=1
    fi

    # Load average
    local load_1min
    load_1min=$(cut -d' ' -f1 /proc/loadavg)
    local cpu_cores
    cpu_cores=$(nproc)
    if [[ $(echo "${load_1min} < ${cpu_cores}" | bc) -eq 1 ]]; then
        track_result "performance_load" "PASS" "System load acceptable: ${load_1min}"
    else
        track_result "performance_load" "WARN" "High system load: ${load_1min}"
    fi

    return ${failed}
}

# Network performance verification
verify_network_performance() {
    local failed=0

    log_info "Verifying network performance metrics"

    # Get environment type for thresholds
    local env_type
    env_type=$(load_state "ENVIRONMENT")
    
    # Set thresholds based on environment
    local latency_max=1.0
    local throughput_min=100
    if [[ "${env_type}" =~ ^qa ]]; then
        latency_max=0.5
        throughput_min=500
    fi

    # Network latency
    local latency
    latency=$(ping -c 5 8.8.8.8 2>/dev/null | tail -1 | awk -F '/' '{print $5}')
    if [[ -n "${latency}" ]]; then
        if [[ $(echo "${latency} < ${latency_max}" | bc) -eq 1 ]]; then
            track_result "performance_latency" "PASS" "Network latency acceptable: ${latency}ms"
        else
            track_result "performance_latency" "FAIL" "Network latency too high: ${latency}ms (max: ${latency_max}ms)"
            failed=1
        fi
    else
        track_result "performance_latency" "FAIL" "Could not measure network latency"
        failed=1
    fi

    # Network throughput (if iperf3 is available)
    if command -v iperf3 >/dev/null 2>&1; then
        # Start iperf3 server in background
        iperf3 -s -1 >/dev/null 2>&1 &
        local server_pid=$!
        
        # Wait for server to start
        sleep 1
        
        # Run throughput test
        local throughput
        throughput=$(iperf3 -c localhost -t 5 2>/dev/null | grep "sender" | awk '{print $7}')
        kill ${server_pid} 2>/dev/null || true
        
        if [[ -n "${throughput}" ]]; then
            if [[ $(echo "${throughput} >= ${throughput_min}" | bc) -eq 1 ]]; then
                track_result "performance_throughput" "PASS" "Network throughput sufficient: ${throughput}Mbits/sec"
            else
                track_result "performance_throughput" "FAIL" "Network throughput too low: ${throughput}Mbits/sec (min: ${throughput_min}Mbits/sec)"
                failed=1
            fi
        else
            track_result "performance_throughput" "WARN" "Could not measure network throughput"
        fi
    else
        track_result "performance_throughput" "WARN" "iperf3 not available for throughput testing"
    fi

    return ${failed}
}

# Resource limits verification
verify_resource_limits() {
    local failed=0

    log_info "Verifying resource limits"

    # Get environment type for thresholds
    local env_type
    env_type=$(load_state "ENVIRONMENT")
    
    # Set thresholds based on environment
    local max_connections=100
    local max_fd=65535
    local max_processes=1000
    if [[ "${env_type}" =~ ^qa ]]; then
        max_connections=1000
        max_fd=131070
        max_processes=5000
    fi

    # File descriptor limits
    local fd_limit
    fd_limit=$(ulimit -n)
    if [[ ${fd_limit} -ge ${max_fd} ]]; then
        track_result "performance_fd_limit" "PASS" "File descriptor limit sufficient: ${fd_limit}"
    else
        track_result "performance_fd_limit" "FAIL" "File descriptor limit too low: ${fd_limit} (min: ${max_fd})"
        failed=1
    fi

    # Process limits
    local process_limit
    process_limit=$(ulimit -u)
    if [[ ${process_limit} -ge ${max_processes} ]]; then
        track_result "performance_process_limit" "PASS" "Process limit sufficient: ${process_limit}"
    else
        track_result "performance_process_limit" "FAIL" "Process limit too low: ${process_limit} (min: ${max_processes})"
        failed=1
    fi

    # Connection tracking
    if [[ -f /proc/sys/net/netfilter/nf_conntrack_max ]]; then
        local conntrack_max
        conntrack_max=$(cat /proc/sys/net/netfilter/nf_conntrack_max)
        if [[ ${conntrack_max} -ge ${max_connections} ]]; then
            track_result "performance_conntrack" "PASS" "Connection tracking limit sufficient: ${conntrack_max}"
        else
            track_result "performance_conntrack" "FAIL" "Connection tracking limit too low: ${conntrack_max} (min: ${max_connections})"
            failed=1
        fi
    else
        track_result "performance_conntrack" "WARN" "Connection tracking not enabled"
    fi

    return ${failed}
}

# Monitoring system verification
verify_monitoring() {
    local failed=0

    log_info "Verifying monitoring system"

    # Get environment type
    local env_type
    env_type=$(load_state "ENVIRONMENT")

    # Check monitoring configuration based on environment
    if [[ "${env_type}" =~ ^qa ]]; then
        # QA environments require monitoring
        if systemctl is-active --quiet sssonector-monitor; then
            track_result "performance_monitoring" "PASS" "Monitoring service is running"
            
            # Check metrics collection
            if [[ -d "/opt/sssonector/tools/metrics" ]] && [[ -n "$(ls -A /opt/sssonector/tools/metrics/)" ]]; then
                track_result "performance_metrics" "PASS" "Metrics are being collected"
            else
                track_result "performance_metrics" "FAIL" "No metrics found"
                failed=1
            fi
        else
            track_result "performance_monitoring" "FAIL" "Monitoring service is not running"
            failed=1
        fi
    else
        # Local development environment
        track_result "performance_monitoring" "PASS" "Monitoring optional in development"
    fi

    return ${failed}
}

# Main verification function
main() {
    local failed=0

    # Run verifications
    verify_system_performance || failed=1
    verify_network_performance || failed=1
    verify_resource_limits || failed=1
    verify_monitoring || failed=1

    return ${failed}
}

# Run main function
main "$@"
