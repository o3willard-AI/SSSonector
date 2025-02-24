#!/bin/bash

# build_verify.sh
# Part of Project SENTINEL - Build System Verification Tool
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

# Configuration
REPO_ROOT="$(git rev-parse --show-toplevel)"
BUILD_DIR="${REPO_ROOT}/build"
EXPECTED_PLATFORMS=(
    "linux-amd64"
    "linux-arm64"
    "linux-arm"
    "darwin-amd64"
    "darwin-arm64"
    "windows-amd64"
)

# Verify build directory exists
verify_build_dir() {
    log_info "Verifying build directory..."
    if [[ ! -d "${BUILD_DIR}" ]]; then
        log_error "Build directory not found at ${BUILD_DIR}"
        return 1
    fi
    log_info "Build directory verified"
}

# Verify version information
verify_version() {
    local binary=$1
    log_info "Verifying version information for $(basename ${binary})..."
    
    if [[ ! -x "${binary}" ]]; then
        log_error "Binary not executable: ${binary}"
        return 1
    }

    local version_output
    version_output=$("${binary}" --version)
    
    # Check version format
    if ! echo "${version_output}" | grep -q "^SSSonector v[0-9]\+\.[0-9]\+\.[0-9]\+"; then
        log_error "Invalid version format in ${binary}"
        return 1
    }

    # Check build timestamp
    if ! echo "${version_output}" | grep -q "Build Time: [0-9]\{4\}-[0-9]\{2\}-[0-9]\{2\}_[0-9]\{2\}:[0-9]\{2\}:[0-9]\{2\}"; then
        log_error "Invalid build timestamp format in ${binary}"
        return 1
    }

    # Check commit hash
    if ! echo "${version_output}" | grep -q "Commit: [0-9a-f]\{7\}"; then
        log_error "Invalid commit hash format in ${binary}"
        return 1
    }

    log_info "Version information verified for $(basename ${binary})"
}

# Verify checksums
verify_checksums() {
    log_info "Verifying checksums..."
    local failed=0

    for platform in "${EXPECTED_PLATFORMS[@]}"; do
        local binary="${BUILD_DIR}/sssonector-v*-${platform}"
        local checksum_file="${binary}.sha256"
        
        if [[ ! -f "${checksum_file}" ]]; then
            log_error "Checksum file not found for ${platform}"
            failed=1
            continue
        }

        if ! (cd "${BUILD_DIR}" && sha256sum -c "$(basename ${checksum_file})"); then
            log_error "Checksum verification failed for ${platform}"
            failed=1
        fi
    done

    if [[ ${failed} -eq 1 ]]; then
        return 1
    fi
    
    log_info "All checksums verified"
}

# Verify platform binaries
verify_platforms() {
    log_info "Verifying platform binaries..."
    local failed=0

    for platform in "${EXPECTED_PLATFORMS[@]}"; do
        local binary="${BUILD_DIR}/sssonector-v*-${platform}"
        if ! ls ${binary} >/dev/null 2>&1; then
            log_error "Binary not found for platform: ${platform}"
            failed=1
            continue
        fi

        if [[ "${platform}" != windows* ]]; then
            verify_version "${binary}" || failed=1
        fi
    done

    if [[ ${failed} -eq 1 ]]; then
        return 1
    fi

    log_info "All platform binaries verified"
}

# Main verification sequence
main() {
    log_info "Starting build verification..."

    verify_build_dir || exit 1
    verify_platforms || exit 1
    verify_checksums || exit 1

    log_info "Build verification completed successfully"
}

# Execute main function
main
