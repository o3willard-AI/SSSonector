#!/bin/bash
set -e

# Get the version from git tag or use a default
VERSION=$(git describe --tags 2>/dev/null || echo "dev")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS="-X main.Version=$VERSION -X main.Commit=$COMMIT -X main.BuildTime=$BUILD_TIME -s -w"

# Supported platforms
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "linux/arm/v7"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/386"
)

# Commands to build
COMMANDS=(
    "cmd/admin"
    "cmd/benchmark"
)

# Clean build directory
rm -rf build/bin/*

# Build for each platform
for platform in "${PLATFORMS[@]}"; do
    # Split platform into OS and architecture
    IFS="/" read -r -a platform_split <<< "$platform"
    GOOS="${platform_split[0]}"
    GOARCH="${platform_split[1]}"
    ARM=""
    
    # Handle ARM version if specified
    if [ "${#platform_split[@]}" -eq 3 ]; then
        ARM="${platform_split[2]}"
        GOARM="${ARM#v}"  # Remove 'v' prefix
    fi

    # Build each command
    for cmd in "${COMMANDS[@]}"; do
        # Get the binary name from the command path
        binary_name=$(basename "$cmd")
        
        # Add extension for Windows
        if [ "$GOOS" = "windows" ]; then
            binary_name="$binary_name.exe"
        fi

        # Create platform-specific directory
        platform_dir="build/bin/$GOOS/$GOARCH"
        if [ -n "$ARM" ]; then
            platform_dir="$platform_dir/$ARM"
        fi
        mkdir -p "$platform_dir"

        # Set environment variables for cross-compilation
        export GOOS="$GOOS"
        export GOARCH="$GOARCH"
        if [ -n "$GOARM" ]; then
            export GOARM="$GOARM"
        fi

        echo "Building $binary_name for $GOOS/$GOARCH${ARM:+/$ARM}..."
        
        # Build the binary
        go build -ldflags "$LDFLAGS" -o "$platform_dir/$binary_name" "./$cmd"

        # Create checksum
        (cd "$platform_dir" && sha256sum "$binary_name" > "$binary_name.sha256")
    done
done

# Create version file
cat > build/bin/version.json << EOF
{
    "version": "$VERSION",
    "commit": "$COMMIT",
    "buildTime": "$BUILD_TIME"
}
EOF

echo "Build complete! Binaries are in build/bin/"
