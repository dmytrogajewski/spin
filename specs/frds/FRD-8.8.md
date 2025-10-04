# Feature Requirements Document: Landlock LSM Sandboxing (8.8)

**Feature ID:** 8.8
**Feature Name:** Landlock LSM Sandboxing Implementation
**Module:** `internal/security/sandbox`
**Priority:** P0 (High - Security Critical)
**Status:** In Progress
**Created:** 2025-10-04
**Parent Feature:** 8.3 (Security Integration)

---

## Overview

Implement proper Landlock LSM (Linux Security Module) sandboxing for command execution on Linux systems. This provides filesystem access control and process isolation without requiring root privileges or external tools.

---

## Business Requirements

### Problem Statement

The current `LinuxSandbox.Wrap()` implementation is a placeholder that:
1. Does not apply any actual restrictions to child processes
2. Only checks for Landlock availability but doesn't use it
3. Provides no filesystem access control
4. Cannot enforce read-only or workspace-write modes

### Success Criteria

- Commands execute with Landlock restrictions applied
- Filesystem access is limited based on sandbox mode
- Read-only mode prevents all writes
- Workspace-write mode allows writes only in workspace
- No root privileges required
- Graceful fallback when Landlock unavailable
- Zero regressions in existing tests

---

## Technical Requirements

### Architecture

Landlock implementation approaches:

**Option 1: CGo with Pre-exec Functions** (Selected)
- Use `syscall.SysProcAttr.PreExec` hook
- Apply Landlock rules before exec
- Most reliable, no external dependencies

**Option 2: Helper Binary**
- Separate binary applies Landlock then execs target
- More complex, harder to maintain

**Option 3: Wrapper Script**
- Shell script wrapper
- Less reliable, harder to debug

### Landlock API Requirements

```go
// Landlock syscalls (via golang.org/x/sys/unix)
unix.LandlockCreateRuleset()
unix.LandlockAddRule()
unix.LandlockRestrictSelf()
```

### Components to Implement

#### 1. Landlock Support Detection

```go
// landlockSupported checks Landlock availability
func landlockSupported() (bool, int) {
    // Return (supported, ABI version)
    // ABI v1: kernel 5.13+
    // ABI v2: kernel 5.19+ (network support)
    // ABI v3: kernel 6.2+ (ioctl support)
}
```

#### 2. Ruleset Creation

```go
// createRuleset creates Landlock ruleset based on mode
func createRuleset(opts SandboxOptions) (int, error) {
    // Create ruleset with appropriate access rights
    // Based on mode: read-only, workspace-write, full-access
}
```

#### 3. Rule Addition

```go
// addPathRule adds filesystem path rule to ruleset
func addPathRule(rulesetFD int, path string, access uint64) error {
    // Add rule for specific path with access rights
}
```

#### 4. Restriction Application

```go
// applyLandlock applies Landlock restrictions in child process
func (s *LinuxSandbox) applyLandlock(opts SandboxOptions) func() {
    // Return pre-exec function that applies restrictions
    // Called after fork, before exec
}
```

#### 5. Command Wrapping

```go
func (s *LinuxSandbox) Wrap(cmd *exec.Cmd, opts SandboxOptions) error {
    // Set up SysProcAttr with pre-exec function
    // Apply Landlock restrictions before command execution
}
```

---

## Definition of Ready (DoR)

- [x] FRD-8.3 (Security Integration) documented
- [x] Existing sandbox interface defined
- [x] Landlock LSM available in kernel 5.13+
- [x] golang.org/x/sys/unix package available
- [x] Test framework ready

---

## Definition of Done (DoD)

### Implementation
- [ ] `landlockSupported()` returns version-aware detection
- [ ] `createRuleset()` creates mode-appropriate rulesets
- [ ] `addPathRule()` adds filesystem rules
- [ ] `applyLandlock()` returns pre-exec function
- [ ] `Wrap()` applies Landlock via SysProcAttr
- [ ] Graceful fallback to NoopSandbox when unavailable
- [ ] Support for ABI v1 (5.13+) minimum

### Access Control Modes

**ModeReadOnly:**
- [ ] Allow read access to ReadPaths
- [ ] Allow read access to system paths (/bin, /usr, /lib)
- [ ] Deny all write operations
- [ ] Block writes to WorkDir

