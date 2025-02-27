# SSSonector Investigation Guide

This guide provides comprehensive instructions for investigating and troubleshooting issues with the SSSonector tunnel. It documents the systematic approach to identifying and resolving connectivity problems in SSSonector deployments.

## Overview

The SSSonector investigation tools implement a methodical troubleshooting approach designed to systematically identify and resolve issues with the SSSonector tunnel. The investigation follows a bottom-up approach, starting with the most fundamental network components and progressing to more complex aspects of the tunnel.

The investigation is divided into six phases, each focusing on a specific aspect of the tunnel:

1. **Firewall Rules Investigation**: Checks and adds firewall rules for ICMP traffic.
2. **Routing Tables Verification**: Verifies and fixes routing tables.
3. **Kernel Parameters Investigation**: Checks and adjusts kernel parameters.
4. **Packet Capture Analysis**: Captures and analyzes packets to identify where packets are lost.
5. **MTU Investigation**: Tests with different MTU values and Path MTU Discovery settings.
6. **Packet Filtering Investigation**: Checks for packet filtering rules that might be blocking traffic.

This phased approach ensures that each potential issue is thoroughly investigated, and the results of each phase inform the subsequent phases. The investigation tools are designed to be run in sequence, but can also be run individually to focus on specific aspects of the tunnel.

## Prerequisites

Before running the investigation scripts, ensure that:

1. You have SSH access to the QA servers.
2. You have sudo privileges on the QA servers.
3. The `sshpass` utility is installed on your local machine.
4. SSSonector is installed on the QA servers.

## Running the Investigation

### Option 1: Run the Complete Investigation

To run the complete investigation, use the `run_investigation.sh` script:

```bash
cd /home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/tools/verification
./run_investigation.sh
```

This script will:

1. Run all investigation scripts in the correct order.
2. Save the output of each script to a results directory.
3. Generate a summary report.

### Option 2: Run Individual Investigation Phases

You can also run individual investigation phases:

```bash
# Phase 1: Firewall Rules Investigation
./investigate_firewall_rules.sh

# Phase 2: Routing Tables Verification
./verify_routing_tables.sh

# Phase 3: Kernel Parameters Investigation
./investigate_kernel_parameters.sh

# Phase 4: Packet Capture Analysis
./analyze_packet_capture.sh

# Phase 5: MTU Investigation
./investigate_mtu.sh

# Phase 6: Packet Filtering Investigation
./investigate_packet_filtering.sh
```

## Investigation Scripts in Detail

### 1. investigate_firewall_rules.sh

This script investigates and adds firewall rules for ICMP traffic, which is essential for ping functionality.

**Purpose**: To identify and resolve firewall-related issues that may be blocking ICMP traffic between the SSSonector server and client.

**Methodology**:
- Checks current firewall rules on both server and client to identify any existing rules that might affect ICMP traffic.
- Tests baseline connectivity to establish a reference point.
- Adds ICMP rules to the FORWARD chain to allow ICMP traffic to be forwarded between interfaces.
- Tests connectivity after adding FORWARD rules to measure the impact.
- Adds ICMP rules to the INPUT chain to allow ICMP traffic to be received by the tunnel interfaces.
- Tests connectivity after adding INPUT rules to measure the impact.

**Common Issues Addressed**:
- Missing FORWARD rules for ICMP traffic
- Missing INPUT rules for ICMP traffic
- Default DROP policies that block ICMP traffic
- Specific REJECT rules targeting ICMP traffic

**Interpreting Results**:
- If connectivity improves after adding FORWARD rules, the issue was likely related to packet forwarding.
- If connectivity improves after adding INPUT rules, the issue was likely related to packet reception.
- If connectivity does not improve after adding both types of rules, the issue is likely not firewall-related.

### 2. verify_routing_tables.sh

This script verifies and fixes routing tables, which are essential for proper packet routing.

**Purpose**: To identify and resolve routing-related issues that may be preventing packets from being properly routed between the SSSonector server and client.

**Methodology**:
- Checks current routing tables on both server and client to identify any missing or incorrect routes.
- Tests baseline connectivity to establish a reference point.
- Fixes routing tables by adding or correcting routes for the tunnel network (10.0.0.0/24).
- Tests connectivity after fixing routing tables to measure the impact.

**Common Issues Addressed**:
- Missing routes for the tunnel network
- Incorrect source IP for tunnel routes
- Missing default routes
- Routing conflicts

**Interpreting Results**:
- If connectivity improves after fixing routing tables, the issue was likely related to routing.
- If connectivity does not improve after fixing routing tables, the issue is likely not routing-related.

### 3. investigate_kernel_parameters.sh

This script investigates and adjusts kernel parameters that affect network behavior.

