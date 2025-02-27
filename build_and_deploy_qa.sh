#!/bin/bash

# Build script for SSSonector specifically for Ubuntu (linux/amd64)
# and deploy to QA systems for testing

set -euo pipefail

# Version from git tag or default
VERSION=$(git describe --tags 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT_HASH=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Build directory
BUILD_DIR="./bin"
mkdir -p "$BUILD_DIR"

# QA systems
QA_SYSTEMS=(
    "qa1"
    "qa2"
)

# Build flags
BUILD_FLAGS=(
    "-ldflags"
    "-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME -X main.CommitHash=$COMMIT_HASH"
)

echo "Building SSSonector version $VERSION ($COMMIT_HASH) at $BUILD_TIME"
echo "Building for Ubuntu (linux/amd64)..."

# Set output binary name
OUTPUT="$BUILD_DIR/sssonector-${VERSION}-linux-amd64"
LOCAL_OUTPUT="sssonector"

# Build for Ubuntu (linux/amd64)
echo "Building for linux/amd64..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build \
    "${BUILD_FLAGS[@]}" \
    -o "$OUTPUT" \
    ./cmd/tunnel

# Create checksum
if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$OUTPUT" > "$OUTPUT.sha256"
elif command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$OUTPUT" > "$OUTPUT.sha256"
fi

# Copy to local directory for easy access
cp "$OUTPUT" "$LOCAL_OUTPUT"
chmod +x "$LOCAL_OUTPUT"

echo "Build complete! Binary is at $OUTPUT and $LOCAL_OUTPUT"

# Deploy to QA systems
echo -e "\nDeploying to QA systems..."

for QA_SYSTEM in "${QA_SYSTEMS[@]}"; do
    echo "Deploying to $QA_SYSTEM..."
    
    # Check if QA system is reachable
    if ping -c 1 -W 1 "$QA_SYSTEM" > /dev/null 2>&1; then
        # Copy binary to QA system
        scp "$LOCAL_OUTPUT" "$QA_SYSTEM:/tmp/sssonector"
        ssh "$QA_SYSTEM" "sudo mv /tmp/sssonector /usr/local/bin/sssonector && sudo chmod +x /usr/local/bin/sssonector"
        echo "Deployed to $QA_SYSTEM successfully"
    else
        echo "Warning: $QA_SYSTEM is not reachable, skipping deployment"
    fi
done

# Run QA tests
echo -e "\nRunning QA tests..."

# Use the deploy_sssonector.sh script to deploy SSSonector to the QA environment
echo "Deploying SSSonector to QA environment..."
./tools/verification/deploy_sssonector.sh

# Add forwarding rules to QA systems
echo "Adding forwarding rules to QA systems..."
./tools/verification/add_forwarding_rules.sh

# Run the sanity checks
echo "Running sanity checks..."
./tools/verification/run_sanity_checks.sh

# Run the QA tests
echo "Running QA tests..."
./tools/verification/run_qa_tests.sh

echo "Build and deployment complete!"
