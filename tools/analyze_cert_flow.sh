#!/bin/bash

# analyze_cert_flow.sh
# Analyzes the certificate generation process flow
set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Global variables
RESULTS_DIR="results/cert_flow_$(date +%Y%m%d_%H%M%S)"
CERT_DIR="certs"
REQUIRED_FILES=(
    "ca.key"
    "ca.crt"
    "server.key"
    "server.crt"
    "client.key"
    "client.crt"
)
REQUIRED_PERMISSIONS=(
    "600:ca.key"
    "644:ca.crt"
    "600:server.key"
    "644:server.crt"
    "600:client.key"
    "644:client.crt"
)

# Create results directory
mkdir -p "${RESULTS_DIR}"

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1" | tee -a "${RESULTS_DIR}/cert_flow_report.txt"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" | tee -a "${RESULTS_DIR}/cert_flow_report.txt"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "${RESULTS_DIR}/cert_flow_report.txt"
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

# Check file existence and permissions
check_file() {
    local file=$1
    local expected_perms=$2
    local actual_perms

    if [[ ! -f "${file}" ]]; then
        track_result "file_${file}" "FAIL" "File not found"
        return 1
    fi

    actual_perms=$(stat -c "%a" "${file}")
    if [[ "${actual_perms}" != "${expected_perms}" ]]; then
        track_result "perms_${file}" "FAIL" "Incorrect permissions: expected ${expected_perms}, got ${actual_perms}"
        return 1
    fi

    track_result "file_${file}" "PASS" "File exists with correct permissions"
    return 0
}

# Verify CA certificate
verify_ca_cert() {
    local failed=0
    local subject
    local key_usage
    local basic_constraints

    log_info "Verifying CA certificate..."

    # Check CA key
    if ! check_file "${CERT_DIR}/ca.key" "600"; then
        failed=1
    fi

    # Check CA certificate
    if ! check_file "${CERT_DIR}/ca.crt" "644"; then
        failed=1
    else
        # Verify certificate properties
        subject=$(openssl x509 -in "${CERT_DIR}/ca.crt" -noout -subject)
        if ! echo "${subject}" | grep -q "CN = SSSonector CA"; then
            track_result "ca_subject" "FAIL" "Invalid CA subject: ${subject}"
            failed=1
        else
            track_result "ca_subject" "PASS" "Valid CA subject"
        fi

        # Check key usage
        key_usage=$(openssl x509 -in "${CERT_DIR}/ca.crt" -noout -text | grep "Key Usage" -A1)
        if ! echo "${key_usage}" | grep -q "Certificate Sign"; then
            track_result "ca_key_usage" "FAIL" "Missing required key usage"
            failed=1
        else
            track_result "ca_key_usage" "PASS" "Valid key usage"
        fi

        # Check basic constraints
        basic_constraints=$(openssl x509 -in "${CERT_DIR}/ca.crt" -noout -text | grep "Basic Constraints" -A1)
        if ! echo "${basic_constraints}" | grep -q "CA:TRUE"; then
            track_result "ca_constraints" "FAIL" "Not marked as CA"
            failed=1
        else
            track_result "ca_constraints" "PASS" "Valid basic constraints"
        fi
    fi

    return ${failed}
}

# Verify server certificate
verify_server_cert() {
    local failed=0
    local subject
    local key_usage
    local ext_key_usage
    local san

    log_info "Verifying server certificate..."

    # Check server key
    if ! check_file "${CERT_DIR}/server.key" "600"; then
        failed=1
    fi

    # Check server certificate
    if ! check_file "${CERT_DIR}/server.crt" "644"; then
        failed=1
    else
        # Verify certificate properties
        subject=$(openssl x509 -in "${CERT_DIR}/server.crt" -noout -subject)
        if ! echo "${subject}" | grep -q "CN = "; then
            track_result "server_subject" "FAIL" "Invalid server subject: ${subject}"
            failed=1
        else
            track_result "server_subject" "PASS" "Valid server subject"
        fi

        # Check key usage
        key_usage=$(openssl x509 -in "${CERT_DIR}/server.crt" -noout -text | grep "Key Usage" -A1)
        if ! echo "${key_usage}" | grep -qE "Digital Signature.*Key Encipherment"; then
            track_result "server_key_usage" "FAIL" "Missing required key usage"
            failed=1
        else
            track_result "server_key_usage" "PASS" "Valid key usage"
        fi

        # Check extended key usage
        ext_key_usage=$(openssl x509 -in "${CERT_DIR}/server.crt" -noout -text | grep "Extended Key Usage" -A1)
        if ! echo "${ext_key_usage}" | grep -q "TLS Web Server Authentication"; then
            track_result "server_ext_key_usage" "FAIL" "Missing required extended key usage"
            failed=1
        else
            track_result "server_ext_key_usage" "PASS" "Valid extended key usage"
        fi

        # Check SAN
        san=$(openssl x509 -in "${CERT_DIR}/server.crt" -noout -text | grep -A1 "Subject Alternative Name")
        if ! echo "${san}" | grep -qE "DNS:|IP Address:"; then
            track_result "server_san" "FAIL" "Missing required SAN entries"
            failed=1
        else
            track_result "server_san" "PASS" "Valid SAN entries"
        fi
    fi

    return ${failed}
}

