.PHONY: all clean build test install package

VERSION := 1.0.0
COMMIT := $(shell git rev-parse --short HEAD)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')

GO_FLAGS := -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME)"

all: clean build test package

clean:
	rm -rf build/
	rm -rf dist/
	go clean

build:
	@echo "Building SSSonector..."
	mkdir -p build/bin
	GOOS=linux GOARCH=amd64 go build $(GO_FLAGS) -o build/bin/sssonector-linux-amd64 ./cmd/tunnel
	GOOS=darwin GOARCH=amd64 go build $(GO_FLAGS) -o build/bin/sssonector-darwin-amd64 ./cmd/tunnel
	GOOS=windows GOARCH=amd64 go build $(GO_FLAGS) -o build/bin/sssonector-windows-amd64.exe ./cmd/tunnel

test:
	@echo "Running tests..."
	go test -v ./...

install-deps:
	@echo "Installing build dependencies..."
	go mod download
	go mod tidy

installer-deps:
	@echo "Installing installer build dependencies..."
	which dpkg-deb || (echo "Installing dpkg..." && sudo apt-get install -y dpkg)
	which rpmbuild || (echo "Installing rpm..." && sudo apt-get install -y rpm)
	which makensis || (echo "Installing NSIS..." && sudo apt-get install -y nsis)
	which pkgbuild || (echo "macOS pkgbuild not available on this platform")

package: package-deb package-rpm package-macos package-windows

package-deb:
	@echo "Building Debian package..."
	./scripts/build-installers.sh deb

package-rpm:
	@echo "Building RPM package..."
	./scripts/build-installers.sh rpm

package-macos:
	@echo "Building macOS package..."
	./scripts/build-installers.sh macos

package-windows:
	@echo "Building Windows installer..."
	./scripts/build-installers.sh windows

install: build
	@echo "Installing SSSonector..."
	sudo install -D -m 755 build/bin/sssonector-linux-amd64 /usr/bin/sssonector
	sudo mkdir -p /etc/sssonector/certs
	sudo mkdir -p /var/log/sssonector
	sudo install -D -m 644 configs/server.yaml /etc/sssonector/config.yaml
	sudo install -D -m 644 configs/client.yaml /etc/sssonector/client.yaml
	sudo install -D -m 644 scripts/service/systemd/sssonector.service /etc/systemd/system/
	sudo systemctl daemon-reload
	@echo "Installation complete. Edit /etc/sssonector/config.yaml and run: sudo systemctl start sssonector"

uninstall:
	@echo "Uninstalling SSSonector..."
	sudo systemctl stop sssonector || true
	sudo systemctl disable sssonector || true
	sudo rm -f /usr/bin/sssonector
	sudo rm -f /etc/systemd/system/sssonector.service
	sudo rm -rf /etc/sssonector
	sudo rm -rf /var/log/sssonector
	sudo systemctl daemon-reload
	@echo "Uninstallation complete"

generate-certs:
	@echo "Generating certificates..."
	./scripts/generate-certs.sh

release:
	@echo "Creating release..."
	./scripts/release.sh $(VERSION)

# Development helpers
fmt:
	go fmt ./...

lint:
	golangci-lint run

dev: build
	./build/bin/sssonector-linux-amd64 -config configs/server.yaml

.DEFAULT_GOAL := all
