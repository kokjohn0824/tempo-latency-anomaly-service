.PHONY: help test test-coverage test-short test-verbose clean build docker-build docker-up docker-down run dev-up dev-down dev-restart swagger

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
	@echo "  make run            - Run the service locally (foreground)"
	@echo "  make swagger        - Regenerate Swagger documentation"
	@echo "  make docker-build   - Build Docker image (with pre-test)"
	@echo "  make docker-up      - Start local Docker Compose services"
	@echo "  make docker-down    - Stop and remove local Docker Compose services"
	@echo "  make dev-up         - Start Redis and run service locally"
	@echo "  make dev-down       - Stop local development environment"
	@echo "  make dev-restart    - Restart local service (keep Redis running)"

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

# Start Docker Compose services (port 8081)
docker-up:
	@echo "Starting Docker Compose services..."
	docker compose -f docker/compose.yml up -d
	@echo ""
	@echo "Services started!"
	@echo "  - Redis: localhost:6379"
	@echo "  - API: http://localhost:8081"
	@echo "  - Swagger UI: http://localhost:8081/swagger/index.html"
	@echo "  - Health Check: http://localhost:8081/healthz"
	@echo ""
	@echo "Waiting for services to be ready..."
	@sleep 5
	@curl -s http://localhost:8081/healthz > /dev/null && echo "✓ Service is healthy!" || echo "⚠ Service health check failed (may need more time)"

# Stop and remove Docker Compose services
docker-down:
	@echo "Stopping Docker Compose services..."
	docker compose -f docker/compose.yml down

# Run the service locally
run:
	@echo "Running service locally..."
	go run ./cmd/server -config=configs/config.dev.yaml

# Regenerate Swagger documentation
swagger:
	@echo "Regenerating Swagger documentation..."
	swag init -g cmd/server/main.go -o docs
	@echo "Swagger documentation updated!"

# Start local development environment (Redis + Service)
dev-up:
	@echo "Starting Redis..."
	docker compose -f docker/compose.yml up -d redis
	@echo "Waiting for Redis to be ready..."
	@sleep 3
	@echo "Starting service..."
	@echo "Run 'make run' in another terminal or use 'go run ./cmd/server -config=configs/config.dev.yaml &'"

# Stop local development environment
dev-down:
	@echo "Stopping service..."
	@-pkill -f "go run.*cmd/server" || true
	@-lsof -ti :8081 | xargs kill 2>/dev/null || true
	@echo "Stopping Redis..."
	docker compose -f docker/compose.yml down redis
	@echo "Development environment stopped"

# Restart local service (keep Redis running)
dev-restart:
	@echo "Stopping service..."
	@-pkill -f "go run.*cmd/server" || true
	@-lsof -ti :8081 | xargs kill 2>/dev/null || true
	@sleep 2
	@echo "Starting service..."
	@nohup go run ./cmd/server -config=configs/config.dev.yaml > /tmp/tempo-anomaly-service.log 2>&1 &
	@sleep 3
	@echo "Service restarted! Check logs: tail -f /tmp/tempo-anomaly-service.log"
	@curl -s http://localhost:8081/healthz && echo " - Service is healthy!" || echo " - Service health check failed"
