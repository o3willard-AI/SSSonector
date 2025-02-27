#!/bin/bash

# test_framework.sh
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

# Test environment prerequisites
validate_test_prerequisites() {
    local base_dir=$1
    local failed=0

    log_info "Validating test prerequisites..."

    # Check required directories
    local required_dirs=(
        "test/known_good_working"
        "test/qa_scripts"
        "test/startup_logging/unit"
        "test/startup_logging/helpers"
    )

    for dir in "${required_dirs[@]}"; do
        if [[ ! -d "${base_dir}/${dir}" ]]; then
            log_error "Required test directory missing: ${dir}"
            failed=1
        fi
    done

    # Check required test files
    local required_files=(
        "test/known_good_working/server_config.yaml"
        "test/known_good_working/client_config.yaml"
        "test/qa_scripts/core_functionality_test.sh"
        "test/qa_scripts/cleanup_resources.sh"
        "test/qa_scripts/tunnel_control.sh"
    )

    for file in "${required_files[@]}"; do
        if [[ ! -f "${base_dir}/${file}" ]]; then
            log_error "Required test file missing: ${file}"
            failed=1
        elif [[ ! -x "${base_dir}/${file}" && "${file}" == *.sh ]]; then
            log_error "Test script not executable: ${file}"
            failed=1
        fi
    done

    # Check test dependencies
    local test_deps=(
        "go"
        "golangci-lint"
        "gotestsum"
    )

    for dep in "${test_deps[@]}"; do
        if ! command -v "${dep}" &>/dev/null; then
            log_error "Required test dependency missing: ${dep}"
            failed=1
        fi
    done

    return ${failed}
}

# Test environment setup
prepare_test_environment() {
    local base_dir=$1
    local failed=0

    log_info "Preparing test environment..."

    # Create test output directory
    local test_output="${base_dir}/test/output"
    mkdir -p "${test_output}"

    # Setup test certificates
    if [[ ! -d "${base_dir}/test/certs" ]]; then
        mkdir -p "${base_dir}/test/certs"
        if ! "${base_dir}/generate_certs.sh" -d "${base_dir}/test/certs"; then
            log_error "Failed to generate test certificates"
            failed=1
        fi
    fi

    # Setup test configurations
    local test_config="${base_dir}/test_config.yaml"
    if [[ ! -f "${test_config}" ]]; then
        cp "${base_dir}/test/known_good_working/server_config.yaml" "${test_config}"
    fi

    # Setup test database
    local test_db="${base_dir}/test/data/test.db"
    if [[ ! -f "${test_db}" ]]; then
        mkdir -p "${base_dir}/test/data"
        if ! sqlite3 "${test_db}" ".databases"; then
            log_error "Failed to create test database"
            failed=1
        fi
    fi

    # Setup test network namespace
    if ! ip netns list | grep -q "sssonector_test"; then
        if ! sudo ip netns add sssonector_test; then
            log_error "Failed to create test network namespace"
            failed=1
        fi
    fi

    return ${failed}
}

# Test environment validation
verify_test_environment() {
    local base_dir=$1
    local failed=0

    log_info "Verifying test environment..."

    # Verify test certificates
    if ! openssl verify -CAfile "${base_dir}/test/certs/ca.crt" "${base_dir}/test/certs/server.crt"; then
        log_error "Invalid test certificates"
        failed=1
    fi

    # Verify test configuration
    if ! "${base_dir}/tools/config_validator/config_validator.sh" -c "${base_dir}/test_config.yaml"; then
        log_error "Invalid test configuration"
        failed=1
    fi

    # Verify test database
    if ! sqlite3 "${base_dir}/test/data/test.db" "PRAGMA integrity_check"; then
        log_error "Test database integrity check failed"
        failed=1
    fi

    # Verify test network namespace
    if ! ip netns exec sssonector_test ip link show lo &>/dev/null; then
        log_error "Test network namespace not properly configured"
        failed=1
    fi

    return ${failed}
}

# Test result verification
verify_test_results() {
    local base_dir=$1
    local failed=0

    log_info "Verifying test results..."

    # Check test output files
    local test_output="${base_dir}/test/output"
    if [[ ! -d "${test_output}" ]]; then
        log_error "Test output directory not found"
        return 1
    fi

    # Check test logs for errors
    if find "${test_output}" -type f -name "*.log" -exec grep -l "ERROR\|FAIL\|FATAL" {} \+; then
        log_error "Found errors in test logs"
        failed=1
    fi

    # Check test coverage
    local coverage_file="${test_output}/coverage.out"
    if [[ -f "${coverage_file}" ]]; then
        local coverage
        coverage=$(go tool cover -func="${coverage_file}" | grep "total:" | awk '{print $3}' | tr -d '%')
        if (( $(echo "${coverage} < 80" | bc -l) )); then
            log_warn "Test coverage below threshold: ${coverage}%"
            failed=1
        fi
    else
        log_error "Coverage report not found"
        failed=1
    fi

    # Check test timing
    local timing_file="${test_output}/timing.json"
    if [[ -f "${timing_file}" ]]; then
        local slow_tests
        slow_tests=$(jq '.[] | select(.elapsed > 5)' "${timing_file}" | wc -l)
        if [[ ${slow_tests} -gt 0 ]]; then
            log_warn "Found ${slow_tests} slow tests (>5s)"
        fi
    fi

    return ${failed}
}

# Test resource cleanup
cleanup_test_resources() {
    local base_dir=$1
    local failed=0

    log_info "Cleaning up test resources..."

    # Stop test processes
    if ! "${base_dir}/test/qa_scripts/cleanup_resources.sh"; then
        log_error "Failed to clean up test processes"
        failed=1
    fi

    # Remove test network namespace
    if ip netns list | grep -q "sssonector_test"; then
        if ! sudo ip netns delete sssonector_test; then
            log_error "Failed to remove test network namespace"
            failed=1
        fi
    fi

    # Clean up test files
    local test_files=(
        "${base_dir}/test/output"
        "${base_dir}/test/data/test.db"
        "${base_dir}/test_config.yaml"
    )

    for file in "${test_files[@]}"; do
        if [[ -e "${file}" ]]; then
            rm -rf "${file}"
        fi
    done

    return ${failed}
}

# Main test framework validation function
validate_test_framework() {
    local base_dir=$1
    local failed=0

    # Validate test prerequisites
    validate_test_prerequisites "${base_dir}" || failed=1

    # Prepare test environment
    prepare_test_environment "${base_dir}" || failed=1

    # Verify test environment
    verify_test_environment "${base_dir}" || failed=1

    # Verify test results
    verify_test_results "${base_dir}" || failed=1

    # Clean up test resources
    cleanup_test_resources "${base_dir}" || failed=1

    return ${failed}
}

# If script is run directly, show usage
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "This script is meant to be sourced by qa_validator.sh"
    exit 1
fi
