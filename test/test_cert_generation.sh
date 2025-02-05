#!/bin/bash

# Configuration
LOG_FILE="cert_generation_test.log"
SERVER_SYSTEM="sblanken@192.168.50.210"
CLIENT_SYSTEM="sblanken@192.168.50.211"
DEFAULT_CERT_DIR="/etc/sssonector/certs"
CUSTOM_CERT_DIR="/opt/sssonector/certs"
TEST_PORT=8443

# Logging function
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Verify certificate files exist and have correct permissions
verify_cert_files() {
    local system=$1
    local dir=$2
    local files=("ca.crt" "ca.key" "server.crt" "server.key" "client.crt" "client.key")
    
    log "Verifying certificate files in $dir on $system..."
    
    for file in "${files[@]}"; do
        if ! ssh "$system" "test -f $dir/$file"; then
            log "Error: $file not found in $dir"
            return 1
        fi
        
        # Check permissions
        if [[ $file == *.key ]]; then
            if ! ssh "$system" "test \$(stat -c %a $dir/$file) = '600'"; then
                log "Error: $file has incorrect permissions"
                return 1
            fi
        else
            if ! ssh "$system" "test \$(stat -c %a $dir/$file) = '644'"; then
                log "Error: $file has incorrect permissions"
                return 1
            fi
        fi
    done
    
    return 0
}

# Test default certificate generation
test_default_generation() {
    log "Testing default certificate generation..."
    
    # Clean up any existing certificates
    ssh "$SERVER_SYSTEM" "sudo rm -rf $DEFAULT_CERT_DIR"
    
    # Generate certificates
    if ! ssh "$SERVER_SYSTEM" "sssonector -keygen"; then
        log "Failed to generate certificates in default location"
        return 1
    fi
    
    # Verify files and permissions
    if ! verify_cert_files "$SERVER_SYSTEM" "$DEFAULT_CERT_DIR"; then
        return 1
    fi
    
    log "Default certificate generation test passed"
    return 0
}

# Test custom location certificate generation
test_custom_location() {
    log "Testing custom location certificate generation..."
    
    # Clean up any existing certificates
    ssh "$SERVER_SYSTEM" "sudo rm -rf $CUSTOM_CERT_DIR"
    
    # Generate certificates in custom location
    if ! ssh "$SERVER_SYSTEM" "sssonector -keygen -keyfile $CUSTOM_CERT_DIR"; then
        log "Failed to generate certificates in custom location"
        return 1
    fi
    
    # Verify files and permissions
    if ! verify_cert_files "$SERVER_SYSTEM" "$CUSTOM_CERT_DIR"; then
        return 1
    fi
    
    log "Custom location certificate generation test passed"
    return 0
}

# Test certificate validation
test_cert_validation() {
    log "Testing certificate validation..."
    
    # Generate certificates
    ssh "$SERVER_SYSTEM" "sssonector -keygen"
    
    # Test with valid certificates
    if ! ssh "$SERVER_SYSTEM" "sssonector -mode server -validate-certs"; then
        log "Certificate validation failed for valid certificates"
        return 1
    fi
    
    # Test with invalid permissions
    ssh "$SERVER_SYSTEM" "sudo chmod 666 $DEFAULT_CERT_DIR/*.key"
    if ssh "$SERVER_SYSTEM" "sssonector -mode server -validate-certs" 2>/dev/null; then
        log "Certificate validation should fail with incorrect permissions"
        return 1
    fi
    
    # Test with missing certificate
    ssh "$SERVER_SYSTEM" "sudo rm $DEFAULT_CERT_DIR/server.key"
    if ssh "$SERVER_SYSTEM" "sssonector -mode server -validate-certs" 2>/dev/null; then
        log "Certificate validation should fail with missing certificate"
        return 1
    fi
    
    log "Certificate validation test passed"
    return 0
}

# Test certificate location search
test_cert_location() {
    log "Testing certificate location search..."
    
    # Generate certificates in multiple locations
    ssh "$SERVER_SYSTEM" "sssonector -keygen -keyfile $CUSTOM_CERT_DIR"
    ssh "$SERVER_SYSTEM" "sssonector -keygen"
    
    # Test default location priority
    if ! ssh "$SERVER_SYSTEM" "sssonector -mode server -validate-certs"; then
        log "Failed to locate certificates in default location"
        return 1
    fi
    
    # Test custom location
    ssh "$SERVER_SYSTEM" "sudo rm -rf $DEFAULT_CERT_DIR"
    if ! ssh "$SERVER_SYSTEM" "sssonector -mode server -validate-certs -keyfile $CUSTOM_CERT_DIR"; then
        log "Failed to locate certificates in custom location"
        return 1
    fi
    
    log "Certificate location test passed"
    return 0
}

# Main execution
main() {
    log "Starting certificate generation and location tests..."
    
    # Run default generation test
    if ! test_default_generation; then
        log "Default certificate generation test failed"
        exit 1
    fi
    
    # Run custom location test
    if ! test_custom_location; then
        log "Custom location certificate generation test failed"
        exit 1
    fi
    
    # Run validation test
    if ! test_cert_validation; then
        log "Certificate validation test failed"
        exit 1
    fi
    
    # Run location test
    if ! test_cert_location; then
        log "Certificate location test failed"
        exit 1
    fi
    
    log "All certificate generation and location tests completed successfully"
    exit 0
}

# Start execution
main
