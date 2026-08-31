# JOURNEY-013-compact-status-chip-and-operator-escape: Compact status chip and operator escape

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: Compact status chip and operator escape

## 1. Journey

When **Alex runs spin in the TUI after tool-output compact is already on by default** I want **the status bar to show whether compact is on or off and the last-turn output-bytes reduction from the ledger** so I **can trust that compact is actually running, see savings as output-bytes reduction (not bill/token billing), and disable it with a documented env/config escape**.

## 2. CJM

Alex already has the RTK pipeline, shell apply, built-in apply, and `SPIN_COMPACT=0` / `compact.enabled: false` (Steps 9–12). None of that is visible on the TUI. The status bar shows activity, context %, and model — not compact state. `/help` documents task **mode** `compact` (a 4K token budget) and can be confused with tool-output compact. Welcome text never mentions the escape hatch. This journey adds a status-bar on/off marker plus a savings chip (`−N%` from ledger `BytesIn`/`BytesOut`), documents the operator escape in `/help`, and makes welcome/status refuse to claim compact is on when it is disabled. Default compact stays **on**. No TaskFrame (Step 14).

### Phase 1: See compact state on the status bar

**User Intent:** Know at a glance whether tool-output compact is active in this session.

**Actions:** Start TUI with default config. Read the status bar. Restart with `SPIN_COMPACT=0` or `compact.enabled: false`. Read the status bar again.

**Pain / Risk:** Status claims `on` after the escape hatch; `off` is omitted so the bar looks the same either way; compact **mode** (`/mode compact`) is shown as if it were tool-output compact; a zero-value Metrics field defaults to claiming on.

**Success Signal:** Status shows `on` when `CompactV2.Active()` is true and `off` when it is not. Welcome text matches that state and never claims on when disabled.

### Phase 2: Read last-turn savings as output-bytes reduction

**User Intent:** See how much the last compacted turn reduced output bytes, without thinking the number is a billing or tokenizer saving.

**Actions:** Run a turn that compactes tool output. Read the savings chip. Compare the percent to ledger `BytesIn`/`BytesOut` (or `ReductionPct` from those bytes).

**Pain / Risk:** Chip uses a real tokenizer and disagrees with R15; chip implies token/bill savings; percent lacks the minus sign from the spec example `−N%`; 0% / 100% / mid reduction format incorrectly; chip appears when compact is off.

**Success Signal:** Chip is `−N%` from ledger bytes (optionally `·` output size as in `−72% · 14kB`). Zero, mid, and 100% reduction have unit tests. Label is output-bytes reduction.

### Phase 3: Discover and use the operator escape

**User Intent:** Turn compact off without hunting the spec, then confirm the UI agrees.

**Actions:** Run `/help`. Set `SPIN_COMPACT=0` or `compact.enabled: false`. Restart. Confirm raw tool output and status/welcome say off.

**Pain / Risk:** `/help` only mentions task mode `compact`; escape names drift from config (`compact.enabled`); welcome still says compact is on; default flips to off.

**Success Signal:** `/help` documents `SPIN_COMPACT=0` and the config key. Default remains on. Disabled sessions do not claim on.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| Compact runs silently; operator cannot tell if it is on | 1 | Status bar `on`/`off` from `CompactV2.Active()` |
| Savings are invisible; tokenizer math would lie | 2 | Chip from ledger `BytesIn`/`BytesOut` only |
| `/help` `compact` means a task mode | 3 | Separate “tool output compact” section + escape names |
| Welcome/status can claim on after disable | 1 / 3 | Same `Active()` bit for welcome and status |
| Chip could be read as money/token billing | 2 | Copy is output-bytes reduction (RTK disclaimer) |

### North Star Summary

Alex opens the TUI and the status bar shows compact **on** plus a last-turn `−N%` chip computed from ledger bytes. `/help` names `SPIN_COMPACT=0` and `compact.enabled: false`. After either escape, welcome and status say **off** and never claim compact is on. Default compact stays on. The chip does not imply bill or tokenizer savings.

### Stressors

