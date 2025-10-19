# Deadcode Analysis

This document describes the deadcode analysis workflow for the Spin project, using the official Go deadcode analyzer to identify unreachable functions and test-only code.

## Overview

The deadcode analyzer builds a call graph from main packages and flags functions that aren't reachable at runtime, including through interfaces and reflection. This helps identify:

- **Production dead code**: Functions not reachable from main packages
- **Test-only code**: Functions only used by tests
- **Cross-platform dead code**: Functions dead across all platforms

## Installation

Install the deadcode analyzer:

```bash
go install golang.org/x/tools/cmd/deadcode@latest
```

For JSON processing (recommended):

```bash
# Ubuntu/Debian
sudo apt-get install jq

# macOS
brew install jq
```

## Quick Start

### Makefile Targets

```bash
# Basic deadcode analysis (includes tests)
make deadcode

# Production-only dead code
make deadcode-prod

# Find test-only functions
make deadcode-test-only

# JSON output
make deadcode-json

# Investigate specific function
make deadcode-why FUNC=functionName
```

### Analysis Scripts

```bash
# Comprehensive analysis
./scripts/deadcode-analysis.sh

# Production dead code only
./scripts/deadcode-analysis.sh --production

# Test-only functions only
./scripts/deadcode-analysis.sh --test-only

# Cross-platform analysis
./scripts/deadcode-analysis.sh --cross-platform

# Generate summary report
./scripts/deadcode-analysis.sh --summary
```

## Workflow

### 1. Production Dead Code Detection

Identify functions not reachable from main packages:

```bash
deadcode ./cmd/... ./internal/...
```

This reports functions that are truly dead in production.

### 2. Test-Only Code Detection

Find functions used only by tests by comparing results with and without tests:

```bash
# Generate function lists
deadcode -json ./cmd/... ./internal/... | jq -r '.[] | .Funcs[].Name' | sort > dead_prod.txt
deadcode -test -json ./cmd/... ./internal/... | jq -r '.[] | .Funcs[].Name' | sort > dead_with_tests.txt

# Find test-only functions
comm -23 dead_prod.txt dead_with_tests.txt
```

### 3. Cross-Platform Analysis

Analyze dead code across different platforms:

```bash
# Linux/AMD64
GOOS=linux GOARCH=amd64 deadcode -json ./cmd/... ./internal/... | jq -r '.[] | .Funcs[].Name' | sort > dead_linux_amd64.txt

# Darwin/AMD64
GOOS=darwin GOARCH=amd64 deadcode -json ./cmd/... ./internal/... | jq -r '.[] | .Funcs[].Name' | sort > dead_darwin_amd64.txt

# Find common dead functions
comm -12 dead_linux_amd64.txt dead_darwin_amd64.txt
```

## Understanding Results

### Function Classification

- **Unreachable func**: Function is not reachable from main packages
- **Generated func**: Function is in generated code (excluded by default)
- **Test func**: Function is only reachable through tests

### Common Patterns

1. **Interface implementations**: May appear dead but are used through interfaces
2. **Reflection usage**: Functions called via reflection won't be detected
3. **External tools**: Functions used by external tools or scripts
4. **Future features**: Functions reserved for future functionality

### Investigation

Use `deadcode -whylive` to understand why a function is considered reachable:

```bash
deadcode -whylive "bytes.Buffer.String" ./cmd/... ./internal/...
```

This shows the call path from main to the function.

## CI Integration

### GitHub Actions

Add to your workflow:

```yaml
- name: Deadcode Analysis
  run: |
    go install golang.org/x/tools/cmd/deadcode@latest
    ./scripts/deadcode-ci.sh --max-prod 50 --max-tests 100
```

### Custom Thresholds

```bash
# Set custom thresholds
./scripts/deadcode-ci.sh --max-prod 100 --max-tests 200 --max-test-only 50
```

### Exit Codes

- `0`: All checks passed
- `1`: One or more checks failed (exceeded thresholds)
- `2`: Error in script execution

## Best Practices

### Before Removing Code

1. **Review findings**: Not all "dead" code is safe to remove
2. **Check interfaces**: Ensure functions aren't interface implementations
3. **Verify reflection**: Look for `reflect` usage
4. **Consider external usage**: Check if functions are used by external tools
5. **Test thoroughly**: Run full test suite after removals

### Gradual Cleanup

1. Start with obviously dead code
2. Remove test-only functions that aren't needed
3. Keep interface implementations
4. Preserve functions used by external tools

### Monitoring

1. Run deadcode analysis regularly
2. Set up CI thresholds
3. Track dead code trends over time
4. Review new dead code in PRs

## Configuration

### Environment Variables

- `CI`: Set to `true` for CI mode (reduces colored output)
- `GITHUB_OUTPUT`: GitHub Actions output file
- `CI_PROJECT_DIR`: Project directory for reports

### Build Tags

Analyze specific build configurations:

```bash
deadcode -tags "integration" ./cmd/... ./internal/...
```

### Package Filtering

Limit analysis to specific packages:

```bash
deadcode ./internal/core/... ./internal/llm/...
```

## Troubleshooting

### Common Issues

1. **Compilation errors**: Fix build errors before running deadcode
2. **Missing dependencies**: Ensure all dependencies are available
3. **Platform differences**: Run analysis on target platforms
4. **Build constraints**: Consider different build tags

### Performance

For large codebases:

1. Use package filtering
2. Run analysis in parallel
3. Cache results when possible
4. Use JSON output for processing

## Examples

### Basic Analysis

```bash
# Run basic analysis
make deadcode

# Output example:
# internal/auth/auth.go:51:21: unreachable func: Credential.Validate
# internal/config/loader.go:145:18: unreachable func: Loader.GetStringSlice
```

### Test-Only Functions

```bash
# Find test-only functions
make deadcode-test-only

# Output example:
# Functions used only by tests:
# NewMockProvider
# TestHelperFunction
# ValidateTestData
```

### Investigation

```bash
# Investigate specific function
make deadcode-why FUNC=Credential.Validate

# Output example:
# golang.org/x/tools/cmd/deadcode.main
# static@L0117 --> golang.org/x/tools/go/packages.Load
# static@L0262 --> golang.org/x/tools/go/packages.defaultDriver
# static@L0305 --> golang.org/x/tools/go/packages.goListDriver
# static@L0153 --> golang.org/x/tools/go/packages.goListDriver$1
# static@L0154 --> golang.org/x/tools/go/internal/packagesdriver.GetSizesForArgsGolist
# static@L0044 --> bytes.Buffer.String
```

## Integration with Spin Workflow

### Quality Gates

Deadcode analysis is integrated into Spin's quality gates:

```bash
# Run full quality check
make lint

# This includes:
# - golangci-lint
# - deadcode analysis
```

### Development Workflow

1. **Before commit**: Run `make lint` to check for dead code
2. **PR review**: Include deadcode analysis results
3. **CI pipeline**: Automated deadcode checks with thresholds
4. **Regular cleanup**: Schedule periodic dead code removal

### Documentation

- Update this document when adding new analysis features
- Document any custom thresholds or configurations
- Include examples of common dead code patterns

## References

- [Go deadcode tool documentation](https://pkg.go.dev/golang.org/x/tools/cmd/deadcode)
- [Go blog: Finding dead code](https://go.dev/blog/deadcode)
- [Effective Go: Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
