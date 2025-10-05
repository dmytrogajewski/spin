# Package: internal/version

**Path:** `internal/version`
**Purpose:** Version information management and build metadata

---

## Overview

The `version` package provides centralized version information for Spin, including version number, git commit, build date, and Go version. It supports build-time injection of version information via `-ldflags`.

## Package Structure

```
internal/version/
├── version.go       # Version info and exported functions
└── version_test.go  # 100% test coverage
```

## Key Components

### Build-Time Variables

```go
var (
    // Version is the semantic version (e.g., "1.0.0", "dev")
    Version = "dev"

    // Commit is the git commit hash
    Commit = "unknown"

    // BuildDate is the build timestamp
    BuildDate = "unknown"
)
```

These variables are set via `-ldflags` during build:

```bash
go build -ldflags "\
  -X github.com/dmytrogajewski/spin/internal/version.Version=1.0.0 \
  -X github.com/dmytrogajewski/spin/internal/version.Commit=$(git rev-parse HEAD) \
  -X github.com/dmytrogajewski/spin/internal/version.BuildDate=$(date -u +%Y-%m-%d)"
```

### Types

#### VersionInfo

```go
type VersionInfo struct {
    Version   string  // Semantic version
    Commit    string  // Git commit hash
    BuildDate string  // Build timestamp
    GoVersion string  // Go runtime version
}
```

## Public API

### Functions

#### GetVersionInfo()

```go
func GetVersionInfo() VersionInfo
```

Returns complete version information including runtime Go version.

**Example:**
```go
info := version.GetVersionInfo()
fmt.Printf("Version: %s\n", info.Version)
fmt.Printf("Commit: %s\n", info.Commit)
fmt.Printf("Built: %s\n", info.BuildDate)
fmt.Printf("Go: %s\n", info.GoVersion)
```

#### String()

```go
func String() string
```

Returns a formatted version string with all build information.

**Output Format:**
```
spin version 1.0.0 (commit: abc123, built: 2025-10-05, go1.24.7)
```

**Example:**
```go
fmt.Println(version.String())
// Output: spin version dev (commit: unknown, built: unknown, go1.24.7)
```

#### ShortVersion()

```go
func ShortVersion() string
```

Returns just the version number without additional metadata.

**Example:**
```go
fmt.Println(version.ShortVersion())
// Output: dev
```

## Usage Examples

### In CLI Commands

```go
package main

import (
    "github.com/dmytrogajewski/spin/internal/version"
    "github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
    return &cobra.Command{
        Use:   "version",
        Short: "Show version information",
        RunE: func(cmd *cobra.Command, args []string) error {
            fmt.Fprintln(cmd.OutOrStdout(), version.String())
            return nil
        },
    }
}
```

### Setting Root Command Version

```go
rootCmd := &cobra.Command{
    Use:     "spin",
    Version: version.ShortVersion(),
}
```

### Custom Version Display

```go
info := version.GetVersionInfo()
fmt.Printf("Spin v%s\n", info.Version)
fmt.Printf("  Built: %s\n", info.BuildDate)
fmt.Printf("  Commit: %s\n", info.Commit)
fmt.Printf("  Runtime: %s\n", info.GoVersion)
```

## Build Integration

### Development Builds

By default, without `-ldflags`, the version shows as:
```
spin version dev (commit: unknown, built: unknown, go1.24.7)
```

### Release Builds

For production releases, inject version info:

```bash
#!/bin/bash
VERSION="1.0.0"
COMMIT=$(git rev-parse --short HEAD)
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

go build -ldflags "\
  -X github.com/dmytrogajewski/spin/internal/version.Version=${VERSION} \
  -X github.com/dmytrogajewski/spin/internal/version.Commit=${COMMIT} \
  -X github.com/dmytrogajewski/spin/internal/version.BuildDate=${BUILD_DATE}" \
  -o spin ./cmd/spin/
```

### Makefile Integration

```makefile
VERSION ?= $(shell git describe --tags --always --dirty)
COMMIT := $(shell git rev-parse --short HEAD)
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -ldflags "\
  -X github.com/dmytrogajewski/spin/internal/version.Version=$(VERSION) \
  -X github.com/dmytrogajewski/spin/internal/version.Commit=$(COMMIT) \
  -X github.com/dmytrogajewski/spin/internal/version.BuildDate=$(BUILD_DATE)"

.PHONY: build
build:
	go build $(LDFLAGS) -o spin ./cmd/spin/
```

## Testing

The package has 100% test coverage, including:

- Version info retrieval
- String formatting (full and short)
- Go version matching
- Different version scenarios (dev vs release)

**Run tests:**
```bash
go test ./internal/version/
go test -cover ./internal/version/
```

## Design Decisions

### Why Variables Instead of Constants?

Variables allow build-time injection via `-ldflags`, which is the standard Go practice for version information.

### Why Include Go Version?

The Go runtime version (`runtime.Version()`) helps with:
- Debugging runtime-specific issues
- Ensuring compatibility
- Audit trails for production builds

### Why Separate String() and ShortVersion()?

Different use cases require different verbosity:
- `ShortVersion()` - For `--version` flags, API responses
- `String()` - For detailed version display, logs, debugging

## Dependencies

- `fmt` - String formatting
- `runtime` - Go version detection

**No external dependencies.**

## Thread Safety

All functions are thread-safe. They only read package-level variables set at build time or call `runtime.Version()`.

## Performance

- `GetVersionInfo()`: O(1), minimal allocations
- `String()`: O(1), single `fmt.Sprintf()` call
- `ShortVersion()`: O(1), direct variable access

## Related Files

- [FRD-UI-1.1](../../specs/frds/FRD-UI-1.1.md) - Main CLI Entry Point specification
- [cmd/spin/version.go](../../cmd/spin/version.go) - Version command implementation

---

**Last Updated:** 2025-10-05
**Status:** ✅ Complete
**Test Coverage:** 100%
