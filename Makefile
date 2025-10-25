.PHONY: build install install-user uninstall uninstall-user test test-coverage test-race test-e2e test-all lint fmt clean help deadcode

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
INTERNAL_PKGS=./internal/...
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

## install: Build and install spin binary to /usr/local/bin (requires sudo)
install: build
	@echo "Installing spin to /usr/local/bin..."
	@sudo cp bin/spin /usr/local/bin/spin
	@sudo chmod 755 /usr/local/bin/spin
	@echo "✓ Installed successfully to /usr/local/bin/spin"
	@echo "  You can now run 'spin' from anywhere"

## install-user: Build and install spin binary to ~/.local/bin (no sudo required)
install-user: build
	@echo "Installing spin to ~/.local/bin..."
	@mkdir -p ~/.local/bin
	@cp bin/spin ~/.local/bin/spin
	@chmod 755 ~/.local/bin/spin
	@echo "✓ Installed successfully to ~/.local/bin/spin"
	@echo "  Make sure ~/.local/bin is in your PATH"
	@echo "  You can add it with: export PATH=\"$$HOME/.local/bin:$$PATH\""

## uninstall: Remove installed spin binary from /usr/local/bin
uninstall:
	@echo "Removing spin from /usr/local/bin..."
	@sudo rm -f /usr/local/bin/spin
	@echo "✓ Uninstalled successfully"

## uninstall-user: Remove installed spin binary from ~/.local/bin
uninstall-user:
	@echo "Removing spin from ~/.local/bin..."
	@rm -f ~/.local/bin/spin
	@echo "✓ Uninstalled successfully"

## build-internal: Build internal packages (compile check)
build-internal:
	@echo "Building internal packages..."
	@$(GOBUILD) $(INTERNAL_PKGS)
	@echo "✓ Internal packages build successful"

## test: Run all tests with coverage and deadcode analysis (skips slow stress tests)
test:
	@echo "Running tests with coverage..."
	@$(GOTEST) -short -cover -timeout 30s $(INTERNAL_PKGS)
	@echo ""
	@echo "Running deadcode analysis..."
	@./scripts/deadcode-filter.sh github.com/dmytrogajewski/spin/cmd/spin github.com/dmytrogajewski/spin/internal/...

## test-coverage: Run tests with coverage report (HTML)
test-coverage:
	@echo "Running tests with coverage..."
	@$(GOTEST) -short -cover -coverprofile=coverage.out -timeout 30s $(INTERNAL_PKGS)
	@$(GOCMD) tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

## coverage: Show overall coverage percentage and per-package breakdown
coverage:
	@echo "Calculating coverage..."
	@echo ""
	@$(GOTEST) -short -cover -coverprofile=coverage.out -timeout 30s $(INTERNAL_PKGS) 2>&1 | \
		grep "coverage:" | \
		sed 's/ok  *//' | \
		sed 's/github.com\/dmytrogajewski\/spin\///' | \
		sed 's/(cached)//' | \
		sed -E 's/[[:space:]]+[0-9]+\.[0-9]+s[[:space:]]+/ /' | \
		awk '{for(i=1;i<=NF;i++) if($$i~/^[0-9.]+%$$/) {printf "%-45s %s\n", $$1, $$i; break}}' | \
		sort
	@echo ""
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@$(GOCMD) tool cover -func=coverage.out | tail -1 | awk '{printf "TOTAL COVERAGE: %s\n", $$3}'
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@echo ""
	@echo "For detailed HTML report: make test-coverage"

## test-race: Run tests with race detector
test-race:
	@echo "Running tests with race detector..."
	@$(GOTEST) -short -race -timeout 30s $(INTERNAL_PKGS)

## test-stress: Run stress tests (slow, may take several minutes)
test-stress:
	@echo "Running stress tests..."
	@$(GOTEST) -v -timeout 5m -run "Stress" $(INTERNAL_PKGS)
	@echo "✓ Stress tests complete"

