# JOURNEY-014-taskframe-on-every-parent-turn: TaskFrame on every parent turn

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: TaskFrame on every parent turn

## 1. Journey

When **Alex runs a parent spin turn (TUI, exec, or ACP) after choosing a task mode** I want **every parent turn to carry a compact TaskFrame (objective, phase, output_format, tools, sources, boundaries, success_criteria)** so I **can keep the model on a phase-specific contract without stuffing AGENTS.md or file bodies into the cacheable prompt, and so a later A2A child can be spawned from the same stable serialization**.

## 2. CJM

Alex already has Composer (stable vs dynamic) and `/mode` (`regular`, `review`, `compact`, `planning`). There is no per-turn TaskFrame object. The system prompt is a bag of sections; mode only changes session state and token/tool budgets. Children (Step 16+) must not clone the parent transcript — they need a frame. This journey adds `internal/agent/frame.TaskFrame`, injects it as a **dynamic** Composer section on the parent compose path, maps `/mode` 1:1 onto `phase`, and keeps `sources` as paths or retrieval queries. It does **not** call `retrieval.Assemble` (Step 15) and does **not** spawn A2A (Step 16).

Assumption (SPEC vs DoD): SPEC.md lists phases `plan | work | review | ask`. Step 14 DoD requires `/mode review` → phase `review` and likewise for the other modes. This journey maps mode names onto matching phase strings: `regular`, `review`, `compact`, `planning`.

### Phase 1: Name the turn contract

**User Intent:** Know that the current parent turn has an explicit, inspectable frame (objective, phase, format, tools, sources, boundaries, success criteria).

**Actions:** Start a parent session. Inspect the composed system prompt (or a unit-level compose). Read the TaskFrame section.

**Pain / Risk:** Frame is missing on ACP or exec; frame is buried in the cacheable prefix so prompt cache breaks; fields are empty with no heading so the model cannot see a contract; frame copies RegularSystemPrompt and wastes the window.

**Success Signal:** Composer dynamic part contains a TaskFrame section. Stable/cacheable part does not. Section is short and named.

### Phase 2: Switch mode, switch phase

**User Intent:** `/mode review` (and the other three modes) change the frame’s phase to match.

**Actions:** Run `/mode review`. Inspect `Conversation.CurrentFrame()` (and the mode→phase mapper). Repeat for `regular`, `compact`, `planning`.

**Pain / Risk:** Review maps to SPEC `ask` or `work`; only CLI `spin mode` maps and `/mode` does not; empty mode panics; unknown mode invents a phase.

**Success Signal:** `/mode review` yields phase `review`. The other modes yield their own names. Empty/unknown mode falls back to `regular`.

### Phase 3: Point at sources, never paste bodies

**User Intent:** Tell the turn which files or queries matter without dumping file contents into the frame (and without duplicating AGENTS.md).

**Actions:** Attach `AGENTS.md` (or another fixture path) as a source. Serialize the frame. Compare against the file body.

**Pain / Risk:** `ReadFile` of a source lands in JSON; AGENTS.md body is copied into `objective`; a multi-line paste is accepted as a “source”; rendered size grows without a cap.

**Success Signal:** Serialized frame contains the path or query string only. Fixture body sentinels are absent. Rendered size stays under the fixture cap.

### Phase 4: Freeze a spawn-ready encoding

**User Intent:** The same frame bytes can later be handed to a child (not implemented here) without the encoding drifting.

**Actions:** Marshal the same fixture twice. Compare bytes. Confirm JSON keys are the seven spec fields.

**Pain / Risk:** `map` iteration shuffles keys; `null` vs `[]` flips; HTML escaping changes `&`; text render and JSON disagree.

**Success Signal:** `MarshalStable` is deterministic for a fixture. JSON/text form includes all seven fields. No spawn, no A2A types.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| Parent prompt has no per-turn contract | 1 | Dynamic TaskFrame section on compose |
| `/mode` only changes budgets, not phase language | 2 | 1:1 mode → phase mapping |
| AGENTS.md / file bodies explode tokens | 3 | Sources are paths/queries; size cap on fixture |
| Child spawn would need an ad-hoc blob | 4 | Stable JSON/text now; spawn later |
| Cacheable prefix would churn every turn | 1 | `Cacheable: false` only |

### North Star Summary

