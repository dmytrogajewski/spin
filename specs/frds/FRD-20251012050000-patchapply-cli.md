# FRD-20251012050000: patchapply CLI Tool

**Feature:** CLI tool for applying Spin patches from command line
**Priority:** P1 (Useful for testing and manual use)
**Estimated Effort:** 1 day
**Status:** In Progress
**Created:** 2025-10-12
**Roadmap Item:** Feature 2.4 in specs/tools-modules/ROADMAP.md

---

## 1. Overview

### 1.1 Problem Statement

The `internal/patchapply` package (Parser, Matcher, Applier) is complete and fully tested, but there's no standalone CLI tool for:
- Manual patch application during development
- Testing patch generation by AI models
- Scripting and automation workflows
- Debugging patch application issues

Currently, developers must write Go code to use the patchapply package. A CLI tool would enable:
- Quick testing: `cat test.patch | spin apply-patch`
- CI/CD integration: `spin apply-patch --dry-run < changes.patch`
- Manual application: `spin apply-patch -f my-changes.patch`

### 1.2 Goals

**Primary Goals:**
1. Provide a standalone CLI for applying Spin patches
2. Support reading patches from stdin or file
3. Enable dry-run mode for preview
4. Provide clear, actionable error messages
5. Follow Spin's existing CLI patterns (cobra-based)

**Non-Goals:**
- Patch generation (that's AI model's job)
- Interactive patch editing
- GUI or TUI interface (separate tools)

---

## 2. Requirements

### 2.1 Functional Requirements

#### FR-1: Multi-Mode Execution
The CLI tool must be accessible in three ways:
1. **Symlink mode:** `spin-apply-patch` → invokes apply-patch mode
2. **Subcommand mode:** `spin apply-patch` → runs subcommand
3. **Internal flag mode:** `spin --spin-run-as-apply-patch` → internal subprocess execution

**Rationale:** Follows existing Spin pattern in `cmd/spin/main.go:10-29`.

#### FR-2: Input Sources
Support reading patch from:
- **stdin:** `cat patch.txt | spin apply-patch`
- **file:** `spin apply-patch -f patch.txt`
- **Default:** Read from stdin if no file specified

#### FR-3: Workspace Control
- **Flag:** `--workspace <path>` or `-w <path>`
- **Default:** Current working directory
- **Validation:** Must be a valid directory

#### FR-4: Dry-Run Mode
- **Flag:** `--dry-run` (boolean)
- **Behavior:** Parse and validate patch without applying
- **Output:** Show what would be changed

#### FR-5: Force Overwrite Mode
- **Flag:** `--force` (boolean)
- **Behavior:** Allow `Add File` to overwrite existing files
- **Default:** false (reject if file exists)

#### FR-6: Verbose Output
- **Flag:** `--verbose` or `-v` (boolean)
- **Behavior:** Show detailed operation logs
- **Default:** false (show minimal output)

### 2.2 Output Requirements

#### OR-1: Success Output
```
✓ Applied patch successfully
  Created: 2 files
  Updated: 3 files
  Deleted: 1 file
  Moved: 1 file
```

#### OR-2: Dry-Run Output
```
[DRY RUN] Would apply the following changes:
  Would create: internal/handler/user.go (45 lines)
  Would update: internal/handler/handler.go (3 hunks)
  Would delete: internal/deprecated/old.go
  Would move: handler.go → internal/handler/handler.go
```

#### OR-3: Error Output
```
Error: Failed to apply patch
  Line 23: Context not found in file internal/handler.go
  Expected:
    func Process(data string) {
        log.Printf("Processing")
  Actual (around line 45):
    func Process(ctx context.Context, data string) {
        logrus.Infof("Processing")

Hint: The function signature may have changed. Update the patch context.
```

#### OR-4: Validation Error Output
```
Error: Invalid patch syntax
  Line 5: unknown operation: "*** Modify File: test.txt"
  Expected: "Add File", "Delete File", or "Update File"

Hint: Check the patch format specification.
```

### 2.3 Non-Functional Requirements

#### NFR-1: Performance
- Parse 10k line patch: <100ms
- Apply typical patch: <1s
- Memory usage: <50MB

#### NFR-2: Usability
- Help text with examples
- Clear error messages with line numbers
- Exit codes: 0 (success), 1 (error), 2 (validation failure)

#### NFR-3: Safety
- Validate all paths before any file operations
- Atomic operations (all-or-nothing)
- Clear errors on path traversal attempts

---

## 3. Design

### 3.1 Architecture

```
cmd/spin/
├── main.go              # Entry point, mode detection
├── apply_patch.go       # NEW: CLI implementation
└── apply_patch_test.go  # NEW: CLI tests
```

### 3.2 Flow Diagram

