.PHONY: help test test-coverage test-short test-verbose clean build docker-build

# Default target
help:
	@echo "Tempo Latency Anomaly Service - Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  make test           - Run all unit tests"
	@echo "  make test-coverage  - Run tests with coverage report"
	@echo "  make test-short     - Run short tests only (fast)"
	@echo "  make test-verbose   - Run tests with verbose output"
	@echo "  make clean          - Clean build artifacts and test cache"
	@echo "  make build          - Build the service binary"
	@echo "  make docker-build   - Build Docker image (with pre-test)"

# Run all unit tests
test:
	@echo "Running unit tests..."
	go test -race -cover ./internal/...

# Generate coverage report
test-coverage:
	@echo "Generating coverage report..."
	go test -race -coverprofile=coverage.out -covermode=atomic ./internal/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@go tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'

# Run short tests (skip long-running tests)
test-short:
	@echo "Running short tests..."
	go test -short -v ./internal/...

# Run tests with verbose output
test-verbose:
	@echo "Running tests with verbose output..."
	go test -v -race -cover ./internal/...

# Clean build artifacts and test cache
clean:
	@echo "Cleaning..."
	go clean -testcache
	rm -f coverage.out coverage.html
	rm -f tempo-anomaly

# Build the service binary
build: test
	@echo "Building service..."
	go build -o tempo-anomaly ./cmd/server

# Build Docker image (runs tests first)
docker-build: test
	@echo "Tests passed! Building Docker image..."
	docker compose -f docker/compose.yml build
