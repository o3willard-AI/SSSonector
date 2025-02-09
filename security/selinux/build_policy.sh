#!/bin/bash
set -e

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "Error: This script must be run as root"
    exit 1
fi

# Check for required tools
for cmd in checkmodule semodule_package semodule; do
    if ! command -v $cmd >/dev/null; then
        echo "Error: Required command '$cmd' not found"
        echo "Please install the SELinux policy development tools:"
        echo "  RHEL/CentOS: yum install selinux-policy-devel"
        echo "  Fedora: dnf install selinux-policy-devel"
        echo "  Ubuntu/Debian: apt install selinux-policy-dev"
        exit 1
    fi
done

POLICY_NAME="sssonector"
POLICY_DIR="$(dirname "$0")"
cd "$POLICY_DIR"

echo "Building SELinux policy module..."

# Compile policy module
echo "Compiling policy module..."
checkmodule -M -m -o ${POLICY_NAME}.mod ${POLICY_NAME}.te

# Create policy package
echo "Creating policy package..."
semodule_package -o ${POLICY_NAME}.pp -m ${POLICY_NAME}.mod -f ${POLICY_NAME}.fc

# Install policy module
echo "Installing policy module..."
semodule -i ${POLICY_NAME}.pp

# Set file contexts
echo "Setting file contexts..."
semanage fcontext -a -t sssonector_exec_t "/usr/local/bin/sssonector"
semanage fcontext -a -t sssonector_exec_t "/usr/local/bin/sssonectorctl"
semanage fcontext -a -t sssonector_conf_t "/etc/sssonector(/.*)?"
semanage fcontext -a -t sssonector_var_lib_t "/var/lib/sssonector(/.*)?"
semanage fcontext -a -t sssonector_var_run_t "/var/run/sssonector(/.*)?"
semanage fcontext -a -t sssonector_log_t "/var/log/sssonector(/.*)?"

# Restore file contexts
echo "Restoring file contexts..."
restorecon -R /usr/local/bin/sssonector
restorecon -R /usr/local/bin/sssonectorctl
restorecon -R /etc/sssonector
restorecon -R /var/lib/sssonector
restorecon -R /var/run/sssonector
restorecon -R /var/log/sssonector

# Add port context
echo "Setting port context..."
semanage port -a -t sssonector_port_t -p tcp 8443

echo "SELinux policy module installed successfully"
echo
echo "To verify the installation:"
echo "  - Check loaded modules: semodule -l | grep sssonector"
echo "  - Check file contexts: ls -Z /usr/local/bin/sssonector"
echo "  - Check port context: semanage port -l | grep sssonector"
echo
echo "To enable enforcing mode:"
echo "  setenforce 1"
echo
echo "To monitor SELinux denials:"
echo "  ausearch -m AVC -ts recent"
echo "  tail -f /var/log/audit/audit.log | grep sssonector"
