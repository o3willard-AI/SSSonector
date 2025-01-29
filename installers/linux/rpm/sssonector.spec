Name:           sssonector
Version:        1.0.0
Release:        1%{?dist}
Summary:        Secure Scalable SSL Connector

License:        MIT
URL:            https://github.com/o3willard-AI/SSSonector
Source0:        %{name}-%{version}.tar.gz

BuildRequires:  golang >= 1.21
BuildRequires:  systemd-rpm-macros
Requires:       systemd
Requires:       iproute

%description
A multipurpose utility for connecting services over the internet
or through insecure network links without having to utilize a VPN.
Features include TLS 1.3 with EU-exportable cipher suites,
automatic certificate management, bandwidth throttling,
SNMP monitoring support, and cross-platform compatibility.

%prep
%autosetup

%build
make build

%install
rm -rf $RPM_BUILD_ROOT
# Binary
install -D -m 755 build/%{name} %{buildroot}%{_bindir}/%{name}

# Configuration
install -d %{buildroot}%{_sysconfdir}/%{name}
install -d %{buildroot}%{_sysconfdir}/%{name}/certs
install -m 644 configs/server.yaml %{buildroot}%{_sysconfdir}/%{name}/server.yaml.example
install -m 644 configs/client.yaml %{buildroot}%{_sysconfdir}/%{name}/client.yaml.example

# Systemd service
install -D -m 644 scripts/service/systemd/%{name}.service %{buildroot}%{_unitdir}/%{name}.service

# Log directory
install -d %{buildroot}%{_localstatedir}/log/%{name}

# Data directory
install -d %{buildroot}%{_sharedstatedir}/%{name}

%pre
getent group %{name} >/dev/null || groupadd -r %{name}
getent passwd %{name} >/dev/null || \
    useradd -r -g %{name} -d %{_sharedstatedir}/%{name} \
    -s /sbin/nologin -c "SSSonector Service Account" %{name}
exit 0

%post
%systemd_post %{name}.service
if [ $1 -eq 1 ] ; then
    # Initial installation
    systemctl preset %{name}.service >/dev/null 2>&1 || :
fi

%preun
%systemd_preun %{name}.service

%postun
%systemd_postun_with_restart %{name}.service
if [ $1 -eq 0 ] ; then
    # Package removal, not upgrade
    userdel %{name} >/dev/null 2>&1 || :
    groupdel %{name} >/dev/null 2>&1 || :
fi

%files
%license LICENSE
%doc README.md
%{_bindir}/%{name}
%dir %{_sysconfdir}/%{name}
%dir %attr(750,%{name},%{name}) %{_sysconfdir}/%{name}/certs
%config(noreplace) %{_sysconfdir}/%{name}/*.yaml.example
%{_unitdir}/%{name}.service
%dir %attr(750,%{name},%{name}) %{_localstatedir}/log/%{name}
%dir %attr(750,%{name},%{name}) %{_sharedstatedir}/%{name}

%changelog
* Mon Jan 29 2025 o3willard-AI <o3willard@yahoo.com> - 1.0.0-1
- Initial package release
