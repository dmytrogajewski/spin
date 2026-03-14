# JOURNEY-dedup-task-validate: Deduplicate Task Validate() and MaxAllowedTokens

## Roadmap Link
- Source roadmap: specs/ref/ROADMAP.md
- Feature: 1.3 — Deduplicate Task Validate() and MaxAllowedTokens
- Cluster: 14 (SPEC.md) | LIST.md findings 61, 62

## 1. Journey

When **a developer adding or modifying task modes in spin** I want to **have a single shared validation function for max tokens** so I can **avoid duplicating the same validation logic across all task types and ensure consistent error messages**.

## 2. CJM

Four task types (`Compact`, `Planning`, `Regular`, `Review`) have identical `Validate()` bodies that check:
1. `maxTokens <= 0` returns `ErrMaxTokensMustBePositive`
2. `maxTokens > MaxAllowedTokens` returns wrapped `ErrMaxTokensExceedsMaximumAllowed`

Additionally, `regular.go` defines a private `maxAllowedTokens = 100000` that duplicates the exported `MaxAllowedTokens` from `constants.go`.

### Phase 1: Discovery

**User Intent:** Understand the scope of duplicated validation.

**Actions:** Read all four `Validate()` methods and confirm they are identical.

**Pain / Risk:**
1. Four copies of the same logic — any fix must be applied four times.
2. `regular.go` uses a private constant instead of the shared one.

**Success Signal:** All four implementations confirmed identical.

### Phase 2: Extraction

**User Intent:** Extract a shared `validateMaxTokens` function.

**Actions:**
1. Create `internal/task/validate.go` with `validateMaxTokens(maxTokens int) error`.
2. Update all four `Validate()` methods to call `validateMaxTokens`.
3. Remove private `maxAllowedTokens` constant from `regular.go`.

**Pain / Risk:**
1. Must ensure error messages remain identical for backward compatibility.
2. The `dupl` linter (threshold 100) should stop flagging these as duplicates.

**Success Signal:** Single validation function, all four modes delegate to it.

### Phase 3: Verification

**User Intent:** Confirm no regressions.

**Actions:**
1. `go vet ./...` passes.
2. `go test ./internal/task/...` passes.
3. `make lint` passes.

**Success Signal:** All tests green, lint clean.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| 4 identical Validate() bodies | Discovery | Single shared function |
| Private maxAllowedTokens shadows public constant | Discovery | Remove duplicate constant |

### North Star Summary

A single `validateMaxTokens` function in `internal/task/validate.go` handles max tokens validation for all task modes. The duplicate private constant is removed. Adding a new task mode requires only calling `validateMaxTokens` instead of copying 10 lines of validation.

## 3. UX Implementation and Assessment

### Time to First Value
- [x] Immediate — reduces code duplication

### Production-Ready Defaults
- [x] Error messages unchanged — no behavioral change

### Golden Path Quality
- [x] New task modes only need one function call for validation

### Failure Safety
- [x] Pure refactoring — no behavioral change
- [x] Existing tests validate correctness

## 4. Tests

### TC-01: validateMaxTokens rejects zero

**Given** maxTokens is 0.
**When** `validateMaxTokens(0)` is called.
**Then** it returns `ErrMaxTokensMustBePositive`.

### TC-02: validateMaxTokens rejects negative

**Given** maxTokens is -1.
**When** `validateMaxTokens(-1)` is called.
**Then** it returns `ErrMaxTokensMustBePositive`.

### TC-03: validateMaxTokens accepts valid value

**Given** maxTokens is 4096.
**When** `validateMaxTokens(4096)` is called.
**Then** it returns nil.

### TC-04: validateMaxTokens accepts MaxAllowedTokens exactly

**Given** maxTokens is MaxAllowedTokens.
**When** `validateMaxTokens(MaxAllowedTokens)` is called.
**Then** it returns nil.

### TC-05: validateMaxTokens rejects MaxAllowedTokens+1

**Given** maxTokens is MaxAllowedTokens+1.
**When** `validateMaxTokens(MaxAllowedTokens+1)` is called.
**Then** it returns `ErrMaxTokensExceedsMaximumAllowed`.

### TC-06: all task Validate() methods still work

**Given** all four task types are created with defaults.
**When** `Validate()` is called on each.
**Then** all return nil.

## Implementation

- Created: `internal/task/validate.go` — shared `validateMaxTokens` function
- Created: `internal/task/validate_test.go` — unit tests for `validateMaxTokens`
- Modified: `internal/task/compact.go` — `Validate()` delegates to `validateMaxTokens`
- Modified: `internal/task/planning.go` — `Validate()` delegates to `validateMaxTokens`
- Modified: `internal/task/regular.go` — `Validate()` delegates to `validateMaxTokens`, removed private `maxAllowedTokens`
- Modified: `internal/task/review.go` — `Validate()` delegates to `validateMaxTokens`