Every parent turn’s Composer dynamic suffix includes a compact TaskFrame whose `phase` matches `/mode`, whose `sources` are paths or queries, and whose JSON/text form is stable enough to become a later A2A spawn payload. The cacheable prefix is unchanged. AGENTS.md stays in its own project-instructions section and is never copied into the frame.

### Stressors

1. `/mode review` must yield phase `review`, not SPEC `ask` or `work`.
2. `/mode regular`, `compact`, and `planning` must yield matching phase strings.
3. Empty mode (ValidateMode allows it) must not panic; phase falls back to `regular`.
4. Unknown mode must not invent a free-form phase; fallback is `regular`.
5. Sources that contain newlines (file bodies) must not appear in the serialized frame.
6. A fixture AGENTS.md body sentinel must appear in project-instructions if loaded, but never inside the TaskFrame section.
7. Rendered frame on a fixture must stay ≤ `MaxRenderedBytes` (frame must not duplicate AGENTS.md).
8. `ComposeTwoPart` stable part must not contain the frame heading or `phase` JSON.
9. Two `MarshalStable` calls on the same fixture must be byte-identical (no map-key shuffle, no `null`/empty flip).
10. Apply path must run for TUI/exec compose **and** ACP compose, or ACP parent turns silently lack a frame.
11. `DefaultRegularSections` must still match `RegularSystemPrompt` (frame is an extra section, like the skill catalog).
12. Nil composer on `ApplyTaskFrame` must be a no-op (no panic).
13. Step 15 `retrieval.Assemble` must not be called from this journey.
14. Step 16 A2A types / child spawn must not start from this journey.

## 3. UX Implementation and Assessment

### Time to First Value
- [x] Parent compose includes a TaskFrame section without a new slash command
- [x] `/mode` already exists; phase mapping needs no extra onboarding step

### Onboarding Clarity
- [x] `spin mode describe` names the TaskFrame phase for that mode
- [x] Invalid `/mode` still uses the existing error path

### Production-Ready Defaults
- [x] Default mode `regular` yields phase `regular`
- [x] Empty sources/tools/boundaries serialize as empty arrays, not omitted maps

### Golden Path Quality
- [x] Dynamic compose contains the frame; stable compose does not
- [x] Mode mapping table covers all four modes

### Decision Load
- [x] User still chooses one of four modes; frame fields default from the mode
- [x] No extra flags to enable the frame

### Progressive Complexity
- [x] Parent defaults stay short (no required source list)
- [x] Sources are opt-in paths/queries

### Error Quality
- [x] Unknown mode stays on the existing `/mode` error
- [x] Body-like sources are dropped, not stored

### Failure Safety
- [x] Nil composer is a no-op
- [x] Marshal errors wrap; Render does not panic

### Runtime Transparency
- [x] Frame JSON is visible in the dynamic prompt
- [x] Phase is derived from the session mode, not hidden config

### Debuggability
- [x] Stable JSON can be compared byte-for-byte in tests
- [x] Frame section heading `# Task Frame` is greppable in compose output

### Cross-Surface Consistency
- [x] TUI/exec `composeSystemPrompt` and ACP compose both apply the frame
- [x] Phase names match `/mode` values

### Workflow Consistency
- [x] Apply helper matches `ApplyCatalog` (extra section, not inside `DefaultRegularSections`)
- [x] Journey comment on new tests points at this file

### Change Safety
- [x] `RegularSystemPrompt` golden via `DefaultRegularSections` stays intact
- [x] Project-instructions section still carries AGENTS.md; frame does not replace it

### Experimentation Safety
- [x] Frame is data; turning mode back to `regular` restores phase `regular`
- [x] No spawn side effects in this step

### Interaction Latency
- [x] Frame render is in-memory JSON; no file reads of sources
- [x] Compose path does not call retrieval

### Developer Feedback Speed
- [x] Unit tests fail on mode-mapping and body-leak without a spin binary
- [x] Size-cap test fails if defaults grow into AGENTS.md territory

### Team Scale
- [x] Frame JSON is the contract later children will consume
- [x] Mode names stay the four existing strings

### System Scale
- [x] Size cap plus path-only sources keep the frame from growing with the repo
- [x] New modes can extend `PhaseForMode` without changing Composer

### Right Behavior by Default
- [x] Sources never read file bodies
- [x] Frame is dynamic / non-cacheable

