#!/bin/bash
set -e

# Build configuration
VERSION="1.0.0"
BINARY_NAME="sssonector"
BINARY_CONTROL="sssonectorctl"
PLATFORMS=("linux/amd64" "linux/arm64" "darwin/amd64" "darwin/arm64" "windows/amd64")
OUTPUT_DIR="build"

# Ensure output directory exists
mkdir -p "$OUTPUT_DIR"/{packages,docker}

# Build for each platform
for platform in "${PLATFORMS[@]}"; do
    # Split platform into OS and architecture
    IFS="/" read -r -a parts <<< "$platform"
    GOOS="${parts[0]}"
    GOARCH="${parts[1]}"
    
    # Set output binary names based on platform
    if [ "$GOOS" = "windows" ]; then
        daemon_name="${BINARY_NAME}.exe"
        control_name="${BINARY_CONTROL}.exe"
    else
        daemon_name="${BINARY_NAME}"
        control_name="${BINARY_CONTROL}"
    fi

    echo "Building for $GOOS/$GOARCH..."
    
    # Create platform-specific directory
    platform_dir="$OUTPUT_DIR/$GOOS/$GOARCH"
    mkdir -p "$platform_dir"
    
    # Build binaries
    GOOS=$GOOS GOARCH=$GOARCH go build -v \
        -ldflags "-X main.Version=$VERSION" \
        -o "$platform_dir/$daemon_name" \
        cmd/daemon/main.go

    GOOS=$GOOS GOARCH=$GOARCH go build -v \
        -ldflags "-X main.Version=$VERSION" \
        -o "$platform_dir/$control_name" \
        cmd/sssonectorctl/main.go
    
    # Copy platform-specific service files
    case $GOOS in
        "linux")
            cp init/systemd/sssonector.service "$platform_dir/"
            cp scripts/install.sh "$platform_dir/"
            ;;
        "darwin")
            cp init/launchd/com.o3willard.sssonector.plist "$platform_dir/"
            cp scripts/install_macos.sh "$platform_dir/"
            ;;
        "windows")
            cp scripts/install.ps1 "$platform_dir/"
            ;;
    esac

    # Create archive
    echo "Creating archive for $GOOS/$GOARCH..."
    cd "$OUTPUT_DIR"
    if [ "$GOOS" = "windows" ]; then
        zip -j "packages/${BINARY_NAME}-${VERSION}-$GOOS-$GOARCH.zip" "$GOOS/$GOARCH/"*
    else
        tar czf "packages/${BINARY_NAME}-${VERSION}-$GOOS-$GOARCH.tar.gz" -C "$GOOS/$GOARCH" .
    fi
    cd ..
done

# Build security policies if on Linux
if [ "$(uname)" = "Linux" ]; then
    echo "Building security policies..."
    if [ -d "security/selinux" ]; then
        cd security/selinux && ./build_policy.sh
        cp *.pp "$OUTPUT_DIR/linux/amd64/"
        cp *.pp "$OUTPUT_DIR/linux/arm64/"
    fi
    if [ -d "security/apparmor" ]; then
        cp security/apparmor/usr.local.bin.sssonector "$OUTPUT_DIR/linux/amd64/"
        cp security/apparmor/usr.local.bin.sssonector "$OUTPUT_DIR/linux/arm64/"
    fi
fi

# Copy documentation
echo "Copying documentation..."
mkdir -p "$OUTPUT_DIR/docs"
cp -r docs/deployment "$OUTPUT_DIR/docs/"
cp -r docs/config "$OUTPUT_DIR/docs/"
cp -r docs/implementation "$OUTPUT_DIR/docs/"
cp README.md LICENSE CHANGELOG.md "$OUTPUT_DIR/"

# Copy Kubernetes manifests
if [ -d "deploy/kubernetes" ]; then
    echo "Copying Kubernetes manifests..."
    cp -r deploy/kubernetes "$OUTPUT_DIR/"
fi

# Create release bundle
echo "Creating release bundle..."
cd "$OUTPUT_DIR"
tar czf "${BINARY_NAME}-${VERSION}-release.tar.gz" *
cd ..

echo "Build complete! Artifacts available in $OUTPUT_DIR"
