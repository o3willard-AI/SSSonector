Name:           sssonector
Version:        %{_version}
Release:        %{_release}%{?dist}
Summary:        Secure SSL Tunnel Service
License:        MIT
URL:            https://github.com/o3willard-AI/SSSonector
BuildArch:      x86_64

%description
SSSonector is a secure SSL tunnel service for remote office connectivity.
It provides a persistent TLS 1.3 tunnel with EU-exportable cipher suites,
virtual network interfaces, and SNMP monitoring capabilities.

%install
rm -rf $RPM_BUILD_ROOT
mkdir -p $RPM_BUILD_ROOT/usr/bin
mkdir -p $RPM_BUILD_ROOT/etc/sssonector/certs
mkdir -p $RPM_BUILD_ROOT/etc/systemd/system
mkdir -p $RPM_BUILD_ROOT/var/log/sssonector

# Install binary
install -m 755 %{_build_dir}/sssonector-linux-amd64 $RPM_BUILD_ROOT/usr/bin/sssonector

# Install config files
install -m 644 %{_sourcedir}/configs/server.yaml $RPM_BUILD_ROOT/etc/sssonector/config.yaml
install -m 644 %{_sourcedir}/configs/client.yaml $RPM_BUILD_ROOT/etc/sssonector/client.yaml

# Install systemd service
install -m 644 %{_sourcedir}/scripts/service/systemd/sssonector.service $RPM_BUILD_ROOT/etc/systemd/system/sssonector.service

%files
%defattr(-,root,root,-)
/usr/bin/sssonector
%config(noreplace) /etc/sssonector/config.yaml
%config(noreplace) /etc/sssonector/client.yaml
/etc/systemd/system/sssonector.service
%dir /etc/sssonector
%dir /etc/sssonector/certs
%dir /var/log/sssonector

%pre
# Create sssonector group if it doesn't exist
getent group sssonector >/dev/null || groupadd -r sssonector

# Create sssonector user if it doesn't exist
getent passwd sssonector >/dev/null || \
    useradd -r -g sssonector -d /etc/sssonector \
    -s /sbin/nologin -c "SSSonector Service User" sssonector

%post
# Set permissions
chown -R sssonector:sssonector /etc/sssonector
chown -R sssonector:sssonector /var/log/sssonector
chmod 755 /etc/sssonector
chmod 700 /etc/sssonector/certs
chmod 755 /var/log/sssonector

# Reload systemd
systemctl daemon-reload

# Enable and start service if this is a fresh install
if [ $1 -eq 1 ]; then
    systemctl enable sssonector.service
    systemctl start sssonector.service
fi

%preun
# Stop service before uninstall
if [ $1 -eq 0 ]; then
    systemctl stop sssonector.service
    systemctl disable sssonector.service
fi

%postun
# Remove user and group on uninstall
if [ $1 -eq 0 ]; then
    userdel sssonector
    groupdel sssonector
fi

# Reload systemd
systemctl daemon-reload

%changelog
* Tue Jan 29 2025 O3Willard <support@o3willard.com> - 1.0.0-1
- Initial release