**ModeWorkspaceWrite:**
- [ ] Allow read access to ReadPaths
- [ ] Allow read/write access to WritePaths
- [ ] Allow read/write access to WorkDir
- [ ] Deny writes outside workspace

**ModeFullAccess:**
- [ ] No Landlock restrictions applied
- [ ] Command runs unrestricted

### Quality Gates
- [ ] Unit tests for each function (≥90% coverage)
- [ ] Integration tests on Linux kernel 5.13+
- [ ] Tests verify actual filesystem restrictions
- [ ] Error handling for all syscall failures
- [ ] Godoc for all exported functions
- [ ] No race conditions
- [ ] Cyclomatic complexity ≤15

---

## Implementation Plan

### Phase 1: Landlock Detection (1 hour)

1. Implement `landlockSupported()` with ABI detection
   ```go
   func landlockSupported() (bool, int) {
       // Try to create minimal ruleset to detect support
       // Return ABI version if supported
   }
   ```

2. Add ABI version constants
   ```go
   const (
       landlockABIv1 = 1 // kernel 5.13+
       landlockABIv2 = 2 // kernel 5.19+
       landlockABIv3 = 3 // kernel 6.2+
   )
   ```

3. Update `NewSandbox()` to use version-aware detection

### Phase 2: Ruleset Creation (2 hours)

1. Implement `createRuleset()` for different modes
   ```go
   // Access rights constants
   const (
       accessFSRead = unix.LANDLOCK_ACCESS_FS_READ_FILE |
                     unix.LANDLOCK_ACCESS_FS_READ_DIR
       accessFSWrite = unix.LANDLOCK_ACCESS_FS_WRITE_FILE |
                      unix.LANDLOCK_ACCESS_FS_REMOVE_FILE |
                      unix.LANDLOCK_ACCESS_FS_MAKE_DIR
   )
   ```

2. Handle mode-specific access rights:
   - ModeReadOnly: only read access
   - ModeWorkspaceWrite: read + write in workspace
   - ModeFullAccess: return -1 (no restrictions)

3. Add error handling for syscall failures

### Phase 3: Rule Addition (1 hour)

1. Implement `addPathRule()` to add path rules
   ```go
   func addPathRule(rulesetFD int, path string, access uint64) error {
       pathFD, err := unix.Open(path, unix.O_PATH|unix.O_CLOEXEC, 0)
       // Add rule using LandlockAddRule
       // Close pathFD
   }
   ```

2. Add multiple paths from opts.ReadPaths and opts.WritePaths

3. Add standard system paths for reading:
   - /bin, /usr/bin, /lib, /usr/lib
   - /etc (for shared libraries, configs)

### Phase 4: Restriction Application (2 hours)

1. Implement pre-exec function
   ```go
   func (s *LinuxSandbox) applyLandlock(opts SandboxOptions) func() {
       return func() {
           rulesetFD := createRuleset(opts)
           // Add rules for all paths
           // Call LandlockRestrictSelf
           // Close rulesetFD
       }
   }
   ```

2. Handle errors in pre-exec context
   - Use `runtime.LockOSThread()` if needed
   - Log errors via stderr (no panic in child)

3. Update `Wrap()` to set SysProcAttr:
   ```go
   func (s *LinuxSandbox) Wrap(cmd *exec.Cmd, opts SandboxOptions) error {
       if cmd.SysProcAttr == nil {
           cmd.SysProcAttr = &syscall.SysProcAttr{}
       }
       cmd.SysProcAttr.PreExec = s.applyLandlock(opts)
       return nil
   }
   ```

### Phase 5: Testing (2 hours)

1. Write unit tests:
   - `TestLandlockSupported`
   - `TestCreateRuleset`
   - `TestAddPathRule`
   - `TestApplyLandlock`

2. Write integration tests:
   - `TestLandlock_ReadRestriction`
   - `TestLandlock_WriteRestriction`
   - `TestLandlock_WorkspaceWrite`
   - `TestLandlock_ModeReadOnly`

3. Test error conditions:
   - Landlock not supported
   - Invalid paths
   - Syscall failures

### Phase 6: Documentation & Polish (1 hour)

