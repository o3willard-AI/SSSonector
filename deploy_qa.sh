#!/bin/bash
set -e

echo "Deploying SSSonector to QA environment..."

# Ensure we're in the project root
cd "$(dirname "$0")"

# Source QA environment config if it exists
if [ -f "test/qa_scripts/config.env" ]; then
    source test/qa_scripts/config.env
else
    echo "Error: test/qa_scripts/config.env not found"
    echo "Please copy config.env.example to config.env and update with your QA environment details"
    exit 1
fi

# Build debug binary
echo "Building debug binary..."
./build_debug.sh

# Create deployment package
DEPLOY_DIR="qa_deploy_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$DEPLOY_DIR"

echo "Preparing deployment package in $DEPLOY_DIR..."

# Copy files
cp sssonector "$DEPLOY_DIR/"
cp test/qa_scripts/server_sanity_check.sh "$DEPLOY_DIR/"
cp test/qa_scripts/core_sanity_check.sh "$DEPLOY_DIR/"
cp test/qa_scripts/common.sh "$DEPLOY_DIR/"
cp test/qa_scripts/config.env "$DEPLOY_DIR/"
cp test/qa_scripts/config/server_template.yaml "$DEPLOY_DIR/"
cp test/qa_scripts/config/client_template.yaml "$DEPLOY_DIR/"
cp configs/server.yaml "$DEPLOY_DIR/"

# Create version info
git rev-parse HEAD > "$DEPLOY_DIR/version.txt"
git log -1 --pretty=format:"%h - %s (%ci)" >> "$DEPLOY_DIR/version.txt"

echo "Deploying to QA servers..."

# Deploy to server VM
echo "Deploying to server ${QA_SERVER_VM}..."

# Clean up existing processes and interfaces
ssh "${QA_SSH_USER}@${QA_SERVER_VM}" "
    sudo pkill -f sssonector || true
    sudo ip link del tun0 2>/dev/null || true
    sudo lsof -ti :8443 | xargs -r sudo kill -9 || true
    sleep 2
"

# Create remote directories
ssh "${QA_SSH_USER}@${QA_SERVER_VM}" "sudo mkdir -p /opt/sssonector/bin /opt/sssonector/scripts /etc/sssonector /var/log/sssonector"

