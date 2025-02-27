#!/bin/bash

# run_all.sh
# Runs all sanity check tests for SSSonector
set -euo pipefail

# Import common utilities
source "$(dirname "${BASH_SOURCE[0]}")/../lib/common.sh"

# Default settings
RESULTS_DIR="results"
SCENARIOS_DIR="scenarios"
DEPLOY_ROOT="/opt/sssonector"

# Parse command line arguments
usage() {
    echo "Usage: $0 -s server_host -c client_host [-u user] [-r results_dir]"
    echo
    echo "Options:"
    echo "  -s    Server host address"
    echo "  -c    Client host address"
    echo "  -u    SSH user (default: qauser)"
    echo "  -r    Results directory (default: results)"
    exit 1
}

while getopts "s:c:u:r:h" opt; do
    case ${opt} in
        s) export SSSONECTOR_SERVER_HOST=${OPTARG} ;;
        c) export SSSONECTOR_CLIENT_HOST=${OPTARG} ;;
        u) export SSSONECTOR_SERVER_USER=${OPTARG}
           export SSSONECTOR_CLIENT_USER=${OPTARG} ;;
        r) RESULTS_DIR=${OPTARG} ;;
        h) usage ;;
        \?) usage ;;
    esac
done

# Set default values
SSSONECTOR_SERVER_HOST="127.0.0.1"
SSSONECTOR_CLIENT_HOST="127.0.0.1"
SSSONECTOR_SERVER_USER="sblanken"
SSSONECTOR_CLIENT_USER="sblanken"

# Export deployment root for test scripts
export SSSONECTOR_DEPLOY_ROOT="${DEPLOY_ROOT}"

# Generate certificates
generate_certs() {
    local failed=0
    local cert_dir="certs"

    log_info "Generating certificates"

    # Create certificates directory
    mkdir -p "${cert_dir}"

    # Set server CN to match server host
    export SERVER_CN="${SSSONECTOR_SERVER_HOST}"

    # Generate certificates using the script
    #if ! ../../scripts/generate-certs.sh > "${RESULTS_DIR}/cert_generation.log" 2>&1; then
    #    log_error "Certificate generation failed"
    #    failed=1
    #fi

    # Copy certificates to configs directory
    if [[ ${failed} -eq 0 ]]; then
        mkdir -p "configs/certs"
        cp "${cert_dir}"/*.{key,crt} "configs/certs/" || failed=1
        chmod 600 "configs/certs"/*.key || failed=1
        chmod 644 "configs/certs"/*.crt || failed=1
    fi

    return ${failed}
}

# Main test function
main() {
    local failed=0
    local test_results=()

    # Create results directory
    export RESULTS_DIR="${RESULTS_DIR}/$(date +%Y%m%d_%H%M%S)"
    mkdir -p "${RESULTS_DIR}"

    # Record environment
    {
        echo "Test Environment:"
        echo "Timestamp: $(date)"
        echo "Server Host: ${SSSONECTOR_SERVER_HOST}"
        echo "Client Host: ${SSSONECTOR_CLIENT_HOST}"
        echo "Server User: ${SSSONECTOR_SERVER_USER}"
        echo "Client User: ${SSSONECTOR_CLIENT_USER}"
        echo "Deploy Root: ${SSSONECTOR_DEPLOY_ROOT}"
        echo
        echo "System Information:"
        uname -a
    } > "${RESULTS_DIR}/environment.log"

    # # Generate certificates
    # if ! generate_certs; then
    #     log_error "Certificate generation failed"
    #     return 1
    # fi

    # Build packages
    log_info "Building packages"
    if ! ./build.sh > "${RESULTS_DIR}/build.log" 2>&1; then
        log_error "Build failed"
        return 1
    fi

    # # Clean up any existing deployments
    # log_info "Cleaning up existing deployments"
    # if ! ./deploy.sh -s "${SSSONECTOR_SERVER_HOST}" -c "${SSSONECTOR_CLIENT_HOST}" \
    #     -u "${SSSONECTOR_SERVER_USER}" --cleanup > "${RESULTS_DIR}/cleanup.log" 2>&1; then
    #     log_error "Cleanup failed"
    #         return 1
    # fi

    # # Deploy packages
    # log_info "Deploying packages to QA systems"
    # if ! ./deploy.sh -s "${SSSONECTOR_SERVER_HOST}" -c "${SSSONECTOR_CLIENT_HOST}" \
    #     -u "${SSSONECTOR_SERVER_USER}" > "${RESULTS_DIR}/deploy.log" 2>&1; then
    #     log_error "Deployment failed"
    #     return 1
    # fi

    # Run test scenarios
    local scenarios=(
        "fg_fg"
        "fg_bg"
        "bg_bg"
    )

    for scenario in "${scenarios[@]}"; do
        log_info "Running scenario: ${scenario}"
        if "${SCENARIOS_DIR}/${scenario}.sh" > "${RESULTS_DIR}/${scenario}.log" 2>&1; then
            test_results+=("✅ ${scenario}: PASS")
        else
            test_results+=("❌ ${scenario}: FAIL")
            failed=1
        fi
    done

    # # Clean up after tests
    # log_info "Cleaning up after tests"
    # ./deploy.sh -s "${SSSONECTOR_SERVER_HOST}" -c "${SSSONECTOR_CLIENT_HOST}" \
    #     -u "${SSSONECTOR_SERVER_USER}" --cleanup >> "${RESULTS_DIR}/cleanup.log" 2>&1 || true

    # Generate summary report
    {
        echo "Test Summary"
        echo "============"
        echo
        for result in "${test_results[@]}"; do
            echo "${result}"
        done
        echo
        echo "Detailed Results"
        echo "================"
        echo "See individual test logs in: ${RESULTS_DIR}"
    } > "${RESULTS_DIR}/summary.md"

    # Display results
    if [[ ${failed} -eq 0 ]]; then
        log_info "All tests completed successfully"
    else
        log_error "Some tests failed"
    fi

    log_info "Results available in: ${RESULTS_DIR}/summary.md"
    cat "${RESULTS_DIR}/summary.md"

    return ${failed}
}

# Run main function
main "$@"
