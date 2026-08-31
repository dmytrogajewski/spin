# JOURNEY-010-compact-command-registry: Compact command registry (RTK table, 1-1)

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: Compact command registry (RTK table, 1-1)

## 1. Journey

When **the harness has a known tool or shell result (ls, read, grep, git, tests, ruff, docker)** I want **a default filter registry that maps that command to an RTK compact form** so I **keep exit codes, drop boilerplate the model does not need, and can prove each table row with a golden fixture that never talks to the network or real git**.

## 2. CJM

Alex has the Step 9 pipeline: `Apply` is fail-safe, never rewrites exit codes, unknown commands passthrough, and the ledger is `ceil(bytes/4)`. Production `New()` still has no command filters. The spec table and R1–R11 are the missing default registry. This journey registers those filters behind `Default()` (or an equivalent registry constructor), keeps `New()` empty so Step 9 identity tests stay valid, and treats goldens as the contract — hand-authored from the spec table, not captured from a live `rtk` binary. Auto-rewrite (R11) is a named stub only; shell and built-in wiring stay later steps.

### Phase 1: Resolve command to a table row

**User Intent:** Feed a command string plus stdio into the same `Apply` and have a known command hit its filter.

**Actions:** Construct the default registry pipeline. Call `Apply` with table commands (`ls`, `git status`, `go test`, `read -l minimal`, …) and with an unknown command.

**Pain / Risk:** Exact-key lookup misses `go test -json` and `read -l aggressive`; `git` matches every git subcommand and applies the wrong filter; `New()` silently gains filters and breaks Step 9 identity assumptions; R11 is wired into shell tools in this step.

**Success Signal:** Known table keys (and prefix+space argv) select the right filter. Unknown commands stay R14. `New()` remains empty. R11 is a named stub, not a PreToolUse hook.

### Phase 2: Compact stdout per the spec table

**User Intent:** Each table row has a stable compact form that a golden can lock.

**Actions:** Apply fixtures: raw stdin-like bytes in, compact stdout out. Inspect grouping, truncation, counts, signatures, failure focus, and `×N` dedup.

**Pain / Risk:** Goldens drift toward a live `rtk` binary; `go test` NDJSON drops failure output; `git status` is a flat list instead of state groups; `read` ignores `-l`; grep does not group by file; repeated log lines are not collapsed.

**Success Signal:** Each table row has `testdata/<row>/raw` → `compact` with the same exit code. DoD-specific shapes hold (R2–R10, R4, R8, R9).

### Phase 3: Preserve R12–R15 contracts

**User Intent:** Adding filters must not weaken fail-safe, exit codes, or unknown passthrough.

**Actions:** Run Step 9 tests unchanged. Apply a registered command with a nonzero exit. Apply an unknown command on `Default()`. Confirm filters never import `os/exec`, `net/http`, or invoke git.

**Pain / Risk:** A filter error escapes instead of R12; exit 1 from `go test` becomes 0; unknown commands on `Default()` are invented-filtered; tests shell out to git or the network to build fixtures.

**Success Signal:** Exit codes are copied. Filter errors still fail-safe. Unknown cmds still passthrough. Filter tests are fixture-only.

### Phase 4: Prove goldens without upstream RTK

**User Intent:** Reviewers can see why a fixture looks the way it does.

**Actions:** Read `testdata/README.md`. Each fixture names the spec table row (and RTK id) it encodes.

**Pain / Risk:** README omitted; fixtures look captured from `rtk` and bit-rot when upstream changes; a missing exit file lets tests assume 0 only.

**Success Signal:** README documents that fixtures are hand-authored from the spec table, not a live `rtk` binary. Every golden directory has raw, compact, cmd, and exit.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| `New()` is identity-only; agents still see full `git status` / `go test` blobs | 1 | `Default()` registers the spec table |
| Command strings include flags (`-json`, `-l aggressive`) | 1 | Longest prefix match on `key` + space |
| Live `rtk` goldens drift | 2 / 4 | Hand-authored fixtures + README source note |
| Compact that swallows exit codes breaks CI agents | 3 | Exit code remains an input copy |
| Tests that run git or the network are not hermetic | 3 | Byte-slice filters only |

### North Star Summary

Alex calls `Default().Apply` with a table command and gets the compact form from the spec: trees with counts, read levels, grouped grep, git grouped by state, NDJSON failures-only, `×N` dedup. The same exit code comes back. An unknown command is still raw + 0% ledger. A broken filter is still R12. Nothing in this step rewrites shell argv or paints a TUI chip.

### Stressors

