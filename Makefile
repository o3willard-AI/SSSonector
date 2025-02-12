.PHONY: all build test clean benchmark lint coverage docs

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=sssonector
BENCHMARK_BINARY=benchmark

all: test build

build:
	$(GOBUILD) -o bin/$(BINARY_NAME) -v ./cmd/...
	$(GOBUILD) -o bin/$(BENCHMARK_BINARY) -v ./cmd/benchmark

test:
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -f bin/$(BINARY_NAME)
	rm -f bin/$(BENCHMARK_BINARY)
	rm -f coverage.out

benchmark:
	./bin/$(BENCHMARK_BINARY) \
		-connections 100 \
		-duration 30s \
		-payload 1024 \
		-interval 50ms \
		-warmup 5s

benchmark-stress:
	./bin/$(BENCHMARK_BINARY) \
		-connections 1000 \
		-duration 5m \
		-payload 4096 \
		-interval 10ms \
		-warmup 30s

lint:
	golangci-lint run

coverage:
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

deps:
	$(GOMOD) download
	$(GOMOD) verify
	$(GOGET) -u golang.org/x/lint/golint
	$(GOGET) -u github.com/golangci/golangci-lint/cmd/golangci-lint

docs:
	godoc -http=:6060

# Development targets
dev-setup: deps
	mkdir -p bin
	cp scripts/pre-commit .git/hooks/
	chmod +x .git/hooks/pre-commit

# CI targets
ci-test:
	$(GOTEST) -race -coverprofile=coverage.out ./...

ci-benchmark:
	./bin/$(BENCHMARK_BINARY) \
		-connections 50 \
		-duration 10s \
		-payload 1024 \
		-interval 100ms \
		-warmup 2s

# Docker targets
docker-build:
	docker build -t sssonector .

docker-run:
	docker run --rm sssonector

# Help target
help:
	@echo "Available targets:"
	@echo "  all          - Run tests and build binaries"
	@echo "  build        - Build the project binaries"
	@echo "  test         - Run all tests"
	@echo "  clean        - Clean build artifacts"
	@echo "  benchmark    - Run standard benchmarks"
	@echo "  benchmark-stress - Run stress benchmarks"
	@echo "  lint         - Run linters"
	@echo "  coverage     - Generate test coverage report"
	@echo "  deps         - Install dependencies"
	@echo "  docs         - Start documentation server"
	@echo "  dev-setup   - Set up development environment"
	@echo "  ci-test     - Run tests for CI"
	@echo "  ci-benchmark - Run benchmarks for CI"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
