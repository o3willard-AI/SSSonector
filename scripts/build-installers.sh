#!/bin/bash
set -e

VERSION="1.0.0"
BUILD_DIR="build"
DIST_DIR="dist"
INSTALLER_DIR="installers"
RPM_BUILD_DIR="$BUILD_DIR/rpm"

# Ensure required tools are installed
command -v makensis >/dev/null 2>&1 || { echo "NSIS is required for Windows installer. Install with: apt-get install nsis"; exit 1; }
command -v dpkg-deb >/dev/null 2>&1 || { echo "dpkg-deb is required for Debian package. Install with: apt-get install dpkg"; exit 1; }
command -v rpmbuild >/dev/null 2>&1 || { echo "rpmbuild is required for RPM package. Install with: apt-get install rpm"; exit 1; }

# Create output directories
mkdir -p "$DIST_DIR"

# Build binaries for all platforms
echo "Building binaries..."
make clean
make dist

# Windows Installer
echo "Building Windows installer..."
mkdir -p "$BUILD_DIR/windows"
cp "$BUILD_DIR/windows-amd64/sssonector.exe" "$INSTALLER_DIR/windows/"
cp -r configs "$INSTALLER_DIR/windows/"
makensis "$INSTALLER_DIR/windows/install.nsi"
mv "$INSTALLER_DIR/windows/SSSonector-Setup.exe" "$DIST_DIR/SSSonector-${VERSION}-windows-amd64.exe"

# Linux Package (DEB)
echo "Building Debian package..."
DEBIAN_PKG="$BUILD_DIR/linux/sssonector_${VERSION}_amd64"
mkdir -p "$DEBIAN_PKG/DEBIAN"
mkdir -p "$DEBIAN_PKG/usr/local/bin"
mkdir -p "$DEBIAN_PKG/etc/sssonector"
mkdir -p "$DEBIAN_PKG/lib/systemd/system"

# Copy files for DEB
cp "$BUILD_DIR/linux-amd64/sssonector" "$DEBIAN_PKG/usr/local/bin/"
cp -r configs/* "$DEBIAN_PKG/etc/sssonector/"
cp scripts/service/systemd/sssonector.service "$DEBIAN_PKG/lib/systemd/system/"
cp "$INSTALLER_DIR/linux/DEBIAN/"* "$DEBIAN_PKG/DEBIAN/"

# Set permissions for DEB
chmod 755 "$DEBIAN_PKG/DEBIAN/postinst"
chmod 755 "$DEBIAN_PKG/DEBIAN/prerm"
chmod 755 "$DEBIAN_PKG/usr/local/bin/sssonector"

# Build DEB package
dpkg-deb --build "$DEBIAN_PKG" "$DIST_DIR/sssonector_${VERSION}_amd64.deb"

# Linux Package (RPM)
echo "Building RPM package..."
mkdir -p "$RPM_BUILD_DIR"/{BUILD,RPMS,SOURCES,SPECS,SRPMS}

# Create source tarball
tar czf "$RPM_BUILD_DIR/SOURCES/sssonector-${VERSION}.tar.gz" \
    --transform "s,^,sssonector-${VERSION}/," \
    build/linux-amd64/sssonector configs scripts LICENSE README.md

# Copy spec file
cp "$INSTALLER_DIR/linux/rpm/sssonector.spec" "$RPM_BUILD_DIR/SPECS/"

# Build RPM package
rpmbuild --define "_topdir $PWD/$RPM_BUILD_DIR" \
         --define "version $VERSION" \
         -bb "$RPM_BUILD_DIR/SPECS/sssonector.spec"

# Copy RPM packages to dist
cp "$RPM_BUILD_DIR"/RPMS/*/*.rpm "$DIST_DIR/"

# macOS Package
echo "Building macOS package..."
MACOS_PKG="$BUILD_DIR/macos/sssonector.pkg"
MACOS_PAYLOAD="$BUILD_DIR/macos/payload"

# Create package structure
mkdir -p "$MACOS_PAYLOAD/usr/local/bin"
mkdir -p "$MACOS_PAYLOAD/etc/sssonector"
mkdir -p "$MACOS_PAYLOAD/Library/LaunchDaemons"

# Copy files
cp "$BUILD_DIR/darwin-amd64/sssonector" "$MACOS_PAYLOAD/usr/local/bin/"
cp -r configs/* "$MACOS_PAYLOAD/etc/sssonector/"
cp scripts/service/launchd/com.o3willard.sssonector.plist "$MACOS_PAYLOAD/Library/LaunchDaemons/"

# Set permissions
chmod 755 "$MACOS_PAYLOAD/usr/local/bin/sssonector"
chmod 644 "$MACOS_PAYLOAD/Library/LaunchDaemons/com.o3willard.sssonector.plist"

# Create component package
pkgbuild --root "$MACOS_PAYLOAD" \
         --identifier com.o3willard.sssonector \
         --version "$VERSION" \
         --scripts "$INSTALLER_DIR/macos/scripts" \
         --install-location "/" \
         "$BUILD_DIR/macos/sssonector.pkg"

# Create product archive
productbuild --distribution "$INSTALLER_DIR/macos/distribution.xml" \
             --package-path "$BUILD_DIR/macos" \
             --resources "$INSTALLER_DIR/macos/resources" \
             "$DIST_DIR/SSSonector-${VERSION}-macos.pkg"

echo "Build complete! Installers available in $DIST_DIR:"
ls -l "$DIST_DIR"
