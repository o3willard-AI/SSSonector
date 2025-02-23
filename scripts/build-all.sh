#!/bin/bash

# Exit on error
set -e

# Version from git tag or default
VERSION=$(git describe --tags 2>/dev/null || echo "v2.1.0")

# Create release directory
RELEASE_DIR="bin/release"
mkdir -p $RELEASE_DIR

# Build function
build() {
    local OS=$1
    local ARCH=$2
    local BINARY_NAME="sssonector"
    local OUTPUT="${RELEASE_DIR}/${BINARY_NAME}_${VERSION}_${OS}_${ARCH}"
    
    # Add .exe extension for Windows
    if [ "$OS" = "windows" ]; then
        OUTPUT="${OUTPUT}.exe"
    fi
    
    echo "Building for ${OS}/${ARCH}..."
    # Build main binary
    GOOS=$OS GOARCH=$ARCH CGO_ENABLED=0 go build \
        -ldflags="-X main.Version=${VERSION}" \
        -o "${OUTPUT}" \
        ./cmd/tunnel

    # Build benchmark binary
    BENCHMARK_OUTPUT="${RELEASE_DIR}/benchmark_${VERSION}_${OS}_${ARCH}"
    if [ "$OS" = "windows" ]; then
        BENCHMARK_OUTPUT="${BENCHMARK_OUTPUT}.exe"
    fi
    GOOS=$OS GOARCH=$ARCH CGO_ENABLED=0 go build \
        -ldflags="-X main.Version=${VERSION}" \
        -o "${BENCHMARK_OUTPUT}" \
        ./cmd/benchmark
    
    # Create checksums
    if [ "$OS" = "windows" ]; then
        sha256sum "${OUTPUT}" > "${OUTPUT}.sha256"
        sha256sum "${BENCHMARK_OUTPUT}" > "${BENCHMARK_OUTPUT}.sha256"
    else
        shasum -a 256 "${OUTPUT}" > "${OUTPUT}.sha256"
        shasum -a 256 "${BENCHMARK_OUTPUT}" > "${BENCHMARK_OUTPUT}.sha256"
    fi
    
    echo "Done building ${OS}/${ARCH}"
}

# Clean release directory
rm -rf $RELEASE_DIR/*

# Build all variants
build linux amd64
build linux arm64
build windows amd64
build darwin amd64
build darwin arm64

echo "All builds completed successfully!"
echo "Binaries and checksums are in ${RELEASE_DIR}"
ls -l $RELEASE_DIR
