# Deployment Tools

Part of Project SENTINEL - SSSonector ENvironment Testing & Integration Layer

## Directory Structure Setup Tool

The directory structure setup tool ensures consistent filesystem layout and permissions across all SSSonector deployments. It creates and maintains the required directory structure with appropriate permissions and ownership.

### Standard Directory Structure

```
/opt/sssonector/
├── bin/      # Binary executables
├── config/   # Configuration files
├── certs/    # Certificates and keys
├── log/      # Log files
├── state/    # Runtime state
└── tools/    # Utility scripts
```

### Permission Model

| Type | Path | Permission | Description |
|------|------|------------|-------------|
| Directory | All | 755 | Readable/executable by all, writable by owner |
| File | config/*.yaml | 644 | Readable by all, writable by owner |
| File | certs/*.crt | 644 | Readable by all, writable by owner |
| File | certs/*.key | 600 | Readable/writable by owner only |
| File | bin/* | 755 | Executable by all, writable by owner |
| File | log/* | 644 | Readable by all, writable by owner |

### Usage

```bash
# Make script executable
chmod +x setup_directory_structure.sh

# Create directory structure with default path
./setup_directory_structure.sh -o username:groupname

# Create directory structure with custom path
./setup_directory_structure.sh -d /custom/path -o username:groupname

# Show help
./setup_directory_structure.sh -h
```

### Features

- Creates standardized directory structure
- Sets appropriate permissions for each directory type
- Sets correct ownership for all files and directories
- Verifies directory structure and permissions
- Handles existing directories gracefully
- Provides detailed logging of all operations

### Exit Codes

- 0: Success
- 1: Error (missing arguments, permission issues, verification failure)

### Integration

This tool is typically used:
1. During initial system setup
2. When deploying to new environments
3. To verify and repair existing installations
4. As part of system upgrades

### Error Handling

The tool implements comprehensive error checking:
- Validates all input parameters
- Checks for required permissions
- Verifies all operations
- Provides detailed error messages
- Maintains atomic operations where possible

### Dependencies

- bash
- sudo (for permission operations)
- standard Unix tools (mkdir, chmod, chown)

### Version History

- 1.0.0: Initial release
  - Basic directory structure creation
  - Permission management
  - Structure verification
  - Ownership control
