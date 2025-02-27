# SSSonector Sanity Check Tests

This directory contains sanity check tests for SSSonector that validate core functionality across different deployment modes.

## Test Structure

```
sanity_check/
├── build.sh              # Builds Linux packages
├── deploy.sh             # Deploys to QA servers
├── run_all.sh           # Main test runner
└── scenarios/           # Test scenarios
    ├── fg_fg.sh        # Foreground-Foreground test
    ├── fg_bg.sh        # Foreground-Background test
    └── bg_bg.sh        # Background-Background test
```

## Usage

```bash
./run_all.sh -s <server_host> -c <client_host> [-u <user>] [-r <results_dir>]
```

Options:
- `-s`: Server host address
- `-c`: Client host address
- `-u`: SSH user (default: qauser)
- `-r`: Results directory (default: results)

## Test Scenarios

1. Foreground-Foreground (fg_fg.sh)
   - Server runs in foreground
   - Client runs in foreground
   - Tests bidirectional connectivity
   - Verifies clean shutdown

2. Foreground-Background (fg_bg.sh)
   - Server runs in foreground
   - Client runs in background
   - Tests bidirectional connectivity
   - Verifies clean shutdown

3. Background-Background (bg_bg.sh)
   - Server runs in background
   - Client runs in background
   - Tests bidirectional connectivity
   - Verifies clean shutdown

## Test Flow

Each scenario follows this flow:
1. Start server process
2. Start client process
3. Send 20 test packets client → server
4. Send 20 test packets server → client
5. Stop client process
6. Stop server process
7. Verify clean shutdown

## Results

Test results are stored in `results/YYYYMMDD_HHMMSS/`:
- Individual test logs
- Environment information
- Summary report (summary.md)

## Requirements

- Linux systems for QA testing
- SSH access to test servers
- Go development environment
- Root/sudo access for network operations

## Configuration

Test configurations in `configs/`:
- Server and client YAML files
- Certificate paths
- Network settings
- Security parameters

## Single Source of Truth Policy

Before creating any new:
- Types
- Configurations
- Shared utilities
- Common interfaces

You MUST:
1. First check /internal/config/types/types.go for existing implementations
   - Never create duplicate type definitions
   - Always use types from this central location
   - Add new shared types here, not in individual packages

2. Review /internal/config/validator/validator.go for validation patterns
   - All configuration validation must be centralized here
   - Follow existing validation patterns

## Troubleshooting

Common issues:
1. Build failures:
   - Check Go installation
   - Verify dependencies
   - Check disk space

2. Deployment failures:
   - Verify SSH access
   - Check disk space
   - Verify permissions

3. Test failures:
   - Check network connectivity
   - Verify TUN module
   - Check process permissions
   - Review test logs

4. Clean shutdown issues:
   - Check for leftover processes
   - Verify interface cleanup
   - Check for pidfiles

## Contributing

When adding or modifying tests:
1. Follow existing patterns
2. Update documentation
3. Test thoroughly
4. Consider edge cases
5. Follow Single Source of Truth policy
