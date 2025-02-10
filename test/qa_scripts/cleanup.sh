#!/bin/bash

# Source environment configuration
if [ -f "$(dirname "$0")/config.env" ]; then
    source "$(dirname "$0")/config.env"
else
    echo "Error: config.env not found"
    exit 1
fi

# Source common functions
if [ -f "$(dirname "$0")/common.sh" ]; then
    source "$(dirname "$0")/common.sh"
else
    echo "Error: common.sh not found"
    exit 1
fi

# Validate environment
validate_qa_env || exit 1

# Function for aggressive cleanup
cleanup_vm() {
    local vm=$1
    log "INFO" "Performing aggressive cleanup on $vm..."

    # Stop systemd service
    remote_cmd $vm "systemctl stop sssonector || true"
    sleep 2

    # Kill any running processes with increasing force
    remote_cmd $vm "pkill -f sssonector || true"
    sleep 1
    remote_cmd $vm "pkill -9 -f sssonector || true"
    sleep 1
    
    # Kill any zombie processes
    remote_cmd $vm "ps -ef | grep defunct | grep sssonector | awk '{ print \$3 }' | xargs -r kill -9"
    sleep 1

    # Force remove TUN interface
    remote_cmd $vm "ip link set tun0 down 2>/dev/null || true"
    remote_cmd $vm "ip link delete tun0 2>/dev/null || true"
    sleep 1

    # Clean up any leftover files
    remote_cmd $vm "rm -f /var/run/sssonector.* 2>/dev/null || true"
    remote_cmd $vm "rm -f /var/lock/sssonector.* 2>/dev/null || true"
    remote_cmd $vm "rm -f /tmp/sssonector.* 2>/dev/null || true"

    # Clear logs
    remote_cmd $vm "echo > /var/log/sssonector/service.log"
    remote_cmd $vm "echo > /var/log/sssonector/server.log"
    remote_cmd $vm "echo > /var/log/sssonector/client.log"

    # Reset systemd
    remote_cmd $vm "systemctl reset-failed sssonector || true"
    remote_cmd $vm "systemctl daemon-reload"

    # Double check for any remaining processes
    for i in {1..3}; do
        local procs=$(remote_cmd $vm "ps aux | grep [s]ssonector || true")
        if [ -n "$procs" ]; then
            log "WARN" "Attempt $i: Found processes on $vm, killing again..."
            remote_cmd $vm "ps aux | grep [s]ssonector | awk '{ print \$2 }' | xargs -r kill -9"
            sleep 2
        else
            break
        fi
    done

    # Final verification
    local procs=$(remote_cmd $vm "ps aux | grep [s]ssonector || true")
    if [ -n "$procs" ]; then
        log "ERROR" "Found leftover processes on $vm:"
        echo "$procs"
        return 1
    fi

    local tun=$(remote_cmd $vm "ip link show tun0 2>/dev/null || true")
    if [ -n "$tun" ]; then
        log "ERROR" "TUN interface still exists on $vm"
        return 1
    fi

    log "INFO" "Cleanup successful on $vm"
    return 0
}

# Main cleanup
log "INFO" "Starting aggressive cleanup..."

cleanup_vm "$QA_SERVER_VM"
cleanup_vm "$QA_CLIENT_VM"

log "INFO" "Cleanup complete"
