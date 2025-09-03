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
GO_VERSION ?= $(shell go version | cut -d' ' -f3 | sed 's/go//')

# Public Go proxy configuration
export GOPROXY := https://proxy.golang.org,direct
export GOSUMDB := sum.golang.org

# Build environment
BUILD_ENV ?= $(shell if [ -n "$$CI" ]; then echo "ci"; else echo "local"; fi)
SERVICE_NAME = $(APP_NAME)
PROJECT_TEAM = gomdlint-dev

# Quality gates
COVERAGE_THRESHOLD = 80
MAX_COMPLEXITY = 15
MAX_LINE_LENGTH = 120

# Detect number of processors for parallel builds (improved robustness)
NUM_PROCESSORS := $(shell bash ./scripts/nprocs.sh 2>/dev/null || echo "1")
ifeq ($(NUM_PROCESSORS),)
NUM_PROCESSORS := 1
endif

# Colors for output
RED = \033[31m
GREEN = \033[32m
YELLOW = \033[33m
BLUE = \033[34m
RESET = \033[0m

.PHONY: help build build-all test test-cover test-race fuzz clean install \
		lint lint-sarif gomdlint fmt goimports deps check security security-sarif govulncheck \
		release docker docker-security tools-install pre-commit ci-local dev \
		version-check dependency-check license-check \
		geico-init geico-build geico-env-check geico-proxy-test geico-compliance

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

lint: gomdlint ## Run linters with standard output
	@echo "$(BLUE)Running Go linters...$(RESET)"
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "$(YELLOW)Installing golangci-lint...$(RESET)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	golangci-lint run --timeout=10m -j $(NUM_PROCESSORS)
	@echo "$(GREEN)Linting completed!$(RESET)"

gomdlint: build ## Run markdown linting with gomdlint (Git-aware)
	@echo "$(BLUE)Running markdown linting with gomdlint...$(RESET)"
	@if [ ! -f "$(BUILD_DIR)/$(APP_NAME)" ]; then \
		echo "$(YELLOW)gomdlint binary not found. Building first...$(RESET)"; \
		$(MAKE) build; \
	fi
	@if git rev-parse --verify HEAD >/dev/null 2>&1; then \
		if ! git diff --quiet --exit-code origin/main -- '*.md' 2>/dev/null; then \
			echo "$(YELLOW)Linting changed markdown files...$(RESET)"; \
			git diff --name-only origin/main -- '*.md' 2>/dev/null | xargs -r $(BUILD_DIR)/$(APP_NAME) lint --config .markdownlint.json || \
			git diff --name-only HEAD~1 -- '*.md' 2>/dev/null | xargs -r $(BUILD_DIR)/$(APP_NAME) lint --config .markdownlint.json || \
			$(BUILD_DIR)/$(APP_NAME) lint --config .markdownlint.json *.md docs/*.md 2>/dev/null || echo "$(YELLOW)No markdown files found to lint$(RESET)"; \
		else \
			echo "$(GREEN)No changed markdown files to lint$(RESET)"; \
		fi; \
	else \
		echo "$(YELLOW)Not a git repository or no commits, linting all markdown files...$(RESET)"; \
		$(BUILD_DIR)/$(APP_NAME) lint --config .markdownlint.json *.md docs/*.md 2>/dev/null || echo "$(YELLOW)No markdown files found to lint$(RESET)"; \
	fi
	@echo "$(GREEN)Markdown linting completed!$(RESET)"

lint-sarif: ## Run linters with SARIF output for CI/CD
	@echo "$(BLUE)Running linters with SARIF output...$(RESET)"
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "$(YELLOW)Installing golangci-lint...$(RESET)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	golangci-lint run --timeout=10m --out-format colored-line-number,sarif:golangci-lint-report.sarif
	@echo "$(GREEN)SARIF linting report generated!$(RESET)"

goimports: ## Organize imports
	@echo "$(BLUE)Organizing imports...$(RESET)"
	@if ! command -v goimports >/dev/null 2>&1; then \
		echo "$(YELLOW)Installing goimports...$(RESET)"; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	fi
	goimports -w -local github.com/gomdlint/gomdlint .
	@echo "$(GREEN)Imports organized!$(RESET)"

security: ## Run security checks with standard output
	@echo "$(BLUE)Running security checks...$(RESET)"
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "$(YELLOW)Installing gosec...$(RESET)"; \
		go install github.com/securecodewarrior/sast-scan/cmd/gosec@latest; \
	fi
	gosec -exclude-generated -quiet ./...
	@echo "$(GREEN)Security scan completed!$(RESET)"

security-sarif: ## Run security checks with SARIF output
	@echo "$(BLUE)Running security checks with SARIF output...$(RESET)"
	@if ! command -v gosec >/dev/null 2>&1; then \
		echo "$(YELLOW)Installing gosec...$(RESET)"; \
		go install github.com/securecodewarrior/sast-scan/cmd/gosec@latest; \
	fi
	gosec -fmt sarif -out gosec-report.sarif -exclude-generated -quiet ./...
	@echo "$(GREEN)Security SARIF report generated!$(RESET)"

govulncheck: ## Run Go vulnerability check
	@echo "$(BLUE)Running vulnerability checks...$(RESET)"
	@if ! command -v govulncheck >/dev/null 2>&1; then \
		echo "$(YELLOW)Installing govulncheck...$(RESET)"; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
	fi
	govulncheck ./...
	@echo "$(GREEN)Vulnerability check completed!$(RESET)"

## Testing targets
test: ## Run unit tests
	@echo "$(BLUE)Running unit tests...$(RESET)"
	go test -v ./...
	@echo "$(GREEN)Tests completed!$(RESET)"

test-race: ## Run unit tests with race detection
	@echo "$(BLUE)Running unit tests with race detection...$(RESET)"
	go test -race -v ./...
	@echo "$(GREEN)Race tests completed!$(RESET)"

test-cover: ## Run tests with coverage and enforce threshold
	@echo "$(BLUE)Running tests with coverage (NUM_PROCESSORS=$(NUM_PROCESSORS))...$(RESET)"
	@if command -v gotestsum >/dev/null 2>&1; then \
		echo "$(YELLOW)Using gotestsum for enhanced test output...$(RESET)"; \
		gotestsum --junitfile test-results.xml -- -p $(NUM_PROCESSORS) -race -coverprofile=coverage.out -covermode=atomic -v ./...; \
	else \
		echo "$(YELLOW)Using standard go test...$(RESET)"; \
		go test -race -coverprofile=coverage.out -covermode=atomic ./...; \
	fi
	go tool cover -html=coverage.out -o coverage.html
	@if command -v gocover-cobertura >/dev/null 2>&1; then \
		echo "$(YELLOW)Generating Cobertura coverage report...$(RESET)"; \
		gocover-cobertura < coverage.out > coverage.xml; \
	fi
	@COVERAGE=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Total coverage: $$COVERAGE%"; \
	if [ $$(echo "$$COVERAGE < $(COVERAGE_THRESHOLD)" | bc -l) -eq 1 ]; then \
		echo "$(RED)Coverage $$COVERAGE% is below required $(COVERAGE_THRESHOLD)%$(RESET)"; \
		exit 1; \
	else \
		echo "$(GREEN)Coverage $$COVERAGE% meets requirement!$(RESET)"; \
	fi
	@echo "$(GREEN)Coverage report generated: coverage.html$(RESET)"

fuzz: ## Run fuzz tests (if any exist)
	@echo "$(BLUE)Running fuzz tests...$(RESET)"
	@if go list -f '{{.TestGoFiles}}' ./... | grep -q Fuzz; then \
		echo "Running fuzz tests..."; \
		go test -fuzz=. -fuzztime=30s ./...; \
	else \
		echo "$(YELLOW)No fuzz tests found$(RESET)"; \
	fi
	@echo "$(GREEN)Fuzz testing completed!$(RESET)"

performance: ## Run performance tests
	@echo "$(BLUE)Running performance tests...$(RESET)"
	go test -bench=. -benchmem -cpu=1,2,4 ./...
	@echo "$(GREEN)Performance tests completed!$(RESET)"

## Quality assurance
check: deps fmt goimports lint security govulncheck test-cover ## Run all quality checks
	@echo "$(GREEN)All quality checks passed!$(RESET)"

ci-local: ## Run CI checks locally (mimics GitHub Actions)
	@echo "$(BLUE)Running local CI checks...$(RESET)"
	$(MAKE) deps
	$(MAKE) fmt
	$(MAKE) goimports
	$(MAKE) lint-sarif
	$(MAKE) security-sarif
	$(MAKE) govulncheck
	$(MAKE) test-race
	$(MAKE) test-cover
	$(MAKE) performance
	$(MAKE) build
	@echo "$(GREEN)Local CI checks completed successfully!$(RESET)"

pre-commit: deps fmt goimports lint test ## Quick pre-commit checks
	@echo "$(GREEN)Pre-commit checks passed!$(RESET)"

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
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg BUILD_DATE=$(DATE) \
		--build-arg SERVICE=$(APP_NAME) \
		-t $(APP_NAME):$(VERSION) \
		-t $(APP_NAME):latest .
	@echo "$(GREEN)Docker image built successfully!$(RESET)"

docker-security: docker ## Build Docker image and run security scan
	@echo "$(BLUE)Running Docker security scan...$(RESET)"
	@if command -v trivy >/dev/null 2>&1; then \
		trivy image --severity HIGH,CRITICAL $(APP_NAME):$(VERSION); \
	else \
		echo "$(YELLOW)Trivy not installed, skipping container security scan$(RESET)"; \
		echo "$(YELLOW)Install with: brew install trivy (macOS) or apt install trivy (Ubuntu)$(RESET)"; \
	fi

## Utility targets
clean: ## Clean build artifacts
	@echo "$(BLUE)Cleaning build artifacts...$(RESET)"
	rm -rf $(BUILD_DIR)
	rm -rf releases
	rm -f coverage.out coverage.html coverage.xml
	rm -f test-results.xml test-results.json
	rm -f golangci-lint-report.sarif gosec-report.sarif
	rm -f .air.toml
	rm -f demo.md
	go clean -cache -testcache -modcache
	@echo "$(GREEN)Cleanup completed!$(RESET)"

version: ## Show version information
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Date: $(DATE)"
	@echo "Go Version: $(shell go version)"
	@echo "OS/Arch: $(GOOS)/$(GOARCH)"
	@echo "Processors: $(NUM_PROCESSORS)"
	@echo "Build Environment: $(BUILD_ENV)"
	@echo "Service Name: $(SERVICE_NAME)"
	@echo "Project Team: $(PROJECT_TEAM)"

version-check: ## Check if Go version meets requirements
	@echo "$(BLUE)Checking Go version...$(RESET)"
	@REQUIRED_VERSION="1.23"; \
	CURRENT_VERSION=$(GO_VERSION); \
	if [ "$$CURRENT_VERSION" \< "$$REQUIRED_VERSION" ]; then \
		echo "$(RED)Go version $$CURRENT_VERSION is below required $$REQUIRED_VERSION$(RESET)"; \
		exit 1; \
	else \
		echo "$(GREEN)Go version $$CURRENT_VERSION meets requirements$(RESET)"; \
	fi

dependency-check: ## Check for outdated dependencies
	@echo "$(BLUE)Checking for outdated dependencies...$(RESET)"
	go list -u -m all | grep '\[' || echo "$(GREEN)All dependencies are up to date$(RESET)"

license-check: ## Check for license compliance
	@echo "$(BLUE)Checking licenses...$(RESET)"
	@if command -v go-licenses >/dev/null 2>&1; then \
		go-licenses check ./...; \
	else \
		echo "$(YELLOW)go-licenses not installed, skipping license check$(RESET)"; \
		echo "$(YELLOW)Install with: go install github.com/google/go-licenses@latest$(RESET)"; \
	fi

## Development helpers
tools-install: ## Install all required development tools
	@echo "$(BLUE)Installing development tools...$(RESET)"
	@echo "  $(YELLOW)Installing linters and analyzers...$(RESET)"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/sast-scan/cmd/gosec@latest
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install golang.org/x/tools/cmd/goimports@latest
	@echo "  $(YELLOW)Installing testing tools...$(RESET)"
	go install go.uber.org/mock/mockgen@latest
	go install github.com/onsi/ginkgo/v2/ginkgo@latest
	go install gotest.tools/gotestsum@latest
	go install github.com/boumenot/gocover-cobertura@latest
	@echo "  $(YELLOW)Installing utility tools...$(RESET)"
	go install golang.org/x/tools/cmd/godoc@latest
	go install github.com/securecodewarrior/sast-scan/cmd/nancy@latest
	go install github.com/caarlos0/svu@latest
	@echo "  $(YELLOW)Installing development helpers...$(RESET)"
	@if ! command -v air >/dev/null 2>&1; then \
		go install github.com/cosmtrek/air@latest; \
	fi
	@echo "$(YELLOW)Note: For complete setup, also install:$(RESET)"
	@echo "$(YELLOW)  - trivy: brew install trivy (macOS) or apt install trivy (Ubuntu)$(RESET)"
	@echo "$(GREEN)All development tools installed!$(RESET)"

dev-setup: tools-install deps ## Set up complete development environment
	@echo "$(BLUE)Setting up development environment...$(RESET)"
	$(MAKE) tools-install
	$(MAKE) deps
	@echo "$(GREEN)Development environment ready!$(RESET)"

run: build ## Build and run the application
	@echo "$(BLUE)Running $(APP_NAME)...$(RESET)"
	$(BUILD_DIR)/$(APP_NAME) --help

run-lint: build ## Build and run basic linting
	@echo "$(BLUE)Running $(APP_NAME) lint...$(RESET)"
	$(BUILD_DIR)/$(APP_NAME) lint --help

setup-colima-proxy: ## Setup Colima with proxy support for Zscaler environments
	@echo "$(BLUE)Setting up Colima with proxy support...$(RESET)"
	./scripts/setup-colima-proxy.sh

docker-build-colima: ## Build Docker image with Colima/Zscaler proxy support
	@echo "$(BLUE)Building Docker image for Colima with proxy support...$(RESET)"
	./scripts/docker-build-colima.sh

dev: ## Run development server with hot-reload
	@echo "$(BLUE)Starting development server with hot-reload...$(RESET)"
	@if ! command -v air >/dev/null 2>&1; then \
		echo "$(YELLOW)Air not installed. Installing...$(RESET)"; \
		go install github.com/cosmtrek/air@latest; \
	fi
	@if [ ! -f .air.toml ]; then \
		echo "$(YELLOW)Creating .air.toml configuration...$(RESET)"; \
		air init; \
	fi
	air

demo: build ## Run a demo of the linter
	@echo "$(BLUE)Running demo...$(RESET)"
	@echo "# Test Markdown File" > demo.md
	@echo "" >> demo.md
	@echo "This is a	tab character and a very long line that exceeds the typical line length limit and should trigger the line length rule violation." >> demo.md
	@echo "" >> demo.md
	@echo "##Missing space in heading" >> demo.md
	
	@echo "$(YELLOW)Linting demo.md...$(RESET)"
	$(BUILD_DIR)/$(APP_NAME) lint demo.md || true
	
	@echo "$(YELLOW)Testing plugin system...$(RESET)"
	$(BUILD_DIR)/$(APP_NAME) plugin list || true
	
	@rm -f demo.md
	@echo "$(GREEN)Demo completed!$(RESET)"

## Repository-specific targets
env-init: ## Initialize project build environment
	@echo "$(BLUE)Initializing build environment...$(RESET)"
	@echo "GOPROXY: $(GOPROXY)"
	@echo "GOSUMDB: $(GOSUMDB)"
	@echo "Build Environment: $(BUILD_ENV)"
	go mod tidy
	go mod verify
	$(MAKE) security
	@echo "$(GREEN)Build environment initialized!$(RESET)"

release-build: env-init ## Build with release standards
	@echo "$(BLUE)Building with release standards...$(RESET)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) go build $(LDFLAGS) \
		-ldflags="-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE) \
		-X main.buildEnv=$(BUILD_ENV) -X main.serviceName=$(SERVICE_NAME) \
		-X main.projectTeam=$(PROJECT_TEAM)" \
		-o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)
	@echo "$(GREEN)Release build completed: $(BUILD_DIR)/$(APP_NAME)$(RESET)"

## Repository-specific targets

env-check: ## Check build environment configuration
	@echo "$(BLUE)Checking build environment configuration...$(RESET)"
	@echo "Build Environment: $(BUILD_ENV)"
	@echo "Service Name: $(SERVICE_NAME)"
	@echo "Project Team: $(PROJECT_TEAM)"
	@echo "GOPROXY: $(GOPROXY)"
	@echo "GOSUMDB: $(GOSUMDB)"

proxy-test: ## Test Go proxy connectivity
	@echo "$(BLUE)Testing Go proxy connectivity...$(RESET)"
	@echo "Testing proxy: $(GOPROXY)"
	@if curl -f --connect-timeout 10 "$(GOPROXY)" >/dev/null 2>&1; then \
		echo "$(GREEN)✓ Go proxy is accessible$(RESET)"; \
	else \
		echo "$(YELLOW)⚠ Go proxy not accessible$(RESET)"; \
		echo "$(YELLOW)Check network connectivity$(RESET)"; \
	fi

compliance: ## Run code compliance checks
	@echo "$(BLUE)Running code compliance checks...$(RESET)"
	
	# Check for prohibited patterns
	@echo "Checking for code patterns..."
	@if grep -r "TODO\|FIXME\|XXX\|HACK" . --include="*.go" >/dev/null 2>&1; then \
		echo "$(YELLOW)⚠ Found TODO/FIXME comments - review before production$(RESET)"; \
		grep -rn "TODO\|FIXME\|XXX\|HACK" . --include="*.go" | head -10; \
	else \
		echo "$(GREEN)✓ No prohibited patterns found$(RESET)"; \
	fi
	
	# Check license compliance
	@echo "Checking license compliance..."
	@go list -m all | grep -E "(GPL|AGPL|LGPL)" && { \
		echo "$(RED)✗ Prohibited licenses found!$(RESET)"; \
		exit 1; \
	} || echo "$(GREEN)✓ License compliance check passed$(RESET)"
	
	# Check for security patterns
	@echo "Checking for security anti-patterns..."
	@if grep -r "password\|secret\|key" . --include="*.go" | grep -v "_test.go" | grep -E "(=|:)" >/dev/null 2>&1; then \
		echo "$(YELLOW)⚠ Potential hardcoded credentials found - review security$(RESET)"; \
	else \
		echo "$(GREEN)✓ No obvious credential leaks found$(RESET)"; \
	fi
	
	@echo "$(GREEN)Code compliance check completed!$(RESET)"

# Default target
.DEFAULT_GOAL := help
