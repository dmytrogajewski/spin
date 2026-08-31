# JOURNEY-015-assemble-retrieval-on-the-turn-path: Assemble retrieval on the turn path

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: Assemble retrieval on the turn path

## 1. Journey

When **Alex runs a parent spin turn (TUI, exec, or ACP) with ACE or other retrieval sources configured** I want **`retrieval.Pipeline.Assemble` to run on that turn so assembled fragments enter the model’s context** so I **can rely on ACE bullets and other sources actually appearing in the prompt, without rebuilding the pipeline or breaking ACP/TUI sessions that have no pipeline**.

## 2. CJM

Alex already has a retrieval pipeline (`internal/contexteng/retrieval`) and a Conversation getter (`GetRetrievalPipeline()`). Builder wires `NewBulletSource()` onto the Conversation. The harness ReAct loop, ACE middleware, and Composer (including Step 14 TaskFrame) are live. **Assemble is never called on the production turn path.** ACE middleware may retrieve bullets into `TrajectoryCtx`, but the bridge caller passes `nil` bullets to `ApplyACEPrompt`. ACP/`NewFromAgent` conversations have a nil pipeline. This journey adds **one** Assemble call site on the harness turn, driven from Conversation via `GetRetrievalPipeline()`, injects fragments into the observation/message path, and treats a nil pipeline as a no-op. It does **not** rebuild the pipeline and does **not** start A2A (Step 16).

Assumption (SPEC vs DoD): SPEC lists Assemble fragments as Composer item 5. DoD accepts **dynamic prompt or observation path**. This journey injects formatted fragments as a turn message (observation path) after `BeforeTurn` so `TrajectoryCtx` is populated for `BulletSource`. Frame `sources` stay path/query strings (Step 14); `retrieval.Request` already has `Query`/`Turn`/`TrajectoryCtx` only — this journey does not add a Sources field.

### Phase 1: Bind the existing pipeline on the parent turn

**User Intent:** A production turn uses the Conversation’s retrieval pipeline (the getter is no longer dead).

**Actions:** Start a TUI/exec session (Builder sets a pipeline) or an ACP session (`NewFromAgent`, pipeline nil). Send a user prompt. The turn path calls `GetRetrievalPipeline()`.

**Pain / Risk:** Getter stays unused and deadcode/DoD fail; ACP panics on nil; Builder and harness hold two different pipeline instances so Assemble never sees ACE sources; a second Assemble site doubles fragments.

**Success Signal:** `RunTurn` calls `GetRetrievalPipeline()`. Builder-created conversations bind a non-nil pipeline. ACP/TUI without ACE still complete the turn.

### Phase 2: Assemble once and put fragments in the turn

**User Intent:** Assembled fragment text is visible to the model on that turn.

**Actions:** Register a fake source that returns a unique sentinel. Run a parent turn. Inspect the messages the caller receives (composed turn).

**Pain / Risk:** Assemble runs but the result is discarded; fragments land only in a log; Assemble runs every ReAct iteration and repeats the same block; ACE middleware and Assemble both inject the same bullet.

**Success Signal:** The sentinel appears in the composed turn messages. Token/count of that sentinel is 1. Empty Assemble result adds no message.

### Phase 3: Survive a missing pipeline

**User Intent:** Sessions without ACE or retrieval still run.

**Actions:** Build via `NewFromAgent` with no pipeline. Run a turn. Repeat with a harness that never received `BindRetrieval`.

**Pain / Risk:** Nil dereference in Assemble or bind; TUI without ACE fails; tests that use stub executors (no binder) panic.

**Success Signal:** Nil pipeline is a no-op. Stub `HarnessTurnExecutor` values that do not bind still run.

### Phase 4: Keep the existing pipeline the only implementation

**User Intent:** ACE and other sources enter context through the package that already exists.

**Actions:** Use `retrieval.NewPipeline` + `Source`. Do not add a parallel retriever. Do not change Assemble’s skip-on-error contract.

**Pain / Risk:** A new pipeline type appears; Request grows a Sources field that forks the API; BulletSource is replaced instead of called.

