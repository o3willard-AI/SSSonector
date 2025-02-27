#!/bin/bash

# Setup SSSonector binary
# This script builds and distributes the SSSonector binary to test locations

set -euo pipefail

SERVER_IP=${SERVER_IP:-"192.168.50.210"}
CLIENT_IP=${CLIENT_IP:-"192.168.50.211"}
PROJECT_ROOT="/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector"
RELEASE_DIR="$PROJECT_ROOT/bin/release"

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Validate environment and requirements
validate_environment() {
    # Check if Go is installed
    if ! command -v go >/dev/null 2>&1; then
        log_error "Go is not installed"
        exit 1
    fi

    # Check Go version
    local go_version
    go_version=$(go version | awk '{print $3}' | sed 's/go//')
    if ! [[ "$go_version" =~ ^1\.2[0-9]\. ]]; then
        log_warn "Go version $go_version might not be compatible. Recommended: >= 1.20"
    fi

    # Check SSH connectivity
    if ! ssh -o BatchMode=yes -o ConnectTimeout=5 "$SERVER_IP" "echo 'Connection test'" &>/dev/null; then
        log_error "Cannot connect to server $SERVER_IP. Please ensure SSH access is configured"
        exit 1
    fi
    if ! ssh -o BatchMode=yes -o ConnectTimeout=5 "$CLIENT_IP" "echo 'Connection test'" &>/dev/null; then
        log_error "Cannot connect to client $CLIENT_IP. Please ensure SSH access is configured"
        exit 1
    fi

    # Check if build script exists
    if [ ! -f "$PROJECT_ROOT/scripts/build-all.sh" ]; then
        log_error "Build script not found"
        exit 1
    fi
}

# Build binaries using build-all.sh
build_binaries() {
    log_info "Building SSSonector binaries..."
    
    # Clean any previous builds
    rm -rf "$RELEASE_DIR"
    mkdir -p "$RELEASE_DIR"
    
    # Run build script
    cd "$PROJECT_ROOT"
    if ! bash scripts/build-all.sh; then
        log_error "Failed to build binaries"
        return 1
    fi
    
    # Verify build outputs
    if [ ! -f "$RELEASE_DIR/sssonector_"*"_linux_amd64" ]; then
        log_error "Linux binary not found in release directory"
        return 1
    fi
    
    log_info "Binaries built successfully"
    return 0
}

# Function to verify binary checksum
verify_checksum() {
    local binary=$1
    local checksum_file="${binary}.sha256"
    
    if [ ! -f "$checksum_file" ]; then
        log_error "Checksum file not found for $binary"
        return 1
    fi
    
    if command -v sha256sum >/dev/null 2>&1; then
        sha256sum -c "$checksum_file" >/dev/null 2>&1
    else
        shasum -a 256 -c "$checksum_file" >/dev/null 2>&1
    fi
}

# Function to distribute binary to a host
distribute_binary() {
    local host=$1
    local mode=$2
    local arch="amd64"  # Default to amd64
    local max_retries=3
    local retry_count=0
    local temp_dir
    local binary_path
    local version

    log_info "Distributing binary to $host..."
    
    # Get version from git or default
    version=$(cd "$PROJECT_ROOT" && git describe --tags 2>/dev/null || echo "v2.1.0")
    binary_path="$RELEASE_DIR/sssonector_${version}_linux_${arch}"
    
    # Verify binary exists and checksum
    if [ ! -f "$binary_path" ]; then
        log_error "Binary not found: $binary_path"
        return 1
    fi
    
    if ! verify_checksum "$binary_path"; then
        log_error "Checksum verification failed for $binary_path"
        return 1
    fi
    
    # Create secure temporary directory on remote host
    temp_dir=$(ssh "$host" "mktemp -d")
    if [ -z "$temp_dir" ]; then
        log_error "Failed to create temporary directory on $host"
        return 1
    fi

    # Cleanup function
    cleanup() {
        ssh "$host" "rm -rf $temp_dir" || log_warn "Failed to cleanup temporary directory on $host"
    }
    trap cleanup EXIT

    while [ $retry_count -lt $max_retries ]; do
        if scp "$binary_path" "${host}:${temp_dir}/sssonector" &&
           ssh "$host" "
               # Remove old directory if it exists
               rm -rf ~/sssonector
               
               # Create fresh directory structure
               mkdir -p ~/sssonector/bin ~/sssonector/config
               
               # Move binary to bin directory
               mv ${temp_dir}/sssonector ~/sssonector/bin/
               chmod 755 ~/sssonector/bin/sssonector
               
               # Verify binary runs
               if ! ~/sssonector/bin/sssonector -help >/dev/null 2>&1; then
                   echo 'Error: Binary failed basic execution test'
                   exit 1
               fi
           "; then
            log_info "Successfully distributed binary to $host"
            return 0
        fi
        
        retry_count=$((retry_count + 1))
        if [ $retry_count -lt $max_retries ]; then
            log_warn "Attempt $retry_count failed. Retrying in 5 seconds..."
            sleep 5
        fi
    done

    log_error "Failed to distribute binary to $host after $max_retries attempts"
    return 1
}

# Main execution
main() {
    log_info "Starting binary setup..."
    
    # Validate environment first
    validate_environment
    
    # Build binaries
    build_binaries || exit 1
    
    # Distribute binaries
    distribute_binary "$SERVER_IP" "server" || {
        log_error "Failed to distribute binary to server"
        exit 1
    }
    
    distribute_binary "$CLIENT_IP" "client" || {
        log_error "Failed to distribute binary to client"
        exit 1
    }
    
    log_info "Binary setup complete"
}

# Execute main function
main
