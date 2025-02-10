# Build configuration
BINARY_NAME=sssonector
BINARY_CONTROL=sssonectorctl
VERSION=1.0.0
BUILD_DIR=build
PACKAGE=github.com/o3willard-AI/SSSonector

# Go build flags
LDFLAGS=-ldflags "-X main.Version=${VERSION}"
GO_BUILD=go build ${LDFLAGS}

# Docker configuration
DOCKER_IMAGE=sssonector
DOCKER_TAG=latest

# Default target
.PHONY: all
all: clean build

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf ${BUILD_DIR}

# Create build directory structure
.PHONY: prepare
prepare:
	mkdir -p ${BUILD_DIR}/linux/amd64 ${BUILD_DIR}/linux/arm64
	mkdir -p ${BUILD_DIR}/windows/amd64
	mkdir -p ${BUILD_DIR}/darwin/amd64 ${BUILD_DIR}/darwin/arm64
	mkdir -p ${BUILD_DIR}/docker
	mkdir -p ${BUILD_DIR}/packages
	mkdir -p ${BUILD_DIR}/docs

# Build for all platforms
.PHONY: build
build: prepare deps build-linux build-windows build-darwin build-docker build-security docs k8s
	@for file in README.md LICENSE CHANGELOG.md; do \
		if [ -f $$file ]; then \
			echo "Copying $$file..."; \
			cp $$file ${BUILD_DIR}/; \
		fi \
	done
	tar czf ${BUILD_DIR}/${BINARY_NAME}-${VERSION}-release.tar.gz -C ${BUILD_DIR} .

# Linux builds
.PHONY: build-linux
build-linux: build-linux-amd64 build-linux-arm64

.PHONY: build-linux-amd64
build-linux-amd64:
	GOOS=linux GOARCH=amd64 ${GO_BUILD} -o ${BUILD_DIR}/linux/amd64/${BINARY_NAME} ./cmd/daemon/main.go
	GOOS=linux GOARCH=amd64 ${GO_BUILD} -o ${BUILD_DIR}/linux/amd64/${BINARY_CONTROL} ./cmd/sssonectorctl/main.go
	cp init/systemd/sssonector.service ${BUILD_DIR}/linux/amd64/
	cp scripts/install.sh ${BUILD_DIR}/linux/amd64/
	tar czf ${BUILD_DIR}/packages/${BINARY_NAME}-${VERSION}-linux-amd64.tar.gz -C ${BUILD_DIR}/linux/amd64 .

.PHONY: build-linux-arm64
build-linux-arm64:
	GOOS=linux GOARCH=arm64 ${GO_BUILD} -o ${BUILD_DIR}/linux/arm64/${BINARY_NAME} ./cmd/daemon/main.go
	GOOS=linux GOARCH=arm64 ${GO_BUILD} -o ${BUILD_DIR}/linux/arm64/${BINARY_CONTROL} ./cmd/sssonectorctl/main.go
	cp init/systemd/sssonector.service ${BUILD_DIR}/linux/arm64/
	cp scripts/install.sh ${BUILD_DIR}/linux/arm64/
	tar czf ${BUILD_DIR}/packages/${BINARY_NAME}-${VERSION}-linux-arm64.tar.gz -C ${BUILD_DIR}/linux/arm64 .

# Windows builds
.PHONY: build-windows
build-windows: build-windows-amd64

