.PHONY: build test test-coverage test-race lint fmt clean help deadcode

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOCLEAN=$(GOCMD) clean
GOMOD=$(GOCMD) mod
GOFMT=gofmt
GOLINT=golangci-lint
DEADCODE=deadcode

# Package paths
CORE_PKG=./internal/core/...
ALL_PKGS=./...

# Build targets
.DEFAULT_GOAL := help

## build: Build spin binary (single binary with all modes)
build:
	@echo "Building spin binary..."
	@$(GOBUILD) -o bin/spin ./cmd/spin
	@echo "✓ Build successful (binary at bin/spin)"
	@echo "  Usage: spin          # Start TUI"
	@echo "         spin exec ... # Non-interactive mode"
	@echo "         spin --help   # Show all commands"

## build-core: Build the core module (compile check)
build-core:
	@echo "Building core module..."
	@$(GOBUILD) $(CORE_PKG)
	@echo "✓ Core build successful"

## test: Run all tests
test:
	@echo "Running tests..."
	@$(GOTEST) -v $(CORE_PKG)

## test-coverage: Run tests with coverage report
test-coverage:
	@echo "Running tests with coverage..."
	@$(GOTEST) -cover -coverprofile=coverage.out $(CORE_PKG)
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

## test-race: Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	@$(GOTEST) -race $(CORE_PKG)

## lint: Run linters and deadcode analysis
lint:
	@echo "Running linters..."
	@$(GOLINT) run $(CORE_PKG)
	@echo "Running deadcode analysis..."
	@$(DEADCODE) -test ./cmd/... ./internal/... 2>&1 | grep -v "^$$" || echo "✓ No dead code found"
	@echo "✓ Linting complete"

## deadcode: Run deadcode analysis with detailed output (requires: go install golang.org/x/tools/cmd/deadcode@latest)
deadcode:
	@echo "Running deadcode analysis..."
	@echo "Analyzing cmd/ and internal/ packages..."
	@$(DEADCODE) -test ./cmd/... ./internal/... || echo "Note: Review any unreachable functions listed above"
	@echo ""
	@echo "Tip: Use 'deadcode -whylive <function>' to understand why a function is considered reachable"

## fmt: Format code
fmt:
	@echo "Formatting code..."
	@$(GOFMT) -s -w .
	@echo "✓ Code formatted"

## clean: Clean build artifacts
clean:
	@echo "Cleaning..."
	@$(GOCLEAN)
	@rm -f coverage.out coverage.html
	@rm -rf bin/
	@echo "✓ Cleaned"

## tidy: Tidy go modules
tidy:
	@echo "Tidying go modules..."
	@$(GOMOD) tidy
	@echo "✓ Modules tidied"

## verify: Verify go modules
verify:
	@echo "Verifying go modules..."
	@$(GOMOD) verify
	@echo "✓ Modules verified"

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	@$(GOMOD) download
	@echo "✓ Dependencies downloaded"

## all: Run fmt, lint, build, and test
all: fmt lint build test

## help: Display this help message
help:
	@echo "Spin - Core Module Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/  /'

