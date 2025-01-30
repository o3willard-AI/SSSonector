# SSSonector v1.0.0

## Features

- TLS 1.3 with EU-exportable cipher suites
- Virtual network interfaces for transparent routing
- Persistent tunnel connections with automatic reconnection
- Bandwidth throttling capabilities
- SNMP monitoring and telemetry
- Cross-platform support (Linux, Windows, macOS)
- Comprehensive logging and monitoring
- Systemd/Launchd/Windows Service integration
- Certificate management and rotation

## Installation

### Linux (Debian/Ubuntu)
```bash
wget https://github.com/o3willard-AI/SSSonector/releases/download/v1.0.0/sssonector_1.0.0_amd64.deb
sudo dpkg -i sssonector_1.0.0_amd64.deb
sudo apt-get install -f
```

### Windows
1. Download sssonector-1.0.0-setup.exe from the releases page
2. Run the installer with administrator privileges
3. Follow the installation wizard

### macOS
The macOS installer is pending contribution from the community. Please see [macOS Build Guide](../docs/macos_build_guide.md) if you'd like to help build and submit it.

## Checksums

```

    c2c9b404723a6bc47b1ead2929d207591134f0782c82bf0fa54f594b03b24e24  sssonector_1.0.0_amd64.deb
    e71e1950bfc08229b84b9487a8c2a61c68381e2e305be2bf776cdde216b3377c  sssonector-1.0.0-setup.exe
```

## Documentation

Full documentation is available at: https://github.com/o3willard-AI/SSSonector/tree/v1.0.0/docs

## Notes

- macOS installer will be added in a future release
- RPM package will be added in a future release
