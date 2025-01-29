BINARY_NAME=sssonector
BUILD_DIR=bin
CERT_DIR=certs
CONFIG_DIR=configs

.PHONY: all build clean test generate-certs install uninstall

all: clean generate-certs build

build:
	@echo "Building..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/tunnel

clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@go clean

test:
	@echo "Running tests..."
	@go test -v ./...

generate-certs:
	@echo "Generating certificates..."
	@mkdir -p $(CERT_DIR)
	@openssl req -x509 -newkey rsa:4096 -keyout $(CERT_DIR)/server.key -out $(CERT_DIR)/server.crt -days 365 -nodes -subj "/CN=sssonector-server"
	@openssl req -x509 -newkey rsa:4096 -keyout $(CERT_DIR)/client.key -out $(CERT_DIR)/client.crt -days 365 -nodes -subj "/CN=sssonector-client"
	@chmod 600 $(CERT_DIR)/*.key
	@chmod 644 $(CERT_DIR)/*.crt

install: build
	@echo "Installing..."
	@sudo mkdir -p /etc/sssonector/certs
	@sudo cp $(CERT_DIR)/* /etc/sssonector/certs/
	@sudo cp $(CONFIG_DIR)/*.yaml /etc/sssonector/
	@sudo chmod 600 /etc/sssonector/certs/*.key
	@sudo chmod 644 /etc/sssonector/certs/*.crt
	@sudo install -m 755 $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/

uninstall:
	@echo "Uninstalling..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@sudo rm -rf /etc/sssonector

# Cross-compilation targets
.PHONY: build-linux build-darwin build-windows

build-linux:
	@echo "Building for Linux..."
	@GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/tunnel

build-darwin:
	@echo "Building for macOS..."
	@GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/tunnel

build-windows:
	@echo "Building for Windows..."
	@GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/tunnel

# Build all platforms
.PHONY: build-all
build-all: build-linux build-darwin build-windows

# Development helpers
.PHONY: fmt vet lint
fmt:
	@echo "Formatting code..."
	@go fmt ./...

vet:
	@echo "Running go vet..."
	@go vet ./...

lint:
	@echo "Running linter..."
	@golangci-lint run
