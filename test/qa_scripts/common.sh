#!/bin/bash

# Logging function
log() {
    local level=$1
    local message=$2
    echo "[$(date +%Y-%m-%d\ %H:%M:%S)] [$level] $message"
}

# Remote command execution
remote_cmd() {
    local vm=$1
    local cmd=$2
    
    # If running on the target VM, execute locally
    if [[ "$vm" == "$(hostname -I | cut -d' ' -f1)" ]]; then
        if [[ "$cmd" =~ ^(ip|pkill|systemctl|kill|rm) ]]; then
            echo "$QA_SUDO_PASSWORD" | sudo -S bash -c "$cmd"
        elif [[ "$cmd" == *"sssonector"* ]]; then
            echo "$QA_SUDO_PASSWORD" | sudo -S -u sssonector bash -c "$cmd"
        else
            eval "$cmd"
        fi
        return $?
    fi
    
    # Otherwise, execute remotely using sshpass
    local output
    # Create a temporary script to execute remotely
    local tmpfile=$(mktemp)
    echo "#!/bin/bash" > "$tmpfile"
    echo "export QA_SUDO_PASSWORD='$QA_SUDO_PASSWORD'" >> "$tmpfile"
    
    if [[ "$cmd" =~ ^(ip|pkill|systemctl|kill|rm) ]]; then
        echo "echo \"\$QA_SUDO_PASSWORD\" | sudo -S bash -c '$cmd'" >> "$tmpfile"
    elif [[ "$cmd" == *"sssonector"* ]]; then
        echo "echo \"\$QA_SUDO_PASSWORD\" | sudo -S -u sssonector bash -c '$cmd'" >> "$tmpfile"
    else
        echo "$cmd" >> "$tmpfile"
    fi
    
    chmod +x "$tmpfile"
    
    # Execute the script remotely
    output=$(sshpass -p "$QA_SUDO_PASSWORD" ssh -o StrictHostKeyChecking=no "$QA_USER@$vm" "bash -s" < "$tmpfile" 2>&1)
    local status=$?
    
    # Clean up
    rm -f "$tmpfile"
    
    if [ $status -ne 0 ] && [ "$output" != "Connection to $vm closed." ]; then
        log "ERROR" "Command failed on $vm: $cmd"
        log "ERROR" "Output: $output"
    fi
    
    echo "$output"
    return $status
}

# Check command status
check_status() {
    local message=$1
    local status=$?
    if [ $status -eq 0 ]; then
        log "INFO" "[✓] $message"
        return 0
    else
        log "ERROR" "[✗] $message"
        return 1
    fi
}

# Clean up VM state
cleanup_vm() {
    local vm=$1
    log "INFO" "Cleaning up VM state on $vm..."
    
    # Stop service
    remote_cmd $vm "systemctl stop sssonector || true"
    
    # Kill any remaining processes
    log "INFO" "Executing on $vm: pkill -f sssonector"
    remote_cmd $vm "pkill -f sssonector || true"
    sleep 1
    log "INFO" "Executing on $vm: pkill -9 -f sssonector"
    remote_cmd $vm "pkill -9 -f sssonector || true"
    
    # Clean up network interfaces
    log "INFO" "Executing on $vm: ip link delete tun0"
    remote_cmd $vm "ip link delete tun0 || true"
    
    # Clean up logs
    log "INFO" "Executing on $vm: rm -rf /var/log/sssonector/*"
    remote_cmd $vm "rm -rf /var/log/sssonector/* || true"
}