## test-e2e: Run end-to-end TUI tests (requires Ollama running)
test-e2e: build
	@echo "Running E2E TUI tests (requires: Ollama, models qwen3:0.6b, qwen3:1.7b, qwen2.5-coder:1.5b)..."
	@$(GOTEST) -v -timeout 5m ./tests
	@echo "✓ E2E tests complete"

## test-e2e-quick: Run quick E2E tests only
test-e2e-quick: build
	@echo "Running quick E2E tests..."
	@$(GOTEST) -v -timeout 30s -run "TestTUILaunch|TestTUIExit" ./tests
	@echo "✓ Quick E2E tests complete"

## test-all: Run all tests (unit + e2e)
test-all: test test-e2e
	@echo "✓ All tests passed"

## lint: Run linters and deadcode analysis
lint:
	@echo "Running linters..."
	@$(GOLINT) run $(INTERNAL_PKGS)
	@echo "Running deadcode analysis..."
	@./scripts/deadcode-filter.sh -test github.com/dmytrogajewski/spin/cmd/spin github.com/dmytrogajewski/spin/internal/...
	@echo "✓ Linting complete"

## deadcode: Run deadcode analysis with detailed output (requires: go install golang.org/x/tools/cmd/deadcode@latest)
deadcode:
	@echo "Running deadcode analysis..."
	@echo "Analyzing cmd/ and internal/ packages..."
	@$(DEADCODE) -test github.com/dmytrogajewski/spin/cmd/spin github.com/dmytrogajewski/spin/internal/... || echo "Note: Review any unreachable functions listed above"
	@echo ""
	@echo "Tip: Use 'deadcode -whylive <function>' to understand why a function is considered reachable"

## deadcode-prod: Run deadcode analysis excluding tests (production-only dead code)
deadcode-prod:
	@echo "Running deadcode analysis (production only)..."
	@echo "Analyzing cmd/ and internal/ packages..."
	@./scripts/deadcode-filter.sh github.com/dmytrogajewski/spin/cmd/spin github.com/dmytrogajewski/spin/internal/...

## deadcode-test-only: Find functions used only by tests (requires jq)
deadcode-test-only:
	@echo "Finding functions used only by tests..."
	@echo "This compares deadcode results with and without tests..."
	@mkdir -p .deadcode-tmp
	@$(DEADCODE) -json github.com/dmytrogajewski/spin/cmd/spin github.com/dmytrogajewski/spin/internal/... | jq -r '.[] | .Funcs[].Name' | sort > .deadcode-tmp/dead_prod.txt
	@$(DEADCODE) -test -json github.com/dmytrogajewski/spin/cmd/spin github.com/dmytrogajewski/spin/internal/... | jq -r '.[] | .Funcs[].Name' | sort > .deadcode-tmp/dead_with_tests.txt
	@echo "Functions used only by tests:"
	@comm -23 .deadcode-tmp/dead_prod.txt .deadcode-tmp/dead_with_tests.txt || echo "No test-only functions found"
	@rm -rf .deadcode-tmp

## deadcode-json: Run deadcode analysis with JSON output
deadcode-json:
	@echo "Running deadcode analysis (JSON output)..."
	@$(DEADCODE) -json github.com/dmytrogajewski/spin/cmd/spin github.com/dmytrogajewski/spin/internal/...

## deadcode-why: Show why a function is not dead (usage: make deadcode-why FUNC=functionName)
deadcode-why:
	@if [ -z "$(FUNC)" ]; then \
		echo "Usage: make deadcode-why FUNC=functionName"; \
		echo "Example: make deadcode-why FUNC=bytes.Buffer.String"; \
		exit 1; \
	fi
	@echo "Showing why function '$(FUNC)' is not dead..."
	@$(DEADCODE) -whylive="$(FUNC)" github.com/dmytrogajewski/spin/cmd/spin github.com/dmytrogajewski/spin/internal/...

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

