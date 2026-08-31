# JOURNEY-012-apply-compact-to-built-in-read-grep-glob-ls: Apply compact to built-in read, grep, glob, and ls

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: Apply compact to built-in read, grep, glob, and ls

## 1. Journey

When **the harness runs built-in `read_file`, directory listing, grep/search, or glob tools that never go through the Bash hook** I want **the same RTK filters (R8 / R2+R3 / R10) applied before those results become conversation history** so I **close the documented hole where savings vanish unless the agent uses shell, keep the default read level at `minimal`, and can restore raw output with `SPIN_COMPACT=0` or `compact.enabled: false`**.

## 2. CJM

Alex already has `Default()` filters (Step 10) and shell post-filter plus the env/config escape hatch (Step 11). Built-in `read_file`, `list_directory`, and `file_search` still dump raw bytes into the transcript. RTK’s hook never sees those tools. This journey applies the existing pipeline once on each built-in result — `read` levels, grep grouping + line truncation, tree compression for listings — and reuses `ShouldApply` so the same escape hatch disables them. No TUI chip (Step 13). Shell compact stays as shipped except shared helpers.

### Phase 1: Compact `read_file` with R8 levels

**User Intent:** Reading a source file returns the configured code-filter level, not the full commented body by default.

**Actions:** Read a fixture file that contains comments and a function body. Inspect the tool result. Pass `level=none` (or config `read_level`) when the full body is required.

**Pain / Risk:** Default `aggressive` strips a body the agent needed; compact never runs and comments still fill the window; `level` is ignored and config cannot request `none`; existing reads of comment-free fixtures gain a surprising trailing shape.

**Success Signal:** Default level is `minimal` (comments stripped, bodies kept). Tool arg or config can request `none` or `aggressive`. Output matches `Default().Apply("read -l …")`.

### Phase 2: Compact directory listing with R10

**User Intent:** `list_directory` observations are a hierarchy with per-dir counts, not one fat line per entry.

**Actions:** List a temp fixture with files and a subdirectory. Compare the result to `Default().Apply("ls", …)` on a path-per-line listing.

**Pain / Risk:** Tab-separated `name type bytes` lines are treated as single path names and the tree is garbage; empty-directory message is compacted into a fake filename; compact runs on the error path and hides `failed to read directory`.

**Success Signal:** Successful listings are tree-compressed (R10). Empty dirs still say they are empty. Errors stay errors.

### Phase 3: Compact grep/search (R2+R3) and glob (R10)

**User Intent:** Content search is grouped by file with long lines truncated. Glob / path search is tree-compressed like `ls`.

**Actions:** Grep a temp fixture with two files and one over-long match line. Glob/search the same fixture for paths. Inspect both observations.

**Pain / Risk:** Path search is forced through the grep parser and all hits drop; content grep never runs because there is no apply site; long lines stay unbounded; grouping order is unstable.

**Success Signal:** Grep/search output is grouped by file with line truncation (R2+R3). Glob / `file_search` paths go through tree compression (R10). Fixtures are temp/in-memory, never the user’s repo.

### Phase 4: Shared escape hatch

**User Intent:** Operators who already disable compact for shell get raw built-in output too.

**Actions:** Set `SPIN_COMPACT=0` or `SetCompactEnabled(false)` / `compact.enabled: false`. Repeat read, list, grep, and glob on the same fixtures.

**Pain / Risk:** Escape skips shell but not builtins; builtins default off so nobody gets savings; `level=none` is confused with the global hatch; tests read the real workspace.

**Success Signal:** Either escape skips filters on all four surfaces. Default compact stays on. Tests use `t.TempDir()` / in-memory blobs only.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| Built-in Read/Grep/Glob bypass the Bash hook | 1–3 | Same `Default().Apply` site as shell, keyed by `read` / `grep` / `ls` |
| Aggressive default strips needed bodies | 1 | Default `minimal`; tool arg or config can request `none` |
| Raw `ls` / glob dumps one line per file | 2 / 3 | R10 tree + counts |
| Long grep lines blow the window | 3 | Existing R2 grouping + R3 truncation |
| No way to see raw built-in output | 4 | Same `SPIN_COMPACT=0` / `compact.enabled: false` as Step 11 |

### North Star Summary

Alex runs the agent as today. Compact is on. Built-in read, list, grep/search, and glob observations are compact text — comments stripped at `minimal`, listings tree-compressed, grep grouped and truncated — without the model typing a wrapper. `SPIN_COMPACT=0` or `compact.enabled: false` restores raw output on these tools and on shell. The TUI savings chip stays out of this journey.

