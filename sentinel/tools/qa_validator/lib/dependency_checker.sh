#!/bin/bash

# dependency_checker.sh
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

# Version comparison function
version_compare() {
    local version1=$1
    local version2=$2
    local operator=$3

    # Convert versions to arrays
    IFS='.' read -ra v1_parts <<< "${version1}"
    IFS='.' read -ra v2_parts <<< "${version2}"

    # Pad arrays with zeros
    while [[ ${#v1_parts[@]} -lt 3 ]]; do
        v1_parts+=("0")
    done
    while [[ ${#v2_parts[@]} -lt 3 ]]; do
        v2_parts+=("0")
    done

    # Compare version parts
    for i in {0..2}; do
        if [[ ${v1_parts[i]} -lt ${v2_parts[i]} ]]; then
            [[ ${operator} == "<" || ${operator} == "<=" ]] && return 0
            return 1
        elif [[ ${v1_parts[i]} -gt ${v2_parts[i]} ]]; then
            [[ ${operator} == ">" || ${operator} == ">=" ]] && return 0
            return 1
        fi
    done

    # Versions are equal
    [[ ${operator} == "=" || ${operator} == "<=" || ${operator} == ">=" ]] && return 0
    return 1
}

# System library version validation
validate_system_libraries() {
    local failed=0

    log_info "Validating system libraries..."

    # Required system libraries and their minimum versions
    declare -A required_libs=(
        ["libc.so.6"]="2.31"
        ["libssl.so.1.1"]="1.1.1"
        ["libcrypto.so.1.1"]="1.1.1"
    )

    for lib in "${!required_libs[@]}"; do
        local min_version=${required_libs[${lib}]}
        
        # Check if library exists
        if ! ldconfig -p | grep -q "${lib}"; then
            log_error "Required library not found: ${lib}"
            failed=1
            continue
        fi

        # Get library version
        local lib_path
        lib_path=$(ldconfig -p | grep "${lib}" | awk 'NR==1{print $NF}')
        local version
        version=$(strings "${lib_path}" | grep -E "^[0-9]+\.[0-9]+\.[0-9]+$" | head -n1)

        if [[ -z "${version}" ]]; then
            log_warn "Could not determine version for library: ${lib}"
            continue
        fi

        # Compare versions
        if ! version_compare "${version}" "${min_version}" ">="; then
            log_error "Library ${lib} version too old. Required: ${min_version}, Found: ${version}"
            failed=1
        fi
    done

    return ${failed}
}

# Tool version validation
validate_tool_versions() {
    local failed=0

    log_info "Validating tool versions..."

    # Required tools and their version constraints
    declare -A required_tools=(
        ["openssl"]="1.1.1"
        ["bash"]="5.0"
        ["yq"]="4.0"
        ["curl"]="7.0"
    )

    for tool in "${!required_tools[@]}"; do
        local min_version=${required_tools[${tool}]}
        
        # Check if tool exists
        if ! command -v "${tool}" &>/dev/null; then
            log_error "Required tool not found: ${tool}"
            failed=1
            continue
        fi

        # Get tool version
        local version
        case ${tool} in
            "openssl")
                version=$(openssl version | awk '{print $2}' | cut -c1-5)
                ;;
            "bash")
                version=${BASH_VERSION%%.*}
                ;;
            "yq")
                version=$(yq --version 2>&1 | awk '{print $NF}')
                ;;
            "curl")
                version=$(curl --version | awk 'NR==1{print $2}')
                ;;
        esac

        # Compare versions
        if ! version_compare "${version}" "${min_version}" ">="; then
            log_error "Tool ${tool} version too old. Required: ${min_version}, Found: ${version}"
            failed=1
        fi
    done

    return ${failed}
}

# Dependency chain validation
validate_dependency_chain() {
    local base_dir=$1
    local failed=0

    log_info "Validating dependency chain..."

    # Check binary dependencies
    local binary="${base_dir}/bin/sssonector"
    if [[ ! -x "${binary}" ]]; then
        log_error "Binary not found or not executable: ${binary}"
        return 1
    fi

    # Get binary dependencies
    local deps
    deps=$(ldd "${binary}")

    # Check each dependency
    while IFS= read -r line; do
        if [[ ${line} == *"not found"* ]]; then
            local missing_lib
            missing_lib=$(echo "${line}" | awk '{print $1}')
            log_error "Missing dependency: ${missing_lib}"
            failed=1
        elif [[ ${line} == *"=>"* ]]; then
            local lib_path
            lib_path=$(echo "${line}" | awk '{print $3}')
            if [[ ! -f "${lib_path}" ]]; then
                local lib_name
                lib_name=$(echo "${line}" | awk '{print $1}')
                log_error "Dependency not found: ${lib_name} (${lib_path})"
                failed=1
            fi
        fi
    done <<< "${deps}"

    return ${failed}
}

# Version conflict detection
detect_version_conflicts() {
    local base_dir=$1
    local failed=0

    log_info "Detecting version conflicts..."

    # Create temporary directory for dependency analysis
    local temp_dir
    temp_dir=$(mktemp -d)
    trap 'rm -rf "${temp_dir}"' EXIT

    # Get all shared libraries used by the binary
    local binary="${base_dir}/bin/sssonector"
    if [[ ! -x "${binary}" ]]; then
        log_error "Binary not found or not executable: ${binary}"
        return 1
    fi

    # Create dependency map
    declare -A lib_versions
    while IFS= read -r line; do
        if [[ ${line} == *"=>"* && ${line} != *"not found"* ]]; then
            local lib_path
            lib_path=$(echo "${line}" | awk '{print $3}')
            local lib_name
            lib_name=$(basename "${lib_path}")
            local version
            version=$(strings "${lib_path}" | grep -E "^[0-9]+\.[0-9]+\.[0-9]+$" | head -n1)

            if [[ -n "${version}" ]]; then
                if [[ -n "${lib_versions[${lib_name}]}" && "${lib_versions[${lib_name}]}" != "${version}" ]]; then
                    log_error "Version conflict detected for ${lib_name}: Found versions ${lib_versions[${lib_name}]} and ${version}"
                    failed=1
                fi
                lib_versions[${lib_name}]=${version}
            fi
        fi
    done < <(ldd "${binary}")

    return ${failed}
}

# Main dependency validation function
validate_dependencies() {
    local base_dir=$1
    local failed=0

    # Validate system libraries
    validate_system_libraries || failed=1

    # Validate tool versions
    validate_tool_versions || failed=1

    # Validate dependency chain
    validate_dependency_chain "${base_dir}" || failed=1

    # Detect version conflicts
    detect_version_conflicts "${base_dir}" || failed=1

    return ${failed}
}

# If script is run directly, show usage
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    echo "This script is meant to be sourced by qa_validator.sh"
    exit 1
fi
