#!/bin/bash

# Security test scenario
set -euo pipefail

# Import utilities
source "$(dirname "${BASH_SOURCE[0]}")/../../lib/common.sh"
source "$(dirname "${BASH_SOURCE[0]}")/../../lib/process_utils.sh"

# Initialize results directory
SCENARIO_NAME="04_security"
RESULTS_DIR=$(init_results "${SCENARIO_NAME}")

# Test certificate validation
test_cert_validation() {
    local failed=0
    local invalid_cert_dir="${RESULTS_DIR}/invalid_certs"
    mkdir -p "${invalid_cert_dir}"

    log_info "Testing certificate validation"

    # Generate invalid certificates
    openssl req -x509 -newkey rsa:4096 -keyout "${invalid_cert_dir}/invalid.key" \
        -out "${invalid_cert_dir}/invalid.crt" -days 1 -nodes \
        -subj "/CN=invalid.test" 2>/dev/null

    # Try to start server with invalid certificate
    log_info "Testing server with invalid certificate"
    if "${PROJECT_ROOT}/bin/sssonector" \
        -config "${PROJECT_ROOT}/configs/server.yaml" \
        --cert "${invalid_cert_dir}/invalid.crt" \
        --key "${invalid_cert_dir}/invalid.key" \
        > "${RESULTS_DIR}/invalid_cert_test.log" 2>&1; then
        
        log_error "Server started with invalid certificate"
        failed=1
    fi

    return ${failed}
}

# Test TLS version enforcement
test_tls_version() {
    local failed=0

    log_info "Testing TLS version enforcement"

    # Start server with normal config
    local server_pid
    server_pid=$(start_sssonector "${PROJECT_ROOT}/configs/server.yaml" "server" "${RESULTS_DIR}")
    if [[ $? -ne 0 ]]; then
        log_error "Failed to start server process"
        return 1
    fi

    # Try to connect with different TLS versions
    local versions=("1.0" "1.1" "1.2" "1.3")
    for version in "${versions[@]}"; do
        log_info "Testing TLS ${version}"
        if openssl s_client -connect localhost:443 -tls${version} \
            > "${RESULTS_DIR}/tls_${version}.log" 2>&1; then
            if [[ "${version}" < "1.2" ]]; then
                log_error "Connection succeeded with TLS ${version}"
                failed=1
            fi
        else
            if [[ "${version}" > "1.1" ]]; then
                log_error "Connection failed with TLS ${version}"
                failed=1
            fi
        fi
    done

    # Stop server
    stop_process "${server_pid}"

    return ${failed}
}

# Test unauthorized access
test_unauthorized_access() {
    local failed=0

    log_info "Testing unauthorized access prevention"

    # Start server
    local server_pid
    server_pid=$(start_sssonector "${PROJECT_ROOT}/configs/server.yaml" "server" "${RESULTS_DIR}")
    if [[ $? -ne 0 ]]; then
        log_error "Failed to start server process"
        return 1
    fi

    # Try to connect without client certificate
    if curl -k https://localhost:443 > "${RESULTS_DIR}/unauthorized.log" 2>&1; then
        log_error "Connection succeeded without client certificate"
        failed=1
    fi

    # Stop server
    stop_process "${server_pid}"

    return ${failed}
}

# Test process isolation
test_process_isolation() {
    local failed=0

    log_info "Testing process isolation"

    # Start server
    local server_pid
    server_pid=$(start_sssonector "${PROJECT_ROOT}/configs/server.yaml" "server" "${RESULTS_DIR}")
    if [[ $? -ne 0 ]]; then
        log_error "Failed to start server process"
        return 1
    fi

    # Check process capabilities
    local caps
    caps=$(getcap "/proc/${server_pid}/exe" 2>/dev/null || true)
    if [[ -n "${caps}" ]]; then
        log_error "Process has unexpected capabilities: ${caps}"
        failed=1
    fi

    # Check process user/group
    local process_user
    process_user=$(ps -o user= -p "${server_pid}")
    if [[ "${process_user}" == "root" ]]; then
        log_error "Process running as root"
        failed=1
    fi

    # Stop server
    stop_process "${server_pid}"

    return ${failed}
}

# Test network isolation
test_network_isolation() {
    local failed=0

    log_info "Testing network isolation"

    # Start server
    local server_pid
    server_pid=$(start_sssonector "${PROJECT_ROOT}/configs/server.yaml" "server" "${RESULTS_DIR}")
    if [[ $? -ne 0 ]]; then
        log_error "Failed to start server process"
        return 1
    fi

    # Check listening ports
    local ports
    ports=$(netstat -tuln | grep LISTEN)
    
    # Should only be listening on configured port
    if echo "${ports}" | grep -v ":443 "; then
        log_error "Process listening on unexpected ports"
        failed=1
    fi

    # Stop server
    stop_process "${server_pid}"

    return ${failed}
}

# Main test function
main() {
    local failed=0

    # Check requirements
    check_requirements || exit 1

    # Test certificate validation
    if ! test_cert_validation; then
        save_result "${SCENARIO_NAME}" "FAIL" "Certificate validation test failed"
        return 1
    fi

    # Test TLS version enforcement
    if ! test_tls_version; then
        save_result "${SCENARIO_NAME}" "FAIL" "TLS version enforcement test failed"
        return 1
    fi

    # Test unauthorized access
    if ! test_unauthorized_access; then
        save_result "${SCENARIO_NAME}" "FAIL" "Unauthorized access test failed"
        return 1
    fi

    # Test process isolation
    if ! test_process_isolation; then
        save_result "${SCENARIO_NAME}" "FAIL" "Process isolation test failed"
        return 1
    fi

    # Test network isolation
    if ! test_network_isolation; then
        save_result "${SCENARIO_NAME}" "FAIL" "Network isolation test failed"
        return 1
    fi

    save_result "${SCENARIO_NAME}" "PASS" "Security tests completed successfully"
    return 0
}

# Run main function
main "$@"
