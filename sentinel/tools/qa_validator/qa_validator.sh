#!/bin/bash

# qa_validator.sh
# Part of Project SENTINEL - QA Environment Validation Tool
# Version: 1.0.0

set -euo pipefail

# Source validation modules
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/process_monitor.sh"
source "${SCRIPT_DIR}/lib/resource_validator.sh"
source "${SCRIPT_DIR}/lib/dependency_checker.sh"
source "${SCRIPT_DIR}/lib/security_validator.sh"
source "${SCRIPT_DIR}/lib/test_framework.sh"
source "${SCRIPT_DIR}/lib/access_validator.sh"

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

# Print usage
usage() {
    echo "Usage: $0 [-d base_directory]"
    echo
    echo "Options:"
    echo "  -d    Base directory (default: ${DEFAULT_BASE_DIR})"
    echo "  -h    Show this help message"
    exit 1
}

# Main function
main() {
    local base_dir="${DEFAULT_BASE_DIR}"

    # Parse command line arguments
    while getopts "d:h" opt; do
        case ${opt} in
            d)
                base_dir="${OPTARG}"
                ;;
            h)
                usage
                ;;
            \?)
                usage
                ;;
        esac
    done

    log_info "Starting QA environment validation..."
    log_info "Base directory: ${base_dir}"

    # Run validation checks
    local failed=0

    # Process monitoring
    log_info "Running process validation..."
    validate_processes "${base_dir}" || failed=1

    # Resource validation
    log_info "Running resource validation..."
    validate_resources "${base_dir}" || failed=1

    # Dependency validation
    log_info "Running dependency validation..."
    validate_dependencies "${base_dir}" || failed=1

    # Security validation
    log_info "Running security validation..."
    validate_security "${base_dir}" || failed=1

    # Test framework validation
    log_info "Running test framework validation..."
    validate_test_framework "${base_dir}" || failed=1

    # Access validation
    log_info "Running access validation..."
    validate_access "${base_dir}" || failed=1

    if [[ ${failed} -eq 0 ]]; then
        log_info "QA environment validation completed successfully"
    else
        log_error "QA environment validation failed"
        exit 1
    fi
}

# Execute main function
main "$@"
