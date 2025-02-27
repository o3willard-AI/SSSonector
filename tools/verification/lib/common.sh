#!/bin/bash

# common.sh
# Common utilities for the unified verification system
set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Global variables
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BASE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
RESULTS_DIR="${BASE_DIR}/verification/reports/$(date +%Y%m%d_%H%M%S)"
STATE_FILE="${BASE_DIR}/verification/.state"

# Required commands
REQUIRED_COMMANDS=(
    "openssl"
    "ip"
    "sysctl"
    "dig"
    "nc"
    "ss"
    "lsof"
)

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1" | tee -a "${RESULTS_DIR}/verification.log"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" | tee -a "${RESULTS_DIR}/verification.log"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "${RESULTS_DIR}/verification.log"
    return 1
}

log_debug() {
    if [[ "${DEBUG:-false}" == "true" ]]; then
        echo -e "${BLUE}[DEBUG]${NC} $1" | tee -a "${RESULTS_DIR}/verification.log"
    fi
}

# Result tracking
declare -A CHECK_RESULTS
track_result() {
    local check=$1
    local status=$2
    local message=$3
    CHECK_RESULTS["${check}"]="${status}|${message}"
    
    case ${status} in
        PASS)
            log_info "${check}: ${message}"
            ;;
        WARN)
            log_warn "${check}: ${message}"
            ;;
        FAIL)
            log_error "${check}: ${message}"
            ;;
    esac
}

# Environment detection
detect_environment() {
    # Check if running in QA environment
    if [[ -d "/opt/sssonector" ]]; then
        # Check if server or client
        if [[ -f "/opt/sssonector/config/server.yaml" ]]; then
            echo "qa_server"
        else
            echo "qa_client"
        fi
    else
        echo "local"
    fi
}

# Version comparison
version_gt() {
    test "$(printf '%s\n' "$@" | sort -V | head -n 1)" != "$1"
}

# Check command availability
check_command() {
    local cmd=$1
    if ! command -v "${cmd}" &>/dev/null; then
        track_result "command_${cmd}" "FAIL" "Required command not found: ${cmd}"
        return 1
    fi
    track_result "command_${cmd}" "PASS" "Command available: ${cmd}"
    return 0
}

# Check system requirements
check_system_requirement() {
    local name=$1
    local current=$2
    local required=$3
    local unit=${4:-""}
    
    if [[ ${current} -lt ${required} ]]; then
        track_result "requirement_${name}" "FAIL" "${name} too low: ${current}${unit} (required: ${required}${unit})"
        return 1
    fi
    track_result "requirement_${name}" "PASS" "${name} sufficient: ${current}${unit}"
    return 0
}

# Save state
save_state() {
    local key=$1
    local value=$2
    echo "${key}=${value}" >> "${STATE_FILE}"
}

# Load state
load_state() {
    local key=$1
    if [[ -f "${STATE_FILE}" ]]; then
        grep "^${key}=" "${STATE_FILE}" | cut -d'=' -f2
    fi
}

# Cleanup function
cleanup() {
    log_info "Cleaning up verification environment"
    rm -f "${STATE_FILE}"
}

# Initialize verification environment
init_verification() {
    # Create results directory
    mkdir -p "${RESULTS_DIR}"
    
    # Initialize state file
    : > "${STATE_FILE}"
    
    # Register cleanup handler
    trap cleanup EXIT
    
    # Check required commands
    local failed=0
    for cmd in "${REQUIRED_COMMANDS[@]}"; do
        check_command "${cmd}" || failed=1
    done
    
    # Detect and save environment
    local env
    env=$(detect_environment)
    save_state "ENVIRONMENT" "${env}"
    log_info "Detected environment: ${env}"
    
    return ${failed}
}

# Main verification function
verify() {
    local module=$1
    shift
    local failed=0
    
    log_info "Running ${module} verification"
    
    if [[ -f "${SCRIPT_DIR}/../modules/${module}/verify.sh" ]]; then
        if ! "${SCRIPT_DIR}/../modules/${module}/verify.sh" "$@"; then
            log_error "${module} verification failed"
            failed=1
        fi
    else
        log_error "Module not found: ${module}"
        failed=1
    fi
    
    return ${failed}
}

# Export functions
export -f log_info log_warn log_error log_debug
export -f track_result version_gt check_command
export -f check_system_requirement save_state load_state
export -f verify
