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

# Ensure sssonector binary is installed
ensure_binary() {
    local system=$1
    log "Building and installing sssonector on $system..."
    
    # Clean up any existing repository
    ssh "$system" "rm -rf /tmp/SSSonector"
    
    # Clone and build
    if ! ssh "$system" "cd /tmp && \
                        git clone https://github.com/o3willard-AI/SSSonector.git && \
                        cd SSSonector && \
                        git checkout main && \
                        git pull && \
                        echo 'require gopkg.in/yaml.v2 v2.4.0' >> go.mod && \
                        echo 'require github.com/sirupsen/logrus v1.9.3' >> go.mod && \
                        echo 'require golang.org/x/crypto v0.17.0' >> go.mod && \
                        echo 'require golang.org/x/sys v0.15.0' >> go.mod && \
                        GOPROXY=direct go mod download && \
                        GOPROXY=direct go mod tidy && \
                        make clean && \
                        GOPROXY=direct make build && \
                        sudo cp bin/sssonector /usr/local/bin/ && \
                        sudo chmod +x /usr/local/bin/sssonector"; then
        log "Failed to build and install sssonector on $system"
        return 1
    fi
    
    return 0
}

# Generate certificates on server
generate_certificates() {
    log "Generating certificates on server..."
    
    # Create certificate directory
    ssh "$SERVER_SYSTEM" "sudo mkdir -p $CERT_DIR && sudo chown \$(whoami):\$(whoami) $CERT_DIR"
    
    # Create test configuration file
    local config_file="/tmp/server.yaml"
    ssh "$SERVER_SYSTEM" "cat > $config_file << EOL
mode: server
network:
  interface: tun0
  mtu: 1500
tunnel:
  listen_address: 0.0.0.0
  listen_port: 8443
  max_clients: 10
logging:
  level: debug
  file: /tmp/sssonector.log
throttle:
  enabled: false
  rate_limit: 1000000
  burst_limit: 2000000
monitor:
  enabled: false
  snmp_enabled: false
  snmp_address: 127.0.0.1
  snmp_port: 161
  snmp_community: public
  snmp_version: 2c
  log_file: /tmp/sssonector_metrics.log
  update_interval: 30
EOL"
    
    # Generate certificates
    if ! ssh "$SERVER_SYSTEM" "sssonector -keygen -config $config_file"; then
        log "Failed to generate certificates"
        return 1
    fi
    
    return 0
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
    
    # Ensure binary is installed on all systems
    if ! ensure_binary "$SERVER_SYSTEM"; then
        log "Failed to install binary on server"
        exit 1
    fi
    
    if ! ensure_binary "$CLIENT_SYSTEM"; then
        log "Failed to install binary on client"
        exit 1
    fi
    
    # Generate certificates on server
    if ! generate_certificates; then
        log "Failed to generate certificates on server"
        exit 1
    fi
    
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