**Success Signal:** One `Assemble` call site. Existing `pipeline_test.go` / `pipeline_integration_test.go` still pass. No new retrieval implementation.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| Pipeline exists but Assemble never runs | 1–2 | One call site on the harness turn |
| `GetRetrievalPipeline()` is unused | 1 | Bind from `RunTurn` |
| ACE bullets stay in TrajectoryCtx only | 2 | Assemble after BeforeTurn |
| ACP/TUI without ACE would crash | 3 | Nil pipeline no-op |
| Double-injection with ACE middleware | 2 | Dedup by fragment content; one Assemble |
| Rebuilding retrieval | 4 | Call the existing Pipeline |

### North Star Summary

Every parent turn that has a retrieval pipeline calls `Assemble` once (after middleware `BeforeTurn` on ReAct turn 0) and the formatted fragments appear in the messages the model sees. A nil pipeline changes nothing. The same fragment is not injected twice. The existing `retrieval.Pipeline` is the only assembler.

### Stressors

1. `GetRetrievalPipeline()` remains uncalled on the production turn path (DoD miss / deadcode).
2. ACP `NewFromAgent` conversations have a nil pipeline and must not panic.
3. Stub executors used in conversation tests do not implement bind and must still `RunTurn`.
4. Assemble after `BeforeTurn` so `BulletSource` can read `TrajectoryCtx`; Assemble before it yields empty ACE fragments.
5. Assemble on every ReAct iteration would duplicate the same fragment (token-count risk).
6. ACE middleware `ApplyACEPrompt` and Assemble both injecting the same bullet (double-injection).
7. Empty fragment list must not append a blank retrieved-context message.
8. A source error must not abort the turn (existing Assemble skip-on-error).
9. Fake source sentinel must appear in the composed turn, not only in a unit-level Assemble result.
10. Builder default `NewBulletSource()` with no trajectory bullets must stay a no-op (no empty heading).
11. Two `RunTurn` calls must not accumulate duplicate sentinels in a way the token-count test rejects.
12. Step 16 A2A types / child spawn must not start from this journey.
13. `retrieval.Request` must not grow a new Sources API (do not rebuild the pipeline).
14. Frame `sources` are paths/queries (Step 14); they must not be expanded to file bodies here.

## 3. UX Implementation and Assessment

### Time to First Value
- [x] Parent turn with a pipeline assembles without a new slash command
- [x] No extra onboarding step to “enable retrieval” when Builder already wired a pipeline

### Onboarding Clarity
- [x] Feature is not a user-facing command; it is on the existing turn path
- [x] Missing pipeline is silent (no error toast required)

### Production-Ready Defaults
- [x] Builder keeps `NewPipeline(NewBulletSource())`
- [x] Nil pipeline (ACP without ACE) is a no-op

### Golden Path Quality
- [x] Fake source fragment is present in the composed turn
- [x] Assemble output is not discarded

### Decision Load
- [x] Operator does not choose a second retrieval implementation
- [x] No new flags to run Assemble

### Progressive Complexity
- [x] Sessions without sources stay unchanged
- [x] Additional `Source` values are opt-in on the existing Pipeline

### Error Quality
- [x] Source errors stay skip-and-continue (existing Assemble)
- [x] Nil pipeline does not surface a user-facing error

### Failure Safety
- [x] Nil pipeline is a no-op
- [x] Missing binder on a stub executor is a no-op

### Runtime Transparency
- [x] Retrieved text is visible in turn messages (`# Retrieved Context`)
- [x] Empty Assemble adds no hidden blank block

### Debuggability
- [x] Sentinel tests prove the fragment reached the caller messages
- [x] Token-count test fails if the same fragment is injected twice

### Cross-Surface Consistency
- [x] TUI/exec Builder conversations bind the pipeline on `RunTurn`
- [x] ACP `NewFromAgent` nil pipeline still runs

### Workflow Consistency
- [x] One Assemble call site; existing Pipeline type
- [x] Journey comment on new tests points at this file

### Change Safety
- [x] Existing retrieval unit/integration tests remain the Assemble contract
- [x] TaskFrame compose path is unchanged

### Experimentation Safety
- [x] Disabling ACE / omitting the pipeline restores the previous turn
- [x] No A2A spawn side effects

### Interaction Latency
- [x] Assemble runs in-process against registered sources
- [x] No extra model call to retrieve fragments

### Developer Feedback Speed
- [x] Integration test fails if the sentinel is missing from the turn
- [x] Dedup test fails if the same fragment is counted twice

