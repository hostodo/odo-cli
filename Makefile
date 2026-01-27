.PHONY: build install clean test release snapshot help

# Variables
BINARY_NAME=hostodo
VERSION?=0.1.0
BUILD_DIR=dist
INSTALL_PATH=/usr/local/bin

# Build information
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS=-ldflags "-s -w -X github.com/hostodo/hostodo-cli/cmd.Version=$(VERSION) -X github.com/hostodo/hostodo-cli/cmd.Commit=$(COMMIT) -X github.com/hostodo/hostodo-cli/cmd.Date=$(DATE)"

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the binary
	@echo "Building $(BINARY_NAME)..."
	@go build $(LDFLAGS) -o $(BINARY_NAME) .
	@echo "Build complete: ./$(BINARY_NAME)"

install: build ## Install the binary to /usr/local/bin
	@echo "Installing $(BINARY_NAME) to $(INSTALL_PATH)..."
	@sudo mv $(BINARY_NAME) $(INSTALL_PATH)/
	@echo "Installed successfully!"
	@echo "Run '$(BINARY_NAME) --version' to verify"

clean: ## Clean build artifacts
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME)
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

test: ## Run tests
	@echo "Running tests..."
	@go test -v ./...

fmt: ## Format code
	@echo "Formatting code..."
	@go fmt ./...
	@gofmt -s -w .

lint: ## Run linter
	@echo "Running linter..."
	@golangci-lint run

deps: ## Download dependencies
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy

snapshot: ## Create a snapshot release with GoReleaser
	@echo "Creating snapshot release..."
	@goreleaser release --snapshot --clean

release: ## Create a production release with GoReleaser
	@echo "Creating release..."
	@goreleaser release --clean

release-check: ## Check release configuration
	@echo "Checking release configuration..."
	@goreleaser check

# Platform-specific builds
build-linux: ## Build for Linux
	@echo "Building for Linux..."
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 .
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 .

build-darwin: ## Build for macOS
	@echo "Building for macOS..."
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 .
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 .

build-windows: ## Build for Windows
	@echo "Building for Windows..."
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe .

build-all: build-linux build-darwin build-windows ## Build for all platforms
	@echo "All platform builds complete!"

run: ## Run the CLI
	@go run . $(ARGS)

dev: ## Run in development mode (example: make dev ARGS="instances list")
	@go run . $(ARGS)
