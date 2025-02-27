#!/bin/bash

# Build script for SSSonector
# Builds binaries for all supported platforms

set -euo pipefail

# Version from git tag or default
VERSION=$(git describe --tags 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build directory
BUILD_DIR="build"
mkdir -p "$BUILD_DIR"

# Supported platforms
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "linux/arm"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

# Build flags
BUILD_FLAGS=(
    "-ldflags"
    "-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME -X main.CommitHash=$COMMIT_HASH"
)

echo "Building SSSonector version $VERSION ($COMMIT_HASH) at $BUILD_TIME"
echo "Building for ${#PLATFORMS[@]} platforms..."

# Build for each platform
for PLATFORM in "${PLATFORMS[@]}"; do
    OS="${PLATFORM%/*}"
    ARCH="${PLATFORM#*/}"
    
    # Set output binary name based on OS
    if [ "$OS" = "windows" ]; then
        OUTPUT="$BUILD_DIR/sssonector-${VERSION}-${OS}-${ARCH}.exe"
    else
        OUTPUT="$BUILD_DIR/sssonector-${VERSION}-${OS}-${ARCH}"
    fi
    
    echo "Building for $OS/$ARCH..."
    GOOS=$OS GOARCH=$ARCH CGO_ENABLED=0 go build \
        "${BUILD_FLAGS[@]}" \
        -o "$OUTPUT" \
        ./cmd/tunnel
    
    # Create checksum
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$OUTPUT" > "$OUTPUT.sha256"
    elif command -v shasum >/dev/null 2>&1; then
        shasum -a 256 "$OUTPUT" > "$OUTPUT.sha256"
    fi
done

echo "Build complete! Binaries are in the $BUILD_DIR directory"

# List built files
echo -e "\nBuilt files:"
ls -lh "$BUILD_DIR"
