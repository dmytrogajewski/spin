# Roadmap: Lint Cleanup (25,378 issues)

**Created:** 2026-03-12
**Spec:** [spec-lint-issues.md](./spec-lint-issues.md)
**Strategy:** Progressive waves — each wave is independently valuable, testable, and brings the codebase measurably closer to `make lint` passing.

> **RULE: `//nolint` directives are PROHIBITED.** Every issue must be fixed by changing the code, not by suppressing the linter. No exceptions. If a linter rule is truly inapplicable to the project, disable it globally in `.golangci.yml` — never per-line.

---

## Wave 0: Auto-fixable Formatting (est. ~8,300 issues)

Mechanical fixes that require zero human judgment. Safe to batch.

- [x] **0.1 — `gofmt` + `goimports` formatting** (~183 issues) — [JOURNEY-0.1](../journeys/JOURNEY-0.1.md)
  - **Description:** Run `gofmt -w` and `goimports -w` across the codebase. Purely mechanical.
  - **DoR:** golangci-lint installed, codebase compiles
  - **DoD:** `golangci-lint run --enable-only gofmt,goimports ./...` reports 0 issues; `make test` passes
  - **How:** `gofmt -w .` then `goimports -w -local github.com/dmytrogajewski/spin .`
  - **`//nolint` prohibited** — all formatting issues must be fixed in source

- [x] **0.2 — `godot` comment periods** (~7,946 issues) — [JOURNEY-0.2](../journeys/JOURNEY-0.2.md)
  - **Description:** Add trailing periods to comments. Largest single linter. Fully auto-fixable.
  - **DoR:** 0.1 complete
  - **DoD:** `golangci-lint run --enable-only godot ./...` reports 0; `make test` passes
  - **How:** `golangci-lint run --fix --enable-only godot ./...`
  - **`//nolint` prohibited** — every comment must end with a period

- [x] **0.3 — `misspell` spelling corrections** (~107 issues)
  - **Description:** Fix typos in comments and strings.
  - **DoR:** 0.2 complete
  - **DoD:** `golangci-lint run --enable-only misspell ./...` reports 0
  - **How:** `golangci-lint run --fix --enable-only misspell ./...`
  - **`//nolint` prohibited** — fix every typo

- [x] **0.4 — `dupword` duplicate words** (~3 issues)
  - **Description:** Fix duplicate adjacent words in comments.
  - **DoR:** 0.3 complete
  - **DoD:** 0 issues from dupword
  - **`//nolint` prohibited** — fix duplicate words in source

- [x] **0.5 — `tagalign` struct tag alignment** (~144 issues)
  - **Description:** Align struct tags consistently.
  - **DoR:** 0.4 complete
  - **DoD:** `golangci-lint run --enable-only tagalign ./...` reports 0
  - **How:** `golangci-lint run --fix --enable-only tagalign ./...`
  - **`//nolint` prohibited** — align all struct tags

**Wave 0 Checkpoint:** ~8,300 issues resolved. Verify: `make test` passes, diff is formatting-only.

---

## Wave 1: Whitespace & Style Noise (est. ~5,900 issues)

Stylistic rules that mostly add/remove blank lines. Low risk, high count.

- [x] **1.1 — `wsl_v5` whitespace rules** (~4,196 issues)
  - **Description:** Add required blank lines before blocks, remove unnecessary ones. Partially auto-fixable.
  - **DoR:** Wave 0 complete
  - **DoD:** 0 wsl_v5 issues; `make test` passes
  - **How:** `golangci-lint run --fix --enable-only wsl_v5 ./...` for auto-fixable; manual fix for remainder
  - **Risk:** Some conflicts with other whitespace linters — verify no regressions
  - **`//nolint` prohibited** — fix whitespace in source

- [x] **1.2 — `nlreturn` newlines before returns** (~1,267 issues)
  - **Description:** Add blank line before return/break/continue statements.
  - **DoR:** 1.1 complete
  - **DoD:** 0 nlreturn issues
  - **How:** `golangci-lint run --fix --enable-only nlreturn ./...`
  - **`//nolint` prohibited** — add the blank lines

- [x] **1.3 — `perfsprint` string formatting** (~251 issues)
  - **Description:** Replace `fmt.Sprintf` with `strconv` equivalents where applicable.
  - **DoR:** 1.2 complete
  - **DoD:** 0 perfsprint issues; `make test` passes
  - **`//nolint` prohibited** — use proper string conversion functions

