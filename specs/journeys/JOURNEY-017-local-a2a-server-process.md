# JOURNEY-017-local-a2a-server-process: Local A2A server process (`spin a2a`)

<!-- Journey for /implement. Slug is the topic — not a date. -->

## Roadmap Link
- Source roadmap: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md)
- Feature: Local A2A server process (`spin a2a`)

## 1. Journey

When **a parent harness (or an operator probing the child) starts a local A2A peer** I want **`spin a2a --spec explorer --stdio` to announce an Agent Card built from `subagent.Spec` and then answer `message/send` over NDJSON-RPC** so I **can drive an isolated child with its own TaskFrame and conversation, without copying the parent transcript and without HTTP**.

## 2. CJM

Alex already has A2A types and an in-memory client/server over a pipe (Step 16). Subagent builtins (`explorer`, `planner`, `reviewer`, `ask_user`) exist as `subagent.Spec` values. There is no `spin a2a` command, no child process entry, and no Agent Card derived from a Spec allowlist. Parent spawn (Step 18) still returns `ErrSubagentSpawnNotSupported`. This journey adds the **child-side server**: cobra `spin a2a`, card-from-Spec, isolated harness + TaskFrame, stdio binding, and `unix://` as the documented alternate. It does **not** replace the parent spawn stub and does **not** add `SUBAGENT_*` hook veto.

Assumption: Step 16 `internal/protocol/a2a` types, `Serve`, `Client`, and `MemoryHandler` are reused — no forked Card/Task/Message types. Assumption: integration tests may start the server in-process (`Serve` on pipes) and/or exec `build/bin/spin`; in-process is enough if the cobra command is wired and a test covers `--stdio` card+send. Assumption: child logs must not write to the RPC stdout stream (stderr or a file). Assumption: p99 < 200 ms card-received is a measured gate in an integration test, not an estimate.

### Phase 1: Discover and start the child

**User Intent:** Start a named builtin as a local A2A server without guessing flags.

**Actions:** Run `spin a2a --spec explorer --stdio` (or `--listen unix://…`). The process binds the stream and does not start TUI/exec.

**Pain / Risk:** Unknown spec name is a silent hang; `--stdio` and `--listen` both claimed; logs leak onto stdout and corrupt the RPC stream; cobra never registers `a2a` so help lies.

**Success Signal:** Command exists on the root cobra tree. Unknown spec fails fast. Default binding is stdio when `--listen` is unset. `--listen unix://…` is accepted or documented with a test.

### Phase 2: Read the Agent Card

**User Intent:** Learn who is on the other end before sending work.

**Actions:** Open the stream. Read the first framed NDJSON-RPC message. Inspect name, skills, capabilities, and supported interface.

**Pain / Risk:** First bytes are a log line or help text; card skills are hardcoded and ignore the Spec allowlist; capabilities advertise tools the Spec forbids; types are forked from `internal/protocol/a2a`.

**Success Signal:** First framed message is `agent/card` with an `AgentCard` whose name matches the Spec and whose skills/capabilities derive from `AllowedTools`. Client constructed via `a2a.NewClient` succeeds. Local spawn ready (card received) p99 < 200 ms.

### Phase 3: Send work into an isolated conversation

**User Intent:** Drive `message/send` against the child without leaking parent history.

**Actions:** Call `message/send` with a user text part. Observe the Task. Inspect the child's conversation and TaskFrame.

**Pain / Risk:** Child clones parent transcript; TaskFrame is missing or still the parent's; handler is a stub that never answers; stdout mixing breaks the client decode.

**Success Signal:** `message/send` returns a Task. Child history contains only messages received on this stream. Parent history is not copied. TaskFrame tools match the Spec allowlist.

### Phase 4: Bind the alternate transport

**User Intent:** Use a Unix socket when stdio is already the parent's TUI.

**Actions:** Pass `--listen unix://$path`. Connect as an A2A client. Read the card.

**Pain / Risk:** Scheme rejected silently; TCP/HTTP sneaks in; leftover socket file blocks the next start; docs claim unix without a test.

**Success Signal:** `unix://` is parsed and served, or the flag is documented as the alternate binding and a test proves the parser/server path.

### Friction and Opportunity

| Friction | Phase | Opportunity |
|----------|-------|-------------|
| No child CLI entry; only in-memory Step 16 pipes | 1 | `spin a2a` on the cobra root |
| Card identity invented in the parent | 2 | Card from `subagent.Spec` allowlist |
| Parent transcript cloned into the child | 3 | Isolated harness + TaskFrame only |
| Stdio logs corrupt NDJSON | 1 | Logs to stderr/file, never RPC stdout |
| Stdio occupied by TUI | 4 | `unix://` alternate binding |
| Spawn latency unknown | 2 | Measured p99 < 200 ms card-received gate |

### North Star Summary

An operator or a later parent (Step 18) can start `spin a2a --spec explorer --stdio`, receive an Agent Card whose skills match the explorer allowlist, send a message, and get a Task — while the child holds its own empty-then-local conversation and a TaskFrame, never the parent transcript. Unix socket is a tested alternate. Card announce is fast enough that local spawn-ready is a gate, not a hope.