**Purpose**: To identify and resolve kernel parameter issues that may be affecting tunnel functionality.

**Methodology**:
- Checks current kernel parameters related to IP forwarding, reverse path filtering, ICMP handling, and source routing.
- Tests baseline connectivity to establish a reference point.
- Adjusts kernel parameters to optimal values for tunnel operation.
- Tests connectivity after adjusting kernel parameters to measure the impact.

**Common Issues Addressed**:
- Disabled IP forwarding
- Strict reverse path filtering
- ICMP echo ignore settings
- Disabled source routing

**Interpreting Results**:
- If connectivity improves after adjusting kernel parameters, the issue was likely related to kernel configuration.
- If connectivity does not improve after adjusting kernel parameters, the issue is likely not kernel-related.

### 4. analyze_packet_capture.sh

This script captures and analyzes packets to identify where packets are being lost or dropped.

**Purpose**: To gain visibility into the packet flow through the tunnel and identify exactly where packets are being lost.

**Methodology**:
- Starts packet capture on tun0 interfaces on both server and client to monitor tunnel traffic.
- Starts packet capture on physical interfaces on both server and client to monitor physical network traffic.
- Generates test traffic (ICMP and HTTP) to exercise the tunnel.
- Analyzes packet captures to identify where packets are being lost or dropped.

**Common Issues Addressed**:
- Packets not entering the tunnel
- Packets not exiting the tunnel
- Packets being dropped within the tunnel
- Malformed packets
- Protocol-specific issues

**Interpreting Results**:
- If packets are seen entering but not exiting the tunnel, the issue is likely within the tunnel.
- If packets are not seen entering the tunnel, the issue is likely before the tunnel (routing, firewall, etc.).
- If packets are seen exiting the tunnel but not reaching their destination, the issue is likely after the tunnel.

### 5. investigate_mtu.sh

This script investigates MTU (Maximum Transmission Unit) issues that can cause packet fragmentation and loss.

**Purpose**: To identify and resolve MTU-related issues that may be causing packet fragmentation and loss.

**Methodology**:
- Checks current MTU settings on all interfaces to identify potential mismatches.
- Tests with different MTU values (1500, 1400, 1200, 1000, 576) to find an optimal value.
- Tests with Path MTU Discovery enabled and disabled to determine its impact.
- Tests ping with different packet sizes to identify the maximum working packet size.

**Common Issues Addressed**:
- MTU mismatches between interfaces
- Path MTU Discovery issues
- Packet fragmentation
- Maximum packet size limitations

**Interpreting Results**:
- If connectivity improves with a lower MTU, the issue was likely related to packet fragmentation.
- If connectivity improves with Path MTU Discovery disabled, the issue was likely related to ICMP blocking.
- If connectivity improves only with specific packet sizes, the issue was likely related to maximum packet size limitations.

### 6. investigate_packet_filtering.sh

This script checks for packet filtering rules that might be blocking tunnel traffic.

**Purpose**: To identify and resolve packet filtering issues that may be blocking tunnel traffic.

**Methodology**:
- Checks for DROP or REJECT rules in iptables that might affect tunnel traffic.
- Checks for ebtables or arptables rules that might affect tunnel traffic.
- Tests baseline connectivity to establish a reference point.
- Temporarily disables filtering rules to determine their impact.
- Tests connectivity with filtering disabled to measure the impact.
- Restores filtering rules to maintain security.
- Adds exceptions for tunnel traffic to allow it through the filters.
- Tests connectivity with tunnel exceptions to measure the impact.

**Common Issues Addressed**:
- Explicit DROP or REJECT rules targeting tunnel traffic
- Default DROP policies affecting tunnel traffic
- ebtables or arptables rules affecting tunnel traffic
- nftables rules affecting tunnel traffic

**Interpreting Results**:
- If connectivity improves with filtering disabled, the issue was likely related to packet filtering.
- If connectivity improves with tunnel exceptions, the issue was likely related to specific filtering rules.
- If connectivity does not improve with filtering disabled, the issue is likely not filtering-related.

## Results and Interpretation

The investigation results are saved to a timestamped results directory with the following structure:

```
results_YYYYMMDD_HHMMSS/
  ├── investigate_firewall_rules.log
  ├── investigate_firewall_rules.status
  ├── verify_routing_tables.log
  ├── verify_routing_tables.status
  ├── investigate_kernel_parameters.log
  ├── investigate_kernel_parameters.status
  ├── analyze_packet_capture.log
  ├── analyze_packet_capture.status
  ├── investigate_mtu.log
  ├── investigate_mtu.status
  ├── investigate_packet_filtering.log
  ├── investigate_packet_filtering.status
  └── summary.md
```

