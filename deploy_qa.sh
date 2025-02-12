#!/bin/bash
set -ex

echo "Deploying SSSonector to QA environment..."

# Ensure we're in the project root
cd "$(dirname "$0")"

# Source QA environment config if it exists
if [ -f "test/qa_scripts/config.env" ]; then
    source "test/qa_scripts/config.env"
else
    echo "Error: test/qa_scripts/config.env not found"
    echo "Please copy config.env.example to config.env and update with your QA environment details"
    exit 1
fi

# Check if sshpass is installed
if ! command -v sshpass &> /dev/null; then
    echo "Installing sshpass..."
    sudo apt-get update && sudo apt-get install -y sshpass
fi

# Clean up local environment
echo "Cleaning up local environment..."
sudo pkill -f sssonector || true
sudo lsof -ti :8443 | xargs -r sudo kill -9 || true
sudo ip link delete tun0 2>/dev/null || true
sleep 2

# Build debug binary
echo "Building debug binary..."
./build_debug.sh

# Create deployment package
DEPLOY_DIR="qa_deploy_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$DEPLOY_DIR"/{server,client}

echo "Preparing deployment package in $DEPLOY_DIR..."

# Copy common files
cp sssonector "$DEPLOY_DIR/server/"
cp sssonector "$DEPLOY_DIR/client/"
cp test/qa_scripts/common.sh "$DEPLOY_DIR/server/"
cp test/qa_scripts/common.sh "$DEPLOY_DIR/client/"
cp test/qa_scripts/config.env "$DEPLOY_DIR/server/"
cp test/qa_scripts/config.env "$DEPLOY_DIR/client/"

# Copy server-specific files
cp test/qa_scripts/server_sanity_check.sh "$DEPLOY_DIR/server/"
cp test/qa_scripts/core_sanity_check.sh "$DEPLOY_DIR/server/"
cp test/qa_scripts/config/server_template.yaml "$DEPLOY_DIR/server/server.yaml"

# Copy client-specific files
cp test/qa_scripts/config/client_template.yaml "$DEPLOY_DIR/client/client.yaml"

# Create version info
git rev-parse HEAD > "$DEPLOY_DIR/server/version.txt"
git log -1 --pretty=format:"%h - %s (%ci)" >> "$DEPLOY_DIR/server/version.txt"
cp "$DEPLOY_DIR/server/version.txt" "$DEPLOY_DIR/client/version.txt"

echo "Deploying to QA servers..."

# Deploy to server VM
echo "Deploying to server ${QA_SERVER_VM}..."

echo "Testing SSH connection to server..."
sshpass -p "${QA_SUDO_PASSWORD}" ssh -v -o StrictHostKeyChecking=no "${QA_USER}@${QA_SERVER_VM}" "echo 'SSH connection test successful'"

echo "Cleaning up existing processes on server..."
sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${QA_SERVER_VM}" "echo ${QA_SUDO_PASSWORD} | sudo -S pkill -f sssonector || true"

echo "Cleaning up network interfaces on server..."
sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${QA_SERVER_VM}" "echo ${QA_SUDO_PASSWORD} | sudo -S ip link del tun0 2>/dev/null || true"

echo "Cleaning up ports on server..."
sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${QA_SERVER_VM}" "echo ${QA_SUDO_PASSWORD} | sudo -S lsof -ti :8443 | xargs -r sudo kill -9 || true"
sleep 2

echo "Creating remote directories on server..."
sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${QA_SERVER_VM}" "echo ${QA_SUDO_PASSWORD} | sudo -S mkdir -p /opt/sssonector/bin /opt/sssonector/scripts /etc/sssonector /var/log/sssonector"

echo "Copying files to server..."
sshpass -p "${QA_SUDO_PASSWORD}" scp -v -o StrictHostKeyChecking=no -r "$DEPLOY_DIR/server/"* "${QA_USER}@${QA_SERVER_VM}:/tmp/"