1. Unknown command on `Default()` — bytes unchanged, strategy `R14`, 0% reduction.
2. `New()` stays empty — `ls` on `New()` is still passthrough (R14), not the tree filter.
3. Nonzero exit (for example `go test` fixture exit 1) is preserved after compact.
4. `go test` NDJSON — passing tests collapse to a count; failure names and output stay (R9).
5. `git status` porcelain — entries grouped by state with counts (R5).
6. `ls` / `tree` — hierarchy plus per-directory file counts (R10).
7. `read -l none|minimal|aggressive` — passthrough, comments stripped, signatures only (R8).
8. `grep` / `rg` — lines truncated and grouped by file (R2+R3).
9. Repeated consecutive log lines collapse to `line ×N` (R4).
10. Filter tests do not import or call `os/exec`, `net/http`, or a real git binary.
11. Prefix argv (`go test -json`, `git status --porcelain`) selects the same filter as the table key.
12. Empty stdout for `git add` still yields one confirmation line, not a panic.
13. Malformed NDJSON / non-JSON `go test` text does not panic; fail-safe or a conservative text fallback keeps the exit code.
14. R11 is a named registry stub only — no PreToolUse / shell rewrite in this step.
15. Filter error or panic on a registered command still returns original stdio and `R12`.

## 3. UX Implementation and Assessment

The operator-facing surface is still not a flag or chip. Value is that later shell (Step 11) and built-in tools (Step 12) can call `Default().Apply` and get table-accurate compact stdout.

### Time to First Value
- [ ] `Default().Apply("ls", raw, nil, 0)` returns a tree with counts and no config
- [ ] Goldens show the compact form without running the named binaries

### Onboarding Clarity
- [ ] Registry keys match the spec per-command table
- [ ] `testdata/README.md` states fixtures are spec-authored, not live `rtk`

### Production-Ready Defaults
- [ ] `Default()` registers the table; `New()` stays empty
- [ ] `read` without `-l` defaults to `minimal`
- [ ] Unknown commands remain passthrough

### Golden Path Quality
- [ ] Each spec table row has a golden: raw → compact, same exit code
- [ ] DoD shapes for go test, git status, ls/tree, read levels, grep, dedup hold

### Decision Load
- [ ] Callers pick `New()` vs `Default()`, not a per-command strategy enum
- [ ] Read level is taken from the command string (`-l` / `--level`)

### Progressive Complexity
- [ ] Identity path unchanged for `New()` and unknown cmds
- [ ] R11 rewrite is a named stub, not in the way

### Error Quality
- [ ] Filter panic/error still becomes raw output + `R12`
- [ ] Malformed NDJSON does not crash `Apply`

### Failure Safety
- [ ] Exit code is never derived from compact success
- [ ] Filters operate on copies / return new slices; fail-safe still has originals

### Runtime Transparency
- [ ] Ledger still records bytes in/out on compacted results
- [ ] Unknown vs registered is observable (R14 vs nonempty compact / empty strategy)

### Debuggability
- [ ] Goldens are diffable text files next to the package
- [ ] README maps each fixture directory to a spec row and RTK id

### Cross-Surface Consistency
- [ ] Same `Filter` / `Apply` contract as Step 9
- [ ] Command names in the registry match the spec table spelling

### Workflow Consistency
- [ ] Package stays `internal/contexteng/compact`
- [ ] Journey and tests follow JOURNEY-00N + `// Journey:` comments

### Change Safety
- [ ] No shell tool, read tool, or TUI chip is wired
- [ ] Step 9 `pipeline.go` / `ledger.go` contracts are extended, not replaced

### Experimentation Safety
- [ ] Fixtures are static; reruns do not need git or network
- [ ] Goldens can be updated without an `rtk` binary

### Interaction Latency
- [ ] Filters are pure byte transforms (no subprocess)
- [ ] Compact p99 budget from Step 9 remains a package invariant for unknown cmds

### Developer Feedback Speed
- [ ] A mismatched golden names the fixture directory
- [ ] Exit-code mismatch is a dedicated assertion

### Team Scale
- [ ] Adding a table row is a filter + a testdata directory
- [ ] README is the fixture source of truth for reviewers

### System Scale
- [ ] Prefix match lets flags grow without new registry keys
- [ ] Shared helpers (failures-only, confirm line, dedup) keep cyclomatic cost down

### Right Behavior by Default
- [ ] `Default()` is the spec table
- [ ] Safe defaults: read `minimal`; unknown passthrough; fail-safe on errors

### Anti-Bypass Design
- [ ] There is no API to drop the exit code field
- [ ] Tests that would call git or the network are rejected by construction (no such imports)

## 4. Tests

### TC-01: golden_each_table_row

**Given** a testdata directory per spec table row with `cmd`, `exit`, `raw`, and `compact`.
**When** `Default().Apply` runs on that cmd and raw stdout with that exit.
**Then** stdout equals `compact` and `Result.ExitCode` equals the fixture exit.

### TC-02: gotest_ndjson_failures_kept

**Given** NDJSON with passing and failing tests.
**When** the `go test` filter runs.
**Then** passing tests collapse to a count and failure names plus output remain (R9).

### TC-03: git_status_grouped_by_state

