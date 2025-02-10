# AI QA Automation Guide

## Overview
This guide documents best practices and methodologies for AI-driven QA automation, based on learnings from the SSSonector project. It serves as a reference for AI QA engineers working on automated testing across different environments.

## Key Principles

1. **State Management**
   - Maintain a detailed JSON status file (e.g., qa_environment_status.json) to track:
     - Current state of all test environments
     - Access configurations
     - Service statuses
     - Last known good state
   - Update status file after each significant operation
   - Include timestamps for all state changes

2. **Access Automation**
   - Use SSH key-based authentication for secure, passwordless access
   - Configure passwordless sudo access for automated system operations
   - Verify both SSH and sudo access after configuration changes
   - Store credentials securely and separately from automation scripts

3. **Error Recovery**
   - Save context frequently (at least several times per day)
   - Document last known good state
   - Create recovery scripts for common failure scenarios
   - Maintain separate scripts for setup and verification

4. **Script Organization**
   - Break down complex operations into separate scripts
   - Use expect scripts for interactive operations
   - Include verification steps in each script
   - Add clear success/failure indicators
   - Use consistent error handling and logging

## Best Practices

### Environment Setup
1. Create separate scripts for:
   - Initial environment configuration
   - Access setup (SSH/sudo)
   - Service installation
   - Service configuration
   - Verification and testing

2. Use standardized script structure:
   ```expect
   #!/usr/bin/expect -f
   
   # Configuration variables
   set username "user"
   set password "pass"
   
   # Clear error handling
   if {[catch {
       # Main operations
   } err]} {
       puts "❌ Error: $err"
       exit 1
   }
   
   # Verification step
   if {[verify_operation]} {
       puts "✅ Success"
   } else {
       puts "❌ Failed"
       exit 1
   }
   ```

### Automation Scripts
1. **SSH Key Setup**
   - Generate keys with no passphrase for automation
   - Deploy keys to all target systems
   - Verify key-based login works
   - Document key locations and permissions

2. **Sudo Access**
   - Use sudoers.d directory for configuration
   - Set correct ownership (root:root) and permissions (440)
   - Verify passwordless sudo access
   - Test specific commands needed for automation

3. **Service Management**
   - Create status check scripts
   - Implement clean startup/shutdown procedures
   - Monitor service health during tests
   - Log all service state changes

## Context Management

### Required Context
1. **Environment Information**
   - IP addresses and credentials
   - Network configuration
   - System specifications
   - Installed services and versions

2. **Test Configuration**
   - Test suite locations
   - Test data
   - Expected results
   - Known issues

3. **Recovery Information**
   - Backup locations
   - Recovery procedures
   - Emergency contacts
   - Fallback configurations

### Context Backup
1. Save the following in version control:
   - All automation scripts
   - Configuration files
   - Status JSON files
   - Test results and logs
   - This guide and related documentation

2. Include in context backups:
   - Current environment state
   - Test progress
   - Failed test cases
   - Debugging notes

## Verification Procedures

### Access Verification
```expect
# Example verification script structure
spawn ssh -i $ssh_key $username@$ip "echo 'SSH test'; sudo -n echo 'Sudo test'"
expect {
    "SSH test" {
        puts "✅ SSH access verified"
    }
    "Sudo test" {
        puts "✅ Sudo access verified"
    }
    timeout {
        puts "❌ Verification failed"
        exit 1
    }
}
```

### Service Verification
1. Check service status
2. Verify required ports are open
3. Test basic functionality
4. Monitor resource usage
5. Verify logging is working

## Recovery Procedures

1. **Access Issues**
   - Verify network connectivity
   - Check SSH key permissions
   - Validate sudoers configuration
   - Test with password authentication as fallback

2. **Service Issues**
   - Check service status and logs
   - Verify dependencies
   - Test configuration files
   - Restart services if needed

3. **Environment Issues**
   - Validate VM status
   - Check resource usage
   - Verify network connectivity
   - Test system access

## Notes for Future AI QA Engineers

1. **Environment Understanding**
   - Always map out the complete test environment
   - Document all dependencies
   - Understand service interactions
   - Know the recovery procedures

2. **Automation Strategy**
   - Start with basic access automation
   - Build up to service management
   - Add comprehensive verification
   - Implement robust error handling

3. **Context Management**
   - Save context frequently
   - Include all relevant files
   - Document decision points
   - Maintain clear status information

4. **Continuous Improvement**
   - Update this guide with new learnings
   - Enhance automation scripts
   - Improve error handling
   - Share knowledge with team

Remember: This guide should be treated as a living document. Update it with new learnings, improved methodologies, and better practices as they are discovered.
