#!/bin/bash

# run_investigation.sh
# Master script to run all investigation scripts in the correct order
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

# Create results directory
create_results_dir() {
    local timestamp
    timestamp=$(date +%Y%m%d_%H%M%S)
    local results_dir="results_${timestamp}"
    
    log_info "Creating results directory: ${results_dir}"
    mkdir -p "${results_dir}"
    
    echo "${results_dir}"
}

# Run investigation script and save output
run_investigation() {
    local script=$1
    local results_dir=$2
    local script_name
    script_name=$(basename "${script}" .sh)
    
    log_step "Running ${script_name}"
    
    # Run script and save output
    if "${script}" > "${results_dir}/${script_name}.log" 2>&1; then
        log_info "${script_name} completed successfully"
        echo "SUCCESS" > "${results_dir}/${script_name}.status"
    else
        log_warn "${script_name} failed"
        echo "FAILED" > "${results_dir}/${script_name}.status"
    fi
}

# Generate summary report
generate_summary() {
    local results_dir=$1
    
    log_step "Generating summary report"
    
    # Create summary report
    cat > "${results_dir}/summary.md" << EOF
# SSSonector Investigation Summary

## Overview

This report summarizes the results of the SSSonector investigation conducted on $(date).

## Investigation Results

EOF
    
    # Add results for each investigation
    for status_file in "${results_dir}"/*.status; do
        local script_name
        script_name=$(basename "${status_file}" .status)
        local status
        status=$(cat "${status_file}")
        
        echo "### ${script_name}" >> "${results_dir}/summary.md"
        echo "" >> "${results_dir}/summary.md"
        echo "Status: ${status}" >> "${results_dir}/summary.md"
        echo "" >> "${results_dir}/summary.md"
        echo "#### Key Findings" >> "${results_dir}/summary.md"
        echo "" >> "${results_dir}/summary.md"
        
        # Extract key findings from log file
        grep -E "^\[INFO\]|^\[WARN\]|^\[ERROR\]|^\[STEP\]" "${results_dir}/${script_name}.log" | \
            sed 's/\x1b\[[0-9;]*m//g' | \
            grep -v "Testing SSH connection\|Installing sshpass\|Starting SSSonector\|Stopping SSSonector" | \
            head -n 20 >> "${results_dir}/summary.md"
        
        echo "" >> "${results_dir}/summary.md"
        echo "For detailed results, see [${script_name}.log](${script_name}.log)" >> "${results_dir}/summary.md"
        echo "" >> "${results_dir}/summary.md"
    done
    
    # Add conclusion
    cat >> "${results_dir}/summary.md" << EOF

## Conclusion

Based on the investigation results, the following issues were identified:

1. [Add conclusion based on investigation results]

## Recommendations

Based on the investigation results, the following recommendations are made:

1. [Add recommendations based on investigation results]

EOF
    
    log_info "Summary report generated: ${results_dir}/summary.md"
}

# Main function
main() {
    log_info "Starting SSSonector investigation"
    
    # Create results directory
    local results_dir
    results_dir=$(create_results_dir)
    
    # Run investigations in reverse order
    log_step "Running investigations in reverse order"
    
    # Phase 6: Packet Filtering Investigation
    run_investigation "./investigate_packet_filtering.sh" "${results_dir}"
    
    # Phase 5: MTU Investigation
    run_investigation "./investigate_mtu.sh" "${results_dir}"
    
    # Phase 4: Packet Capture Analysis
    run_investigation "./analyze_packet_capture.sh" "${results_dir}"
    
    # Phase 3: Kernel Parameters Investigation
    run_investigation "./investigate_kernel_parameters.sh" "${results_dir}"
    
    # Phase 2: Routing Tables Verification
    run_investigation "./verify_routing_tables.sh" "${results_dir}"
    
    # Phase 1: Firewall Rules Investigation
    run_investigation "./investigate_firewall_rules.sh" "${results_dir}"
    
    # Generate summary report
    generate_summary "${results_dir}"
    
    log_info "SSSonector investigation completed"
    log_info "Results are available in the ${results_dir} directory"
}

# Run main function
main "$@"
