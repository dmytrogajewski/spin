# JOURNEY-0.1: gofmt + goimports formatting cleanup

## Roadmap Link
- Source roadmap: specs/lint-cleanup/roadmap.md
- Feature: 0.1 — `gofmt` + `goimports` formatting (~183 issues)

## 1. Journey

When **a developer maintaining the spin codebase** I want to **have all Go source files follow canonical gofmt formatting and correctly grouped imports** so I can **pass the gofmt and goimports lint checks with zero issues and establish a clean formatting baseline for subsequent lint fixes**.

## 2. CJM

A developer runs `make lint` and sees ~183 issues from gofmt (incorrect formatting) and goimports (wrong import grouping/ordering). These are purely mechanical — no logic changes required. The fix is running the standard Go formatters. This is the foundation for all subsequent lint cleanup waves because formatting must be stable before addressing style, correctness, or complexity linters.

### Phase 1: Discovery

**User Intent:** Understand the scope of gofmt/goimports violations.

**Actions:** Run `make lint` or `golangci-lint run --enable-only gofmt,goimports ./...` and review output.

**Pain / Risk:**
1. Output is noisy — 183 issues mixed with 25,000+ other lint issues makes triage hard.
2. Developer may not know which files are affected without filtering.
3. Some files may have been intentionally formatted differently (unlikely but possible).

**Success Signal:** Developer has a clear count of gofmt + goimports issues and knows they are all auto-fixable.

### Phase 2: Auto-fix

**User Intent:** Apply canonical formatting to all Go source files.

**Actions:**
1. Run `gofmt -w .` to fix all formatting.
2. Run `goimports -w -local github.com/dmytrogajewski/spin .` to fix import grouping.

**Pain / Risk:**
1. `goimports` binary might not be installed — need `go install golang.org/x/tools/cmd/goimports@latest`.
2. `goimports` could remove imports that are used only via side effects (blank imports) — verify with compilation.
3. Formatting changes could cause merge conflicts with in-flight branches.

**Success Signal:** Both commands exit 0; `go build ./...` still compiles.

### Phase 3: Verification

**User Intent:** Confirm zero gofmt/goimports lint issues and no test regressions.

**Actions:**
1. Run `golangci-lint run --enable-only gofmt,goimports ./...` — expect 0 issues.
2. Run `go vet ./...` — expect no new issues.
3. Run `make test` — expect all tests pass.

**Pain / Risk:**
1. Linter cache might show stale results — clear cache or use `--no-cache`.
2. Tests could fail if formatting tools accidentally modified string literals (extremely unlikely).
3. `go vet` may report pre-existing issues unrelated to formatting — these are out of scope for this item.

**Success Signal:** Zero gofmt/goimports issues reported; all tests green.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| 183 issues mixed in 25k+ total output | Discovery | Run linter with `--enable-only` filter |
| goimports binary may not be installed | Auto-fix | Document required tooling in Makefile |
| Merge conflicts on formatting-only changes | Auto-fix | Do formatting first, before any logic changes |

### North Star Summary

All Go source files in the spin codebase follow canonical `gofmt` formatting and have imports correctly grouped into stdlib / third-party / local sections. The gofmt and goimports linters report zero issues. This establishes the formatting baseline that all subsequent lint cleanup waves build upon.

## 3. UX Implementation and Assessment

### Time to First Value
- [x] Fully automated — single command fixes all issues
- [x] No manual review needed for formatting changes

### Onboarding Clarity
- [x] Standard Go tooling — every Go developer knows gofmt
- [x] No configuration decisions required

### Production-Ready Defaults
- [x] gofmt is the canonical Go formatter — no custom rules
- [x] goimports local prefix configured in .golangci.yml

### Golden Path Quality
- [x] Run formatters, verify with linter, run tests — three steps
- [x] All changes are idempotent — re-running produces no diff

### Error Quality
- [x] gofmt/goimports report clear file:line errors
- [x] Linter output shows exact formatting difference expected

### Failure Safety
- [x] Changes are purely cosmetic — zero logic risk
- [x] Reversible via version control

## 4. Tests

### TC-01: gofmt reports zero issues after formatting

**Given** all `.go` files have been processed by `gofmt -w .`.
**When** `golangci-lint run --enable-only gofmt ./...` is executed.
**Then** exit code is 0 and output contains no issues.

### TC-02: goimports reports zero issues after formatting

**Given** all `.go` files have been processed by `goimports -w -local github.com/dmytrogajewski/spin .`.
**When** `golangci-lint run --enable-only goimports ./...` is executed.
**Then** exit code is 0 and output contains no issues.

### TC-03: codebase compiles after formatting

**Given** gofmt and goimports have been applied.
**When** `go build ./...` is executed.
**Then** exit code is 0 with no compilation errors.

### TC-04: all existing tests pass after formatting

**Given** gofmt and goimports have been applied.
**When** `make test` is executed.
**Then** all tests pass with exit code 0.

### TC-05: go vet reports no new issues from formatting

**Given** gofmt and goimports have been applied.
**When** `go vet ./...` is executed.
**Then** no new issues introduced by the formatting changes.

## Traceability
- Roadmap item: specs/lint-cleanup/roadmap.md — item 0.1
- Implementation files: all `.go` files (mechanical formatting, no logic changes)
- Test files: N/A (verified by linter and existing test suite)
