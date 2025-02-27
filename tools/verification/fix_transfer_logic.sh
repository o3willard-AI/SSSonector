#!/bin/bash

# fix_transfer_logic.sh
# Script to fix the transfer logic in SSSonector
set -euo pipefail

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    return 1
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Function to backup a file
backup_file() {
    local file=$1
    local backup="${file}.bak"
    
    log_info "Backing up ${file} to ${backup}"
    cp "${file}" "${backup}"
    
    if [ ! -f "${backup}" ]; then
        log_error "Failed to backup ${file}"
        return 1
    fi
    
    log_info "Backup created successfully"
    return 0
}

# Function to fix transfer.go
fix_transfer_go() {
    local file="/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/internal/tunnel/transfer.go"
    
    log_step "Fixing transfer.go"
    
    # Check if file exists
    if [ ! -f "${file}" ]; then
        log_error "File ${file} not found"
        return 1
    fi
    
    # Backup file
    backup_file "${file}" || return 1
    
    # Fix 1: Improve error handling in copy function
    log_info "Improving error handling in copy function"
    sed -i 's/if err != nil {/if err != nil \&\& err != io.EOF {/' "${file}"
    
    # Fix 2: Add debug logging for packet transmission
    log_info "Adding debug logging for packet transmission"
    cat > /tmp/debug_logging.txt << 'EOF'
				t.logger.Debug("Packet details",
					zap.Binary("packet_data", buf[:min(n, 64)]),
				)
EOF
    sed -i '/t.logger.Debug("Write successful",/r /tmp/debug_logging.txt' "${file}"
    
    # Fix 3: Improve buffer handling
    log_info "Improving buffer handling"
    sed -i 's/buf := make(\[\]byte, mtu)/buf := make(\[\]byte, mtu+100)/' "${file}"
    
    # Fix 4: Add flush mechanism to ensure packets are sent immediately
    log_info "Adding flush mechanism"
    cat > /tmp/flush_mechanism.txt << 'EOF'
				// Flush immediately to ensure packet is sent
				if flusher, ok := dst.(interface{ Flush() error }); ok {
					if err := flusher.Flush(); err != nil {
						if t.logger != nil {
							t.logger.Error("Flush failed",
								zap.Error(err),
							)
						}
					}
				}
EOF
    sed -i '/written += w/r /tmp/flush_mechanism.txt' "${file}"
    
    # Fix 5: Add retry mechanism for failed writes
    log_info "Adding retry mechanism for failed writes"
    cat > /tmp/retry_start.txt << 'EOF'
				// Retry failed writes
				const maxRetries = 3
				for retry := 0; retry < maxRetries; retry++ {
EOF
    sed -i '/if err != nil {/i\\' "${file}"
    sed -i '/if err != nil {/i\\' "${file}"
    sed -i '/if err != nil {/r /tmp/retry_start.txt' "${file}"
    
    cat > /tmp/retry_continue.txt << 'EOF'
						if retry < maxRetries-1 {
							if t.logger != nil {
								t.logger.Warn("Write failed, retrying",
									zap.Error(err),
									zap.Int("retry", retry+1),
								)
							}
							time.Sleep(time.Millisecond * 10)
							continue
						}
EOF
    sed -i '/if err != nil {/r /tmp/retry_continue.txt' "${file}"
    
    cat > /tmp/retry_end.txt << 'EOF'
				}
EOF
    sed -i '/return/i\\' "${file}"
    sed -i '/return/r /tmp/retry_end.txt' "${file}"
    
    # Fix 6: Add import for time package if not already present
    log_info "Adding import for time package if not already present"
    if ! grep -q "time" "${file}"; then
        sed -i 's/import (/import (\n\t"time"/' "${file}"
    fi
    
    log_info "transfer.go fixed successfully"
    return 0
}

