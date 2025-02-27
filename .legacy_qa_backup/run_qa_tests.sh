#!/bin/bash
# Script to run QA tests for SSSonector

# Configuration
QA_SERVER="192.168.50.210"
QA_CLIENT="192.168.50.211"
SUDO_PASS="101abn"
SOURCE_DIR="/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector"
BUILD_DIR="/tmp/sssonector-build"

echo "Starting QA test run at $(date)"

# Function to check command status
check_status() {
    if [ $? -eq 0 ]; then
        echo "[✓] $1"
    else
        echo "[✗] $1"
        exit 1
    fi
}

# Step 1: Build from source
echo "Step 1: Building from source ==="

# Install dependencies
echo "Installing dependencies..."
sudo apt-get update
sudo apt-get install -y \
    libseccomp-dev \
    pkg-config \
    build-essential \
    golang-go \
    make \
    gcc \
    git \
    openssl
check_status "Dependencies installation"

# Verify Go version
echo "Checking build requirements..."
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
if [[ "$GO_VERSION" < "1.21" ]]; then
    echo "[✗] Go version too old. Found: $GO_VERSION"
    exit 1
fi
echo "[✓] Go version $GO_VERSION (>= 1.21)"

# Prepare build environment
echo "Preparing build environment..."
sudo rm -rf $BUILD_DIR
sudo mkdir -p $BUILD_DIR
sudo chown -R $USER:$USER $BUILD_DIR
check_status "Build directory creation"

