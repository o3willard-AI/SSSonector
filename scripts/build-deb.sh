#!/bin/bash
set -e

# Get version from Makefile or default to 1.0.0
VERSION=$(grep "VERSION :=" Makefile | cut -d' ' -f3 || echo "1.0.0")
ARCH="amd64"
PACKAGE_NAME="sssonector"
PACKAGE_ROOT="build/deb/${PACKAGE_NAME}"

echo "Building Debian package for ${PACKAGE_NAME} v${VERSION}..."

# Create package directory structure
mkdir -p "${PACKAGE_ROOT}/DEBIAN"
mkdir -p "${PACKAGE_ROOT}/usr/bin"
mkdir -p "${PACKAGE_ROOT}/etc/sssonector/certs"
mkdir -p "${PACKAGE_ROOT}/etc/systemd/system"
mkdir -p "${PACKAGE_ROOT}/var/log/sssonector"
mkdir -p "${PACKAGE_ROOT}/etc/logrotate.d"

# Copy binary
echo "Copying binary..."
cp "build/bin/${PACKAGE_NAME}-linux-${ARCH}" "${PACKAGE_ROOT}/usr/bin/${PACKAGE_NAME}"
chmod 755 "${PACKAGE_ROOT}/usr/bin/${PACKAGE_NAME}"

# Copy configuration files
echo "Copying configuration files..."
cp configs/server.yaml "${PACKAGE_ROOT}/etc/sssonector/config.yaml.example"
cp configs/client.yaml "${PACKAGE_ROOT}/etc/sssonector/client.yaml.example"
chmod 644 "${PACKAGE_ROOT}/etc/sssonector/"*.yaml.example

# Copy systemd service
echo "Copying systemd service..."
cp scripts/service/systemd/sssonector.service "${PACKAGE_ROOT}/etc/systemd/system/"
chmod 644 "${PACKAGE_ROOT}/etc/systemd/system/sssonector.service"

# Create logrotate configuration
cat > "${PACKAGE_ROOT}/etc/logrotate.d/sssonector" << EOF
/var/log/sssonector/*.log {
    daily
    rotate 7
    compress
    delaycompress
    missingok
    notifempty
    create 0640 sssonector sssonector
    sharedscripts
    postrotate
        systemctl reload sssonector.service >/dev/null 2>&1 || true
    endscript
}
EOF
chmod 644 "${PACKAGE_ROOT}/etc/logrotate.d/sssonector"

# Copy DEBIAN control files
echo "Copying package control files..."
cp installers/linux/DEBIAN/control "${PACKAGE_ROOT}/DEBIAN/"
cp installers/linux/DEBIAN/postinst "${PACKAGE_ROOT}/DEBIAN/"
cp installers/linux/DEBIAN/prerm "${PACKAGE_ROOT}/DEBIAN/"
chmod 755 "${PACKAGE_ROOT}/DEBIAN/postinst"
chmod 755 "${PACKAGE_ROOT}/DEBIAN/prerm"

# Update version in control file
sed -i "s/Version: .*/Version: ${VERSION}/" "${PACKAGE_ROOT}/DEBIAN/control"

# Calculate installed size in KB
INSTALLED_SIZE=$(du -sk "${PACKAGE_ROOT}" | cut -f1)
sed -i "s/Installed-Size: .*/Installed-Size: ${INSTALLED_SIZE}/" "${PACKAGE_ROOT}/DEBIAN/control"

# Build package
echo "Building package..."
dpkg-deb --build "${PACKAGE_ROOT}" "build/${PACKAGE_NAME}_${VERSION}_${ARCH}.deb"

# Verify package
echo "Verifying package..."
lintian "build/${PACKAGE_NAME}_${VERSION}_${ARCH}.deb" || true

echo "Package built successfully: build/${PACKAGE_NAME}_${VERSION}_${ARCH}.deb"

# Clean up
rm -rf "${PACKAGE_ROOT}"