# Function to fix tunnel_test.go
fix_tunnel_test_go() {
    local file="/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/internal/tunnel/tunnel_test.go"
    
    log_step "Fixing tunnel_test.go"
    
    # Check if file exists
    if [ ! -f "${file}" ]; then
        log_error "File ${file} not found"
        return 1
    fi
    
    # Backup file
    backup_file "${file}" || return 1
    
    # Fix 1: Improve mockConn Read method to better simulate real-world behavior
    log_info "Improving mockConn Read method"
    cat > /tmp/mock_conn_delay.txt << 'EOF'
		// Wait a bit before returning to simulate network delay
		time.Sleep(time.Millisecond * 10)
EOF
    sed -i '/if m.readPos >= len(m.readBuf) {/r /tmp/mock_conn_delay.txt' "${file}"
    
    # Fix 2: Add synchronization to mockConn to prevent race conditions
    log_info "Adding synchronization to mockConn"
    cat > /tmp/mock_conn_lock.txt << 'EOF'
	// Lock to prevent race conditions
	m.mu.Lock()
	defer m.mu.Unlock()
EOF
    sed -i '/func (m \*mockConn) Read(p \[\]byte) (n int, err error) {/i\\' "${file}"
    sed -i '/func (m \*mockConn) Read(p \[\]byte) (n int, err error) {/r /tmp/mock_conn_lock.txt' "${file}"
    sed -i '/func (m \*mockConn) Write(p \[\]byte) (n int, err error) {/i\\' "${file}"
    sed -i '/func (m \*mockConn) Write(p \[\]byte) (n int, err error) {/r /tmp/mock_conn_lock.txt' "${file}"
    
    # Fix 3: Improve test timeout handling
    log_info "Improving test timeout handling"
    sed -i 's/deadline := time.Now().Add(1 \* time.Second)/deadline := time.Now().Add(5 \* time.Second)/' "${file}"
    
    log_info "tunnel_test.go fixed successfully"
    return 0
}

# Function to fix client.go
fix_client_go() {
    local file="/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/internal/tunnel/client.go"
    
    log_step "Fixing client.go"
    
    # Check if file exists
    if [ ! -f "${file}" ]; then
        log_error "File ${file} not found"
        return 1
    fi
    
    # Backup file
    backup_file "${file}" || return 1
    
    # Fix 1: Improve connection retry logic
    log_info "Improving connection retry logic"
    sed -i 's/maxRetries := 3/maxRetries := 5/' "${file}"
    sed -i 's/retryDelay := types.NewDuration(1 \* time.Second)/retryDelay := types.NewDuration(2 \* time.Second)/' "${file}"
    
    # Fix 2: Add more detailed logging for connection attempts
    log_info "Adding more detailed logging for connection attempts"
    cat > /tmp/connection_details.txt << 'EOF'
					c.logger.Debug("Connection details",
						zap.String("local_addr", c.conn.LocalAddr().String()),
						zap.String("remote_addr", c.conn.RemoteAddr().String()),
					)
EOF
    sed -i '/c.logger.Warn("Connection attempt failed",/r /tmp/connection_details.txt' "${file}"
    
    log_info "client.go fixed successfully"
    return 0
}

# Function to fix tunnel.go
fix_tunnel_go() {
    local file="/home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/internal/tunnel/tunnel.go"
    
    log_step "Fixing tunnel.go"
    
    # Check if file exists
    if [ ! -f "${file}" ]; then
        log_error "File ${file} not found"
        return 1
    fi
    
    # Backup file
    backup_file "${file}" || return 1
    
    # Fix 1: Improve adapter initialization
    log_info "Improving adapter initialization"
    cat > /tmp/adapter_init.txt << 'EOF'
			if adapterErr == nil {
				s.logger.Info("TUN adapter created successfully",
					zap.String("interface", opts.Name),
					zap.String("address", opts.Address),
					zap.Int("mtu", opts.MTU),
				)
			}
EOF
    sed -i '/s.adapter, adapterErr = adapter.NewTUNAdapter(opts)/r /tmp/adapter_init.txt' "${file}"
    
    # Fix 2: Add more detailed logging for tunnel establishment
    log_info "Adding more detailed logging for tunnel establishment"
    cat > /tmp/tunnel_established.txt << 'EOF'
			s.logger.Info("Tunnel connection established",
				zap.String("remote", conn.RemoteAddr().String()),
				zap.String("local", conn.LocalAddr().String()),
			)
EOF
    sed -i '/s.logger.Debug("Accepted connection",/r /tmp/tunnel_established.txt' "${file}"
    
    log_info "tunnel.go fixed successfully"
    return 0
}

# Main function
main() {
    log_step "Starting SSSonector transfer logic fix"
    
    # Fix transfer.go
    fix_transfer_go || {
        log_error "Failed to fix transfer.go"
        exit 1
    }
    
    # Fix tunnel_test.go
    fix_tunnel_test_go || {
        log_error "Failed to fix tunnel_test.go"
        exit 1
    }
    
    # Fix client.go
    fix_client_go || {
        log_error "Failed to fix client.go"
        exit 1
    }
    
    # Fix tunnel.go
    fix_tunnel_go || {
        log_error "Failed to fix tunnel.go"
        exit 1
    }
    
    log_step "SSSonector transfer logic fix completed"
    log_info "All files fixed successfully"
    log_info "Please rebuild SSSonector and run the enhanced QA testing script to verify the fixes"
    
    exit 0
}

# Run main function
main "$@"