```
User Input (stdin/file)
        ↓
Parse CLI flags (cobra)
        ↓
Read patch text
        ↓
Create patchapply.Parser
        ↓
Parse patch → AST
        ↓
Create patchapply.Applier
        ↓
Configure (dry-run, force, etc.)
        ↓
Apply patch (or validate)
        ↓
Format output
        ↓
Exit (0 = success, 1 = error)
```

### 3.3 Implementation Details

#### File: cmd/spin/apply_patch.go

```go
package main

import (
    "fmt"
    "io"
    "os"
    "path/filepath"

    "github.com/spf13/cobra"
    "github.com/dmytrogajewski/spin/internal/patchapply"
)

// applyPatchCmd represents the apply-patch subcommand
var applyPatchCmd = &cobra.Command{
    Use:   "apply-patch",
    Short: "Apply a Spin patch to the workspace",
    Long: `Apply a Spin patch from stdin or file.

Examples:
  # Apply from stdin
  cat changes.patch | spin apply-patch

  # Apply from file
  spin apply-patch -f changes.patch

  # Dry-run mode
  spin apply-patch --dry-run -f changes.patch

  # Custom workspace
  spin apply-patch -w /path/to/project -f changes.patch`,
    RunE: runApplyPatch,
}

// Flags
var (
    applyPatchFile      string
    applyPatchWorkspace string
    applyPatchDryRun    bool
    applyPatchForce     bool
    applyPatchVerbose   bool
)

func init() {
    applyPatchCmd.Flags().StringVarP(&applyPatchFile, "file", "f", "", "Patch file (default: stdin)")
    applyPatchCmd.Flags().StringVarP(&applyPatchWorkspace, "workspace", "w", ".", "Workspace directory")
    applyPatchCmd.Flags().BoolVar(&applyPatchDryRun, "dry-run", false, "Validate without applying")
    applyPatchCmd.Flags().BoolVar(&applyPatchForce, "force", false, "Force overwrite existing files")
    applyPatchCmd.Flags().BoolVarP(&applyPatchVerbose, "verbose", "v", false, "Verbose output")
}

// runApplyPatch executes the apply-patch command
func runApplyPatch(cmd *cobra.Command, args []string) error {
    // Read patch text
    patchText, err := readPatchInput()
    if err != nil {
        return fmt.Errorf("read patch: %w", err)
    }

    // Parse patch
    parser := patchapply.NewParser(patchText)
    patch, err := parser.Parse()
    if err != nil {
        return formatParseError(err)
    }

    // Resolve workspace
    workspace, err := filepath.Abs(applyPatchWorkspace)
    if err != nil {
        return fmt.Errorf("resolve workspace: %w", err)
    }

    // Create applier
    applier, err := patchapply.NewApplier(workspace)
    if err != nil {
        return fmt.Errorf("create applier: %w", err)
    }

    // Configure applier
    applier.SetDryRun(applyPatchDryRun)
    applier.SetForceOverwrite(applyPatchForce)

    // Apply or validate
    if applyPatchDryRun {
        return runDryRun(applier, patch)
    }

    result, err := applier.Apply(patch)
    if err != nil {
        return formatApplyError(err)
    }

    // Output results
    printResults(result)
    return nil
}

// readPatchInput reads patch from file or stdin
func readPatchInput() (string, error) {
    var reader io.Reader

    if applyPatchFile != "" {
        f, err := os.Open(applyPatchFile)
        if err != nil {
            return "", fmt.Errorf("open file: %w", err)
        }
        defer f.Close()
        reader = f
    } else {
        reader = os.Stdin
    }

    data, err := io.ReadAll(reader)
    if err != nil {
        return "", fmt.Errorf("read input: %w", err)
    }

    return string(data), nil
}

// runDryRun performs dry-run validation
func runDryRun(applier *patchapply.Applier, patch *patchapply.Patch) error {
    if err := applier.ValidatePatch(patch); err != nil {
        return formatApplyError(err)
    }

    fmt.Println("[DRY RUN] Would apply the following changes:")

    for _, op := range patch.Operations {
        switch v := op.(type) {
        case *patchapply.AddFile:
            fmt.Printf("  Would create: %s (%d lines)\n", v.FilePath, len(v.Lines))
        case *patchapply.DeleteFile:
            fmt.Printf("  Would delete: %s\n", v.FilePath)
        case *patchapply.UpdateFile:
            if v.NewPath != "" {
                fmt.Printf("  Would move: %s → %s (%d hunks)\n", v.FilePath, v.NewPath, len(v.Hunks))
            } else {
                fmt.Printf("  Would update: %s (%d hunks)\n", v.FilePath, len(v.Hunks))
            }
        }
    }

    return nil
}

// printResults prints successful application results
func printResults(result *patchapply.ApplyResult) {
    fmt.Println("✓ Applied patch successfully")

    if len(result.FilesCreated) > 0 {
        fmt.Printf("  Created: %d files\n", len(result.FilesCreated))
        if applyPatchVerbose {
            for _, f := range result.FilesCreated {
                fmt.Printf("    - %s\n", f)
            }
        }
    }

    if len(result.FilesUpdated) > 0 {
        fmt.Printf("  Updated: %d files\n", len(result.FilesUpdated))
        if applyPatchVerbose {
            for _, f := range result.FilesUpdated {
                fmt.Printf("    - %s\n", f)
            }
        }
    }

    if len(result.FilesDeleted) > 0 {
        fmt.Printf("  Deleted: %d files\n", len(result.FilesDeleted))
        if applyPatchVerbose {
            for _, f := range result.FilesDeleted {
                fmt.Printf("    - %s\n", f)
            }
        }
    }

    if len(result.FilesMoved) > 0 {
        fmt.Printf("  Moved: %d files\n", len(result.FilesMoved))
        if applyPatchVerbose {
            for old, new := range result.FilesMoved {
                fmt.Printf("    - %s → %s\n", old, new)
            }
        }
    }
}

// formatParseError formats parse errors with helpful hints
func formatParseError(err error) error {
    return fmt.Errorf(`Error: Invalid patch syntax
