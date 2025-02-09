#!/bin/bash
set -e

# Configuration
SERVICE_NAME="com.o3willard.sssonector"
SERVICE_USER="_sssonector"
SERVICE_GROUP="_sssonector"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/sssonector"
STATE_DIR="/var/lib/sssonector"
RUN_DIR="/var/run/sssonector"
LOG_DIR="/var/log/sssonector"
LAUNCHD_DIR="/Library/LaunchDaemons"

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: This script must be run as root"
    exit 1
fi

# Create service user and group
echo "Creating service user and group..."
if ! dscl . -read /Groups/$SERVICE_GROUP >/dev/null 2>&1; then
    # Create group
    NEXT_GID=$(dscl . -list /Groups gid | awk '{print $2}' | sort -n | tail -1 | xargs -I{} expr {} + 1)
    dscl . -create /Groups/$SERVICE_GROUP
    dscl . -create /Groups/$SERVICE_GROUP PrimaryGroupID $NEXT_GID
    dscl . -create /Groups/$SERVICE_GROUP RealName "SSSonector Service Group"
    dscl . -create /Groups/$SERVICE_GROUP Password "*"
fi

if ! dscl . -read /Users/$SERVICE_USER >/dev/null 2>&1; then
    # Create user
    NEXT_UID=$(dscl . -list /Users UniqueID | awk '{print $2}' | sort -n | tail -1 | xargs -I{} expr {} + 1)
    dscl . -create /Users/$SERVICE_USER
    dscl . -create /Users/$SERVICE_USER UniqueID $NEXT_UID
    dscl . -create /Users/$SERVICE_USER PrimaryGroupID $(dscl . -read /Groups/$SERVICE_GROUP PrimaryGroupID | awk '{print $2}')
    dscl . -create /Users/$SERVICE_USER UserShell /usr/bin/false
    dscl . -create /Users/$SERVICE_USER RealName "SSSonector Service User"
    dscl . -create /Users/$SERVICE_USER Password "*"
    dscl . -create /Users/$SERVICE_USER NFSHomeDirectory /var/empty
fi

# Create required directories
echo "Creating directories..."
install -d -m 750 -o $SERVICE_USER -g $SERVICE_GROUP "$CONFIG_DIR"
install -d -m 750 -o $SERVICE_USER -g $SERVICE_GROUP "$CONFIG_DIR/certs"
install -d -m 750 -o $SERVICE_USER -g $SERVICE_GROUP "$STATE_DIR"
install -d -m 750 -o $SERVICE_USER -g $SERVICE_GROUP "$LOG_DIR"

# Install binaries
echo "Installing binaries..."
install -m 755 bin/sssonector "$INSTALL_DIR/sssonector"
install -m 755 bin/sssonectorctl "$INSTALL_DIR/sssonectorctl"

# Install configuration
echo "Installing configuration..."
if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
    install -m 640 -o $SERVICE_USER -g $SERVICE_GROUP \
        config/config.yaml.example "$CONFIG_DIR/config.yaml"
fi

# Install launchd service
echo "Installing launchd service..."
install -m 644 init/launchd/$SERVICE_NAME.plist "$LAUNCHD_DIR/"

# Set up log rotation
echo "Setting up log rotation..."
cat > /etc/newsyslog.d/sssonector.conf << EOF
# logfilename                       [owner:group]    mode count size when  flags [/pid_file] [sig_num]
$LOG_DIR/service.log               $SERVICE_USER:$SERVICE_GROUP    640  7     *    \$D0   J
$LOG_DIR/error.log                 $SERVICE_USER:$SERVICE_GROUP    640  7     *    \$D0   J
EOF

# Configure firewall
echo "Configuring firewall..."
/usr/libexec/ApplicationFirewall/socketfilterfw --add "$INSTALL_DIR/sssonector"
/usr/libexec/ApplicationFirewall/socketfilterfw --unblock "$INSTALL_DIR/sssonector"

# Load and start service
echo "Loading service..."
launchctl load -w "$LAUNCHD_DIR/$SERVICE_NAME.plist"

# Verify service status
echo "Verifying service status..."
if launchctl list | grep -q "$SERVICE_NAME"; then
    echo "Service loaded successfully"
else
    echo "Warning: Service not loaded properly"
fi

echo "Installation complete!"
echo
echo "Next steps:"
echo "1. Edit configuration: $CONFIG_DIR/config.yaml"
echo "2. Install certificates in: $CONFIG_DIR/certs/"
echo "3. Start service: sudo launchctl start $SERVICE_NAME"
echo "4. Check status: sudo launchctl list | grep $SERVICE_NAME"
echo "5. View logs: tail -f $LOG_DIR/service.log"
echo
echo "Control service with: sssonectorctl [command]"
echo
echo "Service management commands:"
echo "  Start:   sudo launchctl start $SERVICE_NAME"
echo "  Stop:    sudo launchctl stop $SERVICE_NAME"
echo "  Reload:  sudo launchctl kickstart -k $SERVICE_NAME"
echo "  Status:  sudo launchctl list | grep $SERVICE_NAME"
echo "  Unload:  sudo launchctl unload $LAUNCHD_DIR/$SERVICE_NAME.plist"

# Add to PATH if needed
if [[ ":$PATH:" != *":/usr/local/bin:"* ]]; then
    echo 'export PATH="/usr/local/bin:$PATH"' >> /etc/zshrc
    echo 'export PATH="/usr/local/bin:$PATH"' >> /etc/bashrc
    echo "Added /usr/local/bin to PATH. Please restart your shell for changes to take effect."
fi