echo "Installing on server..."
sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${QA_SERVER_VM}" "
    echo '${QA_SUDO_PASSWORD}' | sudo -S bash -c '
        # Create required directories
        mkdir -p /usr/local/bin /opt/sssonector/scripts /etc/sssonector /var/log/sssonector

        # Create sssonector user and group
        groupadd -f sssonector
        useradd -r -g sssonector sssonector 2>/dev/null || true

        # Install binary and set permissions
        mv /tmp/sssonector /usr/local/bin/
        chmod 755 /usr/local/bin/sssonector
        setcap cap_net_admin+ep /usr/local/bin/sssonector
        chown root:root /usr/local/bin/sssonector

        # Install scripts and config
        mv /tmp/server_sanity_check.sh /opt/sssonector/scripts/
        mv /tmp/core_sanity_check.sh /opt/sssonector/scripts/
        mv /tmp/common.sh /opt/sssonector/scripts/
        mv /tmp/config.env /opt/sssonector/scripts/
        mv /tmp/server.yaml /etc/sssonector/config.yaml
        mv /tmp/version.txt /opt/sssonector/

        # Set permissions
        chmod 755 /opt/sssonector/scripts/*.sh
        chmod 644 /etc/sssonector/config.yaml
        chown -R root:root /etc/sssonector
        chmod 755 /etc/sssonector
        chown -R root:root /var/log/sssonector
        chmod 755 /var/log/sssonector

        # Set up TUN device
        mkdir -p /dev/net
        mknod /dev/net/tun c 10 200 2>/dev/null || true
        chown root:sssonector /dev/net/tun
        chmod 0660 /dev/net/tun

        # Add current user to sssonector group
        usermod -a -G sssonector ${QA_USER}
    '
"

# Deploy to client VM
echo "Deploying to client ${QA_CLIENT_VM}..."

echo "Testing SSH connection to client..."
sshpass -p "${QA_SUDO_PASSWORD}" ssh -v -o StrictHostKeyChecking=no "${QA_USER}@${QA_CLIENT_VM}" "echo 'SSH connection test successful'"

echo "Cleaning up existing processes on client..."
sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${QA_CLIENT_VM}" "echo ${QA_SUDO_PASSWORD} | sudo -S pkill -f sssonector || true"

echo "Cleaning up network interfaces on client..."
sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${QA_CLIENT_VM}" "echo ${QA_SUDO_PASSWORD} | sudo -S ip link del tun0 2>/dev/null || true"
sleep 2

echo "Creating remote directories on client..."
sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${QA_CLIENT_VM}" "echo ${QA_SUDO_PASSWORD} | sudo -S mkdir -p /opt/sssonector/bin /opt/sssonector/scripts /etc/sssonector /var/log/sssonector"

echo "Copying files to client..."
sshpass -p "${QA_SUDO_PASSWORD}" scp -v -o StrictHostKeyChecking=no -r "$DEPLOY_DIR/client/"* "${QA_USER}@${QA_CLIENT_VM}:/tmp/"

echo "Installing on client..."
sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${QA_CLIENT_VM}" "
    echo '${QA_SUDO_PASSWORD}' | sudo -S bash -c '
        # Create required directories
        mkdir -p /usr/local/bin /opt/sssonector/scripts /etc/sssonector /var/log/sssonector

        # Create sssonector user and group
        groupadd -f sssonector
        useradd -r -g sssonector sssonector 2>/dev/null || true

        # Install binary and set permissions
        mv /tmp/sssonector /usr/local/bin/
        chmod 755 /usr/local/bin/sssonector
        setcap cap_net_admin+ep /usr/local/bin/sssonector
        chown root:root /usr/local/bin/sssonector

        # Install config
        mv /tmp/client.yaml /etc/sssonector/config.yaml
        mv /tmp/version.txt /opt/sssonector/

        # Set permissions
        chmod 644 /etc/sssonector/config.yaml
        chown -R root:root /etc/sssonector
        chmod 755 /etc/sssonector
        chown -R root:root /var/log/sssonector
        chmod 755 /var/log/sssonector

        # Set up TUN device
        mkdir -p /dev/net
        mknod /dev/net/tun c 10 200 2>/dev/null || true
        chown root:sssonector /dev/net/tun
        chmod 0660 /dev/net/tun

        # Add current user to sssonector group
        usermod -a -G sssonector ${QA_USER}
    '
"

echo "Verifying deployment..."
echo "Server verification:"
sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${QA_SERVER_VM}" "
    echo 'Binary capabilities:' && getcap /usr/local/bin/sssonector
    echo 'Script permissions:' && ls -l /opt/sssonector/scripts/
    echo 'Config file:' && ls -l /etc/sssonector/config.yaml
"

echo "Client verification:"
sshpass -p "${QA_SUDO_PASSWORD}" ssh -o StrictHostKeyChecking=no "${QA_USER}@${QA_CLIENT_VM}" "
    echo 'Binary capabilities:' && getcap /usr/local/bin/sssonector
    echo 'Config file:' && ls -l /etc/sssonector/config.yaml
"

echo "Cleaning up local deployment directory..."
rm -rf "$DEPLOY_DIR"

echo "Deployment complete. You can now run the tests:"
echo "1. Server sanity check:"
echo "   sshpass -p '${QA_SUDO_PASSWORD}' ssh -o StrictHostKeyChecking=no ${QA_USER}@${QA_SERVER_VM} 'cd /opt/sssonector/scripts && ./server_sanity_check.sh'"
echo "2. Core functionality sanity check:"
echo "   sshpass -p '${QA_SUDO_PASSWORD}' ssh -o StrictHostKeyChecking=no ${QA_USER}@${QA_SERVER_VM} 'cd /opt/sssonector/scripts && ./core_sanity_check.sh'"