### Stressors

1. First stdout bytes are slog/help text, so `a2a.NewClient` fails with `ErrUnexpectedCard`.
2. Card skills are a hardcoded list and ignore `Spec.AllowedTools` (explorer gains `write_file` or loses `read_file`).
3. Child constructor accepts parent `[]Message` and copies it into history.
4. `--listen unix://…` is documented but unparsed; the flag is a no-op.
5. Types are duplicated under `internal/agent/child` instead of reusing `internal/protocol/a2a`.
6. `spin a2a` is implemented but never registered on `newRootCmd`, so the binary has no subcommand.
7. Card-received p99 is asserted with a single sample or a wall-clock comment instead of a measured percentile.
8. Unknown `--spec` hangs waiting for stdin instead of failing.
9. Parent spawn stub is replaced in this step (scope leak into Step 18).
10. `SUBAGENT_START` veto is added here (scope leak into Step 19).
11. Unix listen leaves a stale socket and the next bind fails.
12. `message/send` is not wired; the process exits after printing the card.
13. TaskFrame tools are empty or copied from the parent session mode, not the Spec allowlist.
14. Integration test requires a live LLM and fails offline.

## 3. UX Implementation and Assessment

### Time to First Value
- [x] Card is the first framed message after process start (measured p99 < 200 ms)
- [x] `--spec explorer --stdio` is enough to serve; no extra config file

### Onboarding Clarity
- [x] `spin a2a --help` names `--spec`, `--stdio`, and `--listen`
- [x] Unknown spec returns an error that names the missing spec

### Production-Ready Defaults
- [x] Default binding is stdio when `--listen` is unset
- [x] Streaming/push stay off unless the Spec allowlist implies them

### Golden Path Quality
- [x] `--spec explorer --stdio` announces a card then answers `message/send`
- [x] Card name is `explorer`; skills come from the explorer allowlist

### Decision Load
- [x] Operator chooses a builtin spec name; binding defaults to stdio
- [x] Unix listen is opt-in via `--listen`

### Progressive Complexity
- [x] Stdio path needs two flags (`--spec`, `--stdio` or default)
- [x] Unix socket is the documented alternate, not required for the golden path

### Error Quality
- [x] Unknown spec fails with a named error
- [x] Unsupported listen scheme fails instead of binding TCP/HTTP

### Failure Safety
- [x] Child conversation starts empty; a bad send does not import parent history
- [x] No destructive workspace tools are advertised beyond the Spec allowlist

### Runtime Transparency
- [x] Agent Card declares name, skills, binding, and protocol version
- [x] TaskFrame on the child is inspectable (tools + phase from Spec)

### Debuggability
- [x] Logs go to stderr (or a file), not the RPC stdout stream
- [x] NDJSON frames remain JSON-RPC envelopes from line one

### Cross-Surface Consistency
- [x] Slash methods and types match Step 16 (`message/send`, `agent/card`)
- [x] Builtin names match `subagent` (`explorer`, `planner`, `reviewer`, `ask_user`)

### Workflow Consistency
- [x] Cobra registration follows `cmd/spin/root.go` (`AddCommand`)
- [x] Serve/Client stay in `internal/protocol/a2a`

### Change Safety
- [x] Parent spawn stub is untouched
- [x] Existing A2A pipe tests keep using `MemoryHandler` + `Serve`

### Experimentation Safety
- [x] In-process `Serve` on pipes lets tests avoid exec
- [x] Unix listen uses a temp path in tests

### Interaction Latency
- [x] Card-received p99 < 200 ms is a test gate
- [x] `message/send` is answered on the same stream after the card

### Developer Feedback Speed
- [x] Unknown spec fails before Serve
- [x] Client sees protocol errors from Step 16 codes

### Team Scale
- [x] Spec allowlist is the single source for card skills
- [x] Journey + roadmap traceability name the files

### System Scale
- [x] New builtins in `Builtins()` become cards via `Lookup` + `CardFromSpec`
- [x] Bindings (stdio vs unix) share one Server

### Right Behavior by Default
- [x] Isolated harness: empty history, TaskFrame from Spec, no parent copy
- [x] Stdio is the default RPC stream; logs are not

### Anti-Bypass Design
- [x] Card skills cannot advertise a tool absent from `AllowedTools`
- [x] Tests fail if parent history appears on the child

## 4. Tests

### TC-01: Lookup builtin spec

**Given** `subagent.Builtins()`.
**When** `Lookup("explorer")`.
**Then** the spec name is `explorer` and `AllowedTools` includes `read_file`.

### TC-02: Lookup unknown spec

**Given** no builtin named `nope`.
**When** `Lookup("nope")`.
**Then** an error is returned (not a nil spec with success).

### TC-03: Card name and description from Spec

**Given** the explorer builtin Spec.
**When** `CardFromSpec`.
**Then** card `Name` and `Description` match the Spec.

### TC-04: Card skills from allowlist

**Given** the explorer builtin Spec.
**When** `CardFromSpec`.
**Then** every skill id is in `AllowedTools` and every allowlisted tool is a skill; `write_file` is absent.

