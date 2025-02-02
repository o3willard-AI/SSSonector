BINARY_NAME=sssonector
VERSION=1.0.0
BUILD_DIR=build
DIST_DIR=dist/v$(VERSION)
INSTALL_DIR=/usr/bin
CONFIG_DIR=/etc/sssonector
LOG_DIR=/var/log/sssonector

.PHONY: all build clean install uninstall dist

all: build

deps:
	go mod download
	go mod tidy

build: deps
	mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/tunnel
	chmod 755 $(BUILD_DIR)/$(BINARY_NAME)

clean:
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)

install: build
	# Create directories
	mkdir -p $(CONFIG_DIR)
	mkdir -p $(CONFIG_DIR)/certs
	mkdir -p $(LOG_DIR)
	
	# Install binary
	install -m 755 $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	
	# Install config files if they don't exist
	test -f $(CONFIG_DIR)/config.yaml || install -m 644 configs/server.yaml $(CONFIG_DIR)/config.yaml
	test -f $(CONFIG_DIR)/client.yaml || install -m 644 configs/client.yaml $(CONFIG_DIR)/client.yaml
	
	# Set permissions
	chown -R root:root $(CONFIG_DIR)
	chmod -R 755 $(CONFIG_DIR)
	chmod 644 $(CONFIG_DIR)/*.yaml
	chown -R root:root $(LOG_DIR)
	chmod 755 $(LOG_DIR)

uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY_NAME)
	rm -rf $(CONFIG_DIR)
	rm -rf $(LOG_DIR)

dist: build
	mkdir -p $(DIST_DIR)
	cp $(BUILD_DIR)/$(BINARY_NAME) $(DIST_DIR)/
	cp -r configs $(DIST_DIR)/
	cp README.md $(DIST_DIR)/
	cd $(DIST_DIR) && tar czf ../$(BINARY_NAME)-$(VERSION).tar.gz .
	cd $(DIST_DIR) && zip -r ../$(BINARY_NAME)-$(VERSION).zip .
	sha256sum $(DIST_DIR)/../$(BINARY_NAME)-$(VERSION).tar.gz > $(DIST_DIR)/../checksums.txt
	sha256sum $(DIST_DIR)/../$(BINARY_NAME)-$(VERSION).zip >> $(DIST_DIR)/../checksums.txt

test:
	go test -v ./...

lint:
	golangci-lint run

.DEFAULT_GOAL := build