1. Default config — status and welcome show compact **on**; `compact.enabled` stays true.
2. `SPIN_COMPACT=0` — status shows **off**; welcome does not claim on.
3. `compact.enabled: false` — same off claim as the env hatch.
4. Ledger identity (BytesIn == BytesOut) — chip is `−0%`.
5. Mid reduction (e.g. 1000→280 bytes) — chip is `−72%` from those bytes, not `ceil(bytes/4)` tokens.
6. 100% reduction (BytesOut == 0, BytesIn > 0) — chip is `−100%`.
7. Compact disabled with leftover ledger bytes — status is `off`, no `on −N%`.
8. `/help` still lists task mode `compact` and also documents the tool-output escape (`SPIN_COMPACT=0`, `compact.enabled`).
9. Narrow status (`DetailCompact`) still shows on/off; chip percent uses Unicode minus `−`.
10. Last-turn update: tool-complete metadata with ledger bytes replaces the previous chip; empty/missing metadata leaves the last shown savings.
11. Welcome footer with compact off must not contain `compact: on`.
12. Renderer high-usage `N%` coloring must not rewrite the compact `−N%` chip as the context percent.

## 3. UX Implementation and Assessment

### Time to First Value
- [x] Status bar shows compact on/off on first TUI paint (no extra command)
- [x] Last-turn chip appears after a compacted tool result without a tokenizer

### Onboarding Clarity
- [x] `/help` documents `SPIN_COMPACT=0` and `compact.enabled`
- [x] Welcome names the same escape when compact is on; says off when disabled

### Production-Ready Defaults
- [x] Compact remains on by default
- [x] Metrics do not claim on until TUI wires `CompactV2.Active()`

### Golden Path Quality
- [x] Enabled session: status `on` + `−N%` from ledger bytes
- [x] Disabled session: status `off`; welcome does not claim on

### Decision Load
- [x] No new slash command required to see state
- [x] Escape is env or one config key

### Progressive Complexity
- [x] Simple case: default on + chip
- [x] Advanced: optional `·` output-bytes size matching the spec example

### Error Quality
- [x] Zero-byte ledger formats as `−0%`, not NaN or blank
- [x] Missing metadata does not crash the aggregator

### Failure Safety
- [x] Escape hatch restores raw output (Steps 11–12) and the UI agrees
- [x] Disabled compact never shows `on`

### Runtime Transparency
- [x] Chip is last-turn output-bytes reduction from the ledger
- [x] On/off tracks `CompactV2.Active()` (`SPIN_COMPACT` and `enabled`)

### Debuggability
- [x] `/help` is the operator probe for escape names
- [x] Formatter tests pin 0% / mid / 100%

### Cross-Surface Consistency
- [x] Same `Active()` used by shell/builtin compact and the TUI chip
- [x] Terminology: “tool output compact” vs task mode `compact`

### Workflow Consistency
- [x] Chip goes through existing `FormatMetrics` / status manager
- [x] Last-turn bytes arrive via tool-complete metadata the aggregator already sees

### Change Safety
- [x] Existing status fields (context %, YOLO, provider) stay
- [x] Task mode `compact` label is unchanged

### Experimentation Safety
- [x] `SPIN_COMPACT=0` is reversible by unset
- [x] Config `compact.enabled: false` is reversible

### Interaction Latency
- [x] Chip is a format of already-held metrics (no extra IO)
- [x] Status refresh uses the existing tool-complete path

### Developer Feedback Speed
- [x] Formatter unit tests fail on wrong percent or missing minus
- [x] Help test fails if escape names disappear

### Team Scale
- [x] Escape names match config YAML already in-repo
- [x] Journey/roadmap name the same keys

### System Scale
- [x] Chip does not add a tokenizer dependency
- [x] Ledger percent stays `ceil`-free (bytes ratio only)

### Right Behavior by Default
- [x] Compact on; chip shows on
- [x] Disabled sessions do not claim on

### Anti-Bypass Design
- [x] UI cannot show on when `Active()` is false
- [x] Chip cannot be fed tokenizer counts in place of ledger bytes

## 4. Tests

### TC-01: formatter zero reduction

**Given** compact enabled and ledger BytesIn == BytesOut (including both zero).
**When** the status formatter runs.
**Then** the chip contains `−0%` and does not use a tokenizer.

### TC-02: formatter mid reduction

**Given** compact enabled and ledger bytes 1000 in / 280 out.
**When** the status formatter runs.
**Then** the chip contains `−72%` (output-bytes reduction).

### TC-03: formatter 100% reduction

**Given** compact enabled, BytesIn > 0, BytesOut == 0.
**When** the status formatter runs.
**Then** the chip contains `−100%`.

### TC-04: formatter compact off