1. Update godoc comments
2. Add implementation notes to doc.go
3. Update FRD-8.3 completion status
4. Update missing.md roadmap

**Total Estimated Effort:** 9 hours

---

## Test Cases

### Unit Tests

```go
// TestLandlockSupported verifies Landlock detection
func TestLandlockSupported(t *testing.T) {
    supported, version := landlockSupported()
    if supported {
        assert.GreaterOrEqual(t, version, 1)
    }
}

// TestCreateRuleset verifies ruleset creation
func TestCreateRuleset(t *testing.T) {
    tests := []struct {
        mode    Mode
        wantFD  bool
    }{
        {ModeReadOnly, true},
        {ModeWorkspaceWrite, true},
        {ModeFullAccess, false},
    }
    // Test each mode
}

// TestAddPathRule verifies path rule addition
func TestAddPathRule(t *testing.T) {
    // Create ruleset
    // Add rules for various paths
    // Verify no errors
}
```

### Integration Tests

```go
// TestLandlock_WriteRestriction verifies write blocking
func TestLandlock_WriteRestriction(t *testing.T) {
    if runtime.GOOS != "linux" {
        t.Skip("Linux only")
    }

    sb, _ := NewSandbox(ModeReadOnly)
    if !sb.Supported() {
        t.Skip("Landlock not supported")
    }

    workDir := t.TempDir()
    testFile := filepath.Join(workDir, "test.txt")

    // Try to write in read-only mode
    cmd := exec.Command("sh", "-c", "echo test > "+testFile)
    opts := GetDefaultOptions(ModeReadOnly, workDir)
    sb.Wrap(cmd, opts)

    err := cmd.Run()
    assert.Error(t, err) // Should fail
    assert.NoFileExists(t, testFile)
}

// TestLandlock_WorkspaceWrite verifies workspace writes
func TestLandlock_WorkspaceWrite(t *testing.T) {
    // Test writes allowed in workspace
    // Test writes denied outside workspace
}

// TestLandlock_ReadAccess verifies read access
func TestLandlock_ReadAccess(t *testing.T) {
    // Test reads work in all modes
}
```

---

## Security Considerations

### Threat Model

**Threats Mitigated:**
1. Unauthorized filesystem access outside workspace
2. Modification of system files
3. Data exfiltration via file writes
4. Accidental deletion of important files

**Limitations:**
1. Landlock available only on kernel 5.13+
2. No network isolation (requires ABI v2+)
3. Process can still access inherited file descriptors
4. No protection against resource exhaustion

### Security Properties

**Guaranteed:**
- Filesystem access strictly limited to allowed paths
- Restrictions cannot be removed by child process
- No privilege escalation possible
- Stack-based security (inherited by children)

**Not Guaranteed:**
- Network access control (requires kernel 5.19+)
- Memory limits
- CPU limits
- IPC restrictions

---

## Landlock API Reference

### Syscalls Used

```go
// Create ruleset file descriptor
unix.LandlockCreateRuleset(attr *unix.LandlockRulesetAttr, flags int) (fd int, err error)

// Add rule to ruleset
unix.LandlockAddRule(rulesetFD int, ruleType int, ruleAttr unsafe.Pointer, flags int) error

// Apply ruleset to current process
unix.LandlockRestrictSelf(rulesetFD int, flags int) error
```

### Access Rights (ABI v1)

```go
unix.LANDLOCK_ACCESS_FS_EXECUTE         // Execute file
unix.LANDLOCK_ACCESS_FS_WRITE_FILE      // Write to file
unix.LANDLOCK_ACCESS_FS_READ_FILE       // Read file
unix.LANDLOCK_ACCESS_FS_READ_DIR        // Read directory
unix.LANDLOCK_ACCESS_FS_REMOVE_DIR      // Remove directory
unix.LANDLOCK_ACCESS_FS_REMOVE_FILE     // Remove file
unix.LANDLOCK_ACCESS_FS_MAKE_CHAR       // Make char device
unix.LANDLOCK_ACCESS_FS_MAKE_DIR        // Make directory
unix.LANDLOCK_ACCESS_FS_MAKE_REG        // Make regular file
unix.LANDLOCK_ACCESS_FS_MAKE_SOCK       // Make socket
unix.LANDLOCK_ACCESS_FS_MAKE_FIFO       // Make FIFO
unix.LANDLOCK_ACCESS_FS_MAKE_BLOCK      // Make block device
unix.LANDLOCK_ACCESS_FS_MAKE_SYM        // Make symlink
```

