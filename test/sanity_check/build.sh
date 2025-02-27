#!/bin/bash

# build.sh
# Builds SSSonector packages for Linux platforms (Dev system only)
set -euo pipefail

# Import common utilities
source "$(dirname "${BASH_SOURCE[0]}")/../lib/common.sh"

# Build settings
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
)
VERSION=$(git describe --tags --always)
BUILD_DIR="build"
PACKAGE_DIR="packages"

# Build function
build_platform() {
    local platform=$1
    local os=${platform%/*}
    local arch=${platform#*/}
    local binary_name="sssonector"
    local package_name="sssonector_${VERSION}_${arch}"

    log_info "Building for ${platform} on Dev system"

    # Create package structure
    local pkg_root="${BUILD_DIR}/${arch}/${package_name}"
    mkdir -p "${pkg_root}/"{bin,etc/sssonector}

    # Copy binary from project root
    log_info "Copying binary for ${arch}"
    if [[ -f "../../bin/${binary_name}" ]]; then
        cp "../../bin/${binary_name}" "${pkg_root}/bin/"
        chmod 755 "${pkg_root}/bin/${binary_name}"
    else
        log_error "Binary not found: ../../bin/${binary_name}"
        return 1
    fi

    # Copy config files
    log_info "Copying config files"
    if [[ -d "configs" ]]; then
        cp configs/*.yaml "${pkg_root}/etc/sssonector/"
    else
        log_error "Config directory not found: configs"
        return 1
    fi

    # Create package
    mkdir -p "${PACKAGE_DIR}"
    tar -czf "${PACKAGE_DIR}/${package_name}.tar.gz" -C "${BUILD_DIR}/${arch}" "${package_name}"

    # Generate checksums
    (cd "${PACKAGE_DIR}" && sha256sum "${package_name}.tar.gz" > "${package_name}.tar.gz.sha256")

    log_info "Package created: ${PACKAGE_DIR}/${package_name}.tar.gz"
}

# Main build process
main() {
    local failed=0

    # Check if running on Dev system
    if [[ "$(hostname)" != "ai-ws" ]]; then
        log_error "This script must be run on the Dev system (ai-ws)"
        return 1
    fi

    # Clean previous builds
    rm -rf "${BUILD_DIR}" "${PACKAGE_DIR}"
    mkdir -p "${BUILD_DIR}" "${PACKAGE_DIR}"

    # Build for each platform
    for platform in "${PLATFORMS[@]}"; do
        if ! build_platform "${platform}"; then
            log_error "Build failed for ${platform}"
            failed=1
        fi
    done

    if [[ ${failed} -eq 0 ]]; then
        log_info "All builds completed successfully"
        log_info "Packages available in: ${PACKAGE_DIR}"
        # List created packages
        ls -l "${PACKAGE_DIR}"
    else
        log_error "Some builds failed"
        return 1
    fi
}

# Run main function
main "$@"
