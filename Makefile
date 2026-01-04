.PHONY: help build build-server build-worker build-all \
        build-prod-server build-prod-worker build-prod-all build-prod-pwa \
        build-prod-linux build-prod-pi build-prod-darwin build-prod-windows \
        release check-pwa-freshness \
        run run-worker \
        test test-unit test-integration test-config test-coverage test-form \
        clean setup deps deps-tidy deps-verify deps-update \
        lint fmt vet check \
        dev pwa-install pwa-build pwa-dev \
        docker-build docker-run

# Variables
BINARY_NAME=vimesrv
WORKER_BINARY_NAME=vimesrv-worker
BUILD_DIR=./bin
CMD_DIR=./cmd/server
WORKER_CMD_DIR=./cmd/worker
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

run-worker: build-worker ## Build and run the worker
	@echo "Running $(WORKER_BINARY_NAME)..."
	@$(BUILD_DIR)/$(WORKER_BINARY_NAME) -config configs/worker.yaml

# Build commands
build: build-server ## Build the server for development (alias for build-server)

build-server: ## Build the server for development
	@echo "Building $(BINARY_NAME) for development..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-worker: ## Build the worker for development
	@echo "Building $(WORKER_BINARY_NAME) for development..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(WORKER_BINARY_NAME) $(WORKER_CMD_DIR)
	@echo "Build complete: $(BUILD_DIR)/$(WORKER_BINARY_NAME)"

build-all: build-server build-worker ## Build both server and worker

check-pwa-freshness: ## Check if PWA build is up-to-date with sources
	@if [ ! -d web/pwa/dist ]; then \
		echo "ERROR: PWA not built. Run 'make pwa-build' first."; \
		exit 1; \
	fi
	@stale_file=$$(find web/pwa/src -type f -newer web/pwa/dist -print -quit 2>/dev/null); \
	if [ -n "$$stale_file" ]; then \
		echo "ERROR: PWA sources are newer than build (e.g., $$stale_file)"; \
		echo "Run 'make pwa-build' to rebuild the frontend."; \
		exit 1; \
	fi

build-prod-server: check-pwa-freshness ## Build the server for production with optimizations
	@echo "Building $(BINARY_NAME) for production..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -trimpath -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Production build complete: $(BUILD_DIR)/$(BINARY_NAME)"

build-prod-worker: ## Build the worker for production with optimizations
	@echo "Building $(WORKER_BINARY_NAME) for production..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -trimpath -o $(BUILD_DIR)/$(WORKER_BINARY_NAME) $(WORKER_CMD_DIR)
	@echo "Production build complete: $(BUILD_DIR)/$(WORKER_BINARY_NAME)"

build-prod-all: build-prod-server build-prod-worker ## Build both server and worker for production

build-prod-pi: check-pwa-freshness ## Build server for Raspberry Pi (linux/arm64, no worker)
	@echo "Building $(BINARY_NAME) for Raspberry Pi (linux/arm64)..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)
	@echo "Raspberry Pi build complete: $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64"

build-prod-linux: check-pwa-freshness ## Build server and worker for Linux (linux/amd64)
	@echo "Building for Linux (linux/amd64)..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	@GOOS=linux GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -trimpath -o $(BUILD_DIR)/$(WORKER_BINARY_NAME)-linux-amd64 $(WORKER_CMD_DIR)
	@echo "Linux build complete: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64, $(BUILD_DIR)/$(WORKER_BINARY_NAME)-linux-amd64"

build-prod-darwin: check-pwa-freshness ## Build server and worker for macOS (darwin/arm64)
	@echo "Building for macOS (darwin/arm64)..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)
	@GOOS=darwin GOARCH=arm64 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -trimpath -o $(BUILD_DIR)/$(WORKER_BINARY_NAME)-darwin-arm64 $(WORKER_CMD_DIR)
	@echo "macOS build complete: $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64, $(BUILD_DIR)/$(WORKER_BINARY_NAME)-darwin-arm64"

build-prod-windows: check-pwa-freshness ## Build server and worker for Windows (windows/amd64)
	@echo "Building for Windows (windows/amd64)..."
	@mkdir -p $(BUILD_DIR)
	@GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -trimpath -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)
	@GOOS=windows GOARCH=amd64 $(GO) build $(GOFLAGS) -ldflags="$(LDFLAGS)" -trimpath -o $(BUILD_DIR)/$(WORKER_BINARY_NAME)-windows-amd64.exe $(WORKER_CMD_DIR)
	@echo "Windows build complete: $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe, $(BUILD_DIR)/$(WORKER_BINARY_NAME)-windows-amd64.exe"

build-prod-pwa: pwa-build build-prod-all ## Build PWA + server + worker for production
	@echo "Full production build with PWA complete"

# Release
release: test pwa-build build-prod-linux build-prod-pi build-prod-darwin build-prod-windows ## Full release: test, build PWA, build all platforms, create archives
	@echo "Creating release archives..."
	@mkdir -p $(BUILD_DIR)/release
	@# Linux amd64 (server + worker)
	@tar -czf $(BUILD_DIR)/release/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz \
		-C $(BUILD_DIR) $(BINARY_NAME)-linux-amd64 $(WORKER_BINARY_NAME)-linux-amd64
	@# Linux arm64 / Raspberry Pi (server only)
	@tar -czf $(BUILD_DIR)/release/$(BINARY_NAME)-$(VERSION)-linux-arm64.tar.gz \
		-C $(BUILD_DIR) $(BINARY_NAME)-linux-arm64
	@# macOS arm64 (server + worker)
	@tar -czf $(BUILD_DIR)/release/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz \
		-C $(BUILD_DIR) $(BINARY_NAME)-darwin-arm64 $(WORKER_BINARY_NAME)-darwin-arm64
	@# Windows amd64 (server + worker)
	@cd $(BUILD_DIR) && zip -q release/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip \
		$(BINARY_NAME)-windows-amd64.exe $(WORKER_BINARY_NAME)-windows-amd64.exe
	@echo "Release archives created in $(BUILD_DIR)/release/"
	@ls -la $(BUILD_DIR)/release/

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
	@echo "Running all tests"
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

# Setup and cleanup
setup: deps pwa-install ## Install all dependencies (Go + PWA)
	@echo "Setup complete"

clean: ## Remove build artifacts and generated files
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@rm -rf web/pwa/dist
	@echo "Clean complete"

# Docker commands (future)
docker-build: ## Build Docker image
	@echo "Docker support coming soon..."

docker-run: ## Run Docker container
	@echo "Docker support coming soon..."
