#!/bin/bash

# Main test runner for SSSonector
set -euo pipefail

# Import utilities
source "$(dirname "${BASH_SOURCE[0]}")/lib/common.sh"

# Parse command line arguments
SERVER_IP=""
CLIENT_IP=""

while getopts "s:c:h" opt; do
    case ${opt} in
        s)
            SERVER_IP="${OPTARG}"
            ;;
        c)
            CLIENT_IP="${OPTARG}"
            ;;
        h)
            show_usage
            exit 0
            ;;
        \?)
            show_usage
            exit 1
            ;;
    esac
done

# Validate required arguments
if [[ -z "${SERVER_IP}" || -z "${CLIENT_IP}" ]]; then
    log_error "Server IP and Client IP are required"
    show_usage
    exit 1
fi

# Export variables for test scripts
export SSSONECTOR_SERVER_IP="${SERVER_IP}"
export SSSONECTOR_CLIENT_IP="${CLIENT_IP}"

# Initialize results
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Run test scenarios in order
run_test_scenario() {
    local scenario=$1
    local scenario_path="${PROJECT_ROOT}/scenarios/${scenario}/run.sh"
    
    if [[ ! -x "${scenario_path}" ]]; then
        log_error "Test scenario not found or not executable: ${scenario}"
        return 1
    fi

    log_info "Running test scenario: ${scenario}"
    if "${scenario_path}"; then
        ((PASSED_TESTS++))
        return 0
    else
        ((FAILED_TESTS++))
        return 1
    fi
}

# Main test execution
main() {
    local failed=0

    # Create results directory
    mkdir -p "${RESULTS_DIR}"

    # Record test environment
    {
        echo "Test Environment:"
        echo "Timestamp: $(date)"
        echo "Server IP: ${SERVER_IP}"
        echo "Client IP: ${CLIENT_IP}"
        echo "Working Directory: ${PWD}"
        echo
        echo "System Information:"
        uname -a
        echo
        echo "Network Configuration:"
        ip addr show
    } > "${RESULTS_DIR}/environment.log"

    # Run scenarios in order
    local scenarios=(
        "01_cert_generation"
        "02_basic_connectivity"
        "03_performance"
        "04_security"
    )

    for scenario in "${scenarios[@]}"; do
        ((TOTAL_TESTS++))
        if ! run_test_scenario "${scenario}"; then
            failed=1
            # Continue with other tests even if one fails
        fi
    done

    # Generate summary report
    {
        echo "Test Summary"
        echo "============"
        echo "Total Tests: ${TOTAL_TESTS}"
        echo "Passed: ${PASSED_TESTS}"
        echo "Failed: ${FAILED_TESTS}"
        echo
        echo "Detailed Results"
        echo "================"
        
        for scenario in "${scenarios[@]}"; do
            local status_file="${RESULTS_DIR}/${scenario}/status"
            local message_file="${RESULTS_DIR}/${scenario}/message"
            
            echo "${scenario}:"
            if [[ -f "${status_file}" && -f "${message_file}" ]]; then
                echo "Status: $(cat "${status_file}")"
                echo "Message: $(cat "${message_file}")"
            else
                echo "Status: UNKNOWN"
                echo "Message: Results not found"
            fi
            echo
        done
    } > "${RESULTS_DIR}/summary.md"

    # Display results
    if [[ ${failed} -eq 0 ]]; then
        log_info "All tests completed successfully"
        log_info "Results available in: ${RESULTS_DIR}/summary.md"
    else
        log_error "Some tests failed. Check ${RESULTS_DIR}/summary.md for details"
    fi

    return ${failed}
}

# Run main function
main "$@"
