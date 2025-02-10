#!/bin/bash
set -e

# Ensure we're in the project root
cd "$(dirname "$0")"

# Create output directory
mkdir -p dist

# Version from git
VERSION=$(git describe --tags --always --dirty)
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

# Common build flags
LDFLAGS="-X main.version=$VERSION -X main.buildTime=$BUILD_TIME -s -w"
GOFLAGS="-tags=netgo"

# Build function
build() {
    local OS=$1
    local ARCH=$2
    local OUTPUT="dist/sssonector_${VERSION}_${OS}_${ARCH}"
    
    # Add extension for Windows
    if [ "$OS" = "windows" ]; then
        OUTPUT="${OUTPUT}.exe"
    fi
    
    echo "Building for $OS/$ARCH..."
    
    # Enable CGO only for native builds on Linux
    if [ "$OS" = "linux" ] && [ "$ARCH" = "amd64" ]; then
        echo "Building with CGO enabled..."
        CGO_ENABLED=1 GOOS=$OS GOARCH=$ARCH \
        go build \
            -ldflags="$LDFLAGS" \
            -o "$OUTPUT" \
            ./cmd/tunnel
    else
        echo "Building with CGO disabled..."
        CGO_ENABLED=0 GOOS=$OS GOARCH=$ARCH \
        go build \
            -ldflags="$LDFLAGS" \
            -tags netgo \
            -o "$OUTPUT" \
            ./cmd/tunnel
    fi
    
    echo "Built: $OUTPUT"
    
    # Create checksum
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum "$OUTPUT" > "${OUTPUT}.sha256"
    fi
}

echo "Building SSSonector version $VERSION"
echo "Build time: $BUILD_TIME"

# Clean previous builds
echo "Cleaning previous builds..."
rm -rf dist/*

# Native build first (assuming Linux amd64)
echo "Building native version..."
build linux amd64

# Cross-platform builds
echo "Building cross-platform versions..."

# Linux ARM64
build linux arm64

# Windows builds
build windows amd64
build windows arm64

# macOS builds
build darwin amd64
build darwin arm64

echo "Build complete. Binaries are in the dist directory"

# Create version file
cat > dist/version.json << EOF
{
    "version": "$VERSION",
    "build_time": "$BUILD_TIME",
    "platforms": [
        {"os": "linux", "arch": "amd64", "cgo": true},
        {"os": "linux", "arch": "arm64", "cgo": false},
        {"os": "windows", "arch": "amd64", "cgo": false},
        {"os": "windows", "arch": "arm64", "cgo": false},
        {"os": "darwin", "arch": "amd64", "cgo": false},
        {"os": "darwin", "arch": "arm64", "cgo": false}
    ]
}
EOF

# List all built files
echo -e "\nBuilt files:"
ls -lh dist/

# Verify binaries
echo -e "\nVerifying binaries..."
for file in dist/sssonector_*; do
    if [[ -f "$file" && ! "$file" =~ \.sha256$ ]]; then
        echo "Checking $file..."
        file "$file"
    fi
done

echo -e "\nBuild Notes:"
echo "- Native Linux/amd64 build includes full TUN support with CGO"
echo "- Cross-platform builds use pure Go implementation with limited features"
echo "- Windows builds require TAP driver installation"
echo "- macOS builds have basic TUN support"
