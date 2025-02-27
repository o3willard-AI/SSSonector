#!/bin/bash
# Script to integrate the enhanced minimal functionality test with the CI/CD pipeline

# Set strict error handling
set -e
set -o pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENHANCED_SCRIPT="$SCRIPT_DIR/enhanced_minimal_functionality_test.sh"
LOG_FILE="/tmp/ci_cd_integration.log"
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
    
    local required_tools=("ssh" "scp" "git")
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

# Function to deploy enhanced_minimal_functionality_test.sh to QA environment
deploy_enhanced_script() {
    log "Deploying enhanced_minimal_functionality_test.sh to QA environment..."
    
    # Run deploy_to_qa.sh
    log "Running deploy_to_qa.sh..."
    "$SCRIPT_DIR/deploy_to_qa.sh"
    
    log "Enhanced script deployed to QA environment."
    
    return 0
}

# Function to run enhanced_minimal_functionality_test.sh on QA environment
run_enhanced_script() {
    log "Running enhanced_minimal_functionality_test.sh on QA environment..."
    
    # Run run_qa_tests.sh
    log "Running run_qa_tests.sh..."
    "$SCRIPT_DIR/run_qa_tests.sh"
    
    log "Enhanced script ran on QA environment."
    
    return 0
}

# Function to get the latest test results
get_latest_test_results() {
    log "Getting latest test results..."
    
    # Get the latest results directory
    local latest_results_dir=$(find "$RESULTS_DIR" -type d -name "[0-9]*" | sort -r | head -n 1)
    
    if [ -z "$latest_results_dir" ]; then
        log "ERROR: No test results found."
        return 1
    fi
    
    log "Latest test results: $latest_results_dir"
    
    # Check if summary report exists
    local summary_report="$latest_results_dir/summary_report.md"
    
    if file_exists "$summary_report"; then
        log "Summary report found: $summary_report"
    else
        log "ERROR: Summary report not found."
        return 1
    fi
    
    # Return the latest results directory
    echo "$latest_results_dir"
}

# Function to check if tests passed
check_tests_passed() {
    local results_dir="$1"
    
    log "Checking if tests passed..."
    
    # Check if summary report exists
    local summary_report="$results_dir/summary_report.md"
    
    if ! file_exists "$summary_report"; then
        log "ERROR: Summary report not found."
        return 1
    fi
    
    # Check if any test failed
    if grep -q "Status: FAIL" "$summary_report"; then
        log "ERROR: Some tests failed."
        return 1
    fi
    
    log "All tests passed."
    
    return 0
}

# Function to generate certification document
generate_certification() {
    local results_dir="$1"
    
    log "Generating certification document..."
    
    # Get version information
    local version=$(git describe --tags --always --dirty)
    local timestamp=$(date +%Y%m%d)
    
    # Create certification document
    local certification_doc="$results_dir/CERTIFICATION_${version}-${timestamp}.md"
    
    cat > "$certification_doc" << EOF
# SSSonector Functionality Certification

## Certification ID: ${version}-${timestamp}

## Test Information
- **Test Date**: $(date)
- **SSSonector Version**: ${version}
- **QA Environment**: ${QA_SERVER}, ${QA_CLIENT}

## Test Results

The SSSonector version ${version} has successfully passed all tests in the enhanced minimal functionality test suite. This certifies that the software meets the requirements for enterprise-grade communications utilities.

## Summary

$(cat "$results_dir/summary_report.md")

## Certification Authority
SSSonector QA Team
$(date)
EOF
    
    log "Certification document generated: $certification_doc"
    
    # Copy certification document to docs directory
    log "Copying certification document to docs directory..."
    cp "$certification_doc" "$(dirname "$SCRIPT_DIR")/../../../docs/"
    
    log "Certification document copied to docs directory."
    
    return 0
}

