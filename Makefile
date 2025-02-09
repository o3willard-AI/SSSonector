# SSSonector Makefile

# Build variables
BINARY_NAME=sssonector
VERSION=$(shell git describe --tags --always --dirty)
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT_HASH=$(shell git rev-parse HEAD)
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.CommitHash=$(COMMIT_HASH)"

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Directories
CMD_DIR=./cmd
DIST_DIR=./dist
PACKAGE_DIR=./packages

# Supported platforms
PLATFORMS=linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

# Default target
.PHONY: all
all: clean deps test build

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -rf $(DIST_DIR)
	@rm -rf $(PACKAGE_DIR)
	@$(GOCLEAN)

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	@$(GOMOD) download
	@$(GOMOD) verify

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	@$(GOTEST) -v ./...

# Run integration tests
.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	@$(GOTEST) -v -tags=integration ./test/integration/...

# Run benchmarks
.PHONY: bench
bench:
	@echo "Running benchmarks..."
	@$(GOTEST) -bench=. -benchmem ./...

# Build for current platform
.PHONY: build
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(DIST_DIR)
	@$(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME) $(CMD_DIR)/$(BINARY_NAME)
	@$(GOBUILD) $(LDFLAGS) -o $(DIST_DIR)/$(BINARY_NAME)ctl $(CMD_DIR)/$(BINARY_NAME)ctl

# Cross-compile for all platforms
.PHONY: release
release: clean deps test
	@echo "Building release packages..."
	@mkdir -p $(PACKAGE_DIR)
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%/*} \
		GOARCH=$${platform#*/} \
		OUTPUT_NAME=$(BINARY_NAME)_$${platform%/*}_$${platform#*/} \
		OUTPUT_DIR=$(PACKAGE_DIR)/$${platform%/*}_$${platform#*/} \
		; \
		echo "Building for $$GOOS/$$GOARCH..." ; \
		mkdir -p $$OUTPUT_DIR ; \
		GOOS=$$GOOS GOARCH=$$GOARCH $(GOBUILD) $(LDFLAGS) \
			-o $$OUTPUT_DIR/$(BINARY_NAME)$${GOOS:+_}$${GOOS:+.exe} \
			$(CMD_DIR)/$(BINARY_NAME) ; \
		GOOS=$$GOOS GOARCH=$$GOARCH $(GOBUILD) $(LDFLAGS) \
			-o $$OUTPUT_DIR/$(BINARY_NAME)ctl$${GOOS:+_}$${GOOS:+.exe} \
			$(CMD_DIR)/$(BINARY_NAME)ctl ; \
		cp -r config/ $$OUTPUT_DIR/ ; \
		cp -r security/ $$OUTPUT_DIR/ ; \
		cp -r init/ $$OUTPUT_DIR/ ; \
		cp README.md LICENSE $$OUTPUT_DIR/ ; \
		cd $(PACKAGE_DIR) && \
		tar czf $$OUTPUT_NAME.tar.gz $${platform%/*}_$${platform#*/} && \
		cd - ; \
	done

# Install development tools
.PHONY: dev-tools
dev-tools:
	@echo "Installing development tools..."
	@$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint
	@$(GOGET) golang.org/x/tools/cmd/goimports
	@$(GOGET) github.com/golang/mock/mockgen

# Run linter
.PHONY: lint
lint:
	@echo "Running linter..."
	@golangci-lint run

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@goimports -w .

# Generate mocks
.PHONY: generate
generate:
	@echo "Generating mocks..."
	@go generate ./...

# Build Docker image
.PHONY: docker
docker:
	@echo "Building Docker image..."
	@docker build -t sssonector:$(VERSION) .

# Run security audit
.PHONY: audit
audit:
	@echo "Running security audit..."
	@go list -json -m all | nancy sleuth

# Create release tag
.PHONY: tag
tag:
	@echo "Creating release tag $(VERSION)..."
	@git tag -a $(VERSION) -m "Release $(VERSION)"
	@git push origin $(VERSION)

# Install locally
.PHONY: install
install: build
	@echo "Installing locally..."
	@sudo cp $(DIST_DIR)/$(BINARY_NAME) /usr/local/bin/
	@sudo cp $(DIST_DIR)/$(BINARY_NAME)ctl /usr/local/bin/
	@sudo mkdir -p /etc/$(BINARY_NAME)
	@sudo cp -r config/* /etc/$(BINARY_NAME)/
	@sudo mkdir -p /var/lib/$(BINARY_NAME)
	@sudo chown -R root:root /etc/$(BINARY_NAME)
	@sudo chmod -R 644 /etc/$(BINARY_NAME)
	@sudo chmod 755 /etc/$(BINARY_NAME)
	@sudo chown -R root:root /var/lib/$(BINARY_NAME)
	@sudo chmod -R 644 /var/lib/$(BINARY_NAME)
	@sudo chmod 755 /var/lib/$(BINARY_NAME)

# Uninstall locally
.PHONY: uninstall
uninstall:
	@echo "Uninstalling..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)ctl
	@sudo rm -rf /etc/$(BINARY_NAME)
	@sudo rm -rf /var/lib/$(BINARY_NAME)

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  all            - Clean, install dependencies, run tests, and build"
	@echo "  clean          - Remove build artifacts"
	@echo "  deps           - Install dependencies"
	@echo "  test           - Run unit tests"
	@echo "  test-integration - Run integration tests"
	@echo "  bench          - Run benchmarks"
	@echo "  build          - Build for current platform"
	@echo "  release        - Build release packages for all platforms"
	@echo "  dev-tools      - Install development tools"
	@echo "  lint           - Run linter"
	@echo "  fmt            - Format code"
	@echo "  generate       - Generate mocks"
	@echo "  docker         - Build Docker image"
	@echo "  audit          - Run security audit"
	@echo "  tag            - Create release tag"
	@echo "  install        - Install locally"
	@echo "  uninstall      - Uninstall locally"
	@echo "  help           - Show this help"