- [x] **1.4 — `intrange` integer range loops** (~348 issues)
  - **Description:** Replace `for i := 0; i < n; i++` with `for i := range n` (Go 1.22+).
  - **DoR:** 1.3 complete
  - **DoD:** 0 intrange issues; `make test` passes
  - **`//nolint` prohibited** — use modern range syntax

**Wave 1 Checkpoint:** ~14,200 cumulative issues resolved.

---

## Wave 2: Modern Go & Code Hygiene (est. ~2,000 issues)

Adopt modern Go idioms and fix low-severity code issues.

- [x] **2.1 — `modernize` Go modernization** (~657 issues)
  - **Description:** Use modern Go constructs (slices package, errors.Join, range-over-func, etc.).
  - **DoR:** Wave 1 complete
  - **DoD:** 0 modernize issues; `make test` passes
  - **`//nolint` prohibited** — adopt modern idioms in code

- [x] **2.2 — `noinlineerr` error handling style** (~588 issues)
  - **Description:** Extract error checks from inline assignments.
  - **DoR:** 2.1 complete
  - **DoD:** 0 noinlineerr issues
  - **`//nolint` prohibited** — restructure error handling code

- [ ] **2.3 — `revive` miscellaneous style** (~772 issues)
  - **Description:** Fix exported function docs, var declarations, error returns, naming conventions.
  - **DoR:** 2.2 complete
  - **DoD:** 0 revive issues; `make test` passes
  - **Note:** Some revive rules overlap with other linters — fix revive last in this wave
  - **`//nolint` prohibited** — fix all revive violations in source

**Wave 2 Checkpoint:** ~16,200 cumulative issues resolved.

---

## Wave 3: Logging & Forbidden Calls (est. ~500 issues)

- [ ] **3.1 — `forbidigo` replace fmt.Print with slog** (~86 issues)
  - **Description:** Replace `fmt.Print*` calls with structured logging via `log/slog`.
  - **DoR:** Wave 2 complete
  - **DoD:** 0 forbidigo issues; `make test` passes
  - **Note:** Requires deciding on slog handler strategy (context-based vs global)
  - **`//nolint` prohibited** — replace every fmt.Print call with slog

- [ ] **3.2 — `sloglint` structured logging compliance** (~412 issues)
  - **Description:** Fix slog usage: use context-scoped loggers, avoid global slog calls.
  - **DoR:** 3.1 complete
  - **DoD:** 0 sloglint issues; `make test` passes
  - **Note:** `context: scope` setting requires all slog calls to use `slog.InfoContext(ctx, ...)` form
  - **`//nolint` prohibited** — pass context to all slog calls

**Wave 3 Checkpoint:** ~16,700 cumulative issues resolved.

---

## Wave 4: Error Handling & Correctness (est. ~2,500 issues)

High-value fixes that improve reliability.

- [ ] **4.1 — `errcheck` unchecked errors** (~1,239 issues)
  - **Description:** Handle or explicitly ignore all error returns with `_ =` assignment. Largest correctness bucket.
  - **DoR:** Wave 3 complete
  - **DoD:** 0 errcheck issues; `make test` passes
  - **Strategy:** Triage by package — start with `internal/agent`, `internal/llm`, `cmd/spin`
  - **`//nolint` prohibited** — handle every error or use explicit `_ =` assignment

- [ ] **4.2 — `err113` sentinel errors / error wrapping** (~505 issues)
  - **Description:** Define sentinel errors, use `%w` for wrapping, use `errors.Is`/`errors.As`.
  - **DoR:** 4.1 complete
  - **DoD:** 0 err113 issues
  - **Note:** Requires creating package-level `var Err* = errors.New(...)` vars
  - **`//nolint` prohibited** — define proper sentinel errors and wrap with `%w`

- [ ] **4.3 — `errorlint` error comparison** (~69 issues)
  - **Description:** Replace `==` error comparisons with `errors.Is`, type switches with `errors.As`.
  - **DoR:** 4.2 complete
  - **DoD:** 0 errorlint issues
  - **`//nolint` prohibited** — use errors.Is/errors.As

- [ ] **4.4 — `wrapcheck` error wrapping at boundaries** (~92 issues)
  - **Description:** Wrap errors at package boundaries for better stack traces.
  - **DoR:** 4.3 complete
  - **DoD:** 0 wrapcheck issues
  - **`//nolint` prohibited** — wrap errors with `fmt.Errorf("...: %w", err)`

- [ ] **4.5 — `nilerr` + `nilnil` return consistency** (~58 issues)
  - **Description:** Fix functions that return nil error with nil value, or return nil when err != nil.
  - **DoR:** 4.4 complete
  - **DoD:** 0 nilerr + nilnil issues
  - **`//nolint` prohibited** — fix return logic

