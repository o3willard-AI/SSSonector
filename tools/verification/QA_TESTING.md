# SSSonector QA Testing Guide (DEPRECATED)

**IMPORTANT: This document is deprecated as of February 26, 2025. Please refer to the new [QA Methodology 2025](QA_METHODOLOGY_2025.md) document and [Minimal Functionality Test](MINIMAL_FUNCTIONALITY_TEST.md) for the current QA testing process.**

This guide provides instructions for setting up and running QA tests for SSSonector.

## Overview

SSSonector is a high-performance, enterprise-grade communications utility designed to allow critical services to connect to and exchange data with one another over the public internet without needing a VPN. This guide focuses on the QA testing process for SSSonector.

## QA Environment

The QA environment consists of two servers:

- **Server**: 192.168.50.210
- **Client**: 192.168.50.211
- **User**: sblanken

## Prerequisites

- SSH access to QA servers
- Sudo privileges on QA servers
- SSSonector binary

## QA Testing Process

The QA testing process consists of the following steps:

1. Clean up the QA environment
2. Deploy SSSonector to the QA environment
3. Run sanity checks

### 1. Clean up the QA Environment

The `cleanup_qa.sh` script cleans up the QA environment by:

- Removing any existing SSSonector binaries
- Cleaning up certificates
- Removing configuration files
- Killing any running SSSonector processes
- Removing TUN interfaces
- Freeing up required ports

To clean up the QA environment:

```bash
cd /home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector
./tools/verification/cleanup_qa.sh
```

### 2. Deploy SSSonector to the QA Environment

The `deploy_sssonector.sh` script deploys SSSonector to the QA environment by:

- Generating certificates
- Creating configuration files
- Copying the SSSonector binary to the QA servers
- Copying certificates and configuration files to the QA servers

To deploy SSSonector to the QA environment:

```bash
cd /home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector
./tools/verification/deploy_sssonector.sh
```

### 3. Run Sanity Checks

The `run_sanity_checks.sh` script runs sanity checks on SSSonector by:

- Starting SSSonector in server mode
- Starting SSSonector in client mode
- Checking tunnel establishment
- Sending packets from client to server
- Sending packets from server to client
- Verifying clean tunnel closure

The script runs three test scenarios:

1. Client foreground / Server foreground
2. Client background / Server foreground
3. Client background / Server background

To run sanity checks:

```bash
cd /home/sblanken/Desktop/go/src/github.com/o3willard-AI/SSSonector
./tools/verification/run_sanity_checks.sh
```

## Understanding SSSonector

SSSonector is a standalone binary that is placed alongside its configuration file. It is not a service controlled by systemd, nor is it a package to be installed. It is simply an executable binary file that is called with bash.

The only flag that is always used is the option to generate the certificate pair. On the intended server mode side, this should be done first so that the client mode certificate can be copied over to the client before proceeding further.

Other than the certificate pairs, the only thing SSSonector needs is its configuration file, which should be located adjacent to SSSonector. Everything else, including the mode and all mandatory and optional parameters, are defined in the configuration file.

## Configuration Files

The deployment script creates four configuration files:

1. `server.yaml`: Server configuration for background mode
2. `client.yaml`: Client configuration for background mode
3. `server_foreground.yaml`: Server configuration for foreground mode
4. `client_foreground.yaml`: Client configuration for foreground mode

The configuration files are deployed to:

- Server: `/opt/sssonector/config/`
- Client: `/opt/sssonector/config/`

## Running SSSonector

To run SSSonector:

- Server background mode: `sudo /opt/sssonector/bin/sssonector -config /opt/sssonector/config/server.yaml`
- Client background mode: `sudo /opt/sssonector/bin/sssonector -config /opt/sssonector/config/client.yaml`
- Server foreground mode: `sudo /opt/sssonector/bin/sssonector -config /opt/sssonector/config/server_foreground.yaml`
- Client foreground mode: `sudo /opt/sssonector/bin/sssonector -config /opt/sssonector/config/client_foreground.yaml`

## Troubleshooting

If you encounter issues during the QA testing process:

1. Check the logs:
   - Server: `/opt/sssonector/log/server.log`
   - Client: `/opt/sssonector/log/client.log`

2. Verify that the SSSonector binary is executable:
   ```bash
   ls -la /opt/sssonector/bin/sssonector
   ```

3. Verify that the certificates have the correct permissions:
   ```bash
   ls -la /opt/sssonector/certs/
   ```

4. Verify that the configuration files have the correct permissions:
   ```bash
   ls -la /opt/sssonector/config/
   ```

5. Check if SSSonector processes are running:
   ```bash
   pgrep -f sssonector
   ```

6. Check if TUN interfaces exist:
   ```bash
   ip link show | grep tun
   ```

7. Check if required ports are available:
   ```bash
   netstat -tuln | grep 8443
   ```

## Conclusion

This guide provides instructions for setting up and running QA tests for SSSonector. By following these steps, you can ensure that SSSonector is functioning correctly in the QA environment.

**NOTE: This document is maintained for historical reference only. For current QA testing procedures, please refer to the [QA Methodology 2025](QA_METHODOLOGY_2025.md) document.**
