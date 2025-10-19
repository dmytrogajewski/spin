# FRD-20251019-split-validator

**Feature:** Split validator.go  
**Date:** 2025-10-19  
**Owner:** Spin Agent  
**Status:** Draft  
**Priority:** MEDIUM 🟡  
**Related:** Feature 3.2 in [core-refactoring/ROADMAP.md](../core-refactoring/ROADMAP.md)

---

## Summary

Split the oversized `validator.go` file (853 lines) into 4 focused files with clear responsibilities. This improves maintainability, readability, and makes it easier to extend validation rules.

**Current State:** 1 file, 853 lines  
**Target State:** 4 files, each ≤250 lines

---

## Problem Statement

The `validator.go` file violates the single responsibility principle and exceeds the 500-line file size limit established in our coding standards. It mixes:

1. Type definitions (CommandClass, Command, ValidationResult, Pattern)
2. Pattern initialization (4 initialization functions with ~500 lines of patterns)
3. Pattern matching logic (checkPattern methods, matchesPattern)
4. Public API (Classify, IsSafe, IsDangerous, NeedsApproval)
5. Command parsing (ParseCommand)

This makes the file difficult to navigate, test, and extend.

---

## Requirements

### Functional Requirements

**FR1:** Split validator.go into 4 files without changing behavior  
**FR2:** All existing tests must pass without modification  
**FR3:** No import cycles introduced  
**FR4:** Public API remains unchanged  
**FR5:** Pattern matching logic remains deterministic  

### Non-Functional Requirements

**NFR1:** Each file ≤500 lines (target: ~200-250 lines)  
**NFR2:** Test coverage ≥90% maintained  
**NFR3:** Zero complexity increase  
**NFR4:** Clear godoc for each file  
**NFR5:** No performance regression  

---

## Design

### File Split Strategy

#### 1. **validator_types.go** (~150 lines)
Move all type definitions and helpers:
- `CommandClass` type and constants (Safe, Interactive, Dangerous, Forbidden, Unverified)
- `CommandClass.String()` method
- `CommandClass.NeedsApproval()` method
- `Command` struct
- `ValidationResult` struct
- `Pattern` struct
- Error constants (ErrInvalidCommand, ErrParseError, ErrEmptyCommand)
- `ParseCommand` function

**Rationale:** Types and parsing are fundamental building blocks used by all other validator code. Separating them makes the type system clear and reusable.

#### 2. **validator_patterns.go** (~500 lines)
Move all pattern initialization:
- `initializeForbiddenPatterns` method
- `initializeDangerousPatterns` method
- `initializeInteractivePatterns` method
- `initializeSafePatterns` method
- All Pattern definitions inline in these methods

**Rationale:** Pattern definitions are data-heavy and stable. Isolating them makes it easy to add/modify patterns without touching classification logic.

#### 3. **validator_matchers.go** (~100 lines)
Move all pattern matching logic:
- `checkSpecialForbiddenPatterns` method
- `checkPatternList` method
- `checkPatternMap` method
- `matchesPattern` method

**Rationale:** Pattern matching is algorithmic logic separate from pattern data. This separation allows testing matching logic independently.

#### 4. **validator.go** (~100 lines)
Keep only core API:
- `Validator` struct
- `NewValidator` constructor
- `Classify` method (orchestration only)
- `IsSafe` method
- `IsInteractive` method
- `IsDangerous` method
- `IsForbidden` method
- `NeedsApproval` method
- Package godoc

**Rationale:** This becomes the public API surface, easy to understand and use. All implementation details hidden in other files.

### Import Strategy

No additional imports needed. All files remain in `package core` and share the same package namespace.

**Import Cycle Risk:** NONE - All files are in the same package with no circular dependencies.

### Backward Compatibility

**100% Backward Compatible** - No changes to public API:
- All exported functions keep same signatures
- All exported types keep same fields
- All package-level behavior identical

---

## Implementation Plan

### Phase 1: Create validator_types.go

1. Create new file with package declaration and imports
2. Move type definitions:
   - CommandClass + constants + methods
   - Command struct
   - ValidationResult struct
   - Pattern struct
   - Error constants
3. Move ParseCommand function
4. Add file-level godoc
5. Run tests: `go test ./internal/core/ -run TestParseCommand -run TestCommandClass`

### Phase 2: Create validator_patterns.go