### Stressors

1. Fixture file with `//` comments — default read output is `minimal` (comments gone, function body kept).
2. Tool arg `level=none` — read output is the raw fixture bytes (R8 none).
3. Tool arg or config `level=aggressive` — only signatures remain.
4. `SPIN_COMPACT=0` on `read_file` — comments remain even at default level.
5. `compact.enabled: false` / `SetCompactEnabled(false)` on list, grep, and glob — raw listing/search text.
6. Directory fixture with files + one subdirectory — observation is R10 tree with a per-dir count, not `name\tfile\tN bytes`.
7. Empty directory — still a descriptive empty message, not a compacted fake path.
8. Grep fixture with two files and one line longer than the compact limit — grouped by file, long line truncated.
9. Glob / `file_search` path hits — tree-compressed; names from the temp fixture still present.
10. Filter panic or error (R12) — original built-in bytes returned; tool success flag unchanged.
11. Tests must not open the user’s real repo — only `t.TempDir()` / in-memory blobs.
12. Shell compact (Step 11) still applies and still honors the same escape hatch (no double-filter fork).

## 3. UX Implementation and Assessment

### Time to First Value
- [x] Default-on: first built-in read/list/search observation is already compacted
- [x] No new wrapper command for the model to remember

### Onboarding Clarity
- [x] Escape hatch names (`SPIN_COMPACT=0`, `compact.enabled`) match Step 11
- [x] Read `level` values are the spec’s `none|minimal|aggressive`

### Production-Ready Defaults
- [x] Read level defaults to `minimal` (bodies kept)
- [x] Compact defaults on; escape is opt-out

### Golden Path Quality
- [x] Comment fixture → minimal read
- [x] Listing fixture → R10 tree
- [x] Grep fixture → grouped + truncated

### Decision Load
- [x] No extra tool required for default compact
- [x] `level` is optional; omitted means config/default minimal

### Progressive Complexity
- [x] Simple case: default filters on built-in results
- [x] Advanced: `level=none` / `aggressive` or global escape

### Error Quality
- [x] Missing path / missing query still fail as today
- [x] Filter errors stay fail-safe raw (R12) via existing pipeline

### Failure Safety
- [x] Escape hatch restores raw built-in output
- [x] Empty directory is not rewritten into a tree node

### Runtime Transparency
- [x] Apply command names (`read -l …`, `ls`, `grep`) match the registry
- [x] Ledger/strategy stay on the pipeline result (chip is Step 13)

### Debuggability
- [x] `SPIN_COMPACT=0` is the raw-output probe for builtins too
- [x] Goldens remain the compact contract for R8/R2/R10

### Cross-Surface Consistency
- [x] Same `Default()` filters as Step 10
- [x] Same `ShouldApply` / env hatch as Step 11

### Workflow Consistency
- [x] Built-in `Execute` still returns `ToolResult`; compact is post-filter only
- [x] Shell apply site is unchanged except shared helpers

### Change Safety
- [x] Config `read_level` is additive; existing YAML without it stays valid
- [x] Constructor signatures for read/list/search stay compatible

### Experimentation Safety
- [x] Tests use temp fixtures, not the user’s repo
- [x] Escape hatch is revertible per process / config

### Interaction Latency
- [x] Apply is the existing in-memory pipeline (no extra model round-trip)
- [x] No network and no required `rtk` binary

### Developer Feedback Speed
- [x] Failed reads/lists still return errors immediately
- [x] Operator can disable compact without a new protocol

### Team Scale
- [x] `compact.enabled` / `read_level` live in versionable YAML
- [x] `SPIN_COMPACT=0` works uniformly for local debug

### System Scale
- [x] One apply site per built-in result
- [x] TUI chip left for Step 13 — no structural fork

### Right Behavior by Default
- [x] Compact on; read `minimal`; listings tree-compressed
- [x] Safe default over aggressive stripping

### Anti-Bypass Design
- [x] Built-ins cannot skip compact by avoiding shell
- [x] Escape hatch is explicit env/config, not a silent default-off

## 4. Tests

### TC-01: read_file default minimal

**Given** a temp file whose body includes `//` comments and a function.
**When** `read_file` runs with only `path`.
**Then** output equals `Default().Apply("read", raw)` / `read -l minimal` (comments stripped, body kept).

### TC-02: read_file level none