%v

Hint: Check the patch format specification.
Expected format:
  *** Begin Patch
  *** Add File: path/to/file.txt
  +content line
  *** End Patch`, err)
}

// formatApplyError formats application errors with context
func formatApplyError(err error) error {
    // Extract structured error if available
    if applyErr, ok := err.(*patchapply.Error); ok {
        return fmt.Errorf(`Error: Failed to apply patch
  %s operation on %s (line %d)
  %v

Hint: %s`,
            applyErr.Op,
            applyErr.Path,
            applyErr.Line,
            applyErr.Err,
            getHintForError(applyErr))
    }

    return fmt.Errorf("Error: %v", err)
}

// getHintForError provides helpful hints for common errors
func getHintForError(err *patchapply.Error) string {
    switch {
    case err.Err == patchapply.ErrContextNotFound:
        return "The context may have changed. Update the patch with current file content."
    case err.Err == patchapply.ErrPathOutsideWorkspace:
        return "Use relative paths within the workspace only."
    case err.Err == patchapply.ErrFileExists:
        return "Use --force to overwrite existing files."
    case err.Err == patchapply.ErrFileNotFound:
        return "Ensure the file exists before updating."
    default:
        return "Check the error message above for details."
    }
}

// runApplyPatchMode is called from main.go for special execution modes
func runApplyPatchMode() int {
    // Create a minimal cobra command for special mode
    cmd := &cobra.Command{
        Use:   "spin-apply-patch",
        Short: "Apply a Spin patch (standalone mode)",
        RunE:  runApplyPatch,
    }

    // Add flags
    cmd.Flags().StringVarP(&applyPatchFile, "file", "f", "", "Patch file (default: stdin)")
    cmd.Flags().StringVarP(&applyPatchWorkspace, "workspace", "w", ".", "Workspace directory")
    cmd.Flags().BoolVar(&applyPatchDryRun, "dry-run", false, "Validate without applying")
    cmd.Flags().BoolVar(&applyPatchForce, "force", false, "Force overwrite existing files")
    cmd.Flags().BoolVarP(&applyPatchVerbose, "verbose", "v", false, "Verbose output")

    if err := cmd.Execute(); err != nil {
        fmt.Fprintf(os.Stderr, "%v\n", err)
        return 1
    }
    return 0
}
```

---

## 4. Testing Strategy

### 4.1 Test Categories

#### Unit Tests (cmd/spin/apply_patch_test.go)

**Test Functions:**
1. `TestReadPatchInput_Stdin` - Read from stdin
2. `TestReadPatchInput_File` - Read from file
3. `TestReadPatchInput_FileNotFound` - Error on missing file
4. `TestFormatParseError` - Parse error formatting
5. `TestFormatApplyError` - Apply error formatting
6. `TestGetHintForError` - Error hint generation

#### Integration Tests

**Test Functions:**
1. `TestApplyPatch_Success` - Full successful application
2. `TestApplyPatch_DryRun` - Dry-run mode
3. `TestApplyPatch_FileInput` - Apply from file
4. `TestApplyPatch_StdinInput` - Apply from stdin
5. `TestApplyPatch_ParseError` - Invalid patch syntax
6. `TestApplyPatch_ApplyError` - Application failures
7. `TestApplyPatch_PathTraversal` - Security validation
8. `TestApplyPatch_ForceOverwrite` - Force mode

### 4.2 Test Fixtures

