#!/bin/bash
set -e

# SELinux policy configuration
POLICY_NAME="sssonector"
POLICY_VERSION="1.0"
TYPE="apps"

# Check if required tools are installed
if ! command -v checkmodule &> /dev/null || ! command -v semodule_package &> /dev/null; then
    echo "Error: SELinux policy tools not found. Please install selinux-policy-devel package."
    exit 1
fi

# Create temporary files
TEMP_FC=$(mktemp)
TEMP_TE=$(mktemp)
TEMP_MOD=$(mktemp)

# Cleanup on exit
trap 'rm -f $TEMP_FC $TEMP_TE $TEMP_MOD' EXIT

# Create file contexts
cat > "$TEMP_FC" << EOF
/usr/local/bin/sssonector    -- system_u:object_r:${POLICY_NAME}_exec_t:s0
/etc/sssonector(/.*)?        -- system_u:object_r:${POLICY_NAME}_conf_t:s0
/var/log/sssonector(/.*)?    -- system_u:object_r:${POLICY_NAME}_log_t:s0
/var/run/sssonector\.pid     -- system_u:object_r:${POLICY_NAME}_var_run_t:s0
EOF

# Create policy module
cat > "$TEMP_TE" << EOF
policy_module(${POLICY_NAME}, ${POLICY_VERSION})

require {
    type unconfined_t;
    type bin_t;
    type etc_t;
    type var_log_t;
    type var_run_t;
    class file { read write create unlink open getattr setattr execute execute_no_trans };
    class dir { read write add_name remove_name search };
    class process { fork transition };
    class capability { net_admin net_raw };
    class tun_socket { create };
    class netlink_route_socket { create bind };
}

# Define custom types
type ${POLICY_NAME}_t;
type ${POLICY_NAME}_exec_t;
type ${POLICY_NAME}_conf_t;
type ${POLICY_NAME}_log_t;
type ${POLICY_NAME}_var_run_t;

# Allow domain transitions
domain_type(${POLICY_NAME}_t)
domain_entry_file(${POLICY_NAME}_t, ${POLICY_NAME}_exec_t)

# Allow network operations
allow ${POLICY_NAME}_t self:capability { net_admin net_raw };
allow ${POLICY_NAME}_t self:tun_socket create;
allow ${POLICY_NAME}_t self:netlink_route_socket { create bind };

# Allow file operations
allow ${POLICY_NAME}_t ${POLICY_NAME}_conf_t:dir { read search };
allow ${POLICY_NAME}_t ${POLICY_NAME}_conf_t:file { read getattr open };
allow ${POLICY_NAME}_t ${POLICY_NAME}_log_t:dir { add_name write };
allow ${POLICY_NAME}_t ${POLICY_NAME}_log_t:file { create write open };
allow ${POLICY_NAME}_t ${POLICY_NAME}_var_run_t:file { create write open unlink };
EOF

# Compile policy module
echo "Compiling SELinux policy module..."
checkmodule -M -m -o "$TEMP_MOD" "$TEMP_TE"

# Create policy package
echo "Creating policy package..."
semodule_package -o "${POLICY_NAME}.pp" -m "$TEMP_MOD" -f "$TEMP_FC"

echo "SELinux policy package created: ${POLICY_NAME}.pp"
