# Security Documentation

This document provides an overview of the security features and hardening measures implemented in SSSonector.

## Overview

SSSonector implements multiple layers of security to ensure robust protection:

1. Linux Security Features
   - Namespaces for isolation
   - Cgroups for resource control
   - Seccomp for syscall filtering
   - Capabilities for privilege management
   - SELinux/AppArmor for mandatory access control

2. Memory Protection
   - Stack protector
   - Address space randomization
   - Memory page locking
   - Executable space protection

3. Resource Limits
   - Process limits
   - File size limits
   - Memory limits
   - Open file limits

4. Filesystem Security
   - Read-only root filesystem
   - Hidden sensitive paths
   - Private /tmp directory
   - Restricted device access

## Security Components

### Namespace Manager

The namespace manager provides process isolation through Linux namespaces:

- Network namespace for network isolation
- Mount namespace for filesystem isolation
- PID namespace for process isolation
- IPC namespace for IPC isolation
- UTS namespace for hostname isolation
- User namespace for user isolation

Configuration options:
```yaml
namespaces:
  network:
    enabled: true
    veth: "sssonector0"
    bridge: "br0"
    ip: "10.0.0.2"
    mask: "255.255.255.0"
  mount:
    enabled: true
    proc: true
    sys: true
    tmp: true
    rootfs: "/var/lib/sssonector/rootfs"
    readonly: true
  pid:
    enabled: true
  ipc:
    enabled: true
  uts:
    enabled: true
    hostname: "sssonector"
  user:
    enabled: false
```

### Cgroup Manager

The cgroup manager provides resource control and accounting:

- Memory limits and accounting
- CPU shares and quotas
- Block I/O limits
- Process number limits

Configuration options:
```yaml
cgroups:
  memory:
    limit: 1G
    soft_limit: 512M
    swap_limit: 1G
    kernel_limit: 128M
  cpu:
    shares: 1024
    quota: 100ms
    period: 100ms
  blkio:
    weight: 500
    device_weight: []
    read_bps: []
    write_bps: []
  pids:
    limit: 1024
```

### Security Manager

The security manager implements system-wide security policies:

- Seccomp filtering
- Capability management
- Memory protections
- Resource limits

Configuration options:
```yaml
security:
  process:
    no_new_privs: true
    secure_exec: true
    drop_privileges: true
    retain_caps:
      - CAP_NET_ADMIN
      - CAP_NET_RAW
      - CAP_NET_BIND_SERVICE
  memory:
    denysetuid: true
    denyexec: true
    mlock: true
    stack_protect: true
    randomize_va: true
  seccomp:
    mode: filtered
    syscalls:
      - read
      - write
      - open
      # ... (see full list in code)
  limits:
    core: 0
    fsize: 1G
    nofile: 1024
    nproc: 64
    stack: 8M
```

### SELinux Policy

The SELinux policy enforces mandatory access control:

- Domain transitions
- Network access control
- File access control
- Process isolation

Key policy rules:
```
# Domain transition
domain_type(sssonector_t)
domain_entry_file(sssonector_t, sssonector_exec_t)

# Network access
allow sssonector_t sssonector_port_t:tcp_socket name_bind;
allow sssonector_t unreserved_port_t:tcp_socket name_connect;

# File access
allow sssonector_t sssonector_conf_t:dir { getattr open read search };
allow sssonector_t sssonector_conf_t:file { getattr open read };

# Process capabilities
allow sssonector_t self:capability { net_admin net_raw setgid setuid sys_admin };
```

### AppArmor Profile

The AppArmor profile provides an additional layer of mandatory access control:

- File access control
- Network access control
- Capability restrictions
- System call filtering

Key profile rules:
```
profile sssonector /usr/local/bin/sssonector {
  #include <abstractions/base>
  #include <abstractions/nameservice>

  capability {
    net_admin,
    net_raw,
    setgid,
    setuid,
  }

  network {
    inet,
    inet6,
    netlink,
    unix,
  }

  # File access
  /etc/sssonector/** r,
  /var/lib/sssonector/** rwk,
  /var/run/sssonector/** rwk,
  /var/log/sssonector/** rw,
}
```

## Security Best Practices

