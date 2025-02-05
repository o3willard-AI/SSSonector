#!/bin/bash

# Configuration
LOG_DIR="test_logs"
SUMMARY_FILE="$LOG_DIR/test_summary.log"
TEST_SCRIPTS=(
    "test_temp_certs.sh"
    "test_cert_generation.sh"
    "transfer_certs.sh"
)

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging function
log() {
    echo -e "${2:-$NC}[$(date '+%Y-%m-%d %H:%M:%S')] $1${NC}" | tee -a "$SUMMARY_FILE"
}

# Create log directory
mkdir -p "$LOG_DIR"

# Initialize summary file
echo "SSSonector Certificate Management Test Suite" > "$SUMMARY_FILE"
echo "=======================================" >> "$SUMMARY_FILE"
echo "Started at: $(date)" >> "$SUMMARY_FILE"
echo "---------------------------------------" >> "$SUMMARY_FILE"

# Track test results
total_tests=0
passed_tests=0
failed_tests=0

# Run each test script
for script in "${TEST_SCRIPTS[@]}"; do
    total_tests=$((total_tests + 1))
    
    log "Running $script..." "$YELLOW"
    
    # Execute test script
    if ./"$script" > "$LOG_DIR/${script%.sh}.log" 2>&1; then
        log "✓ $script passed" "$GREEN"
        passed_tests=$((passed_tests + 1))
    else
        log "✗ $script failed" "$RED"
        failed_tests=$((failed_tests + 1))
        
        # Show error details
        log "Error details from $script:" "$RED"
        tail -n 10 "$LOG_DIR/${script%.sh}.log" | while read -r line; do
            log "  $line" "$RED"
        done
    fi
    
    echo "---------------------------------------" >> "$SUMMARY_FILE"
done

# Print summary
log "\nTest Summary:"
log "Total tests run: $total_tests"
log "Tests passed:    $passed_tests" "$GREEN"
log "Tests failed:    $failed_tests" "$RED"

# Calculate success rate
success_rate=$(( (passed_tests * 100) / total_tests ))
log "Success rate:    ${success_rate}%"

# Add timestamp
echo "----------------------------------------" >> "$SUMMARY_FILE"
echo "Completed at: $(date)" >> "$SUMMARY_FILE"

# Set exit code based on test results
if [ $failed_tests -eq 0 ]; then
    log "\nAll tests passed successfully!" "$GREEN"
    exit 0
else
    log "\nSome tests failed. Check $SUMMARY_FILE for details." "$RED"
    exit 1
fi
