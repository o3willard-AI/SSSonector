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
