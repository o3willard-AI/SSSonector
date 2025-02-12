#!/bin/bash
set -e

# Version from git tag or default
VERSION=$(git describe --tags 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build directory
BUILD_DIR="build"
mkdir -p $BUILD_DIR

# Supported platforms
PLATFORMS=(
    "linux/amd64"
    "linux/arm64"
    "linux/arm"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
    "windows/386"
)

# Build flags
BUILD_FLAGS=(
    "-trimpath"
    "-ldflags=-s -w -X main.version=$VERSION -X main.buildTime=$BUILD_TIME -X main.commitHash=$COMMIT_HASH"
)

# Build types
TYPES=(
    "admin"
    "benchmark"
)

echo "Building SSSonector $VERSION ($COMMIT_HASH) at $BUILD_TIME"

for PLATFORM in "${PLATFORMS[@]}"; do
    # Split platform into OS and ARCH
    IFS="/" read -r -a platform_split <<< "$PLATFORM"
    GOOS="${platform_split[0]}"
    GOARCH="${platform_split[1]}"
    
    # Set output suffix based on OS
    if [ "$GOOS" = "windows" ]; then
        SUFFIX=".exe"
    else
        SUFFIX=""
    fi

    for TYPE in "${TYPES[@]}"; do
        echo "Building $TYPE for $GOOS/$GOARCH..."
        
        OUTPUT="$BUILD_DIR/ssonector-$TYPE-$VERSION-$GOOS-$GOARCH$SUFFIX"
        
        # Set environment variables for cross-compilation
        env GOOS=$GOOS GOARCH=$GOARCH \
            go build "${BUILD_FLAGS[@]}" \
            -o "$OUTPUT" \
            "./cmd/$TYPE"

        # Create SHA256 checksum
        if command -v sha256sum >/dev/null 2>&1; then
            sha256sum "$OUTPUT" > "$OUTPUT.sha256"
        elif command -v shasum >/dev/null 2>&1; then
            shasum -a 256 "$OUTPUT" > "$OUTPUT.sha256"
        elif command -v certutil >/dev/null 2>&1; then
            certutil -hashfile "$OUTPUT" SHA256 > "$OUTPUT.sha256"
        else
            echo "No checksum tool found (sha256sum, shasum, or certutil)"
            exit 1
        fi
        
        echo "Built $OUTPUT"
    done
done

# Create version file
echo "$VERSION" > "$BUILD_DIR/version.txt"
echo "$BUILD_TIME" >> "$BUILD_DIR/version.txt"
echo "$COMMIT_HASH" >> "$BUILD_DIR/version.txt"

# Create archive for each platform
for PLATFORM in "${PLATFORMS[@]}"; do
    # Split platform into OS and ARCH
    IFS="/" read -r -a platform_split <<< "$PLATFORM"
    GOOS="${platform_split[0]}"
    GOARCH="${platform_split[1]}"
    
    ARCHIVE_NAME="$BUILD_DIR/ssonector-$VERSION-$GOOS-$GOARCH"
    
    if [ "$GOOS" = "windows" ]; then
        zip -j "$ARCHIVE_NAME.zip" "$BUILD_DIR"/*"$GOOS-$GOARCH.exe" "$BUILD_DIR/version.txt"
        zip -j "$ARCHIVE_NAME.zip" "$BUILD_DIR"/*"$GOOS-$GOARCH.exe.sha256"
    else
        tar -czf "$ARCHIVE_NAME.tar.gz" -C "$BUILD_DIR" \
            $(cd "$BUILD_DIR" && ls *"$GOOS-$GOARCH" version.txt *"$GOOS-$GOARCH.sha256")
    fi
    
    echo "Created archive $ARCHIVE_NAME"
done

echo "Build complete! Artifacts are in the $BUILD_DIR directory"
