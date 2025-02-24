#!/bin/bash

# Control script for SSSonector tunnel testing
# This script helps manage the tunnel server and client processes

set -euo pipefail

SERVER_HOST="sblanken@192.168.50.210"
CLIENT_HOST="sblanken@192.168.50.211"
LOCAL_SERVER_IP="127.0.0.1"
LOCAL_SERVER_PORT="8080"

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

# Clean up any existing tunnel processes and interfaces
cleanup() {
    log_info "Running resource cleanup..."
    ./cleanup_resources.sh
}

# Verify and fix config permissions
verify_config() {
    local host=$1
    local mode=$2
    log_info "Verifying $mode config on $host..."

    # Create base directory structure
    if [ "$host" != "localhost" ]; then
        ssh -A -i ~/.ssh/qa_key "$host" "
            sudo rm -rf ~/sssonector
            sudo mkdir -p ~/sssonector/{bin,config,log,state,certs}
            sudo chown -R \$USER:\$USER ~/sssonector
            sudo chmod 755 ~/sssonector
            sudo chmod 755 ~/sssonector/{bin,config,log,state,certs}
        "
    fi

    # Copy binary and set permissions
    if [ "$host" != "localhost" ]; then
        scp -i ~/.ssh/qa_key ../../sssonector-v2.0.0-82-ge5bd185-linux-amd64 "$host:~/sssonector/bin/sssonector" || {
            log_error "Failed to copy binary"
            return 1
        }
        ssh -A -i ~/.ssh/qa_key "$host" "
            sudo chown root:root ~/sssonector/bin/sssonector
            sudo chmod 755 ~/sssonector/bin/sssonector
        "
    fi

    # Copy certificates and set permissions
    if [ "$mode" = "server" ]; then
        if [ "$host" = "localhost" ]; then
            cp ca.crt server.crt server.key .
        else
            # Use scp to transfer certificates
            scp -i ~/.ssh/qa_key ca.crt server.crt server.key "$host:~/sssonector/certs/"
        fi
    else
        if [ "$host" = "localhost" ]; then
            cp ca.crt client.crt client.key .
        else
            # Use scp to transfer certificates
            scp -i ~/.ssh/qa_key ca.crt client.crt client.key "$host:~/sssonector/certs/"
        fi
    fi
    if [ "$host" != "localhost" ]; then
        ssh -A -i ~/.ssh/qa_key "$host" "
            sudo chown root:root ~/sssonector/certs/*
            sudo chmod 644 ~/sssonector/certs/*.crt
            sudo chmod 600 ~/sssonector/certs/*.key
        "
    fi

    # Copy config and set permissions
    if [ "$mode" = "server" ]; then
        if [ "$host" = "localhost" ]; then
            cp ../known_good_working/server_config.yaml .
        else
            scp -i ~/.ssh/qa_key ../known_good_working/server_config.yaml "$host:~/sssonector/config/config.yaml"
        fi
    else
        scp -i ~/.ssh/qa_key ../known_good_working/client_config.yaml "$host:~/sssonector/config/config.yaml"
    fi
    if [ "$host" != "localhost" ]; then
        ssh -A -i ~/.ssh/qa_key "$host" "
            sudo chown \$USER:\$USER ~/sssonector/config/config.yaml
            sudo chmod 644 ~/sssonector/config/config.yaml
        "
    fi

    # Create and set permissions for log and state directories
    if [ "$host" != "localhost" ]; then
        ssh -A -i ~/.ssh/qa_key "$host" "
            sudo chown \$USER:\$USER ~/sssonector/{log,state}
            sudo chmod 755 ~/sssonector/{log,state}
            touch ~/sssonector/log/sssonector.log
            sudo chown \$USER:\$USER ~/sssonector/log/sssonector.log
            sudo chmod 644 ~/sssonector/log/sssonector.log
        "
    fi

    # Verify all files exist with correct permissions
    if [ "$host" != "localhost" ]; then
        if ! ssh -A -i ~/.ssh/qa_key "$host" "
            test -x ~/sssonector/bin/sssonector && \
            test -r ~/sssonector/certs/ca.crt && \
            test -r ~/sssonector/config/config.yaml && \
            test -w ~/sssonector/log/sssonector.log && \
            test -w ~/sssonector/state
        "; then
            log_error "Failed to verify file permissions on $host"
            return 1
        fi
    fi

    # Verify config type and content
    local config_type
    if [ "$host" != "localhost" ]; then
        if ! config_type=$(ssh -A -i ~/.ssh/qa_key "$host" "grep -E '^type:' ~/sssonector/config/config.yaml | cut -d' ' -f2 | tr -d '\"'"); then
            log_error "Failed to read config type from $host"
            return 1
        fi

        if [ "$config_type" != "$mode" ]; then
            log_error "Config type mismatch: expected $mode, got $config_type"
            return 1
        fi
    fi

    # Verify config file permissions
    local file_owner
    if [ "$host" != "localhost" ]; then
        if ! file_owner=$(ssh -A -i ~/.ssh/qa_key "$host" "stat -c %U:%G ~/sssonector/config/config.yaml"); then
            log_error "Failed to check config file ownership on $host"
            return 1
        fi

        if ! ssh -A -i ~/.ssh/qa_key "$host" "test \$(stat -c %U:%G ~/sssonector/config/config.yaml) = \$(echo \$USER):\$(echo \$USER)"; then
            log_error "Incorrect config file ownership: expected \$USER:\$USER, got $file_owner"
            return 1
        fi
    fi

    log_info "$mode config verified successfully"
    return 0
}

# Start the tunnel process
start_tunnel() {
    local host=$1
    local mode=$2
    log_info "Starting $mode on $host..."

    # Verify config first
    if ! verify_config "$host" "$mode"; then
        log_error "Config verification failed for $mode"
        return 1
    fi

    if [ "$host" = "localhost" ]; then
        log_info "Starting $mode locally..."
        sudo ./../../sssonector-v2.0.0-82-ge5bd185-linux-amd64 -config server_config.yaml -debug > startup.log 2>&1 &
        SERVER_PID=$!
        echo "$SERVER_PID" > sssonector.pid
        sleep 2
        if sudo netstat -tlpn | grep -q ":8080.*LISTEN.*sssonector"; then
            log_info "Local server port 8080 is listening"
        else
            log_error "Local server failed to start"
            return 1
        fi
    else
        # Start the process in the foreground and capture PID
        ssh -A -i ~/.ssh/qa_key "$host" "cd ~/sssonector && sudo -b ./bin/sssonector -config config/config.yaml -debug > log/startup.log 2>&1 & echo \$! > ~/sssonector/state/sssonector.pid"

        # Wait for interface and service to be ready
        local retries=0
        local max_retries=10
        while [ $retries -lt $max_retries ]; do
            # Check if process is running and get its PID
            local pid
            pid=$(ssh -A -i ~/.ssh/qa_key "$host" "cat ~/sssonector/state/sssonector.pid 2>/dev/null")
            if [ -z "$pid" ] || ! ssh -A -i ~/.ssh/qa_key "$host" "ps -p $pid > /dev/null 2>&1"; then
                # Check startup log for errors
                local startup_error
                startup_error=$(ssh -A -i ~/.ssh/qa_key "$host" "tail -n 5 ~/sssonector/log/startup.log 2>/dev/null")
                log_error "$mode process not running. Last log entries:"
                echo "$startup_error"
                return 1
            fi

            # For server mode, check if port is listening
            if [ "$mode" = "server" ]; then
                if ssh -A -i ~/.ssh/qa_key "$host" "sudo netstat -tlpn | grep -q ':8080.*LISTEN.*sssonector'"; then
                    log_info "$mode port 8080 is listening"
                    break
                else
                    log_warn "$mode port 8080 not listening yet"
                fi
            fi

            # Check interface status
            if ssh -A -i ~/.ssh/qa_key "$host" "ip link show tun0 | grep -q UP"; then
                if [ "$mode" = "client" ] || [ $retries -gt 5 ]; then
                    log_info "$mode interface is up"
                    break
                else
                    log_warn "$mode interface not up yet"
                    # Show interface status
                    ssh -A -i ~/.ssh/qa_key "$host" "ip addr show tun0" || true
                fi
            fi

            retries=$((retries + 1))
            sleep 2

            # Force interface up if it exists but is down
            if [ $retries -eq 5 ]; then
                ssh -A -i ~/.ssh/qa_key "$host" "sudo ip link set tun0 up 2>/dev/null || true"
            fi
        done

        if [ $retries -eq $max_retries ]; then
            log_error "Failed to start $mode after $max_retries attempts"
            return 1
        fi
    fi

    log_info "$mode started successfully"
    return 0
}

# Check tunnel connectivity with retries
check_connectivity() {
    log_info "Checking tunnel connectivity..."

    local retries=0
    local max_retries=5
    while [ $retries -lt $max_retries ]; do
        if [ "$1" = "localhost" ]; then
            if ping -c 1 -W 2 $LOCAL_SERVER_IP &>/dev/null; then
                log_info "Tunnel connectivity test passed"
                return 0
            fi
        else
            if ssh -A -i ~/.ssh/qa_key "$CLIENT_HOST" "ping -c 1 -W 2 10.0.0.1" &>/dev/null; then
                log_info "Tunnel connectivity test passed"
                return 0
            fi
        fi
        retries=$((retries + 1))
        sleep 2

        # Try to help establish connectivity
        if [ $retries -eq 3 ]; then
            log_info "Attempting to improve connectivity..."
            ssh -A -i ~/.ssh/qa_key "$SERVER_HOST" "sudo ip link set tun0 up" &>/dev/null || true
            ssh -A -i ~/.ssh/qa_key "$CLIENT_HOST" "sudo ip link set tun0 up" &>/dev/null || true
        fi
    done

    log_error "Tunnel connectivity test failed after $max_retries attempts"
    return 1
}

# Show tunnel status
show_status() {
    log_info "=== Server Status ==="
    if [ "$1" = "localhost" ]; then
        echo "Process:"
        ps aux | grep [s]ssonector
        echo
        echo "Interface:"
        ip addr show tun0
        echo
        echo "Connections:"
        sudo netstat -anp | grep sssonector
    else
        ssh -A -i ~/.ssh/qa_key "$SERVER_HOST" "
            echo 'Process:'
            ps aux | grep [s]ssonector
            echo
            echo 'Interface:'
            ip addr show tun0
            echo
            echo 'Connections:'
            sudo netstat -anp | grep sssonector
        "
    fi

    log_info "=== Client Status ==="
    ssh -A -i ~/.ssh/qa_key "$CLIENT_HOST" "
        echo 'Process:'
        ps aux | grep [s]ssonector
        echo
        echo 'Interface:'
        ip addr show tun0
        echo
        echo 'Connections:'
        sudo netstat -anp | grep sssonector
    "
}

# Command line interface
case "${1:-help}" in
    start)
        # Clean up any existing instances
        cleanup

        # Start server and client
        if ! start_tunnel "localhost" "server"; then
            log_error "Failed to start local server"
            exit 1
        fi

        sleep 5  # Wait for server to be ready

        # Check connectivity
        if ! check_connectivity "localhost"; then
            log_error "Connectivity check failed"
            cleanup
            exit 1
        fi

        # Start client on remote host
        if ! start_tunnel "$CLIENT_HOST" "client"; then
            log_error "Failed to start client"
            cleanup
            exit 1
        fi
        ;;

    stop)
        cleanup
        log_info "Tunnel stopped"
        ;;

    status)
        show_status
        ;;

    restart)
        $0 stop
        sleep 2
        $0 start
        ;;

    *)
        echo "Usage: $0 {start|stop|status|restart}"
        echo
        echo "Commands:"
        echo "  start   - Start both server and client"
        echo "  stop    - Stop both server and client"
        echo "  status  - Show tunnel status"
        echo "  restart - Restart both server and client"
        exit 1
        ;;
esac

# Add localhost to cleanup
cleanup_host() {
    local host=$1
    local desc=$2

    log_info "Cleaning up $desc ($host)..."

    if [ "$host" = "localhost" ]; then
        # Local cleanup
        log_info "Cleaning up local resources..."

        # Kill any existing sssonector processes
        sudo pkill -9 -f sssonector || true

        # Remove TUN interface if it exists
        sudo ip link del tun0 2>/dev/null || true

        # Clean up any stale lock files and logs
        sudo rm -f /var/run/sssonector.* 2>/dev/null || true
        sudo rm -f startup.log 2>/dev/null || true
        sudo rm -f sssonector.log 2>/dev/null || true
        sudo rm -f sssonector.pid 2>/dev/null || true
    else
        # Remote cleanup
        # Get PID from state file if it exists
        local pid
        pid=$(ssh "$host" "cat ~/sssonector/state/sssonector.pid 2>/dev/null")

        if [ -n "$pid" ]; then
            # Try graceful shutdown first
            ssh "$host" "sudo kill $pid 2>/dev/null || true"
            sleep 2

            # Force kill if still running
            if ssh "$host" "ps -p $pid > /dev/null 2>&1"; then
                ssh "$host" "sudo kill -9 $pid 2>/dev/null || true"
            fi
        fi

        # Kill any other instances
        ssh "$host" "sudo pkill -f sssonector || true"
        sleep 1
        ssh "$host" "sudo pkill -9 -f sssonector || true"

        # Remove TUN interface if it exists
        ssh "$host" "
            if ip link show tun0 &>/dev/null; then
                sudo ip link set tun0 down
                sudo ip link del tun0
            fi
        "

        # Clean up any stale files and logs
        ssh "$host" "
            # Save last logs for debugging
            if [ -f ~/sssonector/log/startup.log ]; then
                sudo cp ~/sssonector/log/startup.log ~/sssonector/log/startup.log.old
            fi
            if [ -f ~/sssonector/log/sssonector.log ]; then
                sudo cp ~/sssonector/log/sssonector.log ~/sssonector/log/sssonector.log.old
            fi

            # Remove runtime files
            sudo rm -f /var/run/sssonector.* 2>/dev/null || true
            sudo rm -f ~/sssonector/log/startup.log 2>/dev/null || true
            sudo rm -f ~/sssonector/log/sssonector.log 2>/dev/null || true
            sudo rm -f ~/sssonector/state/* 2>/dev/null || true

            # Create fresh log files
            sudo touch ~/sssonector/log/startup.log ~/sssonector/log/sssonector.log
            sudo chown \$USER:\$USER ~/sssonector/log/startup.log ~/sssonector/log/sssonector.log
            sudo chmod 644 ~/sssonector/log/startup.log ~/sssonector/log/sssonector.log
        "
    fi

    # Reset directory permissions
    if [ "$host" = "localhost" ]; then
        log_info "Skipping permission reset for local host"
    else
        ssh "$host" "
            sudo chown -R root:root ~/sssonector/{bin,certs} 2>/dev/null || true
            sudo chmod -R 755 ~/sssonector/{bin,certs} 2>/dev/null || true
            sudo chmod 644 ~/sssonector/certs/*.{crt,key} 2>/dev/null || true
            sudo chown -R \$USER:\$USER ~/sssonector/{config,log,state} 2>/dev/null || true
            sudo chmod -R 755 ~/sssonector/{config,log,state} 2>/dev/null || true
            sudo chmod 644 ~/sssonector/config/config.yaml 2>/dev/null || true
            sudo mkdir -p ~/sssonector/{log,state} 2>/dev/null || true
            sudo chown -R \$USER:\$USER ~/sssonector/{log,state} 2>/dev/null || true
        "
    fi

    # Verify cleanup
    if [ "$host" = "localhost" ]; then
        if pgrep -f sssonector || ip link show tun0 &>/dev/null; then
            log_error "Failed to clean up all resources on $desc"
            return 1
        else
            log_info "$desc cleanup successful"
        fi
    else
        if ssh "$host" "pgrep -f sssonector || ip link show tun0" &>/dev/null; then
            log_error "Failed to clean up all resources on $desc"
            return 1
        else
            log_info "$desc cleanup successful"
        fi
    fi
}

# Main cleanup sequence
main() {
    # Clean up server
    cleanup_host "localhost" "local server"
    
    # Clean up client
    cleanup_host "$CLIENT_HOST" "client"
    
    # Clean up server
    cleanup_host "$SERVER_HOST" "server"

    log_info "All resources cleaned up successfully"
}

# Execute main function
main
