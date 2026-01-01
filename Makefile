.PHONY: help build build-prod build-pwa run test test-unit test-integration test-config clean deps setup install lint fmt vet check dev pwa-install pwa-build pwa-dev

# Variables.1-dev.1-dev
BINARY_NAME=vimesrv
BUILD_DIR=./bin
CMD_DIR=./cmd/server
GO=go
GOFLAGS=

# Version information
VERSION=0.0.1-dev
GIT_COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
PACKAGE=github.com/thesystemicprogrammer/vimesrv/cmd/server

# Build flags
LDFLAGS=-s -w -X 'main.version=$(VERSION)' -X 'main.commit=$(GIT_COMMIT)' -X 'main.date=$(BUILD_DATE)'

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-20s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# Development commands
dev: ## Run server in development mode with hot reload (requires air)
	@./scripts/dev.sh

run: build ## Build and run the server
	@echo "Running $(BINARY_NAME)..."
	@$(BUILD_DIR)/$(BINARY_NAME)

# Build commands
build: ## Build the application for development
	@echo "Building $(BINARY_NAME) for development..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-prod: ## Build the application for production with optimizations
	@echo "Building $(BINARY_NAME) for production..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -trimpath -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Production build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-pwa: pwa-build ## Build the application with embedded PWA for production
	@echo "Building $(BINARY_NAME) with PWA for production..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -trimpath -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Production build with PWA complete: $(BUILD_DIR)/$(BINARY_NAME)"

# PWA commands
pwa-install: ## Install PWA dependencies
	@echo "Installing PWA dependencies..."
	@cd web/pwa && npm install

pwa-build: ## Build the PWA for production
	@echo "Building PWA for production..."
	@cd web/pwa && npm run build

pwa-dev: ## Run PWA development server
	@echo "Starting PWA dev server..."
	@cd web/pwa && npm start

# Testing commands
test: ## Run all tests
	@echo "Running all tests..."
	@$(GO) test -v -race -coverprofile=coverage.out ./...
	@echo "Coverage report generated: coverage.out"

test-unit: ## Run unit tests only
	@echo "Running unit tests..."
	@$(GO) test -v -race -short ./...

test-integration: ## Run integration tests only
	@echo "Running integration tests..."
	@$(GO) test -v -race -run Integration ./test/integration/...

test-coverage: test ## Run tests and show coverage report
	@echo "Generating coverage report..."
	@$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-form: ## Test with nice formatting
	@echo "Runnning all tests"
	gotestsum --format dots

# Code quality commands
fmt: ## Format code with gofmt
	@echo "Formatting code..."
	@$(GO) fmt ./...

vet: ## Run go vet
	@echo "Running go vet..."
	@$(GO) vet ./...

lint: ## Run golangci-lint (requires golangci-lint)
	@if command -v golangci-lint > /dev/null; then \
		echo "Running golangci-lint..."; \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

check: fmt vet ## Run fmt and vet

# Dependencies commands
deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@$(GO) mod download

deps-tidy: ## Tidy dependencies
	@echo "Tidying dependencies..."
	@$(GO) mod tidy

deps-verify: ## Verify dependencies
	@echo "Verifying dependencies..."
	@$(GO) mod verify

deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	@$(GO) get -u ./...
	@$(GO) mod tidy


# Docker commands (future)
docker-build: ## Build Docker image
	@echo "Docker support coming soon..."

docker-run: ## Run Docker container
	@echo "Docker support coming soon..."
