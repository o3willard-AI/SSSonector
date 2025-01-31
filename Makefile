# Build configuration
BINARY_NAME=sssonector
VERSION=1.0.0
BUILD_DIR=build
DIST_DIR=dist/v$(VERSION)
CHECKSUMS_FILE=$(BUILD_DIR)/checksums.txt

# Go build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -s -w"
GOFLAGS=-trimpath

# OS/ARCH pairs
WINDOWS_TARGETS=windows/amd64 windows/386
LINUX_TARGETS=linux/amd64 linux/386 linux/arm linux/arm64
DARWIN_TARGETS=darwin/amd64 darwin/arm64

.PHONY: all clean deps lint test build dist windows linux darwin checksums

all: clean deps lint test build

clean:
rm -rf $(BUILD_DIR) $(DIST_DIR)
mkdir -p $(BUILD_DIR) $(DIST_DIR)

deps:
go mod download
go mod tidy

lint:
golangci-lint run

test:
go test -v -race -cover ./...

build: windows linux darwin

windows: $(WINDOWS_TARGETS)

$(WINDOWS_TARGETS): %:
GOOS=$(word 1,$(subst /, ,$@)) \
GOARCH=$(word 2,$(subst /, ,$@)) \
CGO_ENABLED=0 \
go build $(GOFLAGS) $(LDFLAGS) \
-tags server \
-o $(BUILD_DIR)/$(BINARY_NAME)_server_$(word 1,$(subst /, ,$@))_$(word 2,$(subst /, ,$@)).exe \
./cmd/tunnel
GOOS=$(word 1,$(subst /, ,$@)) \
GOARCH=$(word 2,$(subst /, ,$@)) \
CGO_ENABLED=0 \
go build $(GOFLAGS) $(LDFLAGS) \
-tags client \
-o $(BUILD_DIR)/$(BINARY_NAME)_client_$(word 1,$(subst /, ,$@))_$(word 2,$(subst /, ,$@)).exe \
./cmd/tunnel

linux: $(LINUX_TARGETS)

$(LINUX_TARGETS): %:
GOOS=$(word 1,$(subst /, ,$@)) \
GOARCH=$(word 2,$(subst /, ,$@)) \
CGO_ENABLED=0 \
go build $(GOFLAGS) $(LDFLAGS) \
-tags server \
-o $(BUILD_DIR)/$(BINARY_NAME)_server_$(word 1,$(subst /, ,$@))_$(word 2,$(subst /, ,$@)) \
./cmd/tunnel
GOOS=$(word 1,$(subst /, ,$@)) \
GOARCH=$(word 2,$(subst /, ,$@)) \
CGO_ENABLED=0 \
go build $(GOFLAGS) $(LDFLAGS) \
-tags client \
-o $(BUILD_DIR)/$(BINARY_NAME)_client_$(word 1,$(subst /, ,$@))_$(word 2,$(subst /, ,$@)) \
./cmd/tunnel
tar -czf $(DIST_DIR)/$(BINARY_NAME)_$(VERSION)_$(word 1,$(subst /, ,$@))_$(word 2,$(subst /, ,$@)).tar.gz \
-C $(BUILD_DIR) \
$(BINARY_NAME)_server_$(word 1,$(subst /, ,$@))_$(word 2,$(subst /, ,$@)) \
$(BINARY_NAME)_client_$(word 1,$(subst /, ,$@))_$(word 2,$(subst /, ,$@))

darwin: $(DARWIN_TARGETS)

$(DARWIN_TARGETS): %:
GOOS=$(word 1,$(subst /, ,$@)) \
GOARCH=$(word 2,$(subst /, ,$@)) \
CGO_ENABLED=0 \
go build $(GOFLAGS) $(LDFLAGS) \
-tags server \
-o $(BUILD_DIR)/$(BINARY_NAME)_server_$(word 1,$(subst /, ,$@))_$(word 2,$(subst /, ,$@)) \
./cmd/tunnel
GOOS=$(word 1,$(subst /, ,$@)) \
GOARCH=$(word 2,$(subst /, ,$@)) \
CGO_ENABLED=0 \
go build $(GOFLAGS) $(LDFLAGS) \
-tags client \
-o $(BUILD_DIR)/$(BINARY_NAME)_client_$(word 1,$(subst /, ,$@))_$(word 2,$(subst /, ,$@)) \
./cmd/tunnel
tar -czf $(DIST_DIR)/$(BINARY_NAME)_$(VERSION)_$(word 1,$(subst /, ,$@))_$(word 2,$(subst /, ,$@)).tar.gz \
-C $(BUILD_DIR) \
$(BINARY_NAME)_server_$(word 1,$(subst /, ,$@))_$(word 2,$(subst /, ,$@)) \
$(BINARY_NAME)_client_$(word 1,$(subst /, ,$@))_$(word 2,$(subst /, ,$@))

checksums:
cd $(DIST_DIR) && sha256sum * > ../$(CHECKSUMS_FILE)

dist: clean deps build checksums
