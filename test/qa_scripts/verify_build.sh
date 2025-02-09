#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

echo "Running SSSonector build verification..."

# Check Go version
GO_VERSION=$(go version | awk '{print $3}')
if [[ "$GO_VERSION" < "go1.21" ]]; then
    echo -e "${RED}Error: Go version must be 1.21 or higher${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Go version check passed${NC}"

# Run unit tests
echo "Running unit tests..."
go test -v ./internal/... -count=1
echo -e "${GREEN}✓ Unit tests passed${NC}"

# Run integration tests
echo "Running integration tests..."
go test -v ./test/integration/... -count=1
echo -e "${GREEN}✓ Integration tests passed${NC}"

# Verify binary installation
if [[ ! -f "/usr/local/bin/sssonector" ]]; then
    echo -e "${RED}Error: sssonector binary not found${NC}"
    exit 1
fi
echo -e "${GREEN}✓ Binary installation verified${NC}"

# Check file permissions
echo "Checking file permissions..."

# Configuration directory
if [[ ! -d "/etc/sssonector" ]]; then
    echo -e "${RED}Error: Configuration directory not found${NC}"
    exit 1
fi

if [[ $(stat -c %a /etc/sssonector) != "755" ]]; then
    echo -e "${RED}Error: Invalid configuration directory permissions${NC}"
    exit 1
fi

# Certificate files
CERT_DIR="/etc/sssonector/certs"
if [[ ! -d "$CERT_DIR" ]]; then
    echo -e "${RED}Error: Certificate directory not found${NC}"
    exit 1
fi

for cert in server.crt client.crt; do
    if [[ ! -f "$CERT_DIR/$cert" ]]; then
        echo -e "${RED}Error: $cert not found${NC}"
        exit 1
    fi
    if [[ $(stat -c %a "$CERT_DIR/$cert") != "644" ]]; then
        echo -e "${RED}Error: Invalid $cert permissions${NC}"
        exit 1
    fi
done

for key in server.key client.key; do
    if [[ ! -f "$CERT_DIR/$key" ]]; then
        echo -e "${RED}Error: $key not found${NC}"
        exit 1
    fi
    if [[ $(stat -c %a "$CERT_DIR/$key") != "600" ]]; then
        echo -e "${RED}Error: Invalid $key permissions${NC}"
        exit 1
    fi
done

echo -e "${GREEN}✓ File permissions verified${NC}"

# Check system dependencies
echo "Checking system dependencies..."

DEPS=("ip" "iptables" "ping" "systemctl")
for dep in "${DEPS[@]}"; do
    if ! command -v "$dep" &> /dev/null; then
        echo -e "${RED}Error: Required dependency '$dep' not found${NC}"
        exit 1
    fi
done

echo -e "${GREEN}✓ System dependencies verified${NC}"

# Verify service installation
echo "Checking service installation..."

if ! systemctl list-unit-files | grep -q sssonector.service; then
    echo -e "${RED}Error: sssonector service not installed${NC}"
    exit 1
fi

if ! systemctl is-enabled sssonector &> /dev/null; then
    echo -e "${RED}Error: sssonector service not enabled${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Service installation verified${NC}"

# Verify configuration
echo "Verifying configuration..."

CONFIG_FILE="/etc/sssonector/config.yaml"
if [[ ! -f "$CONFIG_FILE" ]]; then
    echo -e "${RED}Error: Configuration file not found${NC}"
    exit 1
fi

# Basic config validation
if ! sssonector validate-config "$CONFIG_FILE"; then
    echo -e "${RED}Error: Invalid configuration${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Configuration verified${NC}"

# Run configuration tests
echo "Running configuration tests..."

# Create test config directory
TEST_CONFIG_DIR=$(mktemp -d)
trap 'rm -rf "$TEST_CONFIG_DIR"' EXIT

# Copy current config for testing
cp "$CONFIG_FILE" "$TEST_CONFIG_DIR/config.yaml"

# Run config-specific tests
go test -v ./test/integration/config_test.go -count=1
echo -e "${GREEN}✓ Configuration tests passed${NC}"

echo -e "${GREEN}All verification checks passed successfully!${NC}"