### TC-05: Card capabilities derive from allowlist

**Given** a Spec whose allowlist has no streaming/push tools.
**When** `CardFromSpec`.
**Then** `Capabilities.Streaming` and `PushNotifications` are false; `supportedInterfaces` uses `NDJSON-RPC`.

### TC-06: Isolated harness has TaskFrame, not parent history

**Given** a parent-looking message with a unique sentinel.
**When** `NewHarness` / `NewServer` is constructed from the Spec only.
**Then** child history is empty, the sentinel is absent, and TaskFrame `tools` equal the Spec allowlist.

### TC-07: Serve on pipes announces card then answers message/send

**Given** `Server.Serve` on `io.Pipe` or `net.Pipe` with spec `explorer`.
**When** `a2a.NewClient` then `SendMessage`.
**Then** the card name is `explorer` and the Task is non-empty / completed.

### TC-08: cobra `--spec explorer --stdio`

**Given** `newRootCmd` with stdin/stdout pipes.
**When** args are `a2a --spec explorer --stdio`.
**Then** the client reads a card and `message/send` succeeds.

### TC-09: `--listen unix://` accepted

**Given** a temp path.
**When** `--listen unix://$path` is parsed / `ListenAndServe` runs.
**Then** the scheme is accepted and a client on that socket receives the card (or the parser test plus help text documents the alternate binding).

### TC-10: card-received p99 < 200 ms

**Given** in-process `Serve` on pipes, N samples.
**When** each sample measures start→`NewClient` (card received).
**Then** p99 is strictly less than 200 ms.

### TC-11: child logs stay off RPC stdout

**Given** Serve writing to a capture buffer.
**When** the first line is read.
**Then** it is a JSON-RPC envelope (`jsonrpc`), not a log prefix.

### TC-12: unknown spec fails before serve

**Given** `spin a2a --spec nope --stdio`.
**When** the command runs.
**Then** it returns an error naming `nope` and writes no card.

### TC-13: planner/reviewer/ask_user cards

**Given** each remaining builtin name.
**When** `CardFromSpec` after `Lookup`.
**Then** card name matches and skill ids ⊆ allowlist.

## Acceptance Criteria

- [x] `spin a2a --spec explorer --stdio` prints/serves a card then answers `message/send`
- [x] Child process has its own conversation; parent history is not copied
- [x] Card skills/capabilities derive from the Spec allowlist
- [x] `--listen unix://…` accepted or explicitly documented as the alternate binding with a test
- [x] Local spawn ready (card received) p99 < 200 ms in an integration test
- [x] `make test` and `make lint` pass

## Traceability
- Roadmap item: [specs/agent-harness/ROADMAP.md](../agent-harness/ROADMAP.md) Step 17
- Implementation files: `cmd/spin/a2a.go`, `cmd/spin/root.go`, `internal/agent/child/card.go`, `internal/agent/child/harness.go`, `internal/agent/child/server.go`, `internal/agent/child/listen.go`, `internal/agent/subagent/builtins.go`
- Test files: `cmd/spin/a2a_test.go`, `internal/agent/child/card_test.go`, `internal/agent/child/harness_test.go`, `internal/agent/child/server_test.go`, `internal/agent/child/listen_test.go`, `internal/agent/child/p99_test.go`, `internal/agent/subagent/lookup_test.go`

## Implementation

Files created:
- `specs/journeys/JOURNEY-017-local-a2a-server-process.md` — this journey
- `cmd/spin/a2a.go` — `spin a2a` cobra command (`--spec`, `--stdio`, `--listen`)
- `cmd/spin/a2a_test.go` — registration, unix:// help, `--stdio` card+send, unknown spec
- `internal/agent/child/card.go` — `CardFromSpec` from Spec allowlist
- `internal/agent/child/card_test.go` — name/description/skills/capabilities and all builtins
- `internal/agent/child/harness.go` — isolated TaskFrame + empty history (no parent copy)
- `internal/agent/child/harness_test.go` — empty history, tools from allowlist, parent sentinel absent
- `internal/agent/child/server.go` — `NewServer` / `Serve` over Step 16 `a2a.Serve`
- `internal/agent/child/server_test.go` — card then `message/send`, own conversation
- `internal/agent/child/listen.go` — `unix://` parse and `ListenAndServe`
- `internal/agent/child/listen_test.go` — parse + socket card
- `internal/agent/child/p99_test.go` — card-received p99 < 200 ms
- `internal/agent/subagent/lookup_test.go` — `Lookup` explorer / unknown

Files modified:
- `internal/agent/subagent/builtins.go` — `Lookup` by builtin name
- `cmd/spin/root.go` — register `newA2ACmd`
- `docs/testing.md` — journey 017 row
- `specs/agent-harness/ROADMAP.md` — Step 17 DoD and traceability

Deviation: the child answers `message/send` via Step 16 `MemoryHandler` (echo Task) on an isolated harness. A full LLM ReAct loop is not started here — parent process spawn is Step 18. In-process `Serve` on pipes is the integration path; cobra `--stdio` is covered without exec of `build/bin/spin`.
