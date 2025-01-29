#!/bin/bash
set -e

VERSION="1.0.0"
PACKAGE="sssonector"
ARCH="amd64"
BUILD_DIR="build/deb"
INSTALL_ROOT="$BUILD_DIR/$PACKAGE"

# Clean and create build directory
rm -rf "$BUILD_DIR"
mkdir -p "$INSTALL_ROOT"
mkdir -p "$INSTALL_ROOT/DEBIAN"
mkdir -p "$INSTALL_ROOT/usr/bin"
mkdir -p "$INSTALL_ROOT/usr/share/sssonector/config"
mkdir -p "$INSTALL_ROOT/lib/systemd/system"

# Build the binary
echo "Building binary..."
go build -o "$INSTALL_ROOT/usr/bin/sssonector" ./cmd/tunnel

# Copy configuration files
echo "Copying configuration files..."
cp configs/server.yaml "$INSTALL_ROOT/usr/share/sssonector/config/"
cp configs/client.yaml "$INSTALL_ROOT/usr/share/sssonector/config/"

# Copy DEBIAN control files
echo "Copying control files..."
cp installers/linux/DEBIAN/control "$INSTALL_ROOT/DEBIAN/"
cp installers/linux/DEBIAN/postinst "$INSTALL_ROOT/DEBIAN/"
chmod 755 "$INSTALL_ROOT/DEBIAN/postinst"

# Set permissions
chmod 755 "$INSTALL_ROOT/usr/bin/sssonector"

# Build the package
echo "Building Debian package..."
dpkg-deb --build "$INSTALL_ROOT" "$BUILD_DIR/${PACKAGE}_${VERSION}_${ARCH}.deb"

echo "Package built: $BUILD_DIR/${PACKAGE}_${VERSION}_${ARCH}.deb"