- [ ] **4.6 — `govet` vet checks** (~336 issues)
  - **Description:** Fix shadow variables, printf format strings, struct tags, atomic operations, etc.
  - **DoR:** 4.5 complete
  - **DoD:** 0 govet issues; `make test` passes
  - **Note:** Shadow checks (strict mode) will be the bulk — requires variable renames
  - **`//nolint` prohibited** — rename shadowed variables, fix format strings

- [ ] **4.7 — `staticcheck` analysis** (~306 issues)
  - **Description:** Fix deprecated API usage, unreachable code, unnecessary conversions, etc.
  - **DoR:** 4.6 complete
  - **DoD:** 0 staticcheck issues
  - **`//nolint` prohibited** — fix deprecated calls, remove dead code

**Wave 4 Checkpoint:** ~19,200 cumulative issues resolved.

---

## Wave 5: Testing Quality (est. ~3,100 issues)

- [ ] **5.1 — `paralleltest` + `tparallel` parallel tests** (~2,639 issues)
  - **Description:** Add `t.Parallel()` to test functions and subtests.
  - **DoR:** Wave 4 complete
  - **DoD:** 0 paralleltest/tparallel issues; `make test -race` passes
  - **Strategy:** Package-by-package; refactor tests with shared state before adding t.Parallel()
  - **Risk:** Tests with shared file system or global state may break — run with `-race`
  - **`//nolint` prohibited** — make every test parallel-safe

- [ ] **5.2 — `testifylint` testify best practices** (~216 issues)
  - **Description:** Use `require` instead of `assert` for fatal checks, fix comparison order, etc.
  - **DoR:** 5.1 complete
  - **DoD:** 0 testifylint issues
  - **`//nolint` prohibited** — fix testify usage

- [ ] **5.3 — `testpackage` separate test packages** (~217 issues)
  - **Description:** Move tests to `_test` packages to test public API only.
  - **DoR:** 5.2 complete
  - **DoD:** 0 testpackage issues
  - **`//nolint` prohibited** — restructure test packages; export helpers if needed for black-box testing

- [ ] **5.4 — `thelper` + `usetesting` test helpers** (~43 issues)
  - **Description:** Mark test helpers with `t.Helper()`, use `testing.TempDir()`.
  - **DoR:** 5.3 complete
  - **DoD:** 0 thelper/usetesting issues
  - **`//nolint` prohibited** — add t.Helper() calls, use testing.TempDir()

**Wave 5 Checkpoint:** ~22,300 cumulative issues resolved.

---

## Wave 6: Complexity Reduction (est. ~500 issues)

- [ ] **6.1 — `cyclop` + `gocyclo` + `gocognit` complexity** (~235 issues)
  - **Description:** Refactor high-complexity functions. Extract sub-functions, use early returns, table-driven patterns.
  - **DoR:** Wave 5 complete
  - **DoD:** 0 cyclop/gocyclo/gocognit issues; `make test` passes
  - **Key targets:**
    - `agent.go:callLLM` (complexity 46)
    - `loop.go:executeAgentLoop` (complexity 38)
    - `ace_service.go:NewACEService` (complexity 34)
    - `tui.go:runTUI` (complexity 33)
    - `config/loader_v2.go:applyDefaults` (complexity 30)
    - `security/approval.go:RequestApproval` (complexity 30)
  - **Strategy:** Extract helper functions, use strategy/handler patterns, break into sub-methods
  - **`//nolint` prohibited** — refactor functions to reduce complexity

- [ ] **6.2 — `funlen` function length** (~117 issues)
  - **Description:** Split long functions (>80 lines) into focused sub-functions.
  - **DoR:** 6.1 complete (complexity reduction already handles many)
  - **DoD:** 0 funlen issues
  - **`//nolint` prohibited** — split functions, do not suppress

- [ ] **6.3 — `nestif` nested conditionals** (~65 issues)
  - **Description:** Flatten nested ifs using early returns, guard clauses.
  - **DoR:** 6.2 complete
  - **DoD:** 0 nestif issues
  - **`//nolint` prohibited** — restructure control flow

- [ ] **6.4 — `maintidx` maintainability index** (~5 issues)
  - **Description:** Improve worst-scoring functions (likely overlap with 6.1).
  - **DoR:** 6.3 complete
  - **DoD:** 0 maintidx issues
  - **`//nolint` prohibited** — improve function maintainability through refactoring

