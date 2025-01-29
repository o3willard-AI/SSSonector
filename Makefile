.PHONY: all build test clean dist fmt lint vet run-server run-client

# Build configuration
BINARY_NAME=SSSonector
VERSION=1.0.0
LDFLAGS=-ldflags "-X main.Version=$(VERSION)"

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet
GOLINT=golangci-lint

# Directories
DIST_DIR=dist
BUILD_DIR=build

all: clean fmt lint vet test build

build:
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) cmd/tunnel/main.go

test:
	$(GOTEST) -v ./...

clean:
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)

dist:
	./scripts/build.sh

fmt:
	$(GOFMT) ./...

lint:
	$(GOLINT) run

vet:
	$(GOVET) ./...

# Development commands
run-server: build
	sudo ./$(BUILD_DIR)/$(BINARY_NAME) --config configs/server.yaml

run-client: build
	sudo ./$(BUILD_DIR)/$(BINARY_NAME) --config configs/client.yaml

# Docker commands for monitoring
monitoring-up:
	cd monitoring && docker-compose up -d

monitoring-down:
	cd monitoring && docker-compose down

# Testing commands
test-certs:
	./scripts/test-certs.sh

test-snmp:
	./scripts/test-snmp.sh

# Installation commands
install-linux: build
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	sudo mkdir -p /etc/SSSonector
	sudo cp -r configs /etc/SSSonector/
	sudo cp scripts/service/systemd/SSSonector.service /etc/systemd/system/
	sudo systemctl daemon-reload

install-darwin: build
	sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	sudo mkdir -p /etc/SSSonector
	sudo cp -r configs /etc/SSSonector/
	sudo cp scripts/service/launchd/com.SSSonector.plist /Library/LaunchDaemons/
	sudo launchctl load /Library/LaunchDaemons/com.SSSonector.plist

install-windows: build
	powershell -ExecutionPolicy Bypass -File scripts/service/windows/install-service.ps1

# Help target
help:
	@echo "Available targets:"
	@echo "  all          - Run clean, fmt, lint, vet, test, and build"
	@echo "  build        - Build the binary"
	@echo "  test         - Run tests"
	@echo "  clean        - Remove build artifacts"
	@echo "  dist         - Create distribution packages"
	@echo "  fmt          - Format code"
	@echo "  lint         - Run linter"
	@echo "  vet          - Run go vet"
	@echo "  run-server   - Build and run server"
	@echo "  run-client   - Build and run client"
	@echo "  monitoring-up   - Start monitoring stack"
	@echo "  monitoring-down - Stop monitoring stack"
	@echo "  test-certs   - Test certificate generation"
	@echo "  test-snmp    - Test SNMP functionality"
	@echo "  install-linux   - Install on Linux"
	@echo "  install-darwin  - Install on macOS"
	@echo "  install-windows - Install on Windows"