### Log Files

Each `.log` file contains the detailed output of the corresponding investigation script. These logs include:

- All commands executed on the QA servers
- The output of those commands
- Test results at each step
- Error messages and warnings
- Diagnostic information

The log files are useful for detailed analysis and debugging. They contain all the raw data collected during the investigation.

### Status Files

Each `.status` file contains a single word indicating the status of the corresponding investigation script:

- `SUCCESS`: The script completed successfully
- `FAILED`: The script encountered an error and did not complete

The status files are used by the summary generator to indicate which investigations were successful.

### Summary Report

The `summary.md` file contains a summary of the investigation results, including:

- An overview of the investigation
- The status of each investigation phase
- Key findings from each investigation phase
- Conclusions based on the investigation results
- Recommendations for resolving the identified issues

The summary report is designed to be human-readable and provides a high-level overview of the investigation results. It is the primary document for understanding the investigation results and determining the next steps.

### Interpreting the Summary Report

The summary report is organized into sections for each investigation phase. For each phase, it includes:

1. **Status**: Whether the investigation was successful or failed.
2. **Key Findings**: The most important observations from the investigation.
3. **Link to Detailed Log**: A link to the detailed log file for more information.

The conclusion section of the summary report provides an overall assessment of the investigation results and identifies the most likely causes of the issues. The recommendations section provides specific actions to resolve the identified issues.

When interpreting the summary report, focus on:

1. **Phases with Improved Connectivity**: If connectivity improved after a specific phase, that phase likely addressed the root cause.
2. **Consistent Patterns**: Look for patterns across multiple phases that point to a common issue.
3. **Unexpected Results**: Pay attention to unexpected results that might indicate complex or interrelated issues.
4. **Recommendations**: The recommendations are based on the investigation results and provide concrete actions to resolve the issues.

## Advanced Usage

### Customizing the Investigation

The investigation scripts can be customized to focus on specific aspects of the tunnel or to adapt to different environments. Common customizations include:

1. **Modifying QA Server Details**: Edit the QA_SERVER, QA_CLIENT, QA_USER, and QA_SUDO_PASSWORD variables at the top of each script to point to your specific QA environment.

2. **Adjusting Test Parameters**: Modify the test parameters (e.g., ping count, MTU values) to suit your specific requirements.

3. **Adding Custom Tests**: Add custom tests to the scripts to address specific issues in your environment.

4. **Changing the Order of Investigations**: Modify the `run_investigation.sh` script to change the order in which the investigations are run.

### Integration with CI/CD Pipelines

The investigation scripts can be integrated into CI/CD pipelines to automatically test tunnel connectivity after deployments. To integrate with CI/CD:

1. **Automate SSH Access**: Set up SSH keys for passwordless access to the QA servers.

2. **Modify Scripts for Non-Interactive Use**: Remove any interactive elements from the scripts and ensure they exit with appropriate status codes.

3. **Parse Summary Report**: Write a parser for the summary report to extract key information for the CI/CD pipeline.

4. **Set Up Notifications**: Configure notifications for failed investigations.

### Integration with Other SSSonector Tools

The investigation tools are designed to work alongside other SSSonector tools:

1. **Deployment Tools**: Use the investigation tools after deploying SSSonector to verify connectivity.

2. **Monitoring Tools**: Use the investigation tools to troubleshoot issues identified by monitoring tools.

3. **Performance Testing Tools**: Use the investigation tools to identify and resolve performance issues.

4. **Security Testing Tools**: Use the investigation tools to verify that security measures do not interfere with tunnel functionality.

## Troubleshooting

If you encounter issues running the investigation scripts:

1. Ensure that all scripts have execute permissions:
   ```bash
   chmod +x *.sh
   ```

2. Check that you have SSH access to the QA servers:
   ```bash
   ssh sblanken@192.168.50.210
   ssh sblanken@192.168.50.211
   ```

3. Check that you have sudo privileges on the QA servers:
   ```bash
   ssh sblanken@192.168.50.210 "sudo -v"
   ssh sblanken@192.168.50.211 "sudo -v"
   ```

4. Check that SSSonector is installed on the QA servers:
   ```bash
   ssh sblanken@192.168.50.210 "ls -l /opt/sssonector/bin/sssonector"
   ssh sblanken@192.168.50.211 "ls -l /opt/sssonector/bin/sssonector"
   ```

## Common Troubleshooting Scenarios

Here are some common troubleshooting scenarios and their solutions:

### Scenario 1: Ping Fails Between Tunnel Endpoints

**Symptoms**:
- Ping from client to server (10.0.0.1) fails
- Ping from server to client (10.0.0.2) fails
- Tunnel interfaces are up
- SSSonector processes are running