### Anti-Bypass Design
- [x] Body-like sources are filtered, not best-effort included
- [x] Tests assert the stable part cannot contain the frame

## 4. Tests

### TC-01: mode mapping review

**Given** mode `review`.
**When** `PhaseForMode` / `FromMode` / `CurrentFrame` after `/mode review`.
**Then** phase is `review`.

### TC-02: mode mapping table

**Given** modes `regular`, `compact`, `planning`.
**When** mapped.
**Then** phase equals the mode name.

### TC-03: empty and unknown mode

**Given** `""` or `"nope"`.
**When** `PhaseForMode`.
**Then** phase is `regular`.

### TC-04: stable marshal

**Given** a fixture frame.
**When** `MarshalStable` is called twice.
**Then** the byte slices are equal and JSON-valid.

### TC-05: no bodies in frame

**Given** a path source and a multi-line body string.
**When** `WithSources` + `MarshalStable`.
**Then** the path is present and the body sentinel is absent.

### TC-06: rendered size cap

**Given** `FromMode(regular)` with path sources (not bodies).
**When** `Render`.
**Then** `len(rendered) <= MaxRenderedBytes`.

### TC-07: dynamic only

**Given** `DefaultRegularSections` plus `ApplyTaskFrame`.
**When** `ComposeTwoPart`.
**Then** dynamic contains `# Task Frame` and the phase; stable contains neither.

### TC-08: parent compose includes frame

**Given** a conversation Builder.
**When** `composeSystemPrompt`.
**Then** output contains `# Task Frame` and `"phase":"regular"`.

### TC-09: AGENTS.md body not in frame section

**Given** workDir AGENTS.md with a unique sentinel.
**When** compose.
**Then** the TaskFrame suffix does not contain the sentinel (project-instructions may).

### TC-10: ApplyTaskFrame nil composer

**Given** a nil composer.
**When** `ApplyTaskFrame`.
**Then** no panic.

### TC-11: DefaultRegularSections golden

**Given** only `DefaultRegularSections`.
**When** `Compose`.
**Then** still equals `RegularSystemPrompt`.

## Acceptance Criteria

- [x] `TaskFrame` serializes to a stable JSON/text form for child spawn later
- [x] Composer dynamic part includes the frame; stable/cacheable part does not
- [x] `/mode review` yields phase `review` (and likewise for the other modes)
- [x] Sources are paths/queries, not file bodies
- [x] Unit tests for mode mapping and “no bodies in frame”
- [x] `make test` and `make lint` pass

## Traceability
- Roadmap item: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md) Step 14
- Implementation files: `internal/agent/frame/frame.go`, `internal/agent/prompt/sections.go`, `internal/conversation/conversation.go`, `internal/conversation/builder.go`, `cmd/spin/mode.go`, `cmd/spin/acp.go`
- Test files: `internal/agent/frame/frame_test.go`, `internal/agent/prompt/taskframe_test.go`, `internal/conversation/conversation_test.go`, `internal/conversation/builder_test.go`, `cmd/spin/mode_test.go`

## Implementation

Files created:
- `specs/journeys/JOURNEY-014-taskframe-on-every-parent-turn.md` — this journey
- `internal/agent/frame/frame.go` — `TaskFrame`, `FromMode`, `PhaseForMode`, `MarshalStable`, `Render`, `WithSources`
- `internal/agent/frame/frame_test.go` — mode mapping, stable JSON, no bodies, size cap
- `internal/agent/prompt/taskframe_test.go` — dynamic-only section, nil composer

Files modified:
- `internal/agent/prompt/sections.go` — `SectionTaskFrame`, `TaskFrameSection`, `ApplyTaskFrame` (`Cacheable: false`)
- `internal/conversation/conversation.go` — `CurrentFrame()` from `/mode`
- `internal/conversation/conversation_test.go` — review + all-mode phase mapping
- `internal/conversation/builder.go` — `ApplyTaskFrame` on parent compose
- `internal/conversation/builder_test.go` — compose includes frame; AGENTS.md body not in frame suffix
- `cmd/spin/acp.go` — same `ApplyTaskFrame` on ACP compose
- `cmd/spin/mode.go` — `spin mode describe` prints TaskFrame phase
- `cmd/spin/mode_test.go` — describe contains `TaskFrame phase: review`
- `docs/testing.md` — journey 014 row
- `specs/agent-harness/ROADMAP.md` — Step 14 DoD and traceability