1. Create new file with package declaration
2. Move all initialization methods:
   - initializeForbiddenPatterns
   - initializeDangerousPatterns
   - initializeInteractivePatterns
   - initializeSafePatterns
3. Add file-level godoc
4. Run tests: `go test ./internal/core/ -run TestValidator_Classify`

### Phase 3: Create validator_matchers.go

1. Create new file with package declaration
2. Move all matching methods:
   - checkSpecialForbiddenPatterns
   - checkPatternList
   - checkPatternMap
   - matchesPattern
3. Add file-level godoc
4. Run tests: `go test ./internal/core/ -run TestValidator_Classify`

### Phase 4: Clean up validator.go

1. Remove moved code
2. Keep only:
   - Validator struct
   - NewValidator
   - Classify (orchestration)
   - Public API methods (IsSafe, etc.)
3. Update file-level godoc
4. Run full test suite

### Phase 5: Verification

1. Run all validator tests: `go test -v ./internal/core/ -run TestValidator`
2. Check no import cycles: `go build ./internal/core/...`
3. Check file sizes: `wc -l internal/core/validator*.go`
4. Check coverage: `go test -cover ./internal/core/ -run TestValidator`
5. Run linter: `make lint`
6. Check complexity: `gocyclo -over 15 internal/core/validator*.go`

---

## Testing Strategy

### Test Coverage Goals

**Overall Target:** ≥90% coverage for all validator files

**Per-File Coverage:**
- validator_types.go: ≥95% (simple type methods)
- validator_patterns.go: ≥85% (pattern data, tested via Classify)
- validator_matchers.go: ≥95% (matching logic, critical path)
- validator.go: ≥95% (public API)

### Test Approach

**No new tests needed** - Existing test suite already comprehensive:
- `TestParseCommand` - 6 cases covering parsing logic
- `TestCommandClass_String` - 6 cases covering string representation
- `TestCommandClass_NeedsApproval` - 5 cases covering approval logic
- `TestValidator_Classify` - 7 cases covering classification
- `TestValidator_IsSafe` - 3 cases
- `TestValidator_IsDangerous` - 3 cases
- `TestValidator_IsForbidden` - 3 cases
- `TestValidator_NeedsApproval` - 6 cases
- `TestValidator_SpecialForbiddenPatterns` - 3 cases
- `TestValidator_Concurrency` - Concurrent safety test

**Total:** 45 test cases covering all validator functionality.

### Test Execution

Run after each phase:
```bash
# Quick check
go test ./internal/core/ -run TestValidator

# Full check with coverage
go test -v -race -cover ./internal/core/ -run TestValidator

# Coverage report
go test -coverprofile=coverage.out ./internal/core/
go tool cover -func=coverage.out | grep validator
```

---

## Risks and Mitigation

### Risk 1: Import Cycles 🟢 LOW
**Impact:** Code won't compile  
**Probability:** Very Low  
**Mitigation:** All files in same package, no imports needed  
**Detection:** `go build ./internal/core/...` will fail immediately  

### Risk 2: Breaking Tests 🟢 LOW
**Impact:** Existing tests fail  
**Probability:** Very Low (only moving code, not changing logic)  
**Mitigation:** Run tests after each phase, revert if any fail  
**Detection:** `go test ./internal/core/` shows failures immediately  

### Risk 3: Performance Regression 🟢 LOW
**Impact:** Slower command validation  
**Probability:** Very Low (no algorithm changes)  
**Mitigation:** All methods remain in same package, inlining preserved  
**Detection:** Benchmark if concerned: `go test -bench=BenchmarkValidatorClassify`  

### Risk 4: Merge Conflicts 🟡 MEDIUM
**Impact:** Git conflicts if validator.go modified in parallel  
**Probability:** Medium  
**Mitigation:** Complete feature in single session, coordinate with team  
**Detection:** Git merge conflicts during PR  

---

## Acceptance Criteria

### Definition of Done

- [x] **DoR.1:** Current validator.go structure analyzed
- [x] **DoR.2:** Split strategy defined (4 files)
- [x] **DoR.3:** Import cycle risk assessed (NONE - same package)
- [x] **DoR.4:** FRD created and reviewed

- [ ] **DoD.1:** validator.go ≤200 lines (target: ~100)
- [ ] **DoD.2:** 4 new files created:
  - validator_types.go (~150 lines)
  - validator_patterns.go (~500 lines)
  - validator_matchers.go (~100 lines)
  - validator.go (~100 lines)
