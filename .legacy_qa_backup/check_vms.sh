#!/bin/bash

# Function to test SSH connection
test_ssh() {
    local ip=$1
    echo "Testing connection to $ip..."
    ssh -o ConnectTimeout=5 -o BatchMode=yes -o StrictHostKeyChecking=no -i /home/sblanken/.ssh/qa_key sblanken@$ip exit 2>/dev/null
    if [ $? -eq 0 ]; then
        echo "✅ Successfully connected to $ip"
        return 0
    else
        echo "❌ Failed to connect to $ip"
        return 1
    fi
}

# Test ping first
for ip in 192.168.50.210 192.168.50.211 192.168.50.212; do
    echo -n "Pinging $ip... "
    if ping -c 1 -W 2 $ip >/dev/null 2>&1; then
        echo "✅ Ping successful"
        test_ssh $ip
    else
        echo "❌ Ping failed"
    fi
    echo "-------------------"
done
