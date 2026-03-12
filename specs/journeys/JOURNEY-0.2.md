# JOURNEY-0.2: godot comment periods

## Roadmap Link
- Source roadmap: specs/lint-cleanup/roadmap.md
- Feature: 0.2 — `godot` comment periods (~7,946 issues)

## 1. Journey

When **a developer maintaining the spin codebase** I want to **have all comments end with a period** so I can **pass the godot lint check with zero issues and maintain consistent comment style across the project**.

## 2. CJM

The godot linter enforces that all comments end with a period. With ~7,946 violations, this is the single largest lint issue category (31% of all issues). The fix is fully auto-fixable via `golangci-lint run --fix`. No logic changes, no risk.

### Phase 1: Discovery

**User Intent:** Identify all comments missing trailing periods.

**Actions:** Run `golangci-lint run --enable-only godot ./...` (or grep lint output for godot).

**Pain / Risk:**
1. Volume is overwhelming — nearly 8,000 issues across the entire codebase.
2. Some comments may be code examples or structured text where a period is awkward.
3. Multi-line comments may need the period only on the last line.

**Success Signal:** Clear count of godot violations; all are auto-fixable.

### Phase 2: Auto-fix

**User Intent:** Add trailing periods to all comments automatically.

**Actions:** Run `golangci-lint run --fix` with godot enabled, or use the full config which already includes godot.

**Pain / Risk:**
1. Auto-fixer might add periods to comments that are intentionally fragments (e.g., struct field comments).
2. Comments ending with code snippets or URLs could look odd with an appended period.
3. Generated code comments (e.g., protobuf, stringer) might get modified — verify no generated files exist.

**Success Signal:** Fixer exits 0; diff shows only period additions at end of comments.

### Phase 3: Verification

**User Intent:** Confirm zero godot issues and no test regressions.

**Actions:**
1. Run lint and grep for godot — expect 0 issues.
2. Run `make test` — expect all tests pass.

**Pain / Risk:**
1. Adding periods to `// +build` or `//go:` directives would break compilation — godot should skip these.
2. Test string literals containing comments might be affected — unlikely but verify.
3. Linter cache could hide remaining issues.

**Success Signal:** Zero godot issues; all tests green.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| 7,946 issues — largest single category | Discovery | Auto-fix eliminates 31% of all lint issues in one command |
| Potential false positives on structured comments | Auto-fix | godot respects Go conventions for directives |
| Volume of diff makes review impractical | Verification | Changes are mechanical — verify by linter + tests only |

### North Star Summary

Every comment in the spin codebase ends with a period, following Go documentation conventions. The godot linter reports zero issues. This eliminates the single largest category of lint violations.

## 3. UX Implementation and Assessment

### Time to First Value
- [x] Single command auto-fix
- [x] No manual review needed

### Golden Path Quality
- [x] Run fixer, verify with linter, run tests
- [x] Idempotent — re-running produces no diff

### Failure Safety
- [x] Comments-only changes — zero logic risk
- [x] Reversible via version control

## 4. Tests

### TC-01: godot reports zero issues after fix

**Given** all comments have been processed by the godot auto-fixer.
**When** `golangci-lint run ./...` output is filtered for godot.
**Then** zero godot issues are reported.

### TC-02: codebase compiles after comment changes

**Given** godot auto-fixer has been applied.
**When** `go build ./...` is executed.
**Then** compilation succeeds with exit code 0.

### TC-03: all tests pass after comment changes

**Given** godot auto-fixer has been applied.
**When** `make test` is executed.
**Then** all tests pass.

### TC-04: go directives are not affected

**Given** godot auto-fixer has been applied.
**When** files containing `//go:generate`, `//go:build`, or `//go:embed` are inspected.
**Then** directive comments are unchanged.

## Traceability
- Roadmap item: specs/lint-cleanup/roadmap.md — item 0.2
- Implementation files: all `.go` files (comment-only changes)
- Test files: N/A (verified by linter and existing test suite)