- [ ] **DoD.3:** No import cycles (`go build ./internal/core/...` succeeds)
- [ ] **DoD.4:** All tests pass (`go test -v -race ./internal/core/ -run TestValidator`)
- [ ] **DoD.5:** Coverage ≥90% maintained (target: 95%+)
- [ ] **DoD.6:** Each file has clear godoc
- [ ] **DoD.7:** Code review completed
- [ ] **DoD.8:** All files ≤500 lines

### Success Metrics

```bash
# File size check (all ≤500 lines)
wc -l internal/core/validator*.go
# Expected output:
#   ~100 validator.go
#   ~150 validator_types.go
#   ~500 validator_patterns.go
#   ~100 validator_matchers.go

# No import cycles
go build ./internal/core/...
# Expected: Success (no errors)

# All tests pass
go test -v ./internal/core/ -run TestValidator
# Expected: PASS (45/45 tests)

# Coverage maintained
go test -coverprofile=coverage.out ./internal/core/
go tool cover -func=coverage.out | grep validator
# Expected: Each file ≥90% coverage

# Complexity check
gocyclo -over 15 internal/core/validator*.go
# Expected: No output (all functions ≤15)
```

---

## Timeline

**Total Estimated Effort:** 2 hours

| Phase | Task | Duration | Dependencies |
|-------|------|----------|--------------|
| 1 | Create validator_types.go | 30 min | None |
| 2 | Create validator_patterns.go | 30 min | Phase 1 |
| 3 | Create validator_matchers.go | 15 min | Phase 1, 2 |
| 4 | Clean up validator.go | 15 min | Phase 1, 2, 3 |
| 5 | Verification & Documentation | 30 min | Phase 1-4 |

---

## References

### Internal
- [ROADMAP.md](../core-refactoring/ROADMAP.md) - Phase 3, Feature 3.2
- [analysis.md](../core-refactoring/analysis.md) - File size issues
- [AGENTS.md](../../AGENTS.md) - Quality standards
- [docs/packages/core.md](../../docs/packages/core.md) - Core package docs

### Code
- `internal/core/validator.go` - Current implementation (853 lines)
- `internal/core/validator_test.go` - Test suite (536 lines, 45 test cases)

### Standards
- Go file size limit: 500 lines
- Coverage requirement: ≥90%
- Complexity limit: ≤15

---

## Appendix A: File Structure Before/After

### Before
```
internal/core/
  validator.go         (853 lines)
  validator_test.go    (536 lines)
```

### After
```
internal/core/
  validator.go          (~100 lines) - Public API + Classify orchestration
  validator_types.go    (~150 lines) - Types, constants, ParseCommand
  validator_patterns.go (~500 lines) - Pattern initialization
  validator_matchers.go (~100 lines) - Pattern matching logic
  validator_test.go     (536 lines) - Tests (unchanged)
```

---

## Appendix B: Godoc Examples

### validator.go
```go
// Package core provides the validator for command safety classification.
//
// The validator classifies commands into five categories:
//   - Safe: Read-only operations (ls, cat, git status)
//   - Interactive: Write operations (mkdir, npm install)
//   - Dangerous: Destructive operations (rm -rf, sudo)
//   - Forbidden: Catastrophic operations (rm -rf /, fork bombs)
//   - Unverified: Unknown commands
```

### validator_types.go
```go
// validator_types.go defines the core types used by the command validator.
//
// This file contains:
//   - CommandClass: Safety classification levels
//   - Command: Parsed command structure
//   - ValidationResult: Classification result with reason
//   - Pattern: Pattern matching rules
```

### validator_patterns.go
```go
// validator_patterns.go defines pattern initialization for command classification.
//
// This file contains pattern definitions for:
//   - Forbidden commands (catastrophic operations)
//   - Dangerous commands (destructive operations)
//   - Interactive commands (write operations)
//   - Safe commands (read-only operations)
```

### validator_matchers.go
```go
// validator_matchers.go implements pattern matching logic for command classification.
//
// This file contains methods for:
//   - Checking special forbidden patterns (fork bombs, disk overwrite)
//   - Matching patterns against commands
//   - Checking pattern lists and maps
```

---

**FRD Version:** 1.0  
**Created:** 2025-10-19  
**Last Updated:** 2025-10-19  
**Next Review:** After implementation

