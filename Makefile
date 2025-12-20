.PHONY: build test clean run docker-build docker-run help

# Build variables
BINARY_NAME=keyrafted
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags="-s -w -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	@echo "Building ${BINARY_NAME}..."
	@CGO_ENABLED=0 go build ${LDFLAGS} -o ${BINARY_NAME} .
	@echo "✓ Build complete: ${BINARY_NAME}"

build-linux: ## Build for Linux
	@echo "Building ${BINARY_NAME} for Linux..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-linux-amd64 .
	@echo "✓ Build complete: ${BINARY_NAME}-linux-amd64"

build-all: ## Build for all platforms
	@echo "Building for all platforms..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-linux-amd64 .
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-darwin-amd64 .
	@CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BINARY_NAME}-darwin-arm64 .
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BINARY_NAME}-windows-amd64.exe .
	@echo "✓ All builds complete"

test: ## Run tests
	@echo "Running tests..."
	@go test ./tests/unit -v
	@go test ./tests/integration -v
	@echo "✓ Tests passed"

test-coverage: ## Run tests with coverage
	@echo "Running tests with coverage..."
	@go test ./tests/... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -f ${BINARY_NAME}
	@rm -f ${BINARY_NAME}-*
	@rm -f coverage.out coverage.html
	@rm -rf data/
	@echo "✓ Clean complete"

run: build ## Build and run
	@./keyrafted start --data-dir ./data

docker-build: ## Build Docker image
	@echo "Building Docker image..."
	@docker build -t keyraft/keyrafted:latest .
	@docker build -t keyraft/keyrafted:${VERSION} .
	@echo "✓ Docker image built"

docker-run: ## Run with Docker Compose
	@echo "Starting Keyraft with Docker Compose..."
	@docker-compose up -d
	@echo "✓ Keyraft running on http://localhost:7200"

docker-stop: ## Stop Docker Compose
	@docker-compose down

docker-logs: ## View Docker logs
	@docker-compose logs -f

install: build ## Install binary to /usr/local/bin
	@echo "Installing ${BINARY_NAME}..."
	@sudo cp ${BINARY_NAME} /usr/local/bin/
	@echo "✓ Installed to /usr/local/bin/${BINARY_NAME}"

lint: ## Run linter
	@echo "Running linter..."
	@go vet ./...
	@go fmt ./...
	@echo "✓ Lint complete"

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "✓ Dependencies ready"

.DEFAULT_GOAL := help

