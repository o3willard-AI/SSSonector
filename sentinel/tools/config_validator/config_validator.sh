#!/bin/bash

# config_validator.sh
# Part of Project SENTINEL - Configuration Validation Tool
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

# Required fields for all configurations
REQUIRED_FIELDS=(
    "type"              # server/client
    "logging.level"     # debug/info/warn/error
    "logging.output"    # file/stdout
)

# Required fields by type
SERVER_REQUIRED_FIELDS=(
    "server.listen_address"
    "server.listen_port"
    "server.max_connections"
)

CLIENT_REQUIRED_FIELDS=(
    "client.server_address"
    "client.server_port"
    "client.retry_interval"
)

# Security policy checks
validate_security_policy() {
    local config_file=$1
    local failed=0
    log_info "Validating security policy..."

    # Check file permissions
    local file_perms
    file_perms=$(stat -c "%a" "${config_file}")
    if [[ "${file_perms}" != "644" ]]; then
        log_error "Invalid config file permissions: ${file_perms} (expected 644)"
        failed=1
    fi

    # Check certificate paths exist and have correct permissions
    local cert_paths
    cert_paths=$(yq eval '.certs.ca_cert, .certs.cert, .certs.key' "${config_file}")
    while IFS= read -r path; do
        if [[ -n "${path}" && "${path}" != "null" ]]; then
            if [[ ! -f "${path}" ]]; then
                log_error "Certificate file not found: ${path}"
                failed=1
                continue
            fi

            local cert_perms
            cert_perms=$(stat -c "%a" "${path}")
            if [[ "${path}" == *".key" && "${cert_perms}" != "600" ]]; then
                log_error "Invalid key file permissions: ${cert_perms} (expected 600)"
                failed=1
            elif [[ "${path}" == *".crt" && "${cert_perms}" != "644" ]]; then
                log_error "Invalid certificate file permissions: ${cert_perms} (expected 644)"
                failed=1
            fi
        fi
    done <<< "${cert_paths}"

    # Check for sensitive information
    if grep -q "password\|secret\|key" "${config_file}"; then
        log_warn "Config file contains potentially sensitive information"
    fi

    return ${failed}
}

# Validate required fields
validate_required_fields() {
    local config_file=$1
    local fields=("${@:2}")
    local failed=0
    
    for field in "${fields[@]}"; do
        if ! yq eval ".${field}" "${config_file}" > /dev/null 2>&1; then
            log_error "Missing required field: ${field}"
            failed=1
        fi
    done

    return ${failed}
}

# Validate field values
validate_field_values() {
    local config_file=$1
    local failed=0

    # Validate logging level
    local log_level
    log_level=$(yq eval '.logging.level' "${config_file}")
    if [[ "${log_level}" != "debug" && "${log_level}" != "info" && "${log_level}" != "warn" && "${log_level}" != "error" ]]; then
        log_error "Invalid logging level: ${log_level}"
        failed=1
    fi

    # Validate port numbers
    local port
    if port=$(yq eval '.server.listen_port // .client.server_port' "${config_file}"); then
        if ! [[ "${port}" =~ ^[0-9]+$ ]] || [ "${port}" -lt 1 ] || [ "${port}" -gt 65535 ]; then
            log_error "Invalid port number: ${port}"
            failed=1
        fi
    fi

    # Validate addresses
    local address
    if address=$(yq eval '.server.listen_address // .client.server_address' "${config_file}"); then
        if ! [[ "${address}" =~ ^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$ || "${address}" == "localhost" ]]; then
            log_warn "Potentially invalid address format: ${address}"
        fi
    fi

    return ${failed}
}

# Validate configuration type-specific requirements
validate_type_specific() {
    local config_file=$1
    local config_type
    local failed=0

    config_type=$(yq eval '.type' "${config_file}")
    
    case "${config_type}" in
        "server")
            validate_required_fields "${config_file}" "${SERVER_REQUIRED_FIELDS[@]}" || failed=1
            ;;
        "client")
            validate_required_fields "${config_file}" "${CLIENT_REQUIRED_FIELDS[@]}" || failed=1
            ;;
        *)
            log_error "Invalid configuration type: ${config_type}"
            failed=1
            ;;
    esac

    return ${failed}
}

# Print usage
usage() {
    echo "Usage: $0 [-c config_file] [-t template]"
    echo
    echo "Options:"
    echo "  -c    Configuration file to validate"
    echo "  -t    Template to validate against (optional)"
    echo "  -h    Show this help message"
    exit 1
}

# Main function
main() {
    local config_file=""
    local template=""

    # Parse command line arguments
    while getopts "c:t:h" opt; do
        case ${opt} in
            c)
                config_file="${OPTARG}"
                ;;
            t)
                template="${OPTARG}"
                ;;
            h)
                usage
                ;;
            \?)
                usage
                ;;
        esac
    done

    # Validate config file is provided
    if [[ -z "${config_file}" ]]; then
        log_error "Configuration file (-c) must be specified"
        usage
    fi

    # Check config file exists
    if [[ ! -f "${config_file}" ]]; then
        log_error "Configuration file not found: ${config_file}"
        exit 1
    fi

    log_info "Starting configuration validation..."
    log_info "Configuration file: ${config_file}"
    if [[ -n "${template}" ]]; then
        log_info "Template: ${template}"
    fi

    # Validate basic required fields
    validate_required_fields "${config_file}" "${REQUIRED_FIELDS[@]}" || exit 1

    # Validate type-specific requirements
    validate_type_specific "${config_file}" || exit 1

    # Validate field values
    validate_field_values "${config_file}" || exit 1

    # Validate security policy
    validate_security_policy "${config_file}" || exit 1

    log_info "Configuration validation completed successfully"
}

# Execute main function
main "$@"
