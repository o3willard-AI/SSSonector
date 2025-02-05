#!/bin/bash

# Configuration
MAX_RETRIES=3
CHECKSUM_FILE="cert_checksums.sha256"
LOG_FILE="cert_transfer.log"
DEV_SYSTEM="sblanken@192.168.50.100"
SERVER_SYSTEM="sblanken@192.168.50.210"
CLIENT_SYSTEM="sblanken@192.168.50.211"
CERT_DIR="/etc/sssonector/certs"
TEMP_DIR="/tmp/sssonector_certs"

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Calculate checksums for certificate files
calculate_checksums() {
    local dir=$1
    (cd "$dir" && sha256sum ca.crt client.crt client.key > "$CHECKSUM_FILE")
}

# Verify checksums match
verify_checksums() {
    local dir=$1
    local checksum_file=$2
    (cd "$dir" && sha256sum -c "$checksum_file")
    return $?
}

# Ensure directory exists and has correct permissions
ensure_directory() {
    local system=$1
    local dir=$2
    ssh "$system" "sudo mkdir -p $dir && sudo chown \$(whoami):\$(whoami) $dir && sudo chmod 755 $dir"
}

# Direct transfer from server to client
direct_transfer() {
    log "Attempting direct transfer from server to client..."
    
    # Create directories
    ensure_directory "$CLIENT_SYSTEM" "$CERT_DIR"
    
    # Try direct SCP
    for ((i=1; i<=MAX_RETRIES; i++)); do
        log "Attempt $i of $MAX_RETRIES..."
        
        # Transfer each file individually to avoid shell expansion issues
        if scp "$SERVER_SYSTEM:$CERT_DIR/ca.crt" "$CLIENT_SYSTEM:$CERT_DIR/" && \
           scp "$SERVER_SYSTEM:$CERT_DIR/client.crt" "$CLIENT_SYSTEM:$CERT_DIR/" && \
           scp "$SERVER_SYSTEM:$CERT_DIR/client.key" "$CLIENT_SYSTEM:$CERT_DIR/"; then
            log "Direct transfer successful"
            
            # Set correct permissions
            ssh "$CLIENT_SYSTEM" "chmod 644 $CERT_DIR/ca.crt $CERT_DIR/client.crt && chmod 600 $CERT_DIR/client.key"
            return 0
        fi
        
        log "Attempt $i failed, waiting before retry..."
        sleep 5
    done
    
    log "All direct transfer attempts failed"
    return 1
}

# Transfer via dev system
intermediary_transfer() {
    log "Attempting transfer via dev system..."
    
    # Create temporary directory
    mkdir -p "$TEMP_DIR"
    
    # Create target directory
    ensure_directory "$CLIENT_SYSTEM" "$CERT_DIR"
    
    # Copy from server to dev system
    log "Copying from server to dev system..."
    if ! scp "$SERVER_SYSTEM:$CERT_DIR/ca.crt" "$TEMP_DIR/" || \
       ! scp "$SERVER_SYSTEM:$CERT_DIR/client.crt" "$TEMP_DIR/" || \
       ! scp "$SERVER_SYSTEM:$CERT_DIR/client.key" "$TEMP_DIR/"; then
        log "Failed to copy from server to dev system"
        rm -rf "$TEMP_DIR"
        return 1
    fi
    
    # Calculate checksums on dev system
    log "Calculating checksums on dev system..."
    calculate_checksums "$TEMP_DIR"
    
    # Copy from dev system to client
    log "Copying from dev system to client..."
    if ! scp "$TEMP_DIR/ca.crt" "$CLIENT_SYSTEM:$CERT_DIR/" || \
       ! scp "$TEMP_DIR/client.crt" "$CLIENT_SYSTEM:$CERT_DIR/" || \
       ! scp "$TEMP_DIR/client.key" "$CLIENT_SYSTEM:$CERT_DIR/" || \
       ! scp "$TEMP_DIR/$CHECKSUM_FILE" "$CLIENT_SYSTEM:$CERT_DIR/"; then
        log "Failed to copy from dev system to client"
        rm -rf "$TEMP_DIR"
        return 1
    fi
    
    # Verify checksums on client
    log "Verifying checksums on client..."
    if ! ssh "$CLIENT_SYSTEM" "cd $CERT_DIR && sha256sum -c $CHECKSUM_FILE"; then
        log "Checksum verification failed on client"
        rm -rf "$TEMP_DIR"
        return 1
    fi
    
    # Set correct permissions
    ssh "$CLIENT_SYSTEM" "chmod 644 $CERT_DIR/ca.crt $CERT_DIR/client.crt && chmod 600 $CERT_DIR/client.key"
    
    # Cleanup
    rm -rf "$TEMP_DIR"
    
    log "Intermediary transfer successful"
    return 0
}

# Main execution
main() {
    log "Starting certificate transfer process..."
    
    # Try direct transfer first
    if direct_transfer; then
        log "Certificate transfer completed successfully via direct transfer"
        exit 0
    fi
    
    log "Direct transfer failed, attempting intermediary transfer..."
    
    # Try intermediary transfer
    if intermediary_transfer; then
        log "Certificate transfer completed successfully via intermediary transfer"
        exit 0
    fi
    
    log "All transfer methods failed"
    exit 1
}

# Start execution
main
