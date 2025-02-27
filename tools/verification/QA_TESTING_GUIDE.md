# SSSonector QA Testing Guide (DEPRECATED)

**IMPORTANT: This document is deprecated as of February 26, 2025. Please refer to the new [QA Methodology 2025](QA_METHODOLOGY_2025.md) document and [Minimal Functionality Test](MINIMAL_FUNCTIONALITY_TEST.md) for the current QA testing process.**

This comprehensive guide outlines the process for conducting Quality Assurance testing on the SSSonector communications utility.

## Testing Process Overview

The QA testing process for SSSonector follows a structured approach to ensure all aspects of the system are thoroughly tested:

1. **Environment Setup**
   - Clean up any previous test artifacts
   - Deploy SSSonector to QA environment
   - Configure test parameters

2. **Functional Testing**
   - Verify tunnel establishment
   - Test bidirectional communication
   - Validate clean tunnel closure

3. **Performance Testing**
   - Measure startup times
   - Evaluate packet transmission performance
   - Assess resource utilization

4. **Reporting**
   - Generate test reports
   - Document any issues encountered
   - Provide recommendations for improvements

## Test Execution Order

For consistent and reliable testing, follow this execution order:

1. Run `cleanup_qa.sh` to ensure a clean environment
2. Execute `deploy_sssonector.sh` to deploy the latest build
3. Run `run_qa_tests.sh` to perform comprehensive testing
4. Review test reports and logs

## Debugging Procedures

If tests fail or unexpected behavior is observed:

1. Check logs in `/opt/sssonector/log/` on both server and client
2. Verify network connectivity between test machines
3. Ensure IP forwarding is enabled on both systems
4. Check for firewall rules that might be blocking traffic
5. Verify TUN interfaces are properly created
6. Run `run_investigation.sh` for detailed diagnostics

## Best Practices

- Always start with a clean environment
- Test all deployment scenarios (foreground/background combinations)
- Verify both client-to-server and server-to-client communication
- Test with various packet sizes and types
- Monitor resource utilization during tests
- Document all test results thoroughly

## Common Issues and Solutions

| Issue | Possible Cause | Solution |
|-------|----------------|----------|
| Tunnel fails to establish | IP forwarding disabled | Enable IP forwarding with `sysctl -w net.ipv4.ip_forward=1` |
| Packet transmission fails | Firewall blocking traffic | Check and adjust firewall rules |
| High latency | Network congestion | Test during off-peak hours or on isolated network |
| Resource leaks | Improper tunnel closure | Verify clean shutdown procedures |

## Test Scenarios

The QA testing process includes the following scenarios:

1. **Basic Connectivity**
   - Server foreground, Client foreground
   - Server foreground, Client background
   - Server background, Client background

2. **Stress Testing**
   - High volume packet transmission
   - Rapid connection/disconnection cycles
   - Long-running connections

3. **Error Handling**
   - Network interruptions
   - Invalid certificates
   - Resource constraints

## Conclusion

Following this guide ensures thorough and consistent testing of SSSonector. By adhering to these procedures, you can identify and address issues before they impact production deployments.

**NOTE: This document is maintained for historical reference only. For current QA testing procedures, please refer to the [QA Methodology 2025](QA_METHODOLOGY_2025.md) document.**