```
test/fixtures/patches/
├── valid_add.patch        # Valid add operation
├── valid_delete.patch     # Valid delete operation
├── valid_update.patch     # Valid update operation
├── valid_move.patch       # Valid move operation
├── invalid_syntax.patch   # Parse error
├── invalid_path.patch     # Path traversal attempt
└── context_missing.patch  # Context not found
```

### 4.3 Test Coverage Target

- **Target:** ≥85% overall
- **Critical paths:** ≥90% (error handling, input parsing)

---

## 5. Acceptance Criteria

### AC-1: Basic Functionality
- [ ] Can apply patch from stdin: `cat test.patch | spin apply-patch`
- [ ] Can apply patch from file: `spin apply-patch -f test.patch`
- [ ] Supports all operation types (Add, Delete, Update, Move)
- [ ] Respects workspace flag: `spin apply-patch -w /path`

### AC-2: Modes
- [ ] Dry-run shows preview: `spin apply-patch --dry-run`
- [ ] Force mode overwrites: `spin apply-patch --force`
- [ ] Verbose shows details: `spin apply-patch -v`

### AC-3: Error Handling
- [ ] Clear parse errors with line numbers
- [ ] Helpful error messages with hints
- [ ] Correct exit codes (0=success, 1=error)
- [ ] Path traversal attempts rejected

### AC-4: Integration
- [ ] Works as symlink: `ln -s spin spin-apply-patch`
- [ ] Works as subcommand: `spin apply-patch`
- [ ] Works with internal flag: `spin --spin-run-as-apply-patch`

### AC-5: Documentation
- [ ] Help text with examples: `spin apply-patch --help`
- [ ] Error messages reference docs
- [ ] README updated with CLI usage

---

## 6. Success Metrics

### Performance
- Parse + apply 100-line patch: <100ms
- Memory usage: <50MB
- No crashes on malformed input

### Usability
- New users can apply patches without reading docs
- Error messages self-explanatory
- Common workflows require minimal flags

### Quality
- Test coverage ≥85%
- Zero lint errors
- Race detector clean
- All acceptance criteria met

---

## 7. Dependencies

### Internal Dependencies
- `internal/patchapply` (Parser, Matcher, Applier) - **Complete**
- `pkg/pathutil` - **Complete**
- `pkg/strutil` - **Complete**

### External Dependencies
- `github.com/spf13/cobra` - CLI framework (already in use)
- Go standard library: `io`, `os`, `fmt`, `path/filepath`

---

## 8. Risks and Mitigations

### Risk 1: Path Security
**Risk:** CLI might allow path traversal
**Mitigation:** All paths validated by `pkg/pathutil`, comprehensive security tests

### Risk 2: Stdin Blocking
**Risk:** Reading from stdin might block indefinitely
**Mitigation:** Document expected usage, consider timeout flag in future

### Risk 3: Large Patches
**Risk:** Memory issues with huge patches
**Mitigation:** Stream-based parser handles this, add size warning if needed

---

## 9. Implementation Plan

### Phase 1: Core Implementation (4 hours)
1. Create `cmd/spin/apply_patch.go`
2. Implement flag parsing
3. Implement input reading (stdin/file)
4. Integrate with patchapply package
5. Implement output formatting

### Phase 2: Testing (3 hours)
1. Create `cmd/spin/apply_patch_test.go`
2. Write unit tests
3. Write integration tests
4. Create test fixtures
5. Achieve ≥85% coverage

### Phase 3: Polish (1 hour)
1. Improve error messages
2. Add help text examples
3. Test all execution modes
4. Run lint and fix issues

---

## 10. Future Enhancements

### Phase 2 Features (Not in this FRD)
- [ ] `--backup` flag to create backups before applying
- [ ] `--continue-on-error` to apply partial patches
- [ ] `--json` output for scripting
- [ ] `--check` to validate without showing changes
- [ ] Progress indicator for large patches

---

## 11. Related Documentation

- [Roadmap: Feature 2.4](../../specs/tools-modules/ROADMAP.md#feature-24-patchapply-cli-tool)
- [Package: patchapply](../../docs/packages/patchapply.md)
- [AGENTS.md](../../AGENTS.md) - Development workflow
- [Tools & Modules Spec](../../specs/tools-modules/tools-modules.md)

---

## 12. Definition of Done

- [x] FRD written and reviewed
- [ ] `cmd/spin/apply_patch.go` implemented
- [ ] `cmd/spin/apply_patch_test.go` written with ≥85% coverage
- [ ] All tests passing (`go test -v -race ./cmd/spin/...`)
- [ ] `make lint` passes (zero errors)
- [ ] Manual testing with real patches complete
- [ ] Help text and examples verified
- [ ] Roadmap updated (Feature 2.4 marked complete)
- [ ] AGENTS.md updated if needed

---

**Status:** Ready for Implementation
**Next Step:** Implement `cmd/spin/apply_patch.go` and tests
**Estimated Completion:** 2025-10-12 (same day)