# Function to update documentation index
update_documentation_index() {
    local version=$(git describe --tags --always --dirty)
    local timestamp=$(date +%Y%m%d)
    
    log "Updating documentation index..."
    
    # Get documentation index file
    local doc_index="$(dirname "$SCRIPT_DIR")/../../../SSSonector_doc_index.md"
    
    if ! file_exists "$doc_index"; then
        log "ERROR: Documentation index not found."
        return 1
    fi
    
    # Add certification document to documentation index
    log "Adding certification document to documentation index..."
    
    # Check if certification document is already in documentation index
    if grep -q "CERTIFICATION_${version}-${timestamp}.md" "$doc_index"; then
        log "Certification document already in documentation index."
    else
        # Add certification document to documentation index
        sed -i "/## Core Documentation/a\\
8. [CERTIFICATION_${version}-${timestamp}.md](docs/CERTIFICATION_${version}-${timestamp}.md)\\
   - Performance Metrics\\
   - Timing Measurements\\
   - Packet Transmission Statistics\\
   - Bandwidth Metrics\\
   - Test Results\\
   - System Configuration\\
   - Security Verification" "$doc_index"
        
        log "Certification document added to documentation index."
    fi
    
    return 0
}

# Function to create GitHub release
create_github_release() {
    local version=$(git describe --tags --always --dirty)
    local timestamp=$(date +%Y%m%d)
    
    log "Creating GitHub release..."
    
    # Check if GitHub CLI is installed
    if ! command_exists "gh"; then
        log "ERROR: GitHub CLI is not installed."
        return 1
    fi
    
    # Check if user is authenticated with GitHub CLI
    if ! gh auth status >/dev/null 2>&1; then
        log "ERROR: Not authenticated with GitHub CLI."
        return 1
    fi
    
    # Create GitHub release
    log "Creating GitHub release..."
    
    # Create release notes
    local release_notes="/tmp/release_notes.md"
    
    cat > "$release_notes" << EOF
# SSSonector ${version}

## Release Information
- **Version**: ${version}
- **Release Date**: $(date)

## Certification
This version has been certified to meet the requirements for enterprise-grade communications utilities. See [CERTIFICATION_${version}-${timestamp}.md](docs/CERTIFICATION_${version}-${timestamp}.md) for details.

## Changes
$(git log --pretty=format:"- %s" $(git describe --tags --abbrev=0)..HEAD)

## Documentation
- [README.md](README.md)
- [PREREQUISITES.md](docs/PREREQUISITES.md)
- [Advanced Configuration Guide](docs/advanced_configuration_guide.md)
- [Troubleshooting Guide](docs/troubleshooting_guide.md)
EOF
    
    # Create GitHub release
    gh release create "${version}" --title "SSSonector ${version}" --notes-file "$release_notes"
    
    log "GitHub release created."
    
    return 0
}

# Main function
main() {
    log "Starting CI/CD integration..."
    
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
    
    # Deploy enhanced_minimal_functionality_test.sh to QA environment
    if ! deploy_enhanced_script; then
        log "ERROR: Failed to deploy enhanced script to QA environment."
        exit 1
    fi
    
    # Run enhanced_minimal_functionality_test.sh on QA environment
    if ! run_enhanced_script; then
        log "ERROR: Failed to run enhanced script on QA environment."
        exit 1
    fi
    
    # Get the latest test results
    local results_dir=$(get_latest_test_results)
    
    if [ -z "$results_dir" ]; then
        log "ERROR: Failed to get latest test results."
        exit 1
    fi
    
    # Check if tests passed
    if ! check_tests_passed "$results_dir"; then
        log "ERROR: Tests failed. Not generating certification document."
        exit 1
    fi
    
    # Generate certification document
    if ! generate_certification "$results_dir"; then
        log "ERROR: Failed to generate certification document."
        exit 1
    fi
    
    # Update documentation index
    if ! update_documentation_index; then
        log "ERROR: Failed to update documentation index."
        exit 1
    fi
    
    # Create GitHub release
    if ! create_github_release; then
        log "ERROR: Failed to create GitHub release."
        exit 1
    fi
    
    log "CI/CD integration completed successfully."
    
    return 0
}

# Run the main function
main