# Verify client certificate
verify_client_cert() {
    local failed=0
    local subject
    local key_usage
    local ext_key_usage

    log_info "Verifying client certificate..."

    # Check client key
    if ! check_file "${CERT_DIR}/client.key" "600"; then
        failed=1
    fi

    # Check client certificate
    if ! check_file "${CERT_DIR}/client.crt" "644"; then
        failed=1
    else
        # Verify certificate properties
        subject=$(openssl x509 -in "${CERT_DIR}/client.crt" -noout -subject)
        if ! echo "${subject}" | grep -q "CN = "; then
            track_result "client_subject" "FAIL" "Invalid client subject: ${subject}"
            failed=1
        else
            track_result "client_subject" "PASS" "Valid client subject"
        fi

        # Check key usage
        key_usage=$(openssl x509 -in "${CERT_DIR}/client.crt" -noout -text | grep "Key Usage" -A1)
        if ! echo "${key_usage}" | grep -qE "Digital Signature.*Key Encipherment"; then
            track_result "client_key_usage" "FAIL" "Missing required key usage"
            failed=1
        else
            track_result "client_key_usage" "PASS" "Valid key usage"
        fi

        # Check extended key usage
        ext_key_usage=$(openssl x509 -in "${CERT_DIR}/client.crt" -noout -text | grep "Extended Key Usage" -A1)
        if ! echo "${ext_key_usage}" | grep -q "TLS Web Client Authentication"; then
            track_result "client_ext_key_usage" "FAIL" "Missing required extended key usage"
            failed=1
        else
            track_result "client_ext_key_usage" "PASS" "Valid extended key usage"
        fi
    fi

    return ${failed}
}

# Verify certificate chain
verify_cert_chain() {
    local failed=0

    log_info "Verifying certificate chain..."

    # Verify server certificate chain
    if openssl verify -CAfile "${CERT_DIR}/ca.crt" "${CERT_DIR}/server.crt" &>/dev/null; then
        track_result "server_chain" "PASS" "Server certificate chain verified"
    else
        track_result "server_chain" "FAIL" "Server certificate chain verification failed"
        failed=1
    fi

    # Verify client certificate chain
    if openssl verify -CAfile "${CERT_DIR}/ca.crt" "${CERT_DIR}/client.crt" &>/dev/null; then
        track_result "client_chain" "PASS" "Client certificate chain verified"
    else
        track_result "client_chain" "FAIL" "Client certificate chain verification failed"
        failed=1
    fi

    return ${failed}
}

# Generate report
generate_report() {
    local total=0
    local passed=0
    local failed=0
    local warnings=0

    {
        echo "Certificate Flow Analysis Report"
        echo "==============================="
        echo "Generated: $(date)"
        echo
        echo "Certificate Information:"
        echo "----------------------"
        echo "CA Certificate:"
        openssl x509 -in "${CERT_DIR}/ca.crt" -noout -text | grep -E "Subject:|Issuer:|Validity|Key Usage|Basic Constraints"
        echo
        echo "Server Certificate:"
        openssl x509 -in "${CERT_DIR}/server.crt" -noout -text | grep -E "Subject:|Issuer:|Validity|Key Usage|Extended Key Usage|Subject Alternative Name"
        echo
        echo "Client Certificate:"
        openssl x509 -in "${CERT_DIR}/client.crt" -noout -text | grep -E "Subject:|Issuer:|Validity|Key Usage|Extended Key Usage"
        echo
        echo "Check Results:"
        echo "-------------"

        for check in "${!CHECK_RESULTS[@]}"; do
            local status
            local message
            status=$(echo "${CHECK_RESULTS[${check}]}" | cut -d'|' -f1)
            message=$(echo "${CHECK_RESULTS[${check}]}" | cut -d'|' -f2)
            
            case ${status} in
                PASS)
                    ((passed++))
                    ((total++))
                    echo "[✓] ${check}: ${message}"
                    ;;
                FAIL)
                    ((failed++))
                    ((total++))
                    echo "[✗] ${check}: ${message}"
                    ;;
                WARN)
                    ((warnings++))
                    ((total++))
                    echo "[!] ${check}: ${message}"
                    ;;
                INFO)
                    echo "[i] ${check}: ${message}"
                    ;;
            esac
        done

        echo
        echo "Summary:"
        echo "--------"
        echo "Total checks: ${total}"
        echo "Passed: ${passed}"
        echo "Failed: ${failed}"
        echo "Warnings: ${warnings}"
        echo
        echo "Overall Status: $([ ${failed} -eq 0 ] && echo "PASS" || echo "FAIL")"
    } >> "${RESULTS_DIR}/cert_flow_report.txt"

    echo "Report generated: ${RESULTS_DIR}/cert_flow_report.txt"
    return ${failed}
}

# Main function
main() {
    local failed=0

    # Check if certificate directory exists
    if [[ ! -d "${CERT_DIR}" ]]; then
        log_error "Certificate directory not found: ${CERT_DIR}"
        return 1
    fi

    # Run verifications
    verify_ca_cert || failed=1
    verify_server_cert || failed=1
    verify_client_cert || failed=1
    verify_cert_chain || failed=1

    # Generate report
    generate_report || failed=1

    if [[ ${failed} -eq 0 ]]; then
        log_info "Certificate flow analysis completed successfully"
    else
        log_error "Certificate flow analysis failed"
    fi

    return ${failed}
}

# Run main function
main "$@"
