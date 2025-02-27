#!/bin/bash

# unified_verifier.sh
# Main entry point for the unified verification system
set -euo pipefail

# Source common utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/common.sh"

# Default options
DEBUG=false
MODULES=("system" "network" "security" "performance")
SKIP_MODULES=()

# Print usage information
usage() {
    cat << EOF
Usage: $0 [options]

Options:
  -d, --debug          Enable debug output
  -m, --modules        Specify modules to run (comma-separated)
  -s, --skip          Skip specified modules (comma-separated)
  -h, --help          Show this help message

Available modules:
  system              System requirements verification
  network             Network configuration verification
  security            Security settings verification
  performance         Performance baseline verification

Example:
  $0 --modules system,network
  $0 --skip performance
  $0 --debug
EOF
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -d|--debug)
            DEBUG=true
            shift
            ;;
        -m|--modules)
            IFS=',' read -ra MODULES <<< "$2"
            shift 2
            ;;
        -s|--skip)
            IFS=',' read -ra SKIP_MODULES <<< "$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            usage
            exit 1
            ;;
    esac
done

# Export debug setting
export DEBUG

# Initialize verification environment
log_info "Initializing verification environment"
init_verification || {
    log_error "Failed to initialize verification environment"
    exit 1
}

# Run verification modules
failed=0
for module in "${MODULES[@]}"; do
    # Skip if module is in skip list
    if [[ " ${SKIP_MODULES[@]} " =~ " ${module} " ]]; then
        log_info "Skipping ${module} verification"
        continue
    fi
    
    # Run module verification
    if ! verify "${module}"; then
        log_error "${module} verification failed"
        failed=1
    fi
done

# Generate verification report
log_info "Generating verification report"

# Create report header
cat << EOF > "${RESULTS_DIR}/report.md"
# SSSonector Environment Verification Report
Generated: $(date)

## Environment Information
- Type: $(load_state "ENVIRONMENT")
- Base Directory: ${BASE_DIR}
- Hostname: $(hostname)
- OS: $(uname -a)

## Verification Results
EOF

# Add module results
for module in "${MODULES[@]}"; do
    if [[ " ${SKIP_MODULES[@]} " =~ " ${module} " ]]; then
        continue
    fi
    
    echo -e "\n### ${module^} Verification" >> "${RESULTS_DIR}/report.md"
    echo '```' >> "${RESULTS_DIR}/report.md"
    
    # Extract module results from CHECK_RESULTS
    for key in "${!CHECK_RESULTS[@]}"; do
        if [[ "${key}" =~ ^${module} ]]; then
            IFS='|' read -r status message <<< "${CHECK_RESULTS[${key}]}"
            echo "${status}: ${message}" >> "${RESULTS_DIR}/report.md"
        fi
    done
    
    echo '```' >> "${RESULTS_DIR}/report.md"
done

# Add summary
total_checks=${#CHECK_RESULTS[@]}
passed_checks=0
failed_checks=0
warnings=0

for result in "${CHECK_RESULTS[@]}"; do
    IFS='|' read -r status _ <<< "${result}"
    case ${status} in
        PASS)
            ((passed_checks++))
            ;;
        FAIL)
            ((failed_checks++))
            ;;
        WARN)
            ((warnings++))
            ;;
    esac
done

cat << EOF >> "${RESULTS_DIR}/report.md"

## Summary
- Total Checks: ${total_checks}
- Passed: ${passed_checks}
- Failed: ${failed_checks}
- Warnings: ${warnings}

## Overall Status
$(if [[ ${failed_checks} -eq 0 ]]; then echo "✅ PASSED"; else echo "❌ FAILED"; fi)

## Recommendations
$(if [[ ${failed_checks} -gt 0 ]]; then
    echo "- Review and address failed checks"
    echo "- Re-run verification after fixes"
fi)
$(if [[ ${warnings} -gt 0 ]]; then
    echo "- Review warnings for potential issues"
fi)
EOF

log_info "Verification report generated: ${RESULTS_DIR}/report.md"

# Exit with status
if [[ ${failed} -eq 0 ]]; then
    log_info "Verification completed successfully"
    exit 0
else
    log_error "Verification failed"
    exit 1
fi