**Wave 6 Checkpoint:** ~22,800 cumulative issues resolved.

---

## Wave 7: Security & Context Propagation (est. ~300 issues)

- [ ] **7.1 — `gosec` security issues** (~256 issues)
  - **Description:** Fix potential security issues: weak crypto, hardcoded credentials, integer overflow, file permissions.
  - **DoR:** Wave 6 complete
  - **DoD:** 0 gosec issues
  - **Note:** G204 already excluded globally in `.golangci.yml` (subprocess with variable — needed for command execution)
  - **`//nolint` prohibited** — fix security issues in code; if a rule is project-wide inapplicable, disable in `.golangci.yml`

- [ ] **7.2 — `contextcheck` context propagation** (~14 issues)
  - **Description:** Pass `context.Context` through call chains instead of using `context.Background()`.
  - **DoR:** 7.1 complete
  - **DoD:** 0 contextcheck issues
  - **`//nolint` prohibited** — thread context through function signatures

- [ ] **7.3 — `containedctx` context in structs** (~3 issues)
  - **Description:** Remove `context.Context` from struct fields; pass via method parameters.
  - **DoR:** 7.2 complete
  - **DoD:** 0 containedctx issues
  - **Key targets:** `internal/mcp/registry.go:52`, `internal/tui/mapper.go:27`, `internal/ui/testkit/tuitest_helper.go:16`
  - **`//nolint` prohibited** — refactor structs to accept ctx as method parameter

- [ ] **7.4 — `noctx` HTTP without context** (~32 issues)
  - **Description:** Replace `http.Get`/`http.Post` with `http.NewRequestWithContext`.
  - **DoR:** 7.3 complete
  - **DoD:** 0 noctx issues
  - **`//nolint` prohibited** — use context-aware HTTP calls

**Wave 7 Checkpoint:** ~23,100 cumulative issues resolved.

---

## Wave 8: Naming, Constants & Remaining Style (est. ~1,500 issues)

- [ ] **8.1 — `mnd` magic numbers** (~478 issues)
  - **Description:** Extract magic numbers into named constants.
  - **DoR:** Wave 7 complete
  - **DoD:** 0 mnd issues
  - **`//nolint` prohibited** — define named constants for every magic number

- [ ] **8.2 — `goconst` repeated strings** (~57 issues)
  - **Description:** Extract repeated string literals into constants.
  - **DoR:** 8.1 complete
  - **DoD:** 0 goconst issues
  - **`//nolint` prohibited** — extract constants

- [ ] **8.3 — `varnamelen` variable name length** (~344 issues)
  - **Description:** Rename short-lived variables that are used far from declaration.
  - **DoR:** 8.2 complete
  - **DoD:** 0 varnamelen issues
  - **Note:** Add more entries to `ignore-names`/`ignore-decls` in `.golangci.yml` where short names are idiomatic Go
  - **`//nolint` prohibited** — rename variables or update `.golangci.yml` ignore lists

- [ ] **8.4 — `gocritic` code suggestions** (~387 issues)
  - **Description:** Apply gocritic suggestions: append simplification, redundant type conversions, etc.
  - **DoR:** 8.3 complete
  - **DoD:** 0 gocritic issues
  - **`//nolint` prohibited** — apply all suggestions

- [ ] **8.5 — `lll` long lines** (~143 issues)
  - **Description:** Break lines exceeding 140 chars.
  - **DoR:** 8.4 complete
  - **DoD:** 0 lll issues
  - **`//nolint` prohibited** — break long lines

- [ ] **8.6 — `predeclared` + `dogsled` + minor style** (~25 issues)
  - **Description:** Fix predeclared identifier shadowing, excessive blank identifiers.
  - **DoR:** 8.5 complete
  - **DoD:** 0 issues from these linters
  - **`//nolint` prohibited** — rename identifiers, reduce blank assignments

**Wave 8 Checkpoint:** ~24,600 cumulative issues resolved.

---

## Wave 9: Architecture & Interface Cleanup (est. ~200 issues)

- [ ] **9.1 — `ireturn` interface returns** (~69 issues)
  - **Description:** Return concrete types instead of interfaces where possible.
  - **DoR:** Wave 8 complete
  - **DoD:** 0 ireturn issues; `make test` passes
  - **`//nolint` prohibited** — change return types to concrete types

- [ ] **9.2 — `iface` interface hygiene** (~24 issues)
  - **Description:** Remove identical/unused/opaque interfaces.
  - **DoR:** 9.1 complete
  - **DoD:** 0 iface issues
  - **`//nolint` prohibited** — remove or consolidate interfaces

