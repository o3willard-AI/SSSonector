#!/bin/bash
set -e

# Source common functions and environment
if [ -f "$(dirname "$0")/config.env" ]; then
    set -a
    source "$(dirname "$0")/config.env"
    set +a
else
    echo "Error: config.env not found"
    exit 1
fi

# Set paths
BINARY_PATH="../../sssonector"
CONFIG_PATH="/etc/sssonector/config.yaml"
MINIMAL_CONFIG_PATH="../../minimal.yaml"

# Create log directory
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_DIR="startup_verify_${TIMESTAMP}"
mkdir -p "$LOG_DIR"
chmod 777 "$LOG_DIR"

log() {
    local level=$1
    local message=$2
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] [$level] $message" | tee -a "$LOG_DIR/verify.log"
}

check_prerequisites() {
    log "INFO" "Checking system prerequisites..."
    
    # Check TUN device
    if [ ! -c /dev/net/tun ]; then
        log "ERROR" "TUN device not found"
        log "INFO" "Creating TUN device..."
        sudo mkdir -p /dev/net
        sudo mknod /dev/net/tun c 10 200
    fi
    
    # Check TUN permissions
    TUN_PERMS=$(ls -l /dev/net/tun)
    log "INFO" "TUN device permissions: $TUN_PERMS"
    if [[ ! "$TUN_PERMS" =~ "crw-rw----" ]]; then
        log "ERROR" "Incorrect TUN device permissions"
        log "INFO" "Setting correct permissions..."
        sudo chmod 0660 /dev/net/tun
    fi
    
    # Check sssonector group
    if ! getent group sssonector > /dev/null; then
        log "ERROR" "sssonector group not found"
        log "INFO" "Creating sssonector group..."
        sudo groupadd -f sssonector
    fi
    
    # Check TUN group ownership
    TUN_GROUP=$(ls -l /dev/net/tun | awk '{print $4}')
    if [ "$TUN_GROUP" != "sssonector" ]; then
        log "ERROR" "Incorrect TUN device group"
        log "INFO" "Setting correct group..."
        sudo chown root:sssonector /dev/net/tun
    fi
    
    # Check current user group membership
    if ! groups | grep -q sssonector; then
        log "ERROR" "Current user not in sssonector group"
        log "INFO" "Adding user to group..."
        sudo usermod -a -G sssonector $USER
        log "WARNING" "You may need to log out and back in for group changes to take effect"
    fi
    
    # Check binary exists
    if [ ! -f "$BINARY_PATH" ]; then
        log "ERROR" "sssonector binary not found at $BINARY_PATH"
        return 1
    fi
    
    # Check binary capabilities
    CAPS=$(getcap "$BINARY_PATH")
    if [[ ! "$CAPS" =~ "cap_net_admin+ep" ]]; then
        log "ERROR" "Missing required capabilities"
        log "INFO" "Setting capabilities..."
        sudo setcap cap_net_admin+ep "$BINARY_PATH"
    fi
    
    # Check config file
    if [ ! -f "$CONFIG_PATH" ]; then
        log "WARNING" "Config not found at $CONFIG_PATH"
        if [ -f "$MINIMAL_CONFIG_PATH" ]; then
            log "INFO" "Using minimal config from $MINIMAL_CONFIG_PATH"
            sudo mkdir -p "$(dirname "$CONFIG_PATH")"
            sudo cp "$MINIMAL_CONFIG_PATH" "$CONFIG_PATH"
        else
            log "ERROR" "No config file available"
            return 1
        fi
    fi
    
    log "INFO" "Prerequisites check complete"
}

monitor_startup() {
    log "INFO" "Starting server monitoring..."
    
    # Start logging in background
    journalctl -f > "$LOG_DIR/journal.log" 2>&1 &
    JOURNAL_PID=$!
    
    # Start server in foreground
    log "INFO" "Starting server in foreground mode..."
    $BINARY_PATH -debug -config "$CONFIG_PATH" > "$LOG_DIR/server.log" 2>&1 &
    SERVER_PID=$!
    
    # Give the server a moment to start
    sleep 2
    
    # Start strace in background
    sudo strace -f -p $SERVER_PID -o "$LOG_DIR/strace.log" 2>/dev/null &
    STRACE_PID=$!
    
    # Monitor resource usage
    (while true; do
        date >> "$LOG_DIR/resources.log"
        echo "Open files:" >> "$LOG_DIR/resources.log"
        sudo lsof -p $SERVER_PID 2>/dev/null >> "$LOG_DIR/resources.log"
        echo "Memory usage:" >> "$LOG_DIR/resources.log"
        sudo pmap $SERVER_PID 2>/dev/null >> "$LOG_DIR/resources.log"
        echo "Network interfaces:" >> "$LOG_DIR/resources.log"
        ip link >> "$LOG_DIR/resources.log"
        echo "----------------------------------------" >> "$LOG_DIR/resources.log"
        sleep 1
    done) &
    RESOURCE_PID=$!
    
    # Monitor state transitions
    local timeout=30
    local start_time=$(date +%s)
    local current_state=""
    local transitions=0
    
    while true; do
        if ! kill -0 $SERVER_PID 2>/dev/null; then
            log "ERROR" "Server process died"
            break
        fi
        
        local new_state=$(grep "State:" "$LOG_DIR/server.log" | tail -n1 | awk '{print $NF}')
        if [ "$new_state" != "$current_state" ]; then
            log "INFO" "State transition: $current_state -> $new_state"
            current_state=$new_state
            ((transitions++))
        fi
        
        if [ "$current_state" = "Running" ]; then
            log "INFO" "Server reached Running state"
            break
        fi
        
        local current_time=$(date +%s)
        if [ $((current_time - start_time)) -ge $timeout ]; then
            log "ERROR" "Timeout waiting for Running state"
            break
        fi
        
        sleep 1
    done
    
    # Cleanup monitoring processes
    kill $JOURNAL_PID $STRACE_PID $RESOURCE_PID 2>/dev/null || true
    
    # Test cleanup
    log "INFO" "Testing cleanup..."
    kill $SERVER_PID
    sleep 2
    
    # Verify cleanup
    if ip link show tun0 >/dev/null 2>&1; then
        log "ERROR" "TUN interface not cleaned up"
    else
        log "INFO" "TUN interface cleaned up properly"
    fi
    
    if ss -tlpn | grep sssonector >/dev/null 2>&1; then
        log "ERROR" "Sockets not cleaned up"
    else
        log "INFO" "Sockets cleaned up properly"
    fi
    
    if [ -f "/var/lib/sssonector/state/*.lock" ]; then
        log "ERROR" "Lock files not cleaned up"
    else
        log "INFO" "Lock files cleaned up properly"
    fi
}

main() {
    log "INFO" "Starting local startup verification"
    
    # Check prerequisites
    if ! check_prerequisites; then
        log "ERROR" "Prerequisites check failed"
        exit 1
    fi
    
    # Monitor startup
    monitor_startup
    
    log "INFO" "Verification complete. Logs available in: $LOG_DIR"
}

main
