#!/bin/bash

# common.sh
# Common test utilities and functions for SSSonector testing

set -euo pipefail

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Directory setup
export SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
export PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
export RESULTS_DIR="${PROJECT_ROOT}/results/$(date +%Y%m%d_%H%M%S)"

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

# Process management
start_process() {
    local binary=$1
    local config=$2
    local log_file=$3
    local mode=${4:-}

    if [[ -n "${mode}" ]]; then
        ${binary} -config "${config}" --mode "${mode}" > "${log_file}" 2>&1 &
    else
        ${binary} -config "${config}" > "${log_file}" 2>&1 &
    fi
    echo $!
}

wait_for_process() {
    local pid=$1
    local timeout=${2:-30}
    local count=0

    while kill -0 ${pid} 2>/dev/null; do
        sleep 1
        ((count++))
        if [[ ${count} -ge ${timeout} ]]; then
            return 1
        fi
    done
}

# Network utilities
wait_for_interface() {
    local interface=$1
    local timeout=${2:-30}
    local count=0

    while ! ip link show "${interface}" &>/dev/null; do
        sleep 1
        ((count++))
        if [[ ${count} -ge ${timeout} ]]; then
            return 1
        fi
    done
}

check_connectivity() {
    local target=$1
    local count=${2:-3}
    local timeout=${3:-2}

    ping -c ${count} -W ${timeout} "${target}" &>/dev/null
}

# Certificate utilities
verify_certificate() {
    local cert=$1
    local key=$2
    local ca=${3:-}

    # Verify certificate matches private key
    if ! openssl x509 -noout -modulus -in "${cert}" | \
        openssl md5 | \
        grep -q "$(openssl rsa -noout -modulus -in "${key}" | openssl md5)"; then
        log_error "Certificate and private key do not match"
        return 1
    fi

    # Verify certificate against CA if provided
    if [[ -n "${ca}" ]]; then
        if ! openssl verify -CAfile "${ca}" "${cert}" &>/dev/null; then
            log_error "Certificate verification against CA failed"
            return 1
        fi
    fi

    return 0
}

# Result handling
init_results() {
    local scenario=$1
    mkdir -p "${RESULTS_DIR}/${scenario}"
    echo "${RESULTS_DIR}/${scenario}"
}

save_result() {
    local scenario=$1
    local status=$2
    local message=$3
    local results_dir="${RESULTS_DIR}/${scenario}"

    mkdir -p "${results_dir}"
    echo "${status}" > "${results_dir}/status"
    echo "${message}" > "${results_dir}/message"

    if [[ "${status}" == "PASS" ]]; then
        log_info "${message}"
    else
        log_error "${message}"
    fi
}

# Environment checks
check_requirements() {
    local failed=0

    # Check TUN device
    if ! test -c /dev/net/tun; then
        log_error "TUN device not available"
        failed=1
    fi

    # Check binary exists
    if [[ ! -x "${PROJECT_ROOT}/bin/sssonector" ]]; then
        log_error "SSSonector binary not found or not executable"
        failed=1
    fi

    return ${failed}
}

# Show script usage
show_usage() {
    echo "Usage: $0 [-s server_ip] [-c client_ip] [-h]"
    echo
    echo "Options:"
    echo "  -s    Server IP address"
    echo "  -c    Client IP address"
    echo "  -h    Show this help message"
}
