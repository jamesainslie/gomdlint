# gomdlint Makefile - High-Performance Go Markdown Linter

# Build information
VERSION ?= $(shell git describe --tags --abbrev=0 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build configuration
APP_NAME = gomdlint
CMD_DIR = ./cmd/$(APP_NAME)
BUILD_DIR = ./bin
LDFLAGS = -ldflags="-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE) -s -w"

# Go configuration
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
CGO_ENABLED ?= 0

# Detect number of processors for parallel builds
NUM_PROCESSORS := $(shell bash -c 'if command -v nproc >/dev/null 2>&1; then nproc; elif command -v sysctl >/dev/null 2>&1; then sysctl -n hw.ncpu; else echo 1; fi')

# Colors for output
RED = \033[31m
GREEN = \033[32m
YELLOW = \033[33m
BLUE = \033[34m
RESET = \033[0m

.PHONY: help build build-all test test-cover benchmark clean install lint fmt deps check security release docker

## Default target
help: ## Show this help message
	@echo "$(BLUE)gomdlint - High-Performance Go Markdown Linter$(RESET)"
	@echo ""
	@echo "$(YELLOW)Available targets:$(RESET)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(GREEN)%-15s$(RESET) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## Build targets
build: deps ## Build the binary for the current platform
	@echo "$(BLUE)Building $(APP_NAME) for $(GOOS)/$(GOARCH)...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)
	@echo "$(GREEN)Built successfully: $(BUILD_DIR)/$(APP_NAME)$(RESET)"

build-all: deps ## Build binaries for all supported platforms
	@echo "$(BLUE)Building $(APP_NAME) for all platforms...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	
	# Linux
	@echo "  $(YELLOW)Building for Linux/amd64...$(RESET)"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-amd64 $(CMD_DIR)
	
	@echo "  $(YELLOW)Building for Linux/arm64...$(RESET)"
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-linux-arm64 $(CMD_DIR)
	
	# macOS
	@echo "  $(YELLOW)Building for macOS/amd64...$(RESET)"
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-amd64 $(CMD_DIR)
	
	@echo "  $(YELLOW)Building for macOS/arm64...$(RESET)"
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-darwin-arm64 $(CMD_DIR)
	
	# Windows
	@echo "  $(YELLOW)Building for Windows/amd64...$(RESET)"
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe $(CMD_DIR)
	
	@echo "$(GREEN)All builds completed successfully!$(RESET)"

install: build ## Install the binary to $GOPATH/bin
	@echo "$(BLUE)Installing $(APP_NAME)...$(RESET)"
	go install $(LDFLAGS) $(CMD_DIR)
	@echo "$(GREEN)Installed successfully!$(RESET)"

## Development targets
deps: ## Download and verify dependencies
	@echo "$(BLUE)Downloading dependencies...$(RESET)"
	go mod tidy
	go mod verify
	@echo "$(GREEN)Dependencies updated!$(RESET)"

fmt: ## Format Go code
	@echo "$(BLUE)Formatting Go code...$(RESET)"
	go fmt ./...
	@echo "$(GREEN)Code formatted!$(RESET)"

lint: ## Run linters
	@echo "$(BLUE)Running linters...$(RESET)"
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "$(YELLOW)Installing golangci-lint...$(RESET)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	golangci-lint run -j $(NUM_PROCESSORS)
	@echo "$(GREEN)Linting completed!$(RESET)"

security: ## Run security checks
	@echo "$(BLUE)Running security checks...$(RESET)"
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "$(YELLOW)Installing gosec...$(RESET)"; \
		go install github.com/securecodewarrior/sast-scan/cmd/gosec@latest; \
	fi
	@if ! command -v govulncheck >/dev/null 2>&1; then \
		echo "$(YELLOW)Installing govulncheck...$(RESET)"; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	gosec ./...
	govulncheck ./...
	@echo "$(GREEN)Security checks completed!$(RESET)"

## Testing targets
test: ## Run unit tests
	@echo "$(BLUE)Running unit tests...$(RESET)"
	go test -race -v ./...
	@echo "$(GREEN)Tests completed!$(RESET)"

test-cover: ## Run tests with coverage
	@echo "$(BLUE)Running tests with coverage...$(RESET)"
	go test -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(RESET)"

benchmark: ## Run benchmarks
	@echo "$(BLUE)Running benchmarks...$(RESET)"
	go test -bench=. -benchmem -cpu=1,2,4 ./...
	@echo "$(GREEN)Benchmarks completed!$(RESET)"

## Quality assurance
check: deps fmt lint security test ## Run all quality checks
	@echo "$(GREEN)All quality checks passed!$(RESET)"

## Release targets  
release: check build-all ## Prepare a release
	@echo "$(BLUE)Preparing release $(VERSION)...$(RESET)"
	@mkdir -p releases
	
	# Create tarballs for Unix systems
	@for binary in $(BUILD_DIR)/$(APP_NAME)-linux-* $(BUILD_DIR)/$(APP_NAME)-darwin-*; do \
		if [ -f "$$binary" ]; then \
			platform=$$(basename "$$binary" | sed 's/$(APP_NAME)-//'); \
			tar -czf releases/$(APP_NAME)-$(VERSION)-$$platform.tar.gz -C $(BUILD_DIR) $$(basename "$$binary") README.md LICENSE; \
		fi; \
	done
	
	# Create zip for Windows
	@if [ -f "$(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe" ]; then \
		cd $(BUILD_DIR) && zip ../releases/$(APP_NAME)-$(VERSION)-windows-amd64.zip $(APP_NAME)-windows-amd64.exe ../README.md ../LICENSE; \
	fi
	
	@echo "$(GREEN)Release $(VERSION) prepared in ./releases/$(RESET)"

## Docker targets
docker: ## Build Docker image
	@echo "$(BLUE)Building Docker image...$(RESET)"
	docker build -t $(APP_NAME):$(VERSION) -t $(APP_NAME):latest .
	@echo "$(GREEN)Docker image built successfully!$(RESET)"

## Utility targets
clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning build artifacts...$(RESET)"
	rm -rf $(BUILD_DIR)
	rm -rf releases
	rm -f coverage.out coverage.html
	go clean -cache -testcache -modcache
	@echo "$(GREEN)Cleanup completed!$(RESET)"

version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"  
	@echo "Date: $(DATE)"
	@echo "Go Version: $(shell go version)"
	@echo "OS/Arch: $(GOOS)/$(GOARCH)"
	@echo "Processors: $(NUM_PROCESSORS)"

## Development helpers
dev-setup: ## Set up development environment
	@echo "$(BLUE)Setting up development environment...$(RESET)"
	
	# Install development tools
	@echo "  $(YELLOW)Installing development tools...$(RESET)"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/sast-scan/cmd/gosec@latest  
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install golang.org/x/tools/cmd/goimports@latest
	
	# Download dependencies
	$(MAKE) deps
	
	@echo "$(GREEN)Development environment ready!$(RESET)"

run: build ## Build and run the application
	@echo "$(BLUE)Running $(APP_NAME)...$(RESET)"
	$(BUILD_DIR)/$(APP_NAME) --help

run-tui: build ## Build and run the TUI interface
	@echo "$(BLUE)Running $(APP_NAME) TUI...$(RESET)"
	$(BUILD_DIR)/$(APP_NAME) tui

demo: build ## Run a demo of the linter
	@echo "$(BLUE)Running demo...$(RESET)"
	@echo "# Test Markdown File" > demo.md
	@echo "" >> demo.md
	@echo "This is a	tab character and a very long line that exceeds the typical line length limit and should trigger the line length rule violation." >> demo.md
	@echo "" >> demo.md
	@echo "##Missing space in heading" >> demo.md
	
	@echo "$(YELLOW)Linting demo.md...$(RESET)"
	$(BUILD_DIR)/$(APP_NAME) lint demo.md || true
	
	@echo "$(YELLOW)Running TUI on demo.md...$(RESET)"
	$(BUILD_DIR)/$(APP_NAME) tui demo.md || true
	
	@rm -f demo.md
	@echo "$(GREEN)Demo completed!$(RESET)"

# Default target
.DEFAULT_GOAL := help