**Investigation Approach**:
1. Run `investigate_firewall_rules.sh` to check for firewall issues
2. Run `verify_routing_tables.sh` to check for routing issues
3. Run `analyze_packet_capture.sh` to see where packets are being lost

**Common Solutions**:
- Add ICMP rules to FORWARD and INPUT chains
- Add routes for the tunnel network
- Disable reverse path filtering

### Scenario 2: TCP Traffic Works But ICMP Fails

**Symptoms**:
- Ping fails between tunnel endpoints
- TCP traffic (e.g., HTTP) works through the tunnel
- Tunnel interfaces are up
- SSSonector processes are running

**Investigation Approach**:
1. Run `investigate_firewall_rules.sh` to check for ICMP-specific firewall rules
2. Run `investigate_packet_filtering.sh` to check for ICMP-specific filtering
3. Run `analyze_packet_capture.sh` to see if ICMP packets are being dropped

**Common Solutions**:
- Add specific ICMP rules to firewall
- Disable ICMP filtering
- Check for ICMP rate limiting

### Scenario 3: Large Packets Fail But Small Packets Work

**Symptoms**:
- Small ping packets work (e.g., ping -s 64)
- Large ping packets fail (e.g., ping -s 1472)
- TCP traffic with large payloads fails
- Tunnel interfaces are up
- SSSonector processes are running

**Investigation Approach**:
1. Run `investigate_mtu.sh` to check for MTU issues
2. Run `analyze_packet_capture.sh` to see if packets are being fragmented
3. Run `investigate_kernel_parameters.sh` to check Path MTU Discovery settings

**Common Solutions**:
- Reduce MTU on tunnel interfaces
- Enable Path MTU Discovery
- Adjust TCP MSS clamping

### Scenario 4: Intermittent Connectivity Issues

**Symptoms**:
- Tunnel connectivity works sometimes but fails other times
- No clear pattern to the failures
- Tunnel interfaces are up
- SSSonector processes are running

**Investigation Approach**:
1. Run `analyze_packet_capture.sh` during both working and non-working periods
2. Run `investigate_kernel_parameters.sh` to check for timeout settings
3. Run `investigate_packet_filtering.sh` to check for stateful filtering

**Common Solutions**:
- Adjust keepalive settings
- Disable connection tracking for tunnel traffic
- Increase timeout values

## Best Practices

### When to Use the Investigation Tools

The investigation tools should be used in the following scenarios:

1. **Initial Deployment**: After deploying SSSonector to a new environment, run the investigation tools to verify connectivity.

2. **Configuration Changes**: After making configuration changes to SSSonector or the network, run the investigation tools to verify that connectivity still works.

3. **Troubleshooting**: When connectivity issues are reported, run the investigation tools to identify and resolve the issues.

4. **Regular Verification**: Run the investigation tools periodically to verify that connectivity is still working and to catch issues before they affect users.

### How to Use the Investigation Tools Effectively

To use the investigation tools effectively:

1. **Start with the Complete Investigation**: Run the complete investigation first to get a comprehensive view of the tunnel status.

2. **Focus on Specific Issues**: Based on the results of the complete investigation, focus on specific issues by running individual investigation scripts.

3. **Document Results**: Document the results of each investigation, including the actions taken and their impact.

4. **Implement Permanent Fixes**: Once you've identified the issues, implement permanent fixes rather than relying on the investigation scripts to fix the issues each time.

5. **Verify Fixes**: After implementing fixes, run the investigation tools again to verify that the issues have been resolved.

### Security Considerations

When using the investigation tools, keep the following security considerations in mind:

1. **Firewall Rules**: The investigation tools add firewall rules that may affect your security posture. Review these rules and ensure they align with your security policies.

2. **Kernel Parameters**: The investigation tools adjust kernel parameters that may affect system security. Review these changes and ensure they align with your security policies.

3. **Credentials**: The investigation tools require SSH access and sudo privileges on the QA servers. Ensure that these credentials are properly secured.

4. **Network Exposure**: The investigation tools may expose network details that could be useful to attackers. Ensure that the investigation results are properly secured.

## Conclusion

The SSSonector investigation tools provide a systematic approach to identifying and resolving connectivity issues with the SSSonector tunnel. By following the phased approach and using the tools effectively, you can quickly identify and resolve issues that prevent proper packet transmission.

Remember that the investigation tools are designed to be diagnostic and educational. They help you identify issues and understand how to resolve them. For production environments, you should implement permanent fixes based on the investigation results rather than relying on the investigation scripts to fix issues each time.

By understanding the common troubleshooting scenarios and following the best practices outlined in this guide, you can maintain reliable and secure tunnel connectivity in your SSSonector deployments.
