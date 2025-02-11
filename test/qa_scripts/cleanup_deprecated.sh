#!/bin/bash
set -e

# List of deprecated scripts to remove
DEPRECATED_SCRIPTS=(
    # Old test scripts without state transition handling
    "test_ssh_auth.exp"
    "test_local_auth.exp"
    "test_remote_auth.exp"
    "verify_ssh_basic.exp"
    "verify_snmp_basic.exp"
    
    # Scripts now covered by core_sanity_check.sh
    "sanity_check.sh"
    "test_scenarios.sh"
    "test_rate_limit.sh"
    
    # Deprecated setup scripts
    "setup_sssonector_repo.exp"
    "setup_sssonector_repo2.exp"
    "verify_automation.exp"
    
    # Old authentication test scripts
    "test_keyboard_interactive.exp"
    "test_ssh_auth.exp"
    "test_remote_auth.exp"
    "reset_remote_ssh.exp"
    
    # Old SNMP test scripts
    "test_remote_snmp.exp"
    "test_remote_snmp2.exp"
    "test_remote_snmp3.exp"
    "verify_snmp_query.exp"
    "verify_snmp_remote.exp"
    "verify_snmp_passthrough.exp"
    
    # Redundant setup scripts
    "setup_sudo.exp"
    "setup_sudo_access.exp"
    "sudo_pass.exp"
    "sudo_install.sh"
)

# Directory containing QA scripts
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Create backup directory
BACKUP_DIR="${SCRIPT_DIR}/deprecated_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BACKUP_DIR"

echo "Creating backup directory: $BACKUP_DIR"

# Move deprecated scripts to backup directory
for script in "${DEPRECATED_SCRIPTS[@]}"; do
    if [ -f "${SCRIPT_DIR}/${script}" ]; then
        echo "Moving deprecated script: ${script}"
        mv "${SCRIPT_DIR}/${script}" "${BACKUP_DIR}/"
    else
        echo "Warning: Script not found: ${script}"
    fi
done

# Update README.md to reflect changes
README="${SCRIPT_DIR}/README.md"
if [ -f "$README" ]; then
    echo "Updating README.md..."
    
    # Add deprecation notice
    cat >> "$README" << EOF

## Deprecated Scripts

The following scripts have been deprecated and moved to backup as they do not utilize 
the new reliability improvements:

1. Old test scripts without state transition handling:
   - test_ssh_auth.exp
   - test_local_auth.exp
   - test_remote_auth.exp
   - verify_ssh_basic.exp
   - verify_snmp_basic.exp

2. Scripts now covered by core_sanity_check.sh:
   - sanity_check.sh
   - test_scenarios.sh
   - test_rate_limit.sh

3. Deprecated setup scripts:
   - setup_sssonector_repo.exp
   - setup_sssonector_repo2.exp
   - verify_automation.exp

These scripts have been replaced by the enhanced reliability testing framework in:
- server_sanity_check.sh
- core_sanity_check.sh

The new scripts include:
- State transition verification
- Resource cleanup validation
- Connection tracking
- Statistics monitoring
EOF
fi

echo "Cleanup complete. Deprecated scripts moved to: $BACKUP_DIR"
echo "Please verify the changes and remove the backup directory if everything works correctly."
