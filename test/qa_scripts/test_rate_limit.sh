#!/bin/bash

# Start monitoring system
echo "Starting monitoring system..."
python3 monitor_ftp_transfer.py &
MONITOR_PID=$!

# Wait for monitoring system to start
sleep 5

echo "Starting server-to-client transfer test..."
# Test server to client transfer
ftp -n 192.168.50.210 << EOF
user test test
binary
get /home/test/test_200mb /tmp/test_download_1
quit
EOF

echo "Waiting 10 seconds for metrics collection..."
sleep 10

echo "Starting client-to-server transfer test..."
# Test client to server transfer
ftp -n 192.168.50.210 << EOF
user test test
binary
put /tmp/test_download_1 /home/test/test_upload_1
quit
EOF

echo "Waiting 10 seconds for final metrics collection..."
sleep 10

# Clean up test files
rm -f /tmp/test_download_1

# Kill monitoring system
kill $MONITOR_PID

echo "Tests completed. The monitoring interface should be available at http://localhost:8080"
