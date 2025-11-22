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

# Use workspace .gotmp directory to avoid /tmp quota issues
GOTMPDIR?=$(shell pwd)/.gotmp
export GOTMPDIR

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

## test: Run all tests (unit + e2e) with coverage and deadcode analysis
test: build
	@mkdir -p $(GOTMPDIR)
	@echo "Running unit and integration tests with coverage..."
	@$(GOTEST) -short -cover -timeout 30s $(INTERNAL_PKGS)
	@echo ""
	@echo "Running deadcode analysis..."
	@./scripts/deadcode-filter.sh github.com/dmytrogajewski/spin/cmd/spin github.com/dmytrogajewski/spin/internal/...
	@echo ""
	@echo "Running all E2E tests (using test-llm provider, no external LLM required)..."
	@$(GOBUILD) -tags e2e_llm_test -o bin/spin ./cmd/spin
	@SPIN_E2E_SKIP_BUILD=1 $(GOTEST) -tags e2e_llm_test -v -timeout 5m ./tests/e2e/...
	@echo ""
	@echo "Running ACP approval persistence E2E (test provider, no external LLM required)..."
	@SPIN_E2E_SKIP_BUILD=1 $(GOTEST) -tags e2e_llm_test -timeout 60s ./tests/e2e/acp -run TestACP_ApprovalPersistence_PromptToToolCall
	@echo ""
	@echo "✓ All tests complete"

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

## test-e2e: Run end-to-end tests (using test-llm provider, no external LLM required)
test-e2e: build
	@echo "Running E2E tests (using test-llm provider, no external LLM required)..."
	@$(GOBUILD) -tags e2e_llm_test -o bin/spin ./cmd/spin
	@$(GOTEST) -tags e2e_llm_test -v -timeout 30s ./tests/e2e/...
	@echo "✓ E2E tests complete"

## test-all: Run all tests (unit + e2e) - alias for 'test'
test-all: test
	@echo "✓ All tests passed"

## test-strict: Run all tests plus static analysis with uast/herr on all non-test Go files
test-strict: test
	@echo ""
	@echo "Running static analysis with uast/herr on all non-test Go files..."
	@echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	@files=$$(find . -name "*.go" -not -name "*_test.go" -not -name "doc.go" -not -path "./vendor/*" -not -path "./.git/*" -not -path "./examples/*" -not -path "./.gotmp/*"); \
	total=$$(echo "$$files" | wc -l); \
	current=0; \
	failed=0; \
	echo "Analyzing $$total files..."; \
	echo ""; \
	for file in $$files; do \
		current=$$((current + 1)); \
		printf "[%3d/%3d] Analyzing %s\n" $$current $$total "$$file"; \
		lines=$$(wc -l < "$$file"); \
		if [ $$lines -lt 50 ]; then continue; fi; \
		output=$$(uast parse "$$file" 2>/dev/null | herr analyze 2>&1); \
		if echo "$$output" | grep -q "total_functions: 0"; then continue; fi; \
		issues=$$(echo "$$output" | grep -E "^(High complexity|Poor.*cohesion|Poor comment quality)" | wc -l); \
		if [ $$issues -ge 3 ]; then \
			echo "  ⚠️  Multiple issues in $$file"; \
			echo "$$output" | grep -E "^(High complexity|Poor.*cohesion|Poor comment quality|High Halstead)" | head -3; \
			failed=$$((failed + 1)); \
		fi; \
	done; \
	echo ""; \
	echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"; \
	if [ $$failed -eq 0 ]; then \
		echo "✓ Static analysis complete: All files passed ($$total files analyzed)"; \
	else \
		echo "⚠️  Static analysis complete: $$failed file(s) with issues out of $$total analyzed"; \
		echo "   (Run 'uast parse <file> | herr analyze' on specific files for details)"; \
	fi; \
	echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

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
