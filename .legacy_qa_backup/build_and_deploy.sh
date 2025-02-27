#!/bin/bash
# Script to build SSSonector from source and deploy to QA systems

# Configuration
QA_SERVER="192.168.50.210"
QA_CLIENT="192.168.50.211"
BUILD_DIR="/tmp/sssonector-build"
SOURCE_DIR="/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector"
MIN_GO_VERSION="1.21"  # Minimum Go version required
SUDO_PASS="101abn"

# Function to check command status
check_status() {
    if [ $? -eq 0 ]; then
        echo "[✓] $1"
    else
        echo "[✗] $1"
        exit 1
    fi
}

# Function to run sudo command with password
sudo_exec() {
    echo "$SUDO_PASS" | sudo -S $@
}

# Function to compare version numbers
version_ge() {
    test "$(printf '%s\n' "$@" | sort -V | head -n 1)" != "$1"
}

# Install dependencies
echo "Installing dependencies..."
sudo_exec apt-get update
sudo_exec apt-get install -y \
    libseccomp-dev \
    pkg-config \
    build-essential \
    golang-go \
    make \
    gcc \
    git
check_status "Dependencies installation"

# Ensure we have required tools
echo "Checking build requirements..."
for tool in go make gcc git pkg-config; do
    command -v $tool >/dev/null 2>&1 || { echo "[✗] $tool not found"; exit 1; }
done

# Verify Go version
GO_CURRENT=$(go version | awk '{print $3}' | sed 's/go//')
if ! version_ge "$GO_CURRENT" "$MIN_GO_VERSION"; then
    echo "[✗] Go version too old. Minimum required: $MIN_GO_VERSION, Found: $GO_CURRENT"
    exit 1
fi
echo "[✓] Go version $GO_CURRENT (>= $MIN_GO_VERSION)"

# Create clean build directory
echo "Preparing build environment..."
sudo_exec rm -rf $BUILD_DIR
sudo_exec mkdir -p $BUILD_DIR
check_status "Build directory creation"

# Copy source to build directory
echo "Copying source code..."
sudo_exec cp -r $SOURCE_DIR/* $BUILD_DIR/
check_status "Source code copy"

# Run tests before building
echo "Running tests..."
cd $BUILD_DIR
sudo_exec -E go test -v ./...
check_status "Unit tests"

# Build binaries
echo "Building SSSonector..."
sudo_exec make clean
check_status "Clean build directory"

# Build both binaries explicitly
echo "Building sssonector..."
sudo_exec go build -o build/sssonector ./cmd/daemon/
check_status "Build sssonector"

echo "Building sssonectorctl..."
sudo_exec go build -o build/sssonectorctl ./cmd/sssonectorctl/
check_status "Build sssonectorctl"

# Create distribution package
echo "Creating distribution package..."
DIST_DIR="$BUILD_DIR/dist"
sudo_exec mkdir -p $DIST_DIR/{bin,etc,systemd,config}

# Copy binaries
sudo_exec cp build/sssonector $DIST_DIR/bin/
sudo_exec cp build/sssonectorctl $DIST_DIR/bin/
sudo_exec chmod 755 $DIST_DIR/bin/*
check_status "Binary copy"

# Copy configuration
sudo_exec cp configs/sssonector.yaml $DIST_DIR/etc/
sudo_exec cp init/systemd/sssonector.service $DIST_DIR/systemd/
sudo_exec cp -r configs/* $DIST_DIR/config/
check_status "Configuration copy"

# Create version file
git rev-parse HEAD > $DIST_DIR/version.txt
date >> $DIST_DIR/version.txt
check_status "Version file creation"

# Create distribution archive
cd $BUILD_DIR
sudo_exec tar czf sssonector-dist.tar.gz dist/
check_status "Distribution package creation"

# Deploy to QA systems
echo "Deploying to QA systems..."

deploy_to_system() {
    local system=$1
    echo "Deploying to $system..."
    
    # Copy distribution package
    scp sssonector-dist.tar.gz $system:/tmp/
    check_status "Copy to $system"
    
    # Install on remote system
    ssh $system "echo '$SUDO_PASS' | sudo -S bash -s" << 'EOF'
        # Stop and cleanup existing installation
        systemctl stop sssonector 2>/dev/null
        
        # Remove existing files
        rm -rf /usr/local/bin/sssonector*
        rm -rf /etc/sssonector
        rm -f /etc/systemd/system/sssonector.service
        
        # Extract distribution
        cd /tmp
        tar xzf sssonector-dist.tar.gz
        
        # Install binaries
        cp dist/bin/* /usr/local/bin/
        chmod 755 /usr/local/bin/sssonector*
        
        # Install configuration
        mkdir -p /etc/sssonector
        cp -r dist/config/* /etc/sssonector/
        cp dist/etc/* /etc/sssonector/
        chmod 644 /etc/sssonector/*.yaml
        
        # Install systemd service
        cp dist/systemd/sssonector.service /etc/systemd/system/
        chmod 644 /etc/systemd/system/sssonector.service
        
        # Create required directories
        mkdir -p /var/lib/sssonector
        mkdir -p /var/log/sssonector
        chmod 755 /var/lib/sssonector
        chmod 755 /var/log/sssonector
        
        # Set proper ownership
        chown -R root:root /etc/sssonector
        chown -R root:root /var/lib/sssonector
        chown -R root:root /var/log/sssonector
        
        # Reload systemd
        systemctl daemon-reload
        
        # Clean up
        rm -rf /tmp/sssonector-dist.tar.gz /tmp/dist
        
        # Print version
        cat /etc/sssonector/version.txt
EOF
    check_status "Installation on $system"
}

# Deploy to QA server
deploy_to_system $QA_SERVER
# Deploy to QA client
deploy_to_system $QA_CLIENT

echo "Build and deployment complete!"
echo "Version information:"
cat $DIST_DIR/version.txt