1. Principle of Least Privilege
   - Drop unnecessary capabilities
   - Restrict system calls
   - Limit file access
   - Isolate processes

2. Defense in Depth
   - Multiple security layers
   - Redundant controls
   - Fail-safe defaults
   - Secure by default

3. Resource Control
   - Memory limits
   - CPU quotas
   - Process limits
   - I/O limits

4. Isolation
   - Network isolation
   - Filesystem isolation
   - Process isolation
   - IPC isolation

## Installation and Configuration

1. SELinux Policy Installation:
```bash
# Build and install policy module
cd security/selinux
./build_policy.sh

# Verify installation
semodule -l | grep sssonector
ls -Z /usr/local/bin/sssonector
```

2. AppArmor Profile Installation:
```bash
# Install profile
cd security/apparmor
./install_profile.sh

# Verify installation
aa-status | grep sssonector
```

3. Security Configuration:
```yaml
# /etc/sssonector/config.yaml
security:
  enabled: true
  mode: enforcing
  selinux: true
  apparmor: true
  seccomp: filtered
  namespaces: true
  cgroups: true
```

## Monitoring and Auditing

1. SELinux Audit Logs:
```bash
# Monitor denials
ausearch -m AVC -ts recent
tail -f /var/log/audit/audit.log | grep sssonector
```

2. AppArmor Audit Logs:
```bash
# Monitor denials
aa-notify -s 1 -f /var/log/audit/audit.log
tail -f /var/log/syslog | grep "apparmor=\"DENIED\""
```

3. Resource Usage:
```bash
# Monitor cgroup stats
cat /sys/fs/cgroup/sssonector/memory.current
cat /sys/fs/cgroup/sssonector/cpu.stat
```

4. Process Status:
```bash
# Check process isolation
ps -Z | grep sssonector
ip netns list
lsns | grep sssonector
```

## Troubleshooting

1. SELinux Issues:
```bash
# Temporarily disable SELinux
setenforce 0

# Generate allow rules
audit2allow -a -M sssonector_local
```

2. AppArmor Issues:
```bash
# Set profile to complain mode
aa-complain /usr/local/bin/sssonector

# Update profile from logs
aa-logprof
```

3. Namespace Issues:
```bash
# Check namespace isolation
nsenter -t <pid> -n ip addr
nsenter -t <pid> -m findmnt
```

4. Cgroup Issues:
```bash
# Check cgroup hierarchy
systemd-cgls
cat /proc/self/cgroup
```

## Security Updates

1. Policy Updates:
   - Monitor security advisories
   - Update SELinux/AppArmor policies
   - Update seccomp filters
   - Review resource limits

2. Configuration Updates:
   - Review security settings
   - Update capability sets
   - Adjust resource limits
   - Tune performance

3. Monitoring Updates:
   - Review audit logs
   - Monitor resource usage
   - Check isolation status
   - Verify policy enforcement

## References

1. Linux Security
   - [Linux Namespaces](https://man7.org/linux/man-pages/man7/namespaces.7.html)
   - [Control Groups](https://www.kernel.org/doc/html/latest/admin-guide/cgroup-v2.html)
   - [Seccomp](https://www.kernel.org/doc/html/latest/userspace-api/seccomp_filter.html)
   - [Capabilities](https://man7.org/linux/man-pages/man7/capabilities.7.html)

2. Mandatory Access Control
   - [SELinux](https://selinuxproject.org/page/Main_Page)
   - [AppArmor](https://gitlab.com/apparmor/apparmor/-/wikis/Documentation)

3. Resource Management
   - [Cgroups v2](https://docs.kernel.org/admin-guide/cgroup-v2.html)
   - [Linux Memory Management](https://www.kernel.org/doc/html/latest/admin-guide/mm/index.html)

4. Process Isolation
   - [Mount Namespaces](https://man7.org/linux/man-pages/man7/mount_namespaces.7.html)
   - [Network Namespaces](https://man7.org/linux/man-pages/man7/network_namespaces.7.html)
   - [PID Namespaces](https://man7.org/linux/man-pages/man7/pid_namespaces.7.html)
   - [User Namespaces](https://man7.org/linux/man-pages/man7/user_namespaces.7.html)
