#!/bin/bash
set -e

# Configuration
SERVICE_NAME="sssonector"
SERVICE_USER="sssonector"
SERVICE_GROUP="sssonector"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/sssonector"
STATE_DIR="/var/lib/sssonector"
RUN_DIR="/var/run/sssonector"
LOG_DIR="/var/log/sssonector"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: This script must be run as root"
    exit 1
fi

# Create service user and group
echo "Creating service user and group..."
if ! getent group "$SERVICE_GROUP" >/dev/null; then
    groupadd --system "$SERVICE_GROUP"
fi
if ! getent passwd "$SERVICE_USER" >/dev/null; then
    useradd --system \
        --gid "$SERVICE_GROUP" \
        --no-create-home \
        --shell /sbin/nologin \
        --comment "SSSonector Service User" \
        "$SERVICE_USER"
fi

# Create required directories
echo "Creating directories..."
install -d -m 750 -o "$SERVICE_USER" -g "$SERVICE_GROUP" "$CONFIG_DIR"
install -d -m 750 -o "$SERVICE_USER" -g "$SERVICE_GROUP" "$CONFIG_DIR/certs"
install -d -m 750 -o "$SERVICE_USER" -g "$SERVICE_GROUP" "$STATE_DIR"
install -d -m 750 -o "$SERVICE_USER" -g "$SERVICE_GROUP" "$LOG_DIR"

# Install binaries
echo "Installing binaries..."
install -m 755 bin/sssonector "$INSTALL_DIR/sssonector"
install -m 755 bin/sssonectorctl "$INSTALL_DIR/sssonectorctl"

# Install configuration
echo "Installing configuration..."
if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
    install -m 640 -o "$SERVICE_USER" -g "$SERVICE_GROUP" \
        config/config.yaml.example "$CONFIG_DIR/config.yaml"
fi

# Install systemd service
if command -v systemctl >/dev/null; then
    echo "Installing systemd service..."
    install -m 644 init/systemd/sssonector.service /etc/systemd/system/
    systemctl daemon-reload
    
    # Enable and start service if requested
    read -p "Enable and start service now? [y/N] " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        systemctl enable sssonector
        systemctl start sssonector
        echo "Service started. Check status with: systemctl status sssonector"
    else
        echo "Service installed but not started. Start manually with: systemctl start sssonector"
    fi
fi

# Set up log rotation
echo "Setting up log rotation..."
cat > /etc/logrotate.d/sssonector << EOF
$LOG_DIR/*.log {
    daily
    rotate 7
    missingok
    notifempty
    compress
    delaycompress
    create 640 $SERVICE_USER $SERVICE_GROUP
    sharedscripts
    postrotate
        systemctl kill -s HUP sssonector.service
    endscript
}
EOF

# Set up firewall rules if firewalld is present
if command -v firewall-cmd >/dev/null; then
    echo "Setting up firewall rules..."
    firewall-cmd --permanent --add-port=8443/tcp
    firewall-cmd --reload
fi

# Set up SELinux context if SELinux is enabled
if command -v semanage >/dev/null && sestatus | grep -q "enabled"; then
    echo "Setting up SELinux context..."
    semanage fcontext -a -t bin_t "$INSTALL_DIR/sssonector"
    semanage fcontext -a -t bin_t "$INSTALL_DIR/sssonectorctl"
    semanage fcontext -a -t etc_t "$CONFIG_DIR(/.*)?"
    semanage fcontext -a -t var_lib_t "$STATE_DIR(/.*)?"
    semanage fcontext -a -t var_run_t "$RUN_DIR(/.*)?"
    semanage fcontext -a -t var_log_t "$LOG_DIR(/.*)?"
    restorecon -R "$INSTALL_DIR/sssonector" "$INSTALL_DIR/sssonectorctl" \
        "$CONFIG_DIR" "$STATE_DIR" "$RUN_DIR" "$LOG_DIR"
    
    # Add SELinux port context
    semanage port -a -t http_port_t -p tcp 8443
fi

echo "Installation complete!"
echo
echo "Next steps:"
echo "1. Edit configuration: $CONFIG_DIR/config.yaml"
echo "2. Install certificates in: $CONFIG_DIR/certs/"
echo "3. Start service: systemctl start sssonector"
echo "4. Check status: systemctl status sssonector"
echo "5. View logs: journalctl -u sssonector"
echo
echo "Control service with: sssonectorctl [command]"