### Team Scale
- [x] Sources remain `retrieval.Source` implementations
- [x] No parallel retriever for teams to choose between

### System Scale
- [x] New sources register on the existing Pipeline
- [x] One injection site keeps token growth linear in fragment count

### Right Behavior by Default
- [x] Nil pipeline does not change ACP/TUI without ACE
- [x] Empty fragments do not add a heading-only message

### Anti-Bypass Design
- [x] Production turn path must call `GetRetrievalPipeline()`
- [x] Tests assert composed-turn presence, not only Assemble return value

## 4. Tests

### TC-01: nil pipeline no-op

**Given** a Conversation with `retrievalPipeline == nil` (NewFromAgent / ACP).
**When** `RunTurn` with a stub executor.
**Then** the turn completes; no panic; no retrieved-context message.

### TC-02: GetRetrievalPipeline on the turn path

**Given** a Conversation with a non-nil pipeline.
**When** `RunTurn`.
**Then** `GetRetrievalPipeline()` is invoked (binder receives that pointer).

### TC-03: assembled fragments in the composed turn

**Given** a pipeline whose fake source returns a unique sentinel.
**When** `RunTurn` through a real harness caller.
**Then** the caller’s messages contain the sentinel.

### TC-04: empty Assemble adds no message

**Given** a pipeline whose source returns no fragments.
**When** the harness executes.
**Then** messages do not contain `# Retrieved Context`.

### TC-05: nil pipeline on the harness is a no-op

**Given** an Executor with no retrieval pipeline.
**When** `Execute`.
**Then** the loop completes (same as today).

### TC-06: fragment not duplicated

**Given** the same sentinel already in history (ACE-middleware-shaped injection) and a fake source that returns it again.
**When** Assemble injects.
**Then** `strings.Count` of the sentinel across the composed turn is 1.

### TC-07: Assemble once per Execute

**Given** a multi-turn ReAct loop (tool call then stop).
**When** Execute with a counting source.
**Then** `Retrieve` is called once (turn 0 only).

### TC-08: source error does not fail the turn

**Given** a source that returns an error (existing Assemble contract).
**When** the turn runs.
**Then** the turn succeeds and other sources can still contribute.

### TC-09: stub executor without binder

**Given** `harnessExecutor` is a stub that does not implement bind.
**When** `RunTurn`.
**Then** no panic (bind is skipped).

### TC-10: default BulletSource without trajectory bullets

**Given** Builder default pipeline (`NewBulletSource()` only).
**When** a turn with empty `TrajectoryCtx` bullets.
**Then** no retrieved-context heading is injected.

## Acceptance Criteria

- [x] `GetRetrievalPipeline()` is no longer unused on the production turn path
- [x] Assembled fragments appear in the dynamic prompt or observation path
- [x] Nil pipeline is a no-op (ACP/TUI without ACE still run)
- [x] Integration test with a fake source proves Assemble output is present in the composed turn
- [x] `make test` and `make lint` pass

## Traceability
- Roadmap item: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md) Step 15
- Implementation files: `internal/agent/harness/loop.go`, `internal/agent/harness/executor.go`, `internal/conversation/conversation.go`, `internal/agent/harness/bridge/turn_executor.go`
- Test files: `internal/agent/harness/retrieval_test.go`, `internal/conversation/retrieval_turn_integration_test.go`

## Implementation

Files created:
- `specs/journeys/JOURNEY-015-assemble-retrieval-on-the-turn-path.md` — this journey
- `internal/agent/harness/retrieval_test.go` — Assemble inject, nil no-op, empty skip, dedup, once-per-Execute
- `internal/conversation/retrieval_turn_integration_test.go` — fake source in composed turn; nil pipeline no-op

Files modified:
- `internal/agent/harness/loop.go` — `phaseRetrieval` after `BeforeTurn` (turn 0 only)
- `internal/agent/harness/executor.go` — `WithRetrievalPipeline`, `SetRetrievalPipeline`
- `internal/agent/harness/bridge/turn_executor.go` — `BindRetrieval`
- `internal/conversation/conversation.go` — `RunTurn` calls `GetRetrievalPipeline()` and binds
- `docs/testing.md` — journey 015 row
- `specs/agent-harness/ROADMAP.md` — Step 15 DoD and traceability