**Given** porcelain `git status` stdout.
**When** the `git status` filter runs.
**Then** paths are grouped by state with counts (R5).

### TC-04: ls_tree_hierarchy_counts

**Given** `ls` paths and `tree` box-drawing stdout that encode the same tree.
**When** those filters run.
**Then** output is a hierarchy with per-directory file counts (R10).

### TC-05: read_levels

**Given** the same source fixture and cmds `read -l none`, `read -l minimal`, `read -l aggressive`.
**When** each apply runs.
**Then** none is identity, minimal strips comments, aggressive keeps signatures (R8).

### TC-06: grep_rg_grouped_truncated

**Given** grep/rg-style `file:line:text` lines including one over the truncate limit.
**When** those filters run.
**Then** lines are grouped by file and long text is truncated (R2+R3).

### TC-07: dedup_times_n

**Given** consecutive repeated log lines.
**When** the dedup/log filter runs.
**Then** repeats collapse to `line ×N` (R4).

### TC-08: filters_are_hermetic

**Given** the compact package and its tests.
**When** imports and testdata are inspected.
**Then** there is no `os/exec`, `net/http`, or real git invocation.

### TC-09: default_unknown_passthrough

**Given** `Default()` and an unregistered command.
**When** `Apply` runs.
**Then** bytes are unchanged, strategy is `R14`, exit code is unchanged.

### TC-10: new_stays_empty

**Given** `New()` and cmd `ls` with listing stdout.
**When** `Apply` runs.
**Then** stdout is unchanged (no tree filter).

### TC-11: prefix_argv_selects_filter

**Given** `Default()` and cmd `git status --porcelain` (or `go test -json`).
**When** `Apply` runs on the matching raw fixture.
**Then** the same compact form as the table key is produced.

### TC-12: r11_is_stub

**Given** the registry and `StrategyR11`.
**When** production filters are listed.
**Then** R11 is named and no shell/PreToolUse rewrite is registered.

## 5. Acceptance Criteria

Verbatim Definition of Done from Step 10:

- Each table row has a golden fixture: raw stdin-like output → compact stdout, same exit code
- `go test` NDJSON: passing tests collapsed, failures kept (`R9`)
- `git status`: grouped by state (`R5`)
- `ls`/`tree`: hierarchy + per-dir counts (`R10`)
- `read` levels `none|minimal|aggressive` (`R8`)
- `grep`/`rg`: truncated lines, grouped by file (`R2`+`R3`)
- Dedup collapses repeated log lines with `×N` (`R4`)
- Filter tests do not call the network or real git
- `make test` and `make lint` pass

## Traceability
- Roadmap item: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md) Step 10
- Implementation files: `internal/contexteng/compact/registry.go`, `internal/contexteng/compact/git.go`, `internal/contexteng/compact/gotest.go`, `internal/contexteng/compact/tree.go`, `internal/contexteng/compact/read.go`, `internal/contexteng/compact/grep.go`, `internal/contexteng/compact/ruff.go`, `internal/contexteng/compact/docker.go`, `internal/contexteng/compact/dedup.go`, `internal/contexteng/compact/json.go`, `internal/contexteng/compact/text.go`, `internal/contexteng/compact/pipeline.go`
- Test files: `internal/contexteng/compact/registry_test.go`, `internal/contexteng/compact/testdata/`

## Implementation

Files created:
- `specs/journeys/JOURNEY-010-compact-command-registry.md` — this journey
- `internal/contexteng/compact/registry.go` — `Default()` spec-table registry
- `internal/contexteng/compact/registry_test.go` — goldens, prefix argv, hermetic imports, R11 stub
- `internal/contexteng/compact/text.go` — shared line encode/decode and grouped render
- `internal/contexteng/compact/tree.go` — `ls` / `tree` / `find` (R10)
- `internal/contexteng/compact/read.go` — `cat` / `read` / `smart` levels (R8/R1)
- `internal/contexteng/compact/grep.go` — `grep` / `rg` group + truncate (R2+R3)
- `internal/contexteng/compact/git.go` — status/diff/log/confirm
- `internal/contexteng/compact/gotest.go` — `go test` NDJSON (R9) and failures-only runners
- `internal/contexteng/compact/ruff.go` — `ruff check` grouped by rule
- `internal/contexteng/compact/docker.go` — `docker ps` essential fields
- `internal/contexteng/compact/dedup.go` — consecutive `×N` (R4)
- `internal/contexteng/compact/json.go` — structure-only JSON (R7)
- `internal/contexteng/compact/testdata/` — per-row goldens and README (spec-authored, not live `rtk`)

Files modified:
- `internal/contexteng/compact/pipeline.go` — longest prefix match; `StrategyR11` stub name
- `specs/agent-harness/ROADMAP.md` — Step 10 DoD ticks and traceability
- `docs/testing.md` — journey 010 test row
