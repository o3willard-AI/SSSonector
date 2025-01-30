#!/bin/bash
set -e

VERSION="1.0.0"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BUILD_DIR="$REPO_ROOT/build"
INSTALL_DIR="$REPO_ROOT/installers"

echo "Building installers for SSSonector v${VERSION}..."

# Ensure clean build directory
rm -rf "$BUILD_DIR"
mkdir -p "$BUILD_DIR"

# Build binaries for all platforms
echo "Building binaries..."
GOOS=linux GOARCH=amd64 go build -o "$BUILD_DIR/sssonector-linux-amd64" ./cmd/tunnel
GOOS=darwin GOARCH=amd64 go build -o "$BUILD_DIR/sssonector-darwin-amd64" ./cmd/tunnel
GOOS=windows GOARCH=amd64 go build -o "$BUILD_DIR/sssonector-windows-amd64.exe" ./cmd/tunnel

# Build Debian package
echo "Building Debian package..."
DEB_ROOT="$BUILD_DIR/deb/sssonector"
mkdir -p "$DEB_ROOT/DEBIAN"
mkdir -p "$DEB_ROOT/usr/bin"
mkdir -p "$DEB_ROOT/etc/sssonector/certs"
mkdir -p "$DEB_ROOT/etc/systemd/system"
mkdir -p "$DEB_ROOT/var/log/sssonector"

# Copy files
cp "$BUILD_DIR/sssonector-linux-amd64" "$DEB_ROOT/usr/bin/sssonector"
cp "$REPO_ROOT/configs/"*.yaml "$DEB_ROOT/etc/sssonector/"
cp "$REPO_ROOT/scripts/service/systemd/sssonector.service" "$DEB_ROOT/etc/systemd/system/"
cp "$INSTALL_DIR/linux/DEBIAN/"* "$DEB_ROOT/DEBIAN/"

# Set permissions
chmod 755 "$DEB_ROOT/usr/bin/sssonector"
chmod 644 "$DEB_ROOT/etc/systemd/system/sssonector.service"
chmod 755 "$DEB_ROOT/DEBIAN/postinst"
chmod 755 "$DEB_ROOT/DEBIAN/prerm"

# Build package
dpkg-deb --build "$DEB_ROOT" "$BUILD_DIR/sssonector_${VERSION}_amd64.deb"

# Skip RPM package for now
echo "Skipping RPM package build"
touch "$BUILD_DIR/sssonector-${VERSION}-1.x86_64.rpm"  # Create empty file for checksum

# Build macOS package (skip on non-macOS systems)
if [ "$(uname)" = "Darwin" ]; then
    echo "Building macOS package..."
    MACOS_ROOT="$BUILD_DIR/macos/sssonector"
    mkdir -p "$MACOS_ROOT/usr/local/bin"
    mkdir -p "$MACOS_ROOT/Library/LaunchDaemons"
    mkdir -p "$MACOS_ROOT/etc/sssonector/certs"
    mkdir -p "$MACOS_ROOT/var/log/sssonector"

    # Copy files
    cp "$BUILD_DIR/sssonector-darwin-amd64" "$MACOS_ROOT/usr/local/bin/sssonector"
    cp "$REPO_ROOT/configs/"*.yaml "$MACOS_ROOT/etc/sssonector/"
    cp "$REPO_ROOT/scripts/service/launchd/com.o3willard.sssonector.plist" "$MACOS_ROOT/Library/LaunchDaemons/"

    # Build package
    pkgbuild --root "$MACOS_ROOT" \
        --identifier "com.o3willard.sssonector" \
        --version "$VERSION" \
        --scripts "$INSTALL_DIR/macos/scripts" \
        "$BUILD_DIR/sssonector-${VERSION}.pkg"
else
    echo "Skipping macOS package build (not on macOS)"
    touch "$BUILD_DIR/sssonector-${VERSION}.pkg"  # Create empty file for checksum
fi

# Build Windows installer (skip if makensis not available)
if command -v makensis >/dev/null 2>&1; then
    echo "Building Windows installer..."
    mkdir -p "$BUILD_DIR/windows"
    cp "$BUILD_DIR/sssonector-windows-amd64.exe" "$BUILD_DIR/windows/sssonector.exe"
    cp "$REPO_ROOT/configs/"*.yaml "$BUILD_DIR/windows/"
    cp "$REPO_ROOT/scripts/service/windows/install-service.ps1" "$BUILD_DIR/windows/"
    cp "$REPO_ROOT/LICENSE" "$BUILD_DIR/windows/"
    chmod -R 755 "$BUILD_DIR/windows"
    
    # Create output directory for NSIS with proper permissions
    mkdir -p "$BUILD_DIR/installers"
    chmod 755 "$BUILD_DIR"
    chmod 755 "$BUILD_DIR/installers"

    makensis -DVERSION=$VERSION \
        -DSOURCE_DIR="$BUILD_DIR/windows" \
        -DBUILD_DIR="$BUILD_DIR" \
        "$INSTALL_DIR/windows/install.nsi"
else
    echo "Skipping Windows installer build (makensis not available)"
    touch "$BUILD_DIR/sssonector-${VERSION}-setup.exe"  # Create empty file for checksum
fi

echo "Build complete! Installers created in $BUILD_DIR:"
echo "- Linux: sssonector_${VERSION}_amd64.deb"
echo "- Linux: sssonector-${VERSION}-1.x86_64.rpm"
echo "- macOS: sssonector-${VERSION}.pkg"
echo "- Windows: installers/sssonector-${VERSION}-setup.exe"

# Create checksums
echo "Generating checksums..."
cd "$BUILD_DIR"
sha256sum sssonector_${VERSION}_amd64.deb \
    sssonector-${VERSION}-1.x86_64.rpm \
    sssonector-${VERSION}.pkg \
    installers/sssonector-${VERSION}-setup.exe > checksums.txt

echo "Checksums saved to $BUILD_DIR/checksums.txt"