---

## Performance Impact

| Operation | Overhead | Notes |
|-----------|----------|-------|
| Landlock detection | ~1ms | One-time at startup |
| Ruleset creation | ~2ms | Per command |
| Rule addition | ~0.5ms/path | Per path rule |
| Restriction application | ~1ms | In child process |
| **Total per command** | **~5-10ms** | Acceptable overhead |

---

## Platform Requirements

| Requirement | Version | Notes |
|-------------|---------|-------|
| Linux kernel | ≥5.13 | Landlock ABI v1 |
| golang.org/x/sys | Latest | Unix syscalls |
| Go version | ≥1.24 | For syscall support |

### Fallback Behavior

- Kernel <5.13: Use NoopSandbox
- Landlock disabled: Use NoopSandbox
- Syscall errors: Log and continue without sandbox

---

## Dependencies

### External Packages
```go
import (
    "golang.org/x/sys/unix"
    "os/exec"
    "syscall"
    "runtime"
)
```

### Internal Dependencies
- `internal/security/sandbox.Sandbox` interface
- `SandboxOptions` struct
- `Mode` enum

---

## Migration Notes

### Breaking Changes
- None (implementation of existing placeholder)

### Compatibility
- Existing tests must pass
- NoopSandbox used as fallback
- No API changes required

---

## Success Metrics

### Functional
- [ ] Write operations blocked in ModeReadOnly
- [ ] Writes allowed only in workspace for ModeWorkspaceWrite
- [ ] Read operations work in all modes
- [ ] Graceful fallback on unsupported systems

### Quality
- [ ] Test coverage ≥90%
- [ ] All integration tests pass on Linux 5.13+
- [ ] No false positives (blocked safe operations)
- [ ] No false negatives (allowed dangerous operations)

### Performance
- [ ] Command execution overhead <15ms
- [ ] No memory leaks
- [ ] No goroutine leaks

---

## Risks and Mitigations

| Risk | Impact | Probability | Mitigation |
|------|--------|-------------|------------|
| Kernel too old | High | Medium | Fallback to NoopSandbox |
| Syscall errors | Medium | Low | Comprehensive error handling |
| Path canonicalization issues | Medium | Low | Use O_PATH for path FDs |
| Pre-exec failures | High | Low | Log errors, allow execution |
| Performance regression | Low | Low | Benchmarking, optimization |

---

## Acceptance Criteria

### Functional
- [ ] LinuxSandbox.Wrap() applies Landlock restrictions
- [ ] ModeReadOnly prevents all writes
- [ ] ModeWorkspaceWrite allows workspace writes only
- [ ] ModeFullAccess disables restrictions
- [ ] Unsupported systems use NoopSandbox
- [ ] All tests pass on Linux 5.13+

### Technical
- [ ] Implementation uses CGo pre-exec approach
- [ ] ABI version detection works correctly
- [ ] Error handling covers all failure modes
- [ ] Code follows SOLID, DRY, KISS principles
- [ ] Complexity ≤15 for all functions
- [ ] No race conditions

### Documentation
- [ ] Godoc complete for all functions
- [ ] Implementation notes in doc.go
- [ ] Usage examples provided
- [ ] Security limitations documented

---

## References

- [Landlock LSM Documentation](https://landlock.io/)
- [Kernel Documentation](https://www.kernel.org/doc/html/latest/userspace-api/landlock.html)
- [golang.org/x/sys/unix](https://pkg.go.dev/golang.org/x/sys/unix)
- [FRD-8.3 Security Integration](FRD-8.3.md)
- [Sandbox Interface](../../internal/security/sandbox/sandbox.go)

---

## Changelog

| Date | Author | Change |
|------|--------|--------|
| 2025-10-04 | AI Agent | Initial FRD creation |

---

**Next Steps:**
1. Review and approve FRD
2. Write unit tests first (TDD)
3. Implement Landlock functions
4. Run integration tests
5. Update roadmap and mark complete
