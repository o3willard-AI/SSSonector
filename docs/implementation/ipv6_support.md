# IPv6 Support (Experimental)

SSSonector includes experimental support for IPv6 networking. This feature is currently under development and is disabled by default.

## Configuration

IPv6 support can be enabled through the network configuration section:

```yaml
network:
  interface: "tun0"
  mtu: 1500
  address: "10.0.0.1/24"
  ipv6:
    enabled: false  # Set to true to enable IPv6 support
    address: "fd00::1/64"  # IPv6 address for the interface
    prefix: 64  # Network prefix length
```

## Current Status

IPv6 support is currently marked as experimental for the following reasons:

1. Limited testing across different platforms and network environments
2. Potential performance implications that need further investigation
3. Security considerations that require additional review
4. Cross-platform compatibility issues that need to be addressed

## Known Limitations

- IPv6 routing may not work correctly in all network configurations
- Some platforms may have limited or no IPv6 support
- Performance impact when IPv6 is enabled needs further optimization
- Security implications of IPv6 tunneling need additional review

## Future Development

The following improvements are planned:

1. Comprehensive IPv6 testing across different platforms
2. Performance optimization for IPv6 traffic
3. Enhanced security measures for IPv6 tunneling
4. Better error handling and diagnostics for IPv6-related issues
5. Documentation updates with platform-specific IPv6 configuration guides

## Feedback

If you encounter any issues with IPv6 support or have suggestions for improvements, please file an issue on our GitHub repository.
