#!/bin/bash
# Script to run the enhanced minimal functionality test script on the QA environment and collect the results

# Set strict error handling
set -e
set -o pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENHANCED_SCRIPT="$SCRIPT_DIR/enhanced_minimal_functionality_test.sh"
LOG_FILE="/tmp/run_qa_tests.log"
RESULTS_DIR="$SCRIPT_DIR/results"

# Extract QA server and client from enhanced script
QA_SERVER=$(grep -E "^QA_SERVER=" "$ENHANCED_SCRIPT" | cut -d'"' -f2)
QA_CLIENT=$(grep -E "^QA_CLIENT=" "$ENHANCED_SCRIPT" | cut -d'"' -f2)
QA_USER=$(grep -E "^QA_USER=" "$ENHANCED_SCRIPT" | cut -d'"' -f2)

# Function to log messages
log() {
    echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1" | tee -a "$LOG_FILE"
}

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check if a file exists
file_exists() {
    [ -f "$1" ]
}

# Function to check if a directory exists
directory_exists() {
    [ -d "$1" ]
}

# Function to check if required tools are installed
check_required_tools() {
    log "Checking if required tools are installed..."
    
    local required_tools=("ssh" "scp")
    local missing_tools=()
    
    for tool in "${required_tools[@]}"; do
        if command_exists "$tool"; then
            log "Tool $tool is installed."
        else
            log "ERROR: Tool $tool is not installed."
            missing_tools+=("$tool")
        fi
    done
    
    if [ ${#missing_tools[@]} -gt 0 ]; then
        log "ERROR: The following tools are missing: ${missing_tools[*]}"
        return 1
    fi
    
    return 0
}

# Function to check if QA environment is accessible
check_qa_environment() {
    log "Checking if QA environment is accessible..."
    
    log "QA server: $QA_SERVER"
    log "QA client: $QA_CLIENT"
    log "QA user: $QA_USER"
    
    # Check if QA server is accessible
    log "Checking if QA server is accessible..."
    if ping -c 1 "$QA_SERVER" >/dev/null 2>&1; then
        log "QA server is accessible."
    else
        log "ERROR: QA server is not accessible."
        return 1
    fi
    
    # Check if QA client is accessible
    log "Checking if QA client is accessible..."
    if ping -c 1 "$QA_CLIENT" >/dev/null 2>&1; then
        log "QA client is accessible."
    else
        log "ERROR: QA client is not accessible."
        return 1
    fi
    
    # Check if SSH to QA server works
    log "Checking if SSH to QA server works..."
    if ssh -o BatchMode=yes -o ConnectTimeout=5 "$QA_USER@$QA_SERVER" "echo 'SSH to QA server works'" >/dev/null 2>&1; then
        log "SSH to QA server works."
    else
        log "ERROR: SSH to QA server does not work."
        return 1
    fi
    
    # Check if SSH to QA client works
    log "Checking if SSH to QA client works..."
    if ssh -o BatchMode=yes -o ConnectTimeout=5 "$QA_USER@$QA_CLIENT" "echo 'SSH to QA client works'" >/dev/null 2>&1; then
        log "SSH to QA client works."
    else
        log "ERROR: SSH to QA client does not work."
        return 1
    fi
    
    return 0
}

# Function to check if enhanced_minimal_functionality_test.sh is deployed to QA environment
check_enhanced_script_deployment() {
    log "Checking if enhanced_minimal_functionality_test.sh is deployed to QA environment..."
    
    # Check if enhanced script is deployed to QA server
    log "Checking if enhanced script is deployed to QA server..."
    if ssh "$QA_USER@$QA_SERVER" "[ -x /opt/sssonector/tools/verification/qa_docs/enhanced_minimal_functionality_test.sh ] && echo 'Enhanced script is deployed to QA server'"; then
        log "Enhanced script is deployed to QA server."
    else
        log "ERROR: Enhanced script is not deployed to QA server."
        return 1
    fi
    
    # Check if enhanced script is deployed to QA client
    log "Checking if enhanced script is deployed to QA client..."
    if ssh "$QA_USER@$QA_CLIENT" "[ -x /opt/sssonector/tools/verification/qa_docs/enhanced_minimal_functionality_test.sh ] && echo 'Enhanced script is deployed to QA client'"; then
        log "Enhanced script is deployed to QA client."
    else
        log "ERROR: Enhanced script is not deployed to QA client."
        return 1
    fi
    
    return 0
}

# Function to create results directory
create_results_directory() {
    log "Creating results directory..."
    
    # Create results directory
    if directory_exists "$RESULTS_DIR"; then
        log "Results directory already exists."
    else
        log "Creating results directory..."
        mkdir -p "$RESULTS_DIR"
    fi
    
    # Create timestamped results directory
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local timestamped_results_dir="$RESULTS_DIR/$timestamp"
    
    log "Creating timestamped results directory: $timestamped_results_dir"
    mkdir -p "$timestamped_results_dir"
    
    echo "$timestamped_results_dir"
}

# Function to run enhanced_minimal_functionality_test.sh on QA server
run_enhanced_script() {
    local results_dir="$1"
    
    log "Running enhanced_minimal_functionality_test.sh on QA server..."
    
    # Run enhanced script on QA server
    log "Running enhanced script on QA server..."
    ssh "$QA_USER@$QA_SERVER" "cd /opt/sssonector/tools/verification/qa_docs && ./enhanced_minimal_functionality_test.sh" | tee "$results_dir/enhanced_script_output.log"
    
    # Check if enhanced script ran successfully
    if [ ${PIPESTATUS[0]} -eq 0 ]; then
        log "Enhanced script ran successfully."
    else
        log "ERROR: Enhanced script failed."
        return 1
    fi
    
    return 0
}

# Function to collect test reports from QA server
collect_test_reports() {
    local results_dir="$1"
    
    log "Collecting test reports from QA server..."
    
    # Create test reports directory
    log "Creating test reports directory..."
    mkdir -p "$results_dir/test_reports"
    
    # Copy test reports from QA server
    log "Copying test reports from QA server..."
    scp "$QA_USER@$QA_SERVER:/tmp/enhanced_sssonector_test_report_*.md" "$results_dir/test_reports/" || true
    
    # Check if test reports were collected
    if [ "$(ls -A "$results_dir/test_reports/")" ]; then
        log "Test reports collected successfully."
    else
        log "WARNING: No test reports were collected."
    fi
    
    return 0
}

# Function to generate summary report
generate_summary_report() {
    local results_dir="$1"
    
    log "Generating summary report..."
    
    # Create summary report
    local summary_report="$results_dir/summary_report.md"
    
    cat > "$summary_report" << EOF
# SSSonector Enhanced Test Summary Report

## Test Information
- **Test Date**: $(date)
- **QA Server**: $QA_SERVER
- **QA Client**: $QA_CLIENT

## Test Results

EOF
    
    # Add test reports to summary report
    for report in "$results_dir/test_reports"/*.md; do
        if [ -f "$report" ]; then
            local report_name=$(basename "$report")
            
            cat >> "$summary_report" << EOF
### Test Report: $report_name

$(cat "$report")

---

EOF
        fi
    done
    
    # Add conclusion to summary report
    cat >> "$summary_report" << EOF
## Conclusion

The enhanced minimal functionality test script was run on the QA environment. See the test reports for detailed results.

## Next Steps

1. Review the test reports and fix any issues
2. Update the documentation based on the test results
3. Integrate the enhanced minimal functionality test script with the CI/CD pipeline
EOF
    
    log "Summary report generated: $summary_report"
    
    return 0
}

# Main function
main() {
    log "Starting run of enhanced minimal functionality test script on QA environment..."
    
    # Check if required tools are installed
    if ! check_required_tools; then
        log "ERROR: Required tools are missing. Please install them before continuing."
        exit 1
    fi
    
    # Check if QA environment is accessible
    if ! check_qa_environment; then
        log "ERROR: QA environment is not accessible. Please fix the issues before continuing."
        exit 1
    fi
    
    # Check if enhanced_minimal_functionality_test.sh is deployed to QA environment
    if ! check_enhanced_script_deployment; then
        log "ERROR: Enhanced script is not deployed to QA environment. Please deploy it before continuing."
        log "You can deploy it using the deploy_to_qa.sh script."
        exit 1
    fi
    
    # Create results directory
    local results_dir=$(create_results_directory)
    
    # Run enhanced_minimal_functionality_test.sh on QA server
    if ! run_enhanced_script "$results_dir"; then
        log "ERROR: Failed to run enhanced script on QA server."
        exit 1
    fi
    
    # Collect test reports from QA server
    if ! collect_test_reports "$results_dir"; then
        log "ERROR: Failed to collect test reports from QA server."
        exit 1
    fi
    
    # Generate summary report
    if ! generate_summary_report "$results_dir"; then
        log "ERROR: Failed to generate summary report."
        exit 1
    fi
    
    log "Run of enhanced minimal functionality test script on QA environment completed successfully."
    log "Results are available in: $results_dir"
    log "Summary report: $results_dir/summary_report.md"
    
    return 0
}

# Run the main function
main
