#!/bin/bash

# Test data generation script for rate limiting tests
SIZES=(
    "1MB"
    "10MB"
    "100MB"
    "1GB"
)

TEST_DATA_DIR="test/test_data"
mkdir -p "$TEST_DATA_DIR"

echo "Generating test data files..."

for size in "${SIZES[@]}"; do
    filename="$TEST_DATA_DIR/test_data_${size}.dat"
    case $size in
        "1MB")   count=1024 ;;
        "10MB")  count=10240 ;;
        "100MB") count=102400 ;;
        "1GB")   count=1048576 ;;
    esac
    
    echo "Generating $size file: $filename"
    dd if=/dev/urandom of="$filename" bs=1024 count=$count status=progress
    sha256sum "$filename" > "${filename}.sha256"
done

echo "Test data generation complete. Files created:"
ls -lh "$TEST_DATA_DIR"
