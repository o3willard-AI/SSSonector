# SSSonector QA Testing Scripts

This directory contains scripts for testing and managing the SSSonector tunnel service in a QA environment.

## Environment Setup

The test environment consists of:
- Server: 192.168.50.210
- Client: 192.168.50.211

Each machine should have:
- SSSonector binary in `~/sssonector/bin/`
- Configuration in `~/sssonector/config/`
- Certificates in `~/sssonector/certs/`
- Log directory at `~/sssonector/log/`
- State directory at `~/sssonector/state/`

## Scripts

### tunnel_control.sh

A control script for managing the SSSonector tunnel service. It provides commands for starting, stopping, and monitoring the tunnel.

```bash
# Start both server and client
./tunnel_control.sh start

# Stop both server and client
./tunnel_control.sh stop

# Show tunnel status
./tunnel_control.sh status

# Restart both server and client
./tunnel_control.sh restart
```

The script handles:
- Process management
- Interface cleanup
- Connectivity testing
- Status monitoring

### setup_configs.sh

Sets up configuration files on both server and client machines.

```bash
./setup_configs.sh
```

### setup_certificates.sh

Generates and distributes TLS certificates for secure communication.

```bash
./setup_certificates.sh
```

### setup_binary.sh

Copies the SSSonector binary to both machines and sets appropriate permissions.

```bash
./setup_binary.sh
```

### setup_sudo_access.sh

Configures necessary sudo permissions for the tunnel service.

```bash
./setup_sudo_access.sh
```

### core_functionality_test.sh

Runs a series of tests to verify core tunnel functionality.

```bash
./core_functionality_test.sh
```

## Testing Process

1. Ensure all setup scripts have been run:
   ```bash
   ./setup_binary.sh
   ./setup_certificates.sh
   ./setup_configs.sh
   ./setup_sudo_access.sh
   ```

2. Start the tunnel:
   ```bash
   ./tunnel_control.sh start
   ```

3. Verify tunnel status:
   ```bash
   ./tunnel_control.sh status
   ```

4. Run functionality tests:
   ```bash
   ./core_functionality_test.sh
   ```

5. Stop the tunnel when done:
   ```bash
   ./tunnel_control.sh stop
   ```

## Troubleshooting

### Common Issues

1. Permission denied errors:
   - Check file permissions in ~/sssonector directories
   - Verify sudo access is properly configured
   - Ensure TUN device permissions are correct

2. Connection failures:
   - Verify both processes are running (use status command)
   - Check firewall settings for port 8080
   - Ensure TUN interfaces are properly configured

3. Certificate errors:
   - Verify certificate files exist and have correct permissions
   - Check certificate paths in config files
   - Regenerate certificates if needed

### Logs

- Server logs: `~/sssonector/log/sssonector.log` on server
- Client logs: `~/sssonector/log/sssonector.log` on client
- System logs: `journalctl -u sssonector` on both machines

### Debug Mode

Add `-debug` flag when running SSSonector for additional logging:

```bash
sudo ~/sssonector/bin/sssonector -config ~/sssonector/config/config.yaml -debug
```

## Network Configuration

### Server
- TUN Interface: tun0
- IP Address: 10.0.0.1/24
- Listen Port: 8080

### Client
- TUN Interface: tun0
- IP Address: 10.0.0.2/24
- Server Port: 8080

## Security Notes

1. The tunnel service requires root privileges for:
   - Creating and configuring TUN interfaces
   - Binding to privileged ports
   - Managing network routes

2. Security measures in place:
   - TLS encryption for all tunnel traffic
   - Certificate-based authentication
   - Restricted file permissions
   - Limited sudo access
   - Network namespace isolation
