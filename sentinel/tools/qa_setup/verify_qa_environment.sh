#!/bin/bash

# verify_qa_environment.sh
# Part of Project SENTINEL - QA Environment Verification Tool
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

# Default paths
DEFAULT_BASE_DIR="/opt/sssonector"
DEFAULT_SERVER_IP="192.168.50.210"
DEFAULT_CLIENT_IP="192.168.50.211"

# Verify directory structure and permissions
verify_directories() {
    local base_dir=$1
    local failed=0
    log_info "Verifying directory structure and permissions..."

    # Required directories and their permissions
    declare -A dir_perms=(
        ["${base_dir}"]=755
        ["${base_dir}/bin"]=755
        ["${base_dir}/config"]=755
        ["${base_dir}/certs"]=755
        ["${base_dir}/log"]=755
        ["${base_dir}/state"]=755
        ["${base_dir}/tools"]=755
    )

    for dir in "${!dir_perms[@]}"; do
        if [[ ! -d "${dir}" ]]; then
            log_error "Directory not found: ${dir}"
            failed=1
            continue
        fi

        local perms
        perms=$(stat -c "%a" "${dir}")
        if [[ "${perms}" != "${dir_perms[${dir}]}" ]]; then
            log_error "Invalid permissions for ${dir}: ${perms} (expected ${dir_perms[${dir}]})"
            failed=1
        fi

        local owner
        owner=$(stat -c "%U:%G" "${dir}")
        if [[ "${owner}" != "root:root" ]]; then
            log_error "Invalid ownership for ${dir}: ${owner} (expected root:root)"
            failed=1
        fi
    done

    return ${failed}
}

# Verify system configuration
verify_system_config() {
    local failed=0
    log_info "Verifying system configuration..."

    # Check IP forwarding
    local ip_forward
    ip_forward=$(cat /proc/sys/net/ipv4/ip_forward)
    if [[ "${ip_forward}" != "1" ]]; then
        log_error "IP forwarding not enabled"
        failed=1
    fi

    # Check TUN module
    if ! test -c /dev/net/tun; then
        log_error "TUN module not loaded"
        failed=1
    fi

    # Check persistent settings
    if [[ ! -f "/etc/sysctl.d/99-sssonector.conf" ]]; then
        log_error "Missing persistent IP forwarding configuration"
        failed=1
    fi

    if [[ ! -f "/etc/modules-load.d/sssonector.conf" ]]; then
        log_error "Missing persistent TUN module configuration"
        failed=1
    fi

    return ${failed}
}

# Verify monitoring service
verify_monitoring() {
    local base_dir=$1
    local failed=0
    log_info "Verifying monitoring service..."

    # Check service file
    if [[ ! -f "/etc/systemd/system/sssonector-monitor.service" ]]; then
        log_error "Monitoring service file not found"
        failed=1
    fi

    # Check monitoring script
    if [[ ! -x "${base_dir}/tools/monitor.sh" ]]; then
        log_error "Monitoring script not found or not executable"
        failed=1
    fi

    # Check service status
    if ! systemctl is-active --quiet sssonector-monitor; then
        log_error "Monitoring service not running"
        failed=1
    fi

    # Check metrics collection
    if [[ ! -d "${base_dir}/tools/metrics" ]]; then
        log_error "Metrics directory not found"
        failed=1
    elif [[ -z "$(ls -A ${base_dir}/tools/metrics/)" ]]; then
        log_error "No metrics files found"
        failed=1
    fi

    return ${failed}
}

# Verify validation scripts
verify_validation_scripts() {
    local base_dir=$1
    local failed=0
    log_info "Verifying validation scripts..."

    # Check validation directory
    if [[ ! -d "${base_dir}/tools/validation" ]]; then
        log_error "Validation directory not found"
        failed=1
    fi

    # Check main validation script
    if [[ ! -x "${base_dir}/tools/validation/validate_environment.sh" ]]; then
        log_error "Validation script not found or not executable"
        failed=1
    fi

    # Check QA validator libraries
    local required_libs=(
        "process_monitor.sh"
        "resource_validator.sh"
    )

    for lib in "${required_libs[@]}"; do
        if [[ ! -f "${base_dir}/tools/qa_validator/lib/${lib}" ]]; then
            log_error "QA validator library not found: ${lib}"
            failed=1
        fi
    done

    return ${failed}
}

# Verify network connectivity
verify_network() {
    local server_ip=$1
    local client_ip=$2
    local failed=0
    log_info "Verifying network connectivity..."

    # Check server connectivity
    if ! ping -c 1 -W 2 "${server_ip}" &>/dev/null; then
        log_warn "Cannot reach server at ${server_ip}"
        failed=1
    fi

    # Check client connectivity
    if ! ping -c 1 -W 2 "${client_ip}" &>/dev/null; then
        log_warn "Cannot reach client at ${client_ip}"
        failed=1
    fi

    return ${failed}
}

# Main verification function
main() {
    local base_dir="${DEFAULT_BASE_DIR}"
    local server_ip="${DEFAULT_SERVER_IP}"
    local client_ip="${DEFAULT_CLIENT_IP}"
    local failed=0

    # Parse command line arguments
    while getopts "d:s:c:h" opt; do
        case ${opt} in
            d)
                base_dir="${OPTARG}"
                ;;
            s)
                server_ip="${OPTARG}"
                ;;
            c)
                client_ip="${OPTARG}"
                ;;
            h)
                echo "Usage: $0 [-d base_directory] [-s server_ip] [-c client_ip]"
                exit 0
                ;;
            \?)
                echo "Invalid option: -${OPTARG}"
                exit 1
                ;;
        esac
    done

    log_info "Starting QA environment verification..."
    log_info "Base directory: ${base_dir}"
    log_info "Server IP: ${server_ip}"
    log_info "Client IP: ${client_ip}"

    # Run verification steps
    verify_directories "${base_dir}" || failed=1
    verify_system_config || failed=1
    verify_monitoring "${base_dir}" || failed=1
    verify_validation_scripts "${base_dir}" || failed=1
    verify_network "${server_ip}" "${client_ip}" || failed=1

    if [[ ${failed} -eq 0 ]]; then
        log_info "QA environment verification completed successfully"
    else
        log_error "QA environment verification failed"
        exit 1
    fi
}

# Execute main function
main "$@"