**Given** compact disabled, even if leftover BytesIn/BytesOut would reduce.
**When** the status formatter runs.
**Then** the bar contains `off` and does not contain `on`.

### TC-05: help documents escape

**Given** `/help` is executed.
**When** the operator reads the output.
**Then** it contains `SPIN_COMPACT=0` and `compact.enabled`.

### TC-06: welcome disabled does not claim on

**Given** `CompactV2.Active()` is false.
**When** welcome is printed.
**Then** the footer does not claim compact is on.

### TC-07: welcome/status default on

**Given** default config (compact enabled, env unset).
**When** TUI welcome and status are initialized.
**Then** both show compact on.

### TC-08: last-turn from ledger metadata

**Given** a tool-complete event whose metadata carries ledger BytesIn/BytesOut.
**When** the status aggregator processes it.
**Then** the next format uses those bytes for `−N%`.

### TC-09: manager setters

**Given** a status manager.
**When** `SetCompactEnabled` / `SetCompactSavings` run.
**Then** `GetMetrics` reflects enabled and the byte pair.

### TC-10: help still lists task mode compact

**Given** `/help`.
**When** executed.
**Then** task mode `compact` remains documented and the tool-output section is distinct.

## 5. Acceptance Criteria

- Status bar shows compact on/off and a savings chip (`−N%` from the ledger)
- Chip uses ledger bytes, not a tokenizer
- `/help` documents `SPIN_COMPACT=0` and config key
- Welcome/status does not claim compact is on when disabled
- Unit tests for formatter with zero, mid, and 100% reduction
- `make test` and `make lint` pass

## Traceability
- Roadmap item: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md) Step 13
- Implementation files: `internal/ui/status/formatter.go`, `internal/ui/status/aggregator.go`, `internal/ui/status/manager.go`, `cmd/spin/tui.go`, `internal/commands/help.go`, `internal/ui/adapters/puretty.go`, `internal/contexteng/compact/ledger.go`, `internal/tools/shell_command.go`, `internal/tools/compact_apply.go`
- Test files: `internal/ui/status/formatter_test.go`, `internal/ui/status/aggregator_test.go`, `internal/ui/status/manager_test.go`, `internal/commands/commands_test.go`, `cmd/spin/tui_welcome_test.go`, `internal/tools/shell_command_compact_test.go`, `internal/contexteng/compact/pipeline_test.go`

## Implementation

Files created:
- `specs/journeys/JOURNEY-013-compact-status-chip-and-operator-escape.md` — this journey
- `internal/commands/help.go` — `/help` tool-output compact section (`SPIN_COMPACT=0`, `compact.enabled`)

Files modified:
- `internal/ui/status/formatter.go` — `FormatCompactChip` (`on`/`off`, `−N%` from ledger bytes, optional `·` size)
- `internal/ui/status/formatter_test.go` — zero / mid / 100% / off / SPEC `14kB`
- `internal/ui/status/manager.go` — `CompactEnabled`, `CompactBytesIn`/`Out`, setters
- `internal/ui/status/manager_test.go` — unset does not claim on; savings pair
- `internal/ui/status/aggregator.go` — last-turn ledger bytes from tool-complete metadata
- `internal/ui/status/aggregator_test.go` — metadata 1000/280
- `internal/contexteng/compact/ledger.go` — `ByteReductionPct`, `MetaBytesIn`/`Out`
- `internal/contexteng/compact/pipeline_test.go` — `ByteReductionPct` matches ledger bytes
- `cmd/spin/tui.go` — welcome line + `initializeUI` wires `cfg.Compact.Active()`
- `cmd/spin/tui_welcome_test.go` — disabled welcome does not claim on
- `internal/ui/adapters/puretty.go` — `SetCompactEnabled`
- `internal/commands/commands.go` — append compact help
- `internal/commands/commands_test.go` — help contains escape names
- `internal/tools/shell_command.go` — attach ledger bytes to result metadata
- `internal/tools/shell_command_compact_test.go` — metadata equals ledger
- `internal/tools/compact_apply.go` — builtin apply returns result with ledger metadata
- `internal/tools/read_file.go`, `internal/tools/list_directory.go`, `internal/tools/file_search.go`, `internal/tools/grep.go` — use `applyBuiltinCompact` result
- `docs/testing.md` — journey 013 row
- `specs/agent-harness/ROADMAP.md` — Step 13 DoD and traceability