# Copy source code
echo "Copying source code..."
cp -r $SOURCE_DIR/* $BUILD_DIR/
check_status "Source code copy"

# Create test certificates and keys
echo "Generating test certificates..."
mkdir -p $BUILD_DIR/test/certs
cd $BUILD_DIR/test/certs

# Generate CA key and certificate
openssl genrsa -out ca.key 4096
openssl req -new -x509 -days 365 -key ca.key -out ca.crt -subj "/CN=SSSonector Test CA"

# Generate server key and CSR
openssl genrsa -out server.key 4096
openssl req -new -key server.key -out server.csr -subj "/CN=sssonector-server"

# Generate client key and CSR
openssl genrsa -out client.key 4096
openssl req -new -key client.key -out client.csr -subj "/CN=sssonector-client"

# Sign server certificate
openssl x509 -req -days 365 -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out server.crt

# Sign client certificate
openssl x509 -req -days 365 -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial -out client.crt

check_status "Certificate generation"

# Create test configuration
echo "Creating test configuration..."
mkdir -p $BUILD_DIR/configs
cat > $BUILD_DIR/configs/sssonector.yaml << EOF
mode: server
logging:
  level: "info"
  file: "/var/log/sssonector/sssonector.log"
auth:
  certificate: "/etc/sssonector/certs/server.crt"
  key: "/etc/sssonector/certs/server.key"
  ca_certificate: "/etc/sssonector/certs/ca.crt"
network:
  interface: "tun0"
  mtu: 1500
  address: "10.0.0.1/24"
tunnel:
  port: 8443
  keepalive: "30s"
  compression: true
security:
  memory_protections:
    enabled: false
  namespace:
    enabled: false
  capabilities:
    enabled: false
  seccomp:
    enabled: false
snmp:
  enabled: false
EOF

cat > $BUILD_DIR/configs/sssonector_client.yaml << EOF
mode: client
logging:
  level: "info"
  file: "/var/log/sssonector/sssonector.log"
auth:
  certificate: "/etc/sssonector/certs/client.crt"
  key: "/etc/sssonector/certs/client.key"
  ca_certificate: "/etc/sssonector/certs/ca.crt"
network:
  interface: "tun0"
  mtu: 1500
  address: "10.0.0.2/24"
tunnel:
  server: "$QA_SERVER"
  port: 8443
  keepalive: "30s"
  compression: true
security:
  memory_protections:
    enabled: false
  namespace:
    enabled: false
  capabilities:
    enabled: false
  seccomp:
    enabled: false
snmp:
  enabled: false
EOF
check_status "Configuration creation"

# Run tests
echo "Running tests..."
cd $BUILD_DIR

# Create test config directory
mkdir -p $BUILD_DIR/test/config
cp configs/sssonector.yaml test/config/
cp configs/sssonector_client.yaml test/config/
cp -r test/certs test/config/

# Run tests with proper environment
sudo -E SSSONECTOR_TEST_CONFIG_DIR=$BUILD_DIR/test/config \
    SSSONECTOR_TEST_CERTS_DIR=$BUILD_DIR/test/certs \
    go test -v ./...
check_status "Unit tests"

# Build binaries
echo "Building SSSonector..."
make clean
check_status "Clean build directory"

# Build both binaries explicitly
echo "Building sssonector..."
go build -o build/sssonector ./cmd/daemon/
check_status "Build sssonector"

echo "Building sssonectorctl..."
go build -o build/sssonectorctl ./cmd/sssonectorctl/
check_status "Build sssonectorctl"

# Create distribution package
echo "Creating distribution package..."
DIST_DIR="$BUILD_DIR/dist"
mkdir -p $DIST_DIR/{bin,etc/certs,systemd}

# Copy binaries
cp build/sssonector $DIST_DIR/bin/
cp build/sssonectorctl $DIST_DIR/bin/
chmod 755 $DIST_DIR/bin/*
check_status "Binary copy"

# Copy configuration and certificates
cp -r configs/* $DIST_DIR/etc/
cp test/certs/* $DIST_DIR/etc/certs/
cp init/systemd/sssonector.service $DIST_DIR/systemd/
check_status "Configuration copy"

# Create version file
git rev-parse HEAD > $DIST_DIR/version.txt
date >> $DIST_DIR/version.txt
check_status "Version file creation"

# Create distribution archive
cd $BUILD_DIR
tar czf sssonector-dist.tar.gz dist/
check_status "Distribution package creation"

# Deploy to QA systems
echo "Deploying to QA systems..."

deploy_to_system() {
    local system=$1
    local config=$2
    echo "Deploying to $system..."
    
    # Copy distribution package
    scp sssonector-dist.tar.gz $system:/tmp/
    check_status "Copy to $system"
    
    # Install on remote system
    ssh $system "echo '$SUDO_PASS' | sudo -S bash -s" << EOF
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
        mkdir -p /etc/sssonector/certs
        cp -r dist/etc/* /etc/sssonector/
        cp dist/etc/certs/* /etc/sssonector/certs/
        cp dist/etc/$config /etc/sssonector/sssonector.yaml
        chmod 644 /etc/sssonector/*.yaml
        chmod 600 /etc/sssonector/certs/*
        
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
deploy_to_system $QA_SERVER "sssonector.yaml"
# Deploy to QA client
deploy_to_system $QA_CLIENT "sssonector_client.yaml"

echo "[✓] Build and deployment"

# Step 2: Verify build
echo "=== Step 2: Verifying build ==="
./verify_build.sh
check_status "Build verification"

# Step 3: Run test scenarios
echo "=== Step 3: Running test scenarios ==="
./test_scenarios.sh
check_status "Test scenarios"

# Step 4: Collect logs
echo "=== Collecting system logs ==="

# Create log collection script
cat > collect_logs.sh << 'EOF'
#!/bin/bash
echo "=== System Information ==="
uname -a
echo
echo "=== Service Status ==="
systemctl status sssonector
echo
echo "=== Service Logs ==="
journalctl -u sssonector --no-pager -n 100
echo
echo "=== Configuration ==="
cat /etc/sssonector/sssonector.yaml
echo
echo "=== Network Status ==="
ip addr show
ip route show
EOF

# Collect logs from both systems
for system in $QA_SERVER $QA_CLIENT; do
    echo "Collecting logs from $system..."
    scp collect_logs.sh $system:/tmp/
    ssh $system "chmod +x /tmp/collect_logs.sh && echo '$SUDO_PASS' | sudo -S /tmp/collect_logs.sh" > "qa_logs_${system}.txt"
    ssh $system "rm /tmp/collect_logs.sh"
done

rm collect_logs.sh

echo "QA test run completed at $(date)"
