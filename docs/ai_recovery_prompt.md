# AI Recovery Prompt

## Instructions for Human
Copy and paste everything between the `===== BEGIN PROMPT =====` and `===== END PROMPT =====` markers into the AI chat interface to restore the project context. This prompt contains all necessary information for the AI to continue development from the current state.

===== BEGIN PROMPT =====

<task>
Continue development of the SSSonector project, a secure SSL tunnel implementation with the following key features:
- TUN interface-based networking
- Certificate-based authentication
- Rate limiting and monitoring
- Cross-platform support (Linux/macOS/Windows)

The project is located at /home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector/

Key documentation and context files (all paths relative to project root):
1. /docs/project_context.md - Project overview and current status
2. /docs/code_structure_snapshot.md - Codebase organization
3. /docs/ai_context_restoration.md - Development history and decisions
4. /docs/test_results.md - Current test status and findings
5. /docs/certificate_management.md - Certificate handling documentation
6. /docs/installation.md - Installation and setup guide

Recent Development Context:
- Improved TUN interface initialization and validation
- Enhanced certificate expiration monitoring
- Added robust process cleanup
- Implemented comprehensive test suite
- Updated documentation with lessons learned

Current State:
- All core functionality is implemented
- Certificate management system is complete with 5 feature flags:
  * -test-without-certs: Run with temporary certificates
  * -generate-certs-only: Generate certificates without starting service
  * -keyfile: Specify certificate directory
  * -keygen: Generate production certificates
  * -validate-certs: Validate existing certificates
- Test suite is passing
- Documentation is up-to-date

Next Steps:
1. Performance optimization of tunnel implementation
2. Enhanced monitoring and metrics collection
3. Automated certificate rotation
4. Cross-platform testing improvements
5. Security hardening

Critical Files:
- cmd/tunnel/main.go: Entry point and flag handling
- internal/adapter/interface_linux.go: Linux TUN implementation
- internal/tunnel/tunnel.go: Core tunnel logic
- internal/monitor/monitor.go: Monitoring system
- internal/cert/manager.go: Certificate management
- test/test_temp_certs.sh: Certificate testing

Development Environment:
- Go 1.21 or later
- Linux (Ubuntu 24.04)
- TUN/TAP kernel module
- iproute2 package

Please analyze these files and continue development according to the next steps, maintaining the existing code quality and test coverage.
</task>

===== END PROMPT =====

## Notes for Future Updates
1. Update the "Recent Development Context" section when major changes are made
2. Keep "Current State" accurate and up-to-date
3. Adjust "Next Steps" as priorities change
4. Update file paths if project structure changes
5. Maintain list of critical files as architecture evolves

## Verification Steps
After pasting the prompt:
1. AI should acknowledge understanding of the project
2. AI should be able to list key features and current state
3. AI should identify next steps
4. AI should request to examine specific files before proceeding

## Recovery Validation
The AI should demonstrate understanding by:
1. Referencing specific files and their purposes
2. Understanding the certificate management system
3. Being aware of recent changes and improvements
4. Knowing the current development priorities
