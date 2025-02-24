# Build Verification Tool

Part of Project SENTINEL - SSSonector ENvironment Testing & Integration Layer

## Overview

The build verification tool ensures the integrity and completeness of SSSonector builds. It performs comprehensive checks on build outputs, verifying version information, platform coverage, and file integrity through checksum validation.

## Features

- Build directory structure verification
- Multi-platform binary verification
- Version information validation
  - Version number format
  - Build timestamp format
  - Commit hash format
- Checksum verification for all platform builds
- Detailed logging with color-coded output

## Supported Platforms

- linux-amd64
- linux-arm64
- linux-arm
- darwin-amd64
- darwin-arm64
- windows-amd64

## Usage

```bash
# Make script executable
chmod +x build_verify.sh

# Run verification
./build_verify.sh
```

## Exit Codes

- 0: All verifications passed
- 1: One or more verifications failed

## Output Format

The tool provides color-coded output for easy status identification:
- 🟢 Green: Information messages
- 🔴 Red: Error messages
- 🟡 Yellow: Warning messages

Example output:
```
[INFO] Starting build verification...
[INFO] Verifying build directory...
[INFO] Build directory verified
[INFO] Verifying platform binaries...
[INFO] All platform binaries verified
[INFO] Verifying checksums...
[INFO] All checksums verified
[INFO] Build verification completed successfully
```

## Integration

This tool is typically run:
1. After each build operation
2. Before deploying to QA environments
3. As part of the CI/CD pipeline
4. Before running test suites

## Error Handling

The tool implements strict error checking:
- Fails fast on critical errors
- Reports all verification failures
- Provides detailed error messages
- Maintains error state for dependent checks

## Dependencies

- bash
- sha256sum
- git (for repository root detection)

## Version History

- 1.0.0: Initial release
  - Basic build verification
  - Multi-platform support
  - Checksum validation
  - Version information checking