# Copy files to QA server
scp -r "$DEPLOY_DIR"/* "${QA_SSH_USER}@${QA_SERVER_VM}:/tmp/"

# Install on QA server
ssh "${QA_SSH_USER}@${QA_SERVER_VM}" "
    # Create required directories
    sudo mkdir -p /usr/local/bin /opt/sssonector/scripts /etc/sssonector /var/log/sssonector && \
    
    # Create sssonector user and group
    sudo groupadd -f sssonector && \
    sudo useradd -r -g sssonector sssonector 2>/dev/null || true && \
    
    # Install binary and set permissions
    sudo mv /tmp/sssonector /usr/local/bin/ && \
    sudo chmod 755 /usr/local/bin/sssonector && \
    sudo setcap cap_net_admin+ep /usr/local/bin/sssonector && \
    sudo chown root:root /usr/local/bin/sssonector && \
    
    # Install scripts and config
    sudo mv /tmp/server_sanity_check.sh /opt/sssonector/scripts/ && \
    sudo mv /tmp/common.sh /opt/sssonector/scripts/ && \
    sudo mv /tmp/config.env /opt/sssonector/scripts/ && \
    sudo mv /tmp/server.yaml /etc/sssonector/config.yaml && \
    sudo mv /tmp/version.txt /opt/sssonector/ && \
    
    # Set permissions
    sudo chmod 755 /opt/sssonector/scripts/*.sh && \
    sudo chmod 644 /etc/sssonector/config.yaml && \
    sudo chown -R root:root /etc/sssonector && \
    sudo chmod 755 /etc/sssonector && \
    sudo chown -R root:root /var/log/sssonector && \
    sudo chmod 755 /var/log/sssonector && \
    
    # Set up TUN device
    sudo mkdir -p /dev/net && \
    sudo mknod /dev/net/tun c 10 200 2>/dev/null || true && \
    sudo chown root:sssonector /dev/net/tun && \
    sudo chmod 0660 /dev/net/tun && \
    
    # Add current user to sssonector group
    sudo usermod -a -G sssonector \$USER
"

echo "Verifying deployment..."

# Verify installation
ssh "${QA_SSH_USER}@${QA_SERVER_VM}" "
    echo 'Binary capabilities:' && getcap /usr/local/bin/sssonector && \
    echo 'Script permissions:' && ls -l /opt/sssonector/scripts/ && \
    echo 'Config file:' && ls -l /etc/sssonector/config.yaml && \
    echo 'Testing binary:' && sg sssonector -c '/usr/local/bin/sssonector -config /etc/sssonector/config.yaml || true'
"

# Clean up local deployment directory
rm -rf "$DEPLOY_DIR"

# Deploy to client VM
echo "Deploying to client ${QA_CLIENT_VM}..."

# Clean up existing processes and interfaces on client
ssh "${QA_SSH_USER}@${QA_CLIENT_VM}" "
    sudo pkill -f sssonector || true
    sudo ip link del tun0 2>/dev/null || true
    sudo lsof -ti :8443 | xargs -r sudo kill -9 || true
    sleep 2
"

# Create remote directories on client
ssh "${QA_SSH_USER}@${QA_CLIENT_VM}" "sudo mkdir -p /opt/sssonector/bin /opt/sssonector/scripts /etc/sssonector /var/log/sssonector"

# Copy files to QA client
scp -r "$DEPLOY_DIR"/* "${QA_SSH_USER}@${QA_CLIENT_VM}:/tmp/"

# Install on QA client
ssh "${QA_SSH_USER}@${QA_CLIENT_VM}" "
    # Create required directories
    sudo mkdir -p /usr/local/bin /opt/sssonector/scripts /etc/sssonector /var/log/sssonector && \
    
    # Create sssonector user and group
    sudo groupadd -f sssonector && \
    sudo useradd -r -g sssonector sssonector 2>/dev/null || true && \
    
    # Install binary and set permissions
    sudo mv /tmp/sssonector /usr/local/bin/ && \
    sudo chmod 755 /usr/local/bin/sssonector && \
    sudo setcap cap_net_admin+ep /usr/local/bin/sssonector && \
    sudo chown root:root /usr/local/bin/sssonector && \
    
    # Install scripts and config
    sudo mv /tmp/core_sanity_check.sh /opt/sssonector/scripts/ && \
    sudo mv /tmp/common.sh /opt/sssonector/scripts/ && \
    sudo mv /tmp/config.env /opt/sssonector/scripts/ && \
    sudo mv /tmp/client_template.yaml /etc/sssonector/config.yaml && \
    sudo mv /tmp/version.txt /opt/sssonector/ && \
    
    # Set permissions
    sudo chmod 755 /opt/sssonector/scripts/*.sh && \
    sudo chmod 644 /etc/sssonector/config.yaml && \
    sudo chown -R root:root /etc/sssonector && \
    sudo chmod 755 /etc/sssonector && \
    sudo chown -R root:root /var/log/sssonector && \
    sudo chmod 755 /var/log/sssonector && \
    
    # Set up TUN device
    sudo mkdir -p /dev/net && \
    sudo mknod /dev/net/tun c 10 200 2>/dev/null || true && \
    sudo chown root:sssonector /dev/net/tun && \
    sudo chmod 0660 /dev/net/tun && \
    
    # Add current user to sssonector group
    sudo usermod -a -G sssonector \$USER
"

echo "Verifying client deployment..."

# Verify client installation
ssh "${QA_SSH_USER}@${QA_CLIENT_VM}" "
    echo 'Binary capabilities:' && getcap /usr/local/bin/sssonector && \
    echo 'Script permissions:' && ls -l /opt/sssonector/scripts/ && \
    echo 'Config file:' && ls -l /etc/sssonector/config.yaml && \
    echo 'Testing binary:' && sg sssonector -c '/usr/local/bin/sssonector -config /etc/sssonector/config.yaml || true'
"

echo "Deployment complete. You can now run the tests:"
echo "1. Server sanity check:"
echo "   ssh ${QA_SSH_USER}@${QA_SERVER_VM} 'cd /opt/sssonector/scripts && ./server_sanity_check.sh'"
echo "2. Core functionality sanity check:"
echo "   ssh ${QA_SSH_USER}@${QA_SERVER_VM} 'cd /opt/sssonector/scripts && ./core_sanity_check.sh'"
