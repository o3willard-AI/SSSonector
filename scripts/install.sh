#!/bin/bash
set -e

# Check if running as root
if [ "$EUID" -ne 0 ]; then
  echo "Please run as root"
  exit 1
fi

# Create sssonector user if it doesn't exist
if ! id -u sssonector >/dev/null 2>&1; then
  echo "Creating sssonector user..."
  useradd -r -s /sbin/nologin sssonector
fi

# Build and install binary
echo "Building SSSonector..."
cd "$(dirname "$0")/.."
go build -gcflags="all=-N -l" -o sssonector ./cmd/tunnel

echo "Installing binary..."
cp sssonector /usr/local/bin/
chown root:root /usr/local/bin/sssonector
chmod 755 /usr/local/bin/sssonector

# Install startup script
echo "Installing startup script..."
cp scripts/sssonector-server /usr/local/bin/
chown root:root /usr/local/bin/sssonector-server
chmod 755 /usr/local/bin/sssonector-server

# Create required directories
echo "Creating directories..."
mkdir -p /etc/sssonector/certs
mkdir -p /var/log/sssonector

# Set permissions
echo "Setting permissions..."
chown -R root:sssonector /etc/sssonector
chmod 750 /etc/sssonector
chmod 750 /etc/sssonector/certs
chown -R sssonector:sssonector /var/log/sssonector
chmod 755 /var/log/sssonector

# Install systemd service
echo "Installing systemd service..."
cp scripts/sssonector.service /etc/systemd/system/
systemctl daemon-reload
systemctl enable sssonector.service

# Set capabilities
echo "Setting capabilities..."
setcap cap_net_admin,cap_net_raw,cap_net_bind_service+ep /usr/local/bin/sssonector

echo "Installation complete. Please:"
echo "1. Configure /etc/sssonector/config.yaml"
echo "2. Add SSL certificates to /etc/sssonector/certs"
echo "3. Start the service with: systemctl start sssonector"
