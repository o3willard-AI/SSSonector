.PHONY: all build clean dist install test deb rpm win mac

all: build

build:
go mod download
go mod tidy
mkdir -p build
go build -o build/sssonector ./cmd/tunnel
chmod 755 build/sssonector

clean:
rm -rf build dist

deb: build
mkdir -p build/deb/DEBIAN
mkdir -p build/deb/usr/bin
mkdir -p build/deb/etc/sssonector
cp build/sssonector build/deb/usr/bin/
cp -r configs/* build/deb/etc/sssonector/
echo "Package: sssonector\nVersion: 1.0.0\nArchitecture: amd64\nMaintainer: o3willard-AI\nDescription: SSL tunneling application" > build/deb/DEBIAN/control
dpkg-deb --build build/deb dist/sssonector_1.0.0_amd64.deb

rpm: build
mkdir -p build/rpm/{BUILD,RPMS,SOURCES,SPECS,SRPMS}
cp build/sssonector build/rpm/SOURCES/
cp -r configs build/rpm/SOURCES/
echo "Name: sssonector\nVersion: 1.0.0\nRelease: 1\nSummary: SSL tunneling application\nLicense: Proprietary\n\n%description\nSSL tunneling application\n\n%files\n/usr/bin/sssonector\n/etc/sssonector/*" > build/rpm/SPECS/sssonector.spec
rpmbuild -bb --define "_topdir $(PWD)/build/rpm" build/rpm/SPECS/sssonector.spec
cp build/rpm/RPMS/x86_64/sssonector-1.0.0-1.x86_64.rpm dist/

win: build
mkdir -p build/win
cp build/sssonector build/win/
cp -r configs build/win/
makensis installers/windows.nsi
cp build/sssonector-1.0.0-setup.exe dist/

mac: build
mkdir -p build/macos/root/usr/local/bin
mkdir -p build/macos/root/etc/sssonector
cp build/sssonector build/macos/root/usr/local/bin/
cp -r configs/* build/macos/root/etc/sssonector/
pkgbuild --root build/macos/root \
--identifier com.o3willard-ai.sssonector \
--version 1.0.0 \
--install-location / \
dist/sssonector-1.0.0-macos.pkg

dist: build deb rpm win mac
mkdir -p dist/v1.0.0
cp build/sssonector dist/v1.0.0/
cp -r configs dist/v1.0.0/
cp README.md dist/v1.0.0/
cd dist/v1.0.0 && tar czf ../sssonector-1.0.0.tar.gz .
cd dist/v1.0.0 && zip -r ../sssonector-1.0.0.zip .
cp dist/sssonector_1.0.0_amd64.deb dist/v1.0.0/
cp dist/sssonector-1.0.0-1.x86_64.rpm dist/v1.0.0/
cp dist/sssonector-1.0.0-setup.exe dist/v1.0.0/
cp dist/sssonector-1.0.0-macos.pkg dist/v1.0.0/
cd dist && sha256sum sssonector-1.0.0.tar.gz sssonector-1.0.0.zip \
sssonector_1.0.0_amd64.deb sssonector-1.0.0-1.x86_64.rpm \
sssonector-1.0.0-setup.exe sssonector-1.0.0-macos.pkg > checksums.txt

install: build
sudo mkdir -p /usr/bin
sudo cp build/sssonector /usr/bin/
sudo chmod 755 /usr/bin/sssonector
sudo mkdir -p /etc/sssonector
sudo cp -r configs/* /etc/sssonector/

test:
go test -v ./...
