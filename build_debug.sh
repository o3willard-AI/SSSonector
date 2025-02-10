#!/bin/bash
set -e

echo "Building SSSonector with debug symbols..."

# Ensure we're in the project root
cd "$(dirname "$0")"

# Set build environment
export CGO_ENABLED=1
export GODEBUG=netdns=go
export GOFLAGS="-tags=netgo"

# Clean any previous builds
rm -f sssonector

# Build with debug symbols and extra logging
go build \
  -gcflags="all=-N -l" \
  -ldflags="-X main.buildMode=debug -X main.logLevel=debug" \
  -o sssonector \
  ./cmd/tunnel

# Set capabilities for TUN interface management
sudo setcap cap_net_admin+ep sssonector

echo "Build complete. Binary: $(pwd)/sssonector"
echo "Capabilities set: $(getcap sssonector)"

# Set up system requirements
echo "Setting up system requirements..."
sudo groupadd -f sssonector
sudo useradd -r -g sssonector sssonector 2>/dev/null || true
sudo usermod -a -G sssonector $USER

# Set up TUN device
sudo mkdir -p /dev/net
sudo mknod /dev/net/tun c 10 200 2>/dev/null || true
sudo chown root:sssonector /dev/net/tun
sudo chmod 0660 /dev/net/tun

# Ensure current shell has updated group membership
exec sg sssonector -c '

# Copy config file to test location
echo "Setting up test configuration..."
sudo mkdir -p /etc/sssonector
sudo cp configs/server.yaml /etc/sssonector/config.yaml
sudo chown -R root:root /etc/sssonector
sudo chmod 644 /etc/sssonector/config.yaml

# Create log directory
sudo mkdir -p /var/log/sssonector
sudo chown -R root:root /var/log/sssonector
sudo chmod 755 /var/log/sssonector

# Verify build
echo "Build verification:"
./sssonector -config /etc/sssonector/config.yaml || true
'
