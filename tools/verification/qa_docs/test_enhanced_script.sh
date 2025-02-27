#!/bin/bash
# Script to test the enhanced minimal functionality test script in a development environment

# Set strict error handling
set -e
set -o pipefail

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ENHANCED_SCRIPT="$SCRIPT_DIR/enhanced_minimal_functionality_test.sh"
LOG_FILE="/tmp/test_enhanced_script.log"

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

# Function to check if a script is executable
is_executable() {
    [ -x "$1" ]
}

# Function to check if a script is valid bash
is_valid_bash() {
    bash -n "$1" >/dev/null 2>&1
}

# Function to check if a script can be sourced
can_be_sourced() {
    # Create a temporary script that sources the target script
    local temp_script=$(mktemp)
    echo "#!/bin/bash" > "$temp_script"
    echo "source \"$1\" source_only" >> "$temp_script"
    chmod +x "$temp_script"
    
    # Try to execute the temporary script
    if bash "$temp_script" >/dev/null 2>&1; then
        rm "$temp_script"
        return 0
    else
        rm "$temp_script"
        return 1
    fi
}

# Function to check if minimal_functionality_test.sh exists
check_minimal_functionality_test() {
    local minimal_script="$SCRIPT_DIR/../minimal_functionality_test.sh"
    
    log "Checking if minimal_functionality_test.sh exists..."
    if file_exists "$minimal_script"; then
        log "minimal_functionality_test.sh exists."
        
        log "Checking if minimal_functionality_test.sh is executable..."
        if is_executable "$minimal_script"; then
            log "minimal_functionality_test.sh is executable."
        else
            log "ERROR: minimal_functionality_test.sh is not executable."
            return 1
        fi
        
        log "Checking if minimal_functionality_test.sh is valid bash..."
        if is_valid_bash "$minimal_script"; then
            log "minimal_functionality_test.sh is valid bash."
        else
            log "ERROR: minimal_functionality_test.sh is not valid bash."
            return 1
        fi
    else
        log "ERROR: minimal_functionality_test.sh does not exist."
        return 1
    fi
    
    return 0
}

# Function to check if enhanced_minimal_functionality_test.sh is valid
check_enhanced_script() {
    log "Checking if enhanced_minimal_functionality_test.sh exists..."
    if file_exists "$ENHANCED_SCRIPT"; then
        log "enhanced_minimal_functionality_test.sh exists."
        
        log "Checking if enhanced_minimal_functionality_test.sh is executable..."
        if is_executable "$ENHANCED_SCRIPT"; then
            log "enhanced_minimal_functionality_test.sh is executable."
        else
            log "ERROR: enhanced_minimal_functionality_test.sh is not executable."
            return 1
        fi
        
        log "Checking if enhanced_minimal_functionality_test.sh is valid bash..."
        if is_valid_bash "$ENHANCED_SCRIPT"; then
            log "enhanced_minimal_functionality_test.sh is valid bash."
        else
            log "ERROR: enhanced_minimal_functionality_test.sh is not valid bash."
            return 1
        fi
    else
        log "ERROR: enhanced_minimal_functionality_test.sh does not exist."
        return 1
    fi
    
    return 0
}

# Function to check if required tools are installed
check_required_tools() {
    log "Checking if required tools are installed..."
    
    local required_tools=("ssh" "scp" "grep" "sed" "awk" "cat" "ping" "nc" "curl" "python3")
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
    
    # Extract QA server and client from enhanced script
    local qa_server=$(grep -E "^QA_SERVER=" "$ENHANCED_SCRIPT" | cut -d'"' -f2)
    local qa_client=$(grep -E "^QA_CLIENT=" "$ENHANCED_SCRIPT" | cut -d'"' -f2)
    local qa_user=$(grep -E "^QA_USER=" "$ENHANCED_SCRIPT" | cut -d'"' -f2)
    
    log "QA server: $qa_server"
    log "QA client: $qa_client"
    log "QA user: $qa_user"
    
    # Check if QA server is accessible
    log "Checking if QA server is accessible..."
    if ping -c 1 "$qa_server" >/dev/null 2>&1; then
        log "QA server is accessible."
    else
        log "WARNING: QA server is not accessible. This is expected in a development environment."
    fi
    
    # Check if QA client is accessible
    log "Checking if QA client is accessible..."
    if ping -c 1 "$qa_client" >/dev/null 2>&1; then
        log "QA client is accessible."
    else
        log "WARNING: QA client is not accessible. This is expected in a development environment."
    fi
    
    return 0
}

