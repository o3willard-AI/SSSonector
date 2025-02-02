.PHONY: all build clean dist install test

all: build

build:
	go mod download
	go mod tidy
	mkdir -p build
	go build -o build/sssonector ./cmd/tunnel
	chmod 755 build/sssonector

clean:
	rm -rf build dist

dist: build
	mkdir -p dist/v1.0.0
	cp build/sssonector dist/v1.0.0/
	cp -r configs dist/v1.0.0/
	cp README.md dist/v1.0.0/
	cd dist/v1.0.0 && tar czf ../sssonector-1.0.0.tar.gz .
	cd dist/v1.0.0 && zip -r ../sssonector-1.0.0.zip .
	sha256sum dist/v1.0.0/../sssonector-1.0.0.tar.gz > dist/v1.0.0/../checksums.txt
	sha256sum dist/v1.0.0/../sssonector-1.0.0.zip >> dist/v1.0.0/../checksums.txt

install: build
	sudo mkdir -p /usr/bin
	sudo cp build/sssonector /usr/bin/
	sudo chmod 755 /usr/bin/sssonector
	sudo mkdir -p /etc/sssonector
	sudo cp -r configs/* /etc/sssonector/

test:
	go test -v ./...