- [ ] **9.3 — `interfacebloat` large interfaces** (~6 issues)
  - **Description:** Split interfaces with >5 methods into focused role interfaces (ISP).
  - **DoR:** 9.2 complete
  - **DoD:** 0 interfacebloat issues
  - **`//nolint` prohibited** — decompose interfaces

- [ ] **9.4 — `dupl` code duplication** (~107 issues)
  - **Description:** Extract shared logic from duplicate code blocks (>100 token threshold).
  - **DoR:** 9.3 complete
  - **DoD:** 0 dupl issues; `make test` passes
  - **`//nolint` prohibited** — extract shared helpers, eliminate duplication

**Wave 9 Checkpoint:** ~24,800 cumulative issues resolved.

---

## Wave 10: Remaining & Final Pass (est. ~578 issues)

- [ ] **10.1 — Remaining small linters** (~200 combined issues)
  - **Description:** Fix remaining issues: `unconvert` (7), `ineffassign` (7), `wastedassign` (10), `forcetypeassert` (7), `reassign` (28), `exhaustive` (37), `unparam` (46), `usestdlibvars` (17), `inamedparam` (8), `tagliatelle` (15), `embeddedstructfieldcheck` (2), `errchkjson` (2), `errname` (1), `musttag` (2), `recvcheck` (2), `fatcontext` (1), `funcorder` (3), `godoclint` (45), `godox` (5), `gosmopolitan` (17), `mirror` (11), `nolintlint` (4), `prealloc` (5)
  - **DoR:** Waves 0-9 complete
  - **DoD:** Each sub-linter reports 0 issues
  - **`//nolint` prohibited** — fix every issue in code

- [ ] **10.2 — `gochecknoglobals` global variables** (~37 issues)
  - **Description:** Refactor globals into dependency injection or package-level funcs.
  - **DoR:** 10.1 complete
  - **DoD:** 0 gochecknoglobals issues
  - **`//nolint` prohibited** — eliminate globals through DI or functional patterns

- [ ] **10.3 — `gochecknoinits` init functions** (~4 issues)
  - **Description:** Replace `init()` functions with explicit initialization.
  - **DoR:** 10.2 complete
  - **DoD:** 0 gochecknoinits issues
  - **`//nolint` prohibited** — replace init() with explicit setup

- [ ] **10.4 — Final `make lint` green**
  - **Description:** Full `make lint` passes with zero issues and zero `//nolint` directives in codebase.
  - **DoR:** All items 0.1–10.3 complete
  - **DoD:** `make lint` exits 0; `make test` passes; `grep -r '//nolint' . --include='*.go' | wc -l` returns 0
  - **Verification:** `make lint && make test && ! grep -rq '//nolint' --include='*.go' . && echo "CLEAN"`

---

## Summary

| Wave | Focus | Issues | Cumulative |
|------|-------|--------|------------|
| 0 | Auto-fix formatting | ~8,300 | ~8,300 |
| 1 | Whitespace & style noise | ~5,900 | ~14,200 |
| 2 | Modern Go & hygiene | ~2,000 | ~16,200 |
| 3 | Logging | ~500 | ~16,700 |
| 4 | Error handling & correctness | ~2,500 | ~19,200 |
| 5 | Testing quality | ~3,100 | ~22,300 |
| 6 | Complexity reduction | ~500 | ~22,800 |
| 7 | Security & context | ~300 | ~23,100 |
| 8 | Naming & remaining style | ~1,500 | ~24,600 |
| 9 | Architecture & interfaces | ~200 | ~24,800 |
| 10 | Final pass | ~578 | ~25,378 |

## Principles

1. **`//nolint` is PROHIBITED** — every issue is fixed in code; if a rule is project-wide inapplicable, disable it in `.golangci.yml` settings/exclusions
2. **Each wave is testable** — run `make test` after every wave; run linter subset to verify zero issues for completed linters
3. **Progressive value** — Wave 0-1 eliminates >50% of all issues with zero logic changes
4. **Correctness before style** — Wave 4 (errors) and Wave 7 (security) are higher value than Wave 8 (naming), but blocked on formatting being stable first
5. **Safe auto-fix first** — Waves 0-1 are mostly `--fix` compatible; Waves 4+ require human judgment
6. **Test before parallelize** — Wave 5 (parallel tests) comes after correctness fixes to avoid masking race conditions

## Changelog

- **2026-03-12:** Initial roadmap created from `make lint` analysis (25,378 issues across ~80 linters)
