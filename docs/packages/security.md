# Package: internal/security

**Path:** `internal/security`  
**Purpose:** Sandbox and security policy enforcement

---

## Overview

The `security` package provides sandboxing and security controls for safe command execution, using platform-specific mechanisms like macOS sandbox-exec and Linux Landlock LSM.

## Package Structure

```
internal/security/
├── sandbox/              # Sandbox implementation
│   ├── sandbox.go        # Core sandbox interface
│   ├── modes.go          # Sandbox mode constants
│   ├── sandbox_darwin.go # macOS sandbox-exec integration
│   ├── sandbox_linux.go  # Linux Landlock LSM integration
│   ├── sandbox_windows.go# Windows security (future)
│   ├── doc.go            # Package documentation
│   └── *_test.go         # Platform tests
└── hardening/            # Process hardening
    ├── hardening.go      # Core hardening interface
    ├── hardening_darwin.go # macOS hardening
    ├── hardening_linux.go  # Linux capabilities/seccomp
    ├── hardening_windows.go# Windows hardening
    └── hardening_test.go   # Tests
```

**Note:** The package is organized into two main subdirectories:
- **`sandbox/`** - Handles filesystem and process isolation using OS-specific sandboxing mechanisms
- **`hardening/`** - Applies OS-specific process hardening (capability dropping, seccomp, etc.)

## Key Features

- **macOS Sandbox**: sandbox-exec profiles
- **Linux Landlock**: Landlock LSM for filesystem restrictions
- **Command Validation**: Safety classification
- **Policy Enforcement**: Configurable security policies
- **Read-Only Mode**: Prevent file modifications
- **Workspace Isolation**: Restrict to working directory

## Sandbox Modes

```go
const (
    SandboxModeNone           = "none"           // No sandbox
    SandboxModeReadOnly       = "read-only"      // Read-only access
    SandboxModeWorkspaceWrite = "workspace-write" // Write in workspace only
    SandboxModeFullAccess     = "full-access"    // Full system access
)
```

## Usage

### Using Sandbox

```go
import "github.com/dmytrogajewski/spin/internal/security/sandbox"

// Create sandbox
sb := sandbox.New(sandbox.Config{
    Mode:      sandbox.ModeWorkspaceWrite,
    Workspace: "/home/user/project",
})

// Execute command in sandbox
result, err := sb.Execute(ctx, "git status")
if err != nil {
    log.Fatal(err)
}
fmt.Println(result.Output)
```

### Using Hardening

```go
import "github.com/dmytrogajewski/spin/internal/security/hardening"

// Apply process hardening before executing sensitive operations
if err := hardening.Apply(); err != nil {
    log.Fatalf("Failed to apply hardening: %v", err)
}

// On Linux: drops capabilities, applies seccomp filters
// On macOS: applies sandbox profile
// On Windows: applies security restrictions
```

## macOS Sandbox Profile

```scheme
(version 1)
(deny default)
(allow file-read* (subpath "/"))
(allow file-write* (subpath "${WORKSPACE}"))
(allow process-exec (subpath "/usr/bin"))
(allow network-outbound)
```

## Linux Landlock

```go
// Landlock rules for workspace-write mode
rules := []landlock.PathRule{
    {Path: workspace, Access: landlock.AccessFSReadDir | landlock.AccessFSReadFile | landlock.AccessFSWriteFile},
    {Path: "/usr", Access: landlock.AccessFSReadDir | landlock.AccessFSReadFile | landlock.AccessFSExecute},
}
```

## Command Validation

```go
validator := security.NewValidator()
classification := validator.Classify("rm -rf /")
// Returns: SafetyDangerous

if classification == security.SafetyDangerous {
    // Require user approval
}
```

---

**Last Updated:** 2025-10-05  
**Status:** ✅ Production Ready
