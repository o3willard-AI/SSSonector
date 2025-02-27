#!/bin/bash

# Certificate generation test scenario
set -euo pipefail

# Import utilities
source "$(dirname "${BASH_SOURCE[0]}")/../../lib/common.sh"
source "$(dirname "${BASH_SOURCE[0]}")/../../lib/process_utils.sh"

# Initialize results directory
SCENARIO_NAME="01_cert_generation"
RESULTS_DIR=$(init_results "${SCENARIO_NAME}")
CERT_DIR="${RESULTS_DIR}/certs"

# Create certificates directory
mkdir -p "${CERT_DIR}"

# Generate certificates
generate_certs() {
    local failed=0
    
    log_info "Generating certificates in ${CERT_DIR}"

    # Start SSSonector in certificate generation mode
    if ! "${PROJECT_ROOT}/bin/sssonector" \
        --generate-certs \
        --cert-dir "${CERT_DIR}" \
        --server-ip "${SSSONECTOR_SERVER_IP}" \
        > "${RESULTS_DIR}/cert_generation.log" 2>&1; then
        
        log_error "Certificate generation failed"
        return 1
    fi

    # Verify generated files exist
    local required_files=(
        "ca.key"
        "ca.crt"
        "server.key"
        "server.crt"
        "client.key"
        "client.crt"
    )

    for file in "${required_files[@]}"; do
        if [[ ! -f "${CERT_DIR}/${file}" ]]; then
            log_error "Missing required certificate file: ${file}"
            failed=1
        fi
    done

    return ${failed}
}

# Verify certificates
verify_certs() {
    local failed=0

    log_info "Verifying certificates"

    # Verify server certificate
    if ! verify_certificate \
        "${CERT_DIR}/server.crt" \
        "${CERT_DIR}/server.key" \
        "${CERT_DIR}/ca.crt"; then
        failed=1
    fi

    # Verify client certificate
    if ! verify_certificate \
        "${CERT_DIR}/client.crt" \
        "${CERT_DIR}/client.key" \
        "${CERT_DIR}/ca.crt"; then
        failed=1
    fi

    # Verify certificate properties
    local server_cn
    server_cn=$(openssl x509 -noout -subject -in "${CERT_DIR}/server.crt" | grep -o "CN = .*" | cut -d= -f2- | tr -d ' ')
    if [[ "${server_cn}" != "${SSSONECTOR_SERVER_IP}" ]]; then
        log_error "Server certificate CN mismatch. Expected: ${SSSONECTOR_SERVER_IP}, Got: ${server_cn}"
        failed=1
    fi

    return ${failed}
}

# Copy certificates to config directory
install_certs() {
    local failed=0

    log_info "Installing certificates to config directory"

    # Create config certificates directory
    local config_cert_dir="${PROJECT_ROOT}/configs/certs"
    mkdir -p "${config_cert_dir}"

    # Copy certificates
    if ! cp "${CERT_DIR}"/*.{key,crt} "${config_cert_dir}/"; then
        log_error "Failed to copy certificates to config directory"
        failed=1
    fi

    # Set correct permissions
    chmod 600 "${config_cert_dir}"/*.key
    chmod 644 "${config_cert_dir}"/*.crt

    return ${failed}
}

# Main test function
main() {
    local failed=0

    # Check requirements
    check_requirements || exit 1

    # Generate certificates
    if ! generate_certs; then
        save_result "${SCENARIO_NAME}" "FAIL" "Certificate generation failed"
        return 1
    fi

    # Verify certificates
    if ! verify_certs; then
        save_result "${SCENARIO_NAME}" "FAIL" "Certificate verification failed"
        return 1
    fi

    # Install certificates
    if ! install_certs; then
        save_result "${SCENARIO_NAME}" "FAIL" "Certificate installation failed"
        return 1
    fi

    save_result "${SCENARIO_NAME}" "PASS" "Certificate generation completed successfully"
    return 0
}

# Run main function
main "$@"
