#!/bin/bash
# DEPRECATED: This script is deprecated and will be removed in a future release.
# Please use the new verification system: tools/verification/unified_verifier.sh
# See tools/verification/README.md for details.

echo "WARNING: This script is deprecated. Please use tools/verification/unified_verifier.sh instead."
echo "See tools/verification/README.md for details."
echo "Continuing in 5 seconds..."
sleep 5

# verify_environment.sh
# Verifies the environment for SSSonector certificate operations
set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Global variables
RESULTS_DIR="results/env_check_$(date +%Y%m%d_%H%M%S)"
OPENSSL_MIN_VERSION="1.1.1"
MIN_FD_LIMIT=65535
REQUIRED_PORTS=(443)
REQUIRED_COMMANDS=(
    "openssl"
    "dig"
    "nc"
    "timedatectl"
)

# Create results directory
mkdir -p "${RESULTS_DIR}"

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1" | tee -a "${RESULTS_DIR}/environment_report.txt"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" | tee -a "${RESULTS_DIR}/environment_report.txt"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "${RESULTS_DIR}/environment_report.txt"
    return 1
}

# Result tracking
declare -A CHECK_RESULTS
track_result() {
    local check=$1
    local status=$2
    local message=$3
    CHECK_RESULTS["${check}"]="${status}|${message}"
}

# Version comparison function
version_gt() {
    test "$(printf '%s\n' "$@" | sort -V | head -n 1)" != "$1"
}

# OpenSSL verification
check_openssl() {
    local failed=0
    local version
    local config_dir
    local fips_status

    log_info "Checking OpenSSL configuration..."

    # Version check
    version=$(openssl version | cut -d' ' -f2)
    if version_gt "${OPENSSL_MIN_VERSION}" "${version}"; then
        track_result "openssl_version" "FAIL" "OpenSSL version ${version} is older than required ${OPENSSL_MIN_VERSION}"
        failed=1
    else
        track_result "openssl_version" "PASS" "OpenSSL version ${version} meets requirements"
    fi

    # Config directory
    config_dir=$(openssl version -d | cut -d'"' -f2)
    if [[ -d "${config_dir}" ]]; then
        track_result "openssl_config" "PASS" "OpenSSL config directory exists: ${config_dir}"
    else
        track_result "openssl_config" "FAIL" "OpenSSL config directory not found: ${config_dir}"
        failed=1
    fi

    # FIPS mode
    if openssl md5 /dev/null &>/dev/null; then
        fips_status="disabled"
    else
        fips_status="enabled"
    fi
    track_result "openssl_fips" "INFO" "FIPS mode is ${fips_status}"

    # Check providers
    if openssl list -providers | grep -q "OpenSSL Default Provider"; then
        track_result "openssl_providers" "PASS" "Default provider available"
    else
        track_result "openssl_providers" "FAIL" "Default provider not available"
        failed=1
    fi

    # Check legacy provider
    if [[ -f "/usr/lib/x86_64-linux-gnu/ossl-modules/legacy.so" ]]; then
        track_result "openssl_legacy" "PASS" "Legacy provider module available"
    else
        track_result "openssl_legacy" "WARN" "Legacy provider module not found"
    fi

    # Check AES-GCM support (required for TLS 1.3)
    if openssl list -cipher-algorithms | grep -q "id-aes.*-GCM"; then
        track_result "openssl_aes_gcm" "PASS" "AES-GCM support available"
    else
        track_result "openssl_aes_gcm" "FAIL" "AES-GCM support not available"
        failed=1
    fi

    return ${failed}
}

# System state verification
check_system_state() {
    local failed=0
    local time_sync
    local fd_limit
    local disk_space

    log_info "Checking system state..."

    # Time synchronization
    if timedatectl status | grep -q "System clock synchronized: yes"; then
        track_result "time_sync" "PASS" "System time is synchronized"
    else
        track_result "time_sync" "FAIL" "System time not synchronized"
        failed=1
    fi

    # File descriptor limits
    fd_limit=$(ulimit -n)
    if [[ ${fd_limit} -ge ${MIN_FD_LIMIT} ]]; then
        track_result "fd_limits" "PASS" "File descriptor limit sufficient: ${fd_limit}"
    else
        track_result "fd_limits" "FAIL" "File descriptor limit too low: ${fd_limit} (min: ${MIN_FD_LIMIT})"
        failed=1
    fi

    # Disk space
    disk_space=$(df -h . | awk 'NR==2 {print $4}')
    track_result "disk_space" "INFO" "Available disk space: ${disk_space}"

    return ${failed}
}

# Main function
main() {
    local failed=0

    log_info "Starting environment verification..."

    # Check required commands
    for cmd in "${REQUIRED_COMMANDS[@]}"; do
        if ! command -v "${cmd}" &>/dev/null; then
            log_error "Required command not found: ${cmd}"
            failed=1
        else
            log_info "Command available: ${cmd}"
        fi
    done

    # Run verification checks
    check_openssl || failed=1
    check_system_state || failed=1

    # Generate report
    log_info "Generating verification report..."
    {
        echo "SSSonector Environment Verification Report"
        echo "Generated: $(date)"
        echo
        echo "## Summary"
        echo
        if [[ ${failed} -eq 0 ]]; then
            echo "Environment verification PASSED"
        else
            echo "Environment verification FAILED"
        fi
        echo
        echo "## Details"
        echo
        for check in "${!CHECK_RESULTS[@]}"; do
            IFS='|' read -r status message <<< "${CHECK_RESULTS[${check}]}"
            echo "- ${check}: ${status} - ${message}"
        done
    } > "${RESULTS_DIR}/report.md"

    log_info "Verification report saved to: ${RESULTS_DIR}/report.md"

    if [[ ${failed} -eq 0 ]]; then
        log_info "Environment verification completed successfully"
    else
        log_error "Environment verification failed"
        exit 1
    fi
}

# Run main function
main "$@"