**Given** the same fixture and `level=none`.
**When** `read_file` runs.
**Then** output is the raw file bytes.

### TC-03: read_file escape hatch

**Given** `SPIN_COMPACT=0` or `SetCompactEnabled(false)`.
**When** `read_file` runs at default level.
**Then** output is raw (comments remain).

### TC-04: list_directory tree compression

**Given** a temp dir with two files and one subdirectory.
**When** `list_directory` runs.
**Then** output matches `Default().Apply("ls", path-per-line listing)` and contains the names plus a count label.

### TC-05: list_directory empty and errors

**Given** an empty temp dir, or a missing path.
**When** `list_directory` runs.
**Then** empty dirs still mention empty; errors still fail; neither is tree-rewritten into a fake file.

### TC-06: grep grouping and truncation

**Given** two temp files with matches, one line longer than the compact line limit.
**When** the grep/search tool runs.
**Then** output is grouped by file and the long line is truncated (R2+R3).

### TC-07: glob / file_search tree

**Given** a temp workspace with nested matching paths.
**When** `file_search` runs.
**Then** path hits are tree-compressed (R10) and fixture names still appear.

### TC-08: escape hatch on list, grep, glob

**Given** `SPIN_COMPACT=0` or `SetCompactEnabled(false)`.
**When** list, grep, and file_search run on the same fixtures.
**Then** output is the raw listing / raw grep lines / raw path list.

### TC-09: fixtures are isolated

**Given** the new compact tests.
**When** they execute.
**Then** they only touch `t.TempDir()` or in-memory blobs — never the user’s repo root.

### TC-10: config read level

**Given** `CompactV2.ReadLevel` set to `none` (or `aggressive`).
**When** `read_file` runs without a tool `level` arg.
**Then** the configured level is used; a tool arg still wins.

## 5. Acceptance Criteria

- `read_file` output uses code-filter levels from config (default `minimal`)
- Directory listing uses tree compression (`R10`)
- Grep/search grouping + line truncation apply
- Escape hatch disables compact on these tools too
- Tests use in-memory fixtures, not the user’s repo
- `make test` and `make lint` pass

## Traceability
- Roadmap item: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md) Step 12
- Implementation files: `internal/tools/read_file.go`, `internal/tools/list_directory.go`, `internal/tools/file_search.go`, `internal/tools/grep.go`, `internal/tools/compact_apply.go`, `internal/filesearch/grep.go`, `internal/config/config_v2.go`, `internal/conversation/tools.go`, `internal/agent/executor/builtin.go`
- Test files: `internal/tools/read_file_compact_test.go`, `internal/tools/list_directory_compact_test.go`, `internal/tools/file_search_compact_test.go`, `internal/tools/grep_compact_test.go`, `internal/filesearch/grep_test.go`

## Implementation

Files created:
- `specs/journeys/JOURNEY-012-apply-compact-to-built-in-read-grep-glob-ls.md` — this journey
- `internal/tools/compact_apply.go` — shared `ShouldApply` + `Default().Apply` helper and `ApplyCompactSettings`
- `internal/tools/read_file_compact_test.go` — default minimal, `level=none`, env/config escape, `SetReadLevel`
- `internal/tools/list_directory_compact_test.go` — R10 tree, escape hatch, temp fixture
- `internal/tools/file_search_compact_test.go` — glob R10, escape hatch
- `internal/tools/grep.go` — content-search tool that applies R2+R3
- `internal/tools/grep_compact_test.go` — grouping + truncation, escape hatch
- `internal/filesearch/grep.go` — fixture-safe `file:line:text` scan
- `internal/filesearch/grep_test.go` — temp-dir hits only

Files modified:
- `internal/tools/read_file.go` — R8 via `read -l`, optional `level`, `SetReadLevel`
- `internal/tools/list_directory.go` — path listing + `Apply("ls")`
- `internal/tools/file_search.go` — path listing + `Apply("ls")`
- `internal/tools/registry.go` — register `grep`
- `internal/tools/classification.go` — `grep` is CategorySearch
- `internal/config/config_v2.go` — `ReadLevel` default `minimal`
- `internal/config/loader_v2.go` — `compact.read_level` default/env
- `internal/conversation/tools.go` — `ApplyCompactSettings` from config
- `internal/agent/executor/builtin.go` — register grep + apply compact settings
- `specs/agent-harness/ROADMAP.md` — Step 12 DoD ticks and traceability
- `docs/testing.md` — journey 012 test row
