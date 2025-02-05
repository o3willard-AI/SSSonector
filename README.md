# SSSonector - SSL Tunnel Application

SSSonector is a secure SSL tunneling application that provides encrypted communication between servers and clients.

## Features

- Secure SSL/TLS tunneling
- Automatic certificate management
- Built-in certificate generation
- Quick test mode for troubleshooting
- Rate limiting support
- SNMP monitoring (optional)
- Cross-platform support (Linux, macOS, Windows)

## Quick Start

1. Generate certificates:
```bash
sssonector -keygen
```

2. Start the server:
```bash
sssonector -mode server -config /etc/sssonector/config.yaml
```

3. Start the client:
```bash
sssonector -mode client -config /etc/sssonector/config.yaml
```

## Installation

See the installation guides for your platform:
- [Linux Installation](docs/linux_install.md)
- [Windows Installation](docs/windows_install.md)
- [macOS Installation](docs/macos_build.md)
- [Ubuntu Installation](docs/ubuntu_install.md)

## Configuration

SSSonector uses YAML configuration files. Example configurations are provided:
- [Server Configuration](configs/server.yaml)
- [Client Configuration](configs/client.yaml)

## Certificate Management

SSSonector includes comprehensive certificate management features:
- Automatic certificate generation
- Certificate validation and verification
- Test mode with temporary certificates
- Flexible certificate location options

See [Certificate Management](docs/certificate_management.md) for detailed documentation.

## Command Line Options

```
Usage: sssonector [options]

Options:
  -config string
        Path to configuration file
  -keygen
        Generate SSL certificates
  -keyfile string
        Directory containing SSL certificates
  -mode string
        Operation mode (server/client)
  -test-without-certs
        Run a 15-second test connection without certificates
```

## Testing

For quick connectivity testing without setting up certificates:

```bash
# Server side
sssonector -mode server -test-without-certs

# Client side
sssonector -mode client -test-without-certs
```

This will create temporary certificates valid for 15 seconds to test the connection.

## Monitoring

SSSonector supports SNMP monitoring for:
- Connection status
- Bandwidth usage
- Error rates
- Client connections (server mode)

Enable monitoring in the configuration file:
```yaml
monitor:
  snmp_enabled: true
  snmp_port: 161
  snmp_community: "public"
```

## Project Structure

```
.
├── cmd/
│   └── tunnel/          # Main application
├── configs/             # Example configurations
├── docs/               # Documentation
├── internal/
│   ├── adapter/        # Platform-specific code
│   ├── cert/          # Certificate management
│   ├── config/        # Configuration handling
│   ├── connection/    # Connection management
│   ├── monitor/       # Monitoring and metrics
│   ├── throttle/      # Rate limiting
│   └── tunnel/        # Core tunnel functionality
└── installers/        # Platform installers
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Security

Report security issues to security@sssonector.example.com (do not use for support).

## Support

- Documentation: See [docs/](docs/) directory
- Issues: Use GitHub issue tracker
- Community: Join our [Discord server](https://discord.gg/sssonector)
