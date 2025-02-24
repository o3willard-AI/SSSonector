#!/bin/bash

# qa_validator.sh
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

# Default paths
DEFAULT_BASE_DIR="/opt/sssonector"
REQUIRED_TOOLS=(
    "yq"           # YAML processing
    "openssl"      # Certificate operations
    "netstat"      # Network validation
    "curl"         # HTTP checks
    "sha256sum"    # Checksum validation
)

# Validate system tools
validate_tools() {
    log_info "Validating required tools..."
    local failed=0

    for tool in "${REQUIRED_TOOLS[@]}"; do
        if ! command -v "${tool}" &> /dev/null; then
            log_error "Required tool not found: ${tool}"
            failed=1
        fi
    done

    return ${failed}
}

# Validate directory structure
validate_directory_structure() {
    local base_dir=$1
    log_info "Validating directory structure..."
    local failed=0

    # Required directories
    local dirs=(
        "bin"
        "config"
        "certs"
        "log"
        "state"
        "tools"
    )

    for dir in "${dirs[@]}"; do
        if [[ ! -d "${base_dir}/${dir}" ]]; then
            log_error "Missing required directory: ${base_dir}/${dir}"
            failed=1
        fi
    done

    return ${failed}
}

# Validate permissions
validate_permissions() {
    local base_dir=$1
    log_info "Validating permissions..."
    local failed=0

    # Directory permissions
    local dir_perms
    dir_perms=$(stat -c "%a" "${base_dir}")
    if [[ "${dir_perms}" != "755" ]]; then
        log_error "Invalid base directory permissions: ${dir_perms} (expected 755)"
        failed=1
    fi

    # Config file permissions
    if [[ -d "${base_dir}/config" ]]; then
        while IFS= read -r -d '' file; do
            local file_perms
            file_perms=$(stat -c "%a" "${file}")
            if [[ "${file_perms}" != "644" ]]; then
                log_error "Invalid config file permissions: ${file_perms} (expected 644) for ${file}"
                failed=1
            fi
        done < <(find "${base_dir}/config" -type f -name "*.yaml" -print0)
    fi

    # Certificate permissions
    if [[ -d "${base_dir}/certs" ]]; then
        while IFS= read -r -d '' file; do
            local file_perms
            file_perms=$(stat -c "%a" "${file}")
            if [[ "${file}" == *".key" && "${file_perms}" != "600" ]]; then
                log_error "Invalid key file permissions: ${file_perms} (expected 600) for ${file}"
                failed=1
            elif [[ "${file}" == *".crt" && "${file_perms}" != "644" ]]; then
                log_error "Invalid certificate file permissions: ${file_perms} (expected 644) for ${file}"
                failed=1
            fi
        done < <(find "${base_dir}/certs" -type f \( -name "*.key" -o -name "*.crt" \) -print0)
    fi

    return ${failed}
}

# Validate configurations
validate_configurations() {
    local base_dir=$1
    log_info "Validating configurations..."
    local failed=0

    # Check for config validator
    if [[ ! -x "${base_dir}/tools/config_validator/config_validator.sh" ]]; then
        log_error "Config validator not found or not executable"
        failed=1
        return ${failed}
    fi

    # Validate each config file
    while IFS= read -r -d '' config_file; do
        log_info "Validating config file: ${config_file}"
        if ! "${base_dir}/tools/config_validator/config_validator.sh" -c "${config_file}"; then
            log_error "Configuration validation failed for: ${config_file}"
            failed=1
        fi
    done < <(find "${base_dir}/config" -type f -name "*.yaml" -print0)

    return ${failed}
}

# Validate certificates
validate_certificates() {
    local base_dir=$1
    log_info "Validating certificates..."
    local failed=0

    # Check CA certificate
    if [[ ! -f "${base_dir}/certs/ca.crt" ]]; then
        log_error "CA certificate not found"
        failed=1
    else
        # Validate CA certificate
        if ! openssl x509 -in "${base_dir}/certs/ca.crt" -noout -text &> /dev/null; then
            log_error "Invalid CA certificate"
            failed=1
        fi
    fi

    # Check certificate chain
    while IFS= read -r -d '' cert_file; do
        if [[ "${cert_file}" != *"ca.crt" ]]; then
            if ! openssl verify -CAfile "${base_dir}/certs/ca.crt" "${cert_file}" &> /dev/null; then
                log_error "Certificate verification failed for: ${cert_file}"
                failed=1
            fi
        fi
    done < <(find "${base_dir}/certs" -type f -name "*.crt" -print0)

    return ${failed}
}

# Validate network configuration
validate_network() {
    local base_dir=$1
    log_info "Validating network configuration..."
    local failed=0

    # Get configured ports from server and client configs
    local ports
    ports=$(yq eval '.server.listen_port // .client.server_port' "${base_dir}"/config/*.yaml | sort -u)

    # Check port availability
    for port in ${ports}; do
        if netstat -tuln | grep -q ":${port} "; then
            log_warn "Port ${port} is already in use"
        fi
    done

    return ${failed}
}

# Validate binaries
validate_binaries() {
    local base_dir=$1
    log_info "Validating binaries..."
    local failed=0

    # Check for required binaries
    if [[ ! -x "${base_dir}/bin/sssonector" ]]; then
        log_error "SSSonector binary not found or not executable"
        failed=1
        return ${failed}
    fi

    # Validate binary version
    local version_output
    if ! version_output=$("${base_dir}/bin/sssonector" --version 2>/dev/null); then
        log_error "Failed to get binary version information"
        failed=1
    else
        # Check version format
        if ! echo "${version_output}" | grep -q "^SSSonector v[0-9]\+\.[0-9]\+\.[0-9]\+"; then
            log_error "Invalid version format: ${version_output}"
            failed=1
        fi
    fi

    # Validate binary checksum
    if [[ -f "${base_dir}/bin/sssonector.sha256" ]]; then
        if ! (cd "${base_dir}/bin" && sha256sum -c sssonector.sha256); then
            log_error "Binary checksum verification failed"
            failed=1
        fi
    else
        log_warn "Binary checksum file not found"
    fi

    return ${failed}
}

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
    validate_tools || exit 1
    validate_directory_structure "${base_dir}" || exit 1
    validate_permissions "${base_dir}" || exit 1
    validate_configurations "${base_dir}" || exit 1
    validate_certificates "${base_dir}" || exit 1
    validate_network "${base_dir}" || exit 1
    validate_binaries "${base_dir}" || exit 1

    log_info "QA environment validation completed successfully"
}

# Execute main function
main "$@"