.PHONY: build-windows-amd64
build-windows-amd64:
	GOOS=windows GOARCH=amd64 ${GO_BUILD} -o ${BUILD_DIR}/windows/amd64/${BINARY_NAME}.exe ./cmd/daemon/main.go
	GOOS=windows GOARCH=amd64 ${GO_BUILD} -o ${BUILD_DIR}/windows/amd64/${BINARY_CONTROL}.exe ./cmd/sssonectorctl/main.go
	cp scripts/install.ps1 ${BUILD_DIR}/windows/amd64/
	zip -j ${BUILD_DIR}/packages/${BINARY_NAME}-${VERSION}-windows-amd64.zip ${BUILD_DIR}/windows/amd64/*

# macOS builds
.PHONY: build-darwin
build-darwin: build-darwin-amd64 build-darwin-arm64

.PHONY: build-darwin-amd64
build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 ${GO_BUILD} -o ${BUILD_DIR}/darwin/amd64/${BINARY_NAME} ./cmd/daemon/main.go
	GOOS=darwin GOARCH=amd64 ${GO_BUILD} -o ${BUILD_DIR}/darwin/amd64/${BINARY_CONTROL} ./cmd/sssonectorctl/main.go
	cp init/launchd/com.o3willard.sssonector.plist ${BUILD_DIR}/darwin/amd64/
	cp scripts/install_macos.sh ${BUILD_DIR}/darwin/amd64/
	tar czf ${BUILD_DIR}/packages/${BINARY_NAME}-${VERSION}-darwin-amd64.tar.gz -C ${BUILD_DIR}/darwin/amd64 .

.PHONY: build-darwin-arm64
build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 ${GO_BUILD} -o ${BUILD_DIR}/darwin/arm64/${BINARY_NAME} ./cmd/daemon/main.go
	GOOS=darwin GOARCH=arm64 ${GO_BUILD} -o ${BUILD_DIR}/darwin/arm64/${BINARY_CONTROL} ./cmd/sssonectorctl/main.go
	cp init/launchd/com.o3willard.sssonector.plist ${BUILD_DIR}/darwin/arm64/
	cp scripts/install_macos.sh ${BUILD_DIR}/darwin/arm64/
	tar czf ${BUILD_DIR}/packages/${BINARY_NAME}-${VERSION}-darwin-arm64.tar.gz -C ${BUILD_DIR}/darwin/arm64 .

# Docker build (if available)
.PHONY: build-docker
build-docker:
	@if command -v docker >/dev/null 2>&1; then \
		echo "Building Docker image..."; \
		docker build -t ${DOCKER_IMAGE}:${DOCKER_TAG} . && \
		docker save ${DOCKER_IMAGE}:${DOCKER_TAG} > ${BUILD_DIR}/docker/${BINARY_NAME}-${VERSION}.tar; \
	else \
		echo "Docker not found, skipping Docker build"; \
	fi

# Security policy builds (if tools available)
.PHONY: build-security
build-security: build-selinux build-apparmor

.PHONY: build-selinux
build-selinux:
	@if command -v checkmodule >/dev/null 2>&1 && command -v semodule_package >/dev/null 2>&1; then \
		echo "Building SELinux policy..."; \
		cd security/selinux && ./build_policy.sh && \
		cp *.pp ${BUILD_DIR}/linux/amd64/ && \
		cp *.pp ${BUILD_DIR}/linux/arm64/; \
	else \
		echo "SELinux policy tools not found, skipping SELinux policy build"; \
	fi

.PHONY: build-apparmor
build-apparmor:
	@if [ -f security/apparmor/usr.local.bin.sssonector ]; then \
		echo "Installing AppArmor profile..."; \
		cp security/apparmor/usr.local.bin.sssonector ${BUILD_DIR}/linux/amd64/ && \
		cp security/apparmor/usr.local.bin.sssonector ${BUILD_DIR}/linux/arm64/; \
	else \
		echo "AppArmor profile not found, skipping AppArmor installation"; \
	fi

# Test targets
.PHONY: test
test:
	go test -v ./...

.PHONY: test-integration
test-integration:
	go test -v -tags=integration ./test/integration/...

# Install dependencies
.PHONY: deps
deps:
	go mod download

# Generate documentation (if available)
.PHONY: docs
docs:
	@if [ -d docs/deployment ]; then \
		echo "Copying deployment documentation..."; \
		cp -f docs/deployment/DEPLOYMENT.md ${BUILD_DIR}/ 2>/dev/null || true; \
		cp -f docs/deployment/KUBERNETES.md ${BUILD_DIR}/ 2>/dev/null || true; \
	fi
	@if [ -d docs/config ]; then \
		echo "Copying configuration documentation..."; \
		cp -r docs/config ${BUILD_DIR}/; \
	fi
	@if [ -d docs/implementation ]; then \
		echo "Copying implementation documentation..."; \
		cp -r docs/implementation ${BUILD_DIR}/; \
	fi

# Kubernetes manifests (if available)
.PHONY: k8s
k8s:
	@if [ -d deploy/kubernetes ]; then \
		echo "Copying Kubernetes manifests..."; \
		cp -r deploy/kubernetes ${BUILD_DIR}/; \
	else \
		echo "Kubernetes manifests not found, skipping"; \
	fi

# Release bundle
.PHONY: release
release: build build-security docs k8s
	cp README.md ${BUILD_DIR}/
	cp LICENSE ${BUILD_DIR}/
	cp CHANGELOG.md ${BUILD_DIR}/
	tar czf ${BUILD_DIR}/${BINARY_NAME}-${VERSION}-release.tar.gz -C ${BUILD_DIR} .

# Development helpers
.PHONY: run
run: build
	./${BUILD_DIR}/linux/amd64/${BINARY_NAME}

.PHONY: dev
dev:
	go run ./cmd/daemon

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: version
version:
	@echo ${VERSION}
