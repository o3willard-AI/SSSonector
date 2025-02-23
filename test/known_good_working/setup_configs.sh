#!/bin/bash

# Setup configuration files for SSSonector test environment
# This script copies and validates configuration files for server and client machines

set -euo pipefail

SERVER_HOST="sblanken@192.168.50.210"
CLIENT_HOST="sblanken@192.168.50.211"
PROJECT_ROOT="/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector"

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
    # Check SSH connectivity
    if ! ssh -o BatchMode=yes -o ConnectTimeout=5 "$SERVER_HOST" "echo 'Connection test'" &>/dev/null; then
        log_error "Cannot connect to server $SERVER_HOST. Please ensure SSH access is configured"
        exit 1
    fi
    if ! ssh -o BatchMode=yes -o ConnectTimeout=5 "$CLIENT_HOST" "echo 'Connection test'" &>/dev/null; then
        log_error "Cannot connect to client $CLIENT_HOST. Please ensure SSH access is configured"
        exit 1
    fi

    # Check if configuration files exist
    if [ ! -f "$PROJECT_ROOT/configs/server.yaml" ]; then
        log_error "Server configuration file not found"
        exit 1
    fi
    if [ ! -f "$PROJECT_ROOT/configs/client.yaml" ]; then
        log_error "Client configuration file not found"
        exit 1
    fi
}

# Validate YAML syntax
validate_yaml() {
    local file=$1
    if command -v python3 >/dev/null 2>&1; then
        python3 -c "import yaml; yaml.safe_load(open('$file'))" 2>/dev/null || {
            log_error "Invalid YAML syntax in $file"
            return 1
        }
    else
        log_warn "Python3 not found, skipping YAML syntax validation"
    fi
    return 0
}

# Function to setup configuration on a host
setup_host_config() {
    local host=$1
    local mode=$2
    local config_file="$PROJECT_ROOT/configs/${mode}.yaml"
    local temp_dir
    local max_retries=3
    local retry_count=0

    log_info "Setting up $mode configuration on $host..."

    # Create temporary config with substituted values
    local temp_config="/tmp/sssonector_${mode}_config.yaml"
    cp "$config_file" "$temp_config"

    # Validate configuration file
    if ! validate_yaml "$temp_config"; then
        rm -f "$temp_config"
        return 1
    fi

    # Create secure temporary directory on remote host
    temp_dir=$(ssh "$host" "mktemp -d")
    if [ -z "$temp_dir" ]; then
        log_error "Failed to create temporary directory on $host"
        rm -f "$temp_config"
        return 1
    fi

    # Cleanup function
    cleanup() {
        ssh "$host" "rm -rf $temp_dir" || log_warn "Failed to cleanup temporary directory on $host"
        rm -f "$temp_config"
    }
    trap cleanup EXIT

    while [ $retry_count -lt $max_retries ]; do
        if scp "$temp_config" "${host}:${temp_dir}/config.yaml" &&
           ssh "$host" "
               # Create required directory structure
               mkdir -p ~/sssonector/{bin,config,certs,log,state}
               chmod 755 ~/sssonector ~/sssonector/{bin,config,certs,log,state}

               # Move config to sssonector directory
               mv ${temp_dir}/config.yaml ~/sssonector/config/
               chmod 644 ~/sssonector/config/config.yaml

               # Verify directory structure
               if ! test -d ~/sssonector/bin; then
                   echo 'Binary directory not created'
                   exit 1
               fi
               if ! test -d ~/sssonector/config; then
                   echo 'Config directory not created'
                   exit 1
               fi
               if ! test -d ~/sssonector/certs; then
                   echo 'Certificates directory not created'
                   exit 1
               fi
               if ! test -d ~/sssonector/log; then
                   echo 'Log directory not created'
                   exit 1
               fi
               if ! test -d ~/sssonector/state; then
                   echo 'State directory not created'
                   exit 1
               fi

               # Verify config file
               if ! test -f ~/sssonector/config/config.yaml; then
                   echo 'Config file not found'
                   exit 1
               fi

               # List directory structure
               ls -la ~/sssonector/
           "; then
            log_info "Successfully configured $mode on $host"
            return 0
        fi
        
        retry_count=$((retry_count + 1))
        if [ $retry_count -lt $max_retries ]; then
            log_warn "Attempt $retry_count failed. Retrying in 5 seconds..."
            sleep 5
        fi
    done

    log_error "Failed to setup configuration on $host after $max_retries attempts"
    return 1
}

# Main execution
main() {
    log_info "Starting configuration setup..."
    
    # Validate environment first
    validate_environment
    
    # Setup configurations
    setup_host_config "$SERVER_HOST" "server" || {
        log_error "Failed to setup server configuration"
        exit 1
    }
    
    setup_host_config "$CLIENT_HOST" "client" || {
        log_error "Failed to setup client configuration"
        exit 1
    }
    
    log_info "Configuration setup complete and verified"
}

# Execute main function
main
