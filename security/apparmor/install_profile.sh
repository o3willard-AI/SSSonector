#!/bin/bash
set -e

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: This script must be run as root"
    exit 1
fi

# Check for AppArmor
if ! command -v apparmor_parser >/dev/null; then
    echo "Error: AppArmor not found"
    echo "Please install AppArmor:"
    echo "  Ubuntu/Debian: apt install apparmor apparmor-utils"
    echo "  SUSE: zypper install apparmor-parser apparmor-utils"
    exit 1
fi

# Check AppArmor status
if ! systemctl is-active --quiet apparmor.service; then
    echo "Warning: AppArmor service not running"
    echo "Starting AppArmor service..."
    systemctl start apparmor.service
fi

PROFILE_NAME="usr.local.bin.sssonector"
PROFILE_DIR="$(dirname "$0")"
cd "$PROFILE_DIR"

echo "Installing AppArmor profile..."

# Copy profile to system directory
install -m 644 ${PROFILE_NAME} /etc/apparmor.d/

# Create required directories
echo "Creating required directories..."
mkdir -p /etc/sssonector/certs
mkdir -p /var/lib/sssonector
mkdir -p /var/run/sssonector
mkdir -p /var/log/sssonector

# Set correct permissions
echo "Setting directory permissions..."
chown -R sssonector:sssonector /etc/sssonector
chown -R sssonector:sssonector /var/lib/sssonector
chown -R sssonector:sssonector /var/run/sssonector
chown -R sssonector:sssonector /var/log/sssonector

chmod 750 /etc/sssonector
chmod 750 /var/lib/sssonector
chmod 750 /var/run/sssonector
chmod 750 /var/log/sssonector

# Load profile
echo "Loading AppArmor profile..."
apparmor_parser -r -T -W /etc/apparmor.d/${PROFILE_NAME}

# Enable profile
echo "Enabling profile..."
aa-enforce /usr/local/bin/sssonector

echo "AppArmor profile installed and enabled successfully"
echo
echo "To verify the installation:"
echo "  - Check profile status: aa-status | grep sssonector"
echo "  - Check current mode: aa-status --json | jq '.processes | .[] | select(.profile | contains(\"sssonector\"))'"
echo
echo "To monitor AppArmor denials:"
echo "  tail -f /var/log/audit/audit.log | grep -i apparmor"
echo "  tail -f /var/log/syslog | grep -i apparmor"
echo
echo "Profile management commands:"
echo "  - Disable profile:  aa-disable /usr/local/bin/sssonector"
echo "  - Enable profile:   aa-enforce /usr/local/bin/sssonector"
echo "  - Set complain:     aa-complain /usr/local/bin/sssonector"
echo "  - Update profile:   apparmor_parser -r -T -W /etc/apparmor.d/${PROFILE_NAME}"
echo
echo "To test profile in complain mode first:"
echo "  aa-complain /usr/local/bin/sssonector"
echo "  # Run service and check logs for denials"
echo "  aa-logprof  # Update profile based on denials"
echo "  aa-enforce /usr/local/bin/sssonector  # Enable enforcing mode"
