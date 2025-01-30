VERSION := 1.0.0
BINARY_NAME := sssonector
PLATFORMS := linux windows darwin

.PHONY: all clean build package

all: clean build package

clean:
	rm -rf build/
	mkdir -p build/installers
	mkdir -p build/windows
	mkdir -p build/rpm/BUILD
	mkdir -p build/rpm/RPMS
	mkdir -p build/rpm/SOURCES
	mkdir -p build/rpm/SPECS
	mkdir -p build/rpm/SRPMS

build:
	# Linux build
	GOOS=linux GOARCH=amd64 go build -o build/$(BINARY_NAME)-linux-amd64 ./cmd/tunnel
	# Windows build
	GOOS=windows GOARCH=amd64 go build -o build/windows/$(BINARY_NAME).exe ./cmd/tunnel
	# macOS build
	GOOS=darwin GOARCH=amd64 go build -o build/$(BINARY_NAME)-darwin-amd64 ./cmd/tunnel

package: package-deb package-rpm package-windows package-macos

package-deb:
	mkdir -p build/deb/DEBIAN
	mkdir -p build/deb/usr/bin
	cp build/$(BINARY_NAME)-linux-amd64 build/deb/usr/bin/$(BINARY_NAME)
	chmod 755 build/deb/usr/bin/$(BINARY_NAME)
	echo "Package: $(BINARY_NAME)\nVersion: $(VERSION)\nArchitecture: amd64\nMaintainer: o3willard-AI\nDescription: SSL tunneling application" > build/deb/DEBIAN/control
	dpkg-deb --build build/deb build/$(BINARY_NAME)_$(VERSION)_amd64.deb

package-rpm:
	mkdir -p build/rpm/BUILD
	mkdir -p build/rpm/RPMS
	mkdir -p build/rpm/SOURCES
	mkdir -p build/rpm/SPECS
	mkdir -p build/rpm/SRPMS
	cp build/$(BINARY_NAME)-linux-amd64 build/rpm/SOURCES/$(BINARY_NAME)
	echo "Summary: SSL tunneling application\nName: $(BINARY_NAME)\nVersion: $(VERSION)\nRelease: 1\nLicense: MIT\nGroup: Applications/Internet\nBuildRoot: %{_tmppath}/%{name}-%{version}-%{release}-root\n\n%description\nSSL tunneling application\n\n%prep\n\n%build\n\n%install\nmkdir -p %{buildroot}/usr/bin\ncp %{_sourcedir}/$(BINARY_NAME) %{buildroot}/usr/bin/\n\n%files\n/usr/bin/$(BINARY_NAME)" > build/rpm/SPECS/$(BINARY_NAME).spec
	rpmbuild --define "_topdir $(PWD)/build/rpm" -bb build/rpm/SPECS/$(BINARY_NAME).spec
	cp build/rpm/RPMS/x86_64/$(BINARY_NAME)-$(VERSION)-1.x86_64.rpm build/

package-windows:
	makensis -DVERSION=$(VERSION) -DBINARY="$(PWD)/build/windows/$(BINARY_NAME).exe" -DOUTPUT="$(PWD)/build/$(BINARY_NAME)-$(VERSION)-setup.exe" installers/windows.nsi

package-macos:
	# Note: This would typically run on a macOS system
	# For now, we'll create a basic package structure
	mkdir -p build/macos/root/usr/local/bin
	cp build/$(BINARY_NAME)-darwin-amd64 build/macos/root/usr/local/bin/$(BINARY_NAME)
	chmod 755 build/macos/root/usr/local/bin/$(BINARY_NAME)
	# Note: actual pkgbuild command would be run on macOS
