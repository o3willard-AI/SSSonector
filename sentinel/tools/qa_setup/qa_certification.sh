#!/bin/bash

# qa_certification.sh
# Part of Project SENTINEL - QA Certification Tool
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

# Run comprehensive test suite
run_test_suite() {
    local base_dir=$1
    local server_ip=$2
    local client_ip=$3
    local failed=0
    log_info "Running comprehensive test suite..."

    # Create test results directory
    local results_dir="${base_dir}/test/results/$(date +%Y%m%d_%H%M%S)"
    mkdir -p "${results_dir}"

    # Run environment validation
    log_info "Running environment validation..."
    if ! "${base_dir}/tools/qa_setup/verify_qa_environment.sh" -d "${base_dir}" -s "${server_ip}" -c "${client_ip}" > "${results_dir}/environment.log" 2>&1; then
        log_error "Environment validation failed"
        failed=1
    fi

    # Run process validation
    log_info "Running process validation..."
    if ! "${base_dir}/tools/qa_validator/lib/process_monitor.sh" > "${results_dir}/process.log" 2>&1; then
        log_error "Process validation failed"
        failed=1
    fi

    # Run resource validation
    log_info "Running resource validation..."
    if ! "${base_dir}/tools/qa_validator/lib/resource_validator.sh" > "${results_dir}/resource.log" 2>&1; then
        log_error "Resource validation failed"
        failed=1
    fi

    # Run connection tests
    log_info "Running connection tests..."
    if ! "${base_dir}/test/known_good_working/CORE_TEST.sh" > "${results_dir}/connection.log" 2>&1; then
        log_error "Connection tests failed"
        failed=1
    fi

    # Run performance tests
    log_info "Running performance tests..."
    if ! "${base_dir}/test/known_good_working/performance_test.sh" > "${results_dir}/performance.log" 2>&1; then
        log_error "Performance tests failed"
        failed=1
    fi

    # Generate certification report
    generate_report "${results_dir}" "${base_dir}"

    return ${failed}
}

# Generate certification report
generate_report() {
    local results_dir=$1
    local base_dir=$2
    log_info "Generating certification report..."

    local report_file="${results_dir}/certification_report.md"
    
    # Create report header
    cat << EOF > "${report_file}"
# SSSonector QA Certification Report
Generated: $(date)

## Environment Information
- Base Directory: ${base_dir}
- Server IP: ${server_ip}
- Client IP: ${client_ip}
- Version: $(${base_dir}/bin/sssonector --version)

## Test Results Summary
EOF

    # Add test results
    local total_tests=0
    local passed_tests=0

    # Process each log file
    for log in "${results_dir}"/*.log; do
        local test_name=$(basename "${log}" .log)
        local test_status
        if grep -q "ERROR" "${log}"; then
            test_status="❌ FAILED"
        else
            test_status="✅ PASSED"
            ((passed_tests++))
        fi
        ((total_tests++))

        echo "### ${test_name^} Tests" >> "${report_file}"
        echo "Status: ${test_status}" >> "${report_file}"
        echo '```' >> "${report_file}"
        cat "${log}" >> "${report_file}"
        echo '```' >> "${report_file}"
        echo >> "${report_file}"
    done

    # Add summary
    local pass_percentage=$((passed_tests * 100 / total_tests))
    cat << EOF >> "${report_file}"
## Overall Results
- Total Tests: ${total_tests}
- Passed: ${passed_tests}
- Failed: $((total_tests - passed_tests))
- Pass Rate: ${pass_percentage}%

## Certification Status
$(if [[ ${pass_percentage} -eq 100 ]]; then echo "✅ CERTIFIED"; else echo "❌ NOT CERTIFIED"; fi)

## Recommendations
$(if [[ ${pass_percentage} -lt 100 ]]; then
    echo "- Review failed tests and address issues"
    echo "- Re-run certification after fixes"
    echo "- Consult troubleshooting guide for common solutions"
else
    echo "- Environment is fully certified"
    echo "- Continue with regular monitoring"
    echo "- Schedule next certification in 30 days"
fi)
EOF

    log_info "Report generated: ${report_file}"
}

# Main function
main() {
    local base_dir="${DEFAULT_BASE_DIR}"
    local server_ip="${DEFAULT_SERVER_IP}"
    local client_ip="${DEFAULT_CLIENT_IP}"

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

    log_info "Starting QA certification process..."
    log_info "Base directory: ${base_dir}"
    log_info "Server IP: ${server_ip}"
    log_info "Client IP: ${client_ip}"

    # Run test suite
    if run_test_suite "${base_dir}" "${server_ip}" "${client_ip}"; then
        log_info "QA certification completed successfully"
    else
        log_error "QA certification failed"
        exit 1
    fi
}

# Execute main function
main "$@"