# Function to check if test functions are valid
check_test_functions() {
    log "Checking if test functions are valid..."
    
    # Get the list of test functions
    local test_functions=$(grep -E "^test_[a-zA-Z0-9_]+\(\)" "$ENHANCED_SCRIPT" | sed 's/().*//' || echo "")
    
    if [ -z "$test_functions" ]; then
        log "ERROR: No test functions found in enhanced_minimal_functionality_test.sh."
        return 1
    fi
    
    log "Found the following test functions:"
    for func in $test_functions; do
        log "  - $func"
    done
    
    return 0
}

# Function to check if run_test function is valid
check_run_test_function() {
    log "Checking if run_test function is valid..."
    
    # Check if run_test function exists
    if grep -q "^run_test()" "$ENHANCED_SCRIPT"; then
        log "run_test function exists."
    else
        log "ERROR: run_test function does not exist."
        return 1
    fi
    
    return 0
}

# Function to check if initialize_test_report function is valid
check_initialize_test_report_function() {
    log "Checking if initialize_test_report function is valid..."
    
    # Check if initialize_test_report function exists
    if grep -q "^initialize_test_report()" "$ENHANCED_SCRIPT"; then
        log "initialize_test_report function exists."
    else
        log "ERROR: initialize_test_report function does not exist."
        return 1
    fi
    
    return 0
}

# Function to check if update_test_report function is valid
check_update_test_report_function() {
    log "Checking if update_test_report function is valid..."
    
    # Check if update_test_report function exists
    if grep -q "^update_test_report()" "$ENHANCED_SCRIPT"; then
        log "update_test_report function exists."
    else
        log "ERROR: update_test_report function does not exist."
        return 1
    fi
    
    return 0
}

# Function to check if get_version function is valid
check_get_version_function() {
    log "Checking if get_version function is valid..."
    
    # Check if get_version function exists
    if grep -q "^get_version()" "$ENHANCED_SCRIPT"; then
        log "get_version function exists."
    else
        log "ERROR: get_version function does not exist."
        return 1
    fi
    
    return 0
}

# Main function
main() {
    log "Starting test of enhanced minimal functionality test script..."
    
    # Check if minimal_functionality_test.sh exists and is valid
    if ! check_minimal_functionality_test; then
        log "ERROR: minimal_functionality_test.sh is not valid. Please fix it before continuing."
        exit 1
    fi
    
    # Check if enhanced_minimal_functionality_test.sh is valid
    if ! check_enhanced_script; then
        log "ERROR: enhanced_minimal_functionality_test.sh is not valid. Please fix it before continuing."
        exit 1
    fi
    
    # Check if required tools are installed
    if ! check_required_tools; then
        log "ERROR: Required tools are missing. Please install them before continuing."
        exit 1
    fi
    
    # Check if QA environment is accessible
    check_qa_environment
    
    # Check if test functions are valid
    if ! check_test_functions; then
        log "ERROR: Test functions are not valid. Please fix them before continuing."
        exit 1
    fi
    
    # Check if run_test function is valid
    if ! check_run_test_function; then
        log "ERROR: run_test function is not valid. Please fix it before continuing."
        exit 1
    fi
    
    # Check if initialize_test_report function is valid
    if ! check_initialize_test_report_function; then
        log "ERROR: initialize_test_report function is not valid. Please fix it before continuing."
        exit 1
    fi
    
    # Check if update_test_report function is valid
    if ! check_update_test_report_function; then
        log "ERROR: update_test_report function is not valid. Please fix it before continuing."
        exit 1
    fi
    
    # Check if get_version function is valid
    if ! check_get_version_function; then
        log "ERROR: get_version function is not valid. Please fix it before continuing."
        exit 1
    fi
    
    log "All checks passed. The enhanced minimal functionality test script is valid."
    log "To deploy the script to the QA environment, run the deploy_to_qa.sh script."
    
    return 0
}

# Run the main function
main
